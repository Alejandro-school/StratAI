import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import Replay2DViewerV2 from "./Replay2DViewerV2";
import catalog from '../../../../../backend/go-service/pkg/sound/cs2_profiles.json';
import {SOUND_CATALOG_SHA256} from '../domain/replaySoundInformation';

const mocks = vi.hoisted(() => ({ canvas: vi.fn(), setFocus: vi.fn(), setDirector: vi.fn(), sound:null }));
vi.mock("../hooks/useReplayAnnotations", () => ({ useReplayAnnotations: () => ({ annotations: [
  { round: 1, start_tick: 0, end_tick: 99, text: "Hidden enemy annotation" },
], create: vi.fn(), update: vi.fn(), remove: vi.fn() }) }));
vi.mock("./ReplayCanvas", () => ({ ReplayCanvas: (props) => { mocks.canvas(props); return <div>Mapa</div>; } }));
vi.mock("../hooks/useReplayController", () => ({ useReplayController: () => {
  const frames = [{ tick: 10, time_remaining: 80, players: [
    { steam_id: "1", name: "Aliado", team: "T", alive: true, x: 10, y: 10, z: 0 },
    { steam_id: "2", name: "Rival oculto", team: "CT", alive: true, x: 800, y: 800, z: 0, money: 7000 },
  ], shots: [{ shooter_id: "2" }] }];
  return { metadata: { map_name: "de_mirage" }, frames, currentFrame: frames[0], tick: 20,
    roundData: { round: 1, start_tick:10, sound_information:mocks.sound, frames, events: [{ id: "death", tick: 15, type: "kill", killer_name: "Atacante secreto",
      killer_id: "2", killer_x: 700, killer_y: 700, victim_id: "3", victim_team: "CT", victim_name: "CT muerto", victim_x: 10, victim_y: 11 }] },
    events: [], roundsSummary: [{ round: 1 }], roundIndex: 1, actualRound: 1, tickRate: 64, startTick: 10, endTick: 99,
    progress: .1, focusPlayerId: "2", aiAnnotations: [{ text: "Hidden AI annotation" }],
    setFocusPlayerId: mocks.setFocus, setDirectorMode: mocks.setDirector, setIsPlaying: vi.fn(), togglePlayback: vi.fn(),
    changeRound: vi.fn(), seekProgress: vi.fn(), seekEvent: vi.fn(), seekBySeconds: vi.fn(),
    setPlaybackRate: vi.fn(), playbackRate: 1, effectiveRate: 1 };
} }));

describe("professional team replay integration", () => {
  it('passes demo sound points separately to the map while keeping the factual callback free of hidden coordinates',()=>{
    const p=catalog.events.find(p=>p.event==='Base.Footstep');
    mocks.sound={schema_id:'stratai.calculated_sound_information@1',producer:'go.replay.native_sound_profile_3d_v1',
      catalog_id:catalog.catalog_id,catalog_sha256:SOUND_CATALOG_SHA256,
      emissions:[{id:'sound-emission:1',tick:15,availability_tick:15,profile:p.event,family:p.family,
        source:'net.CMsgSosStartSoundEvent',native_sound_hash:p.hash,native_position:{X:0,Y:0,Z:100},
        emitter_id:'2',emitter_side:'CT',radius_units:p.radius_units,radius_basis:p.radius_basis,
        native_observers:[{player_id:'1',side:'T',alive:true,native_position:{X:0,Y:0,Z:0}},
          {player_id:'2',side:'CT',alive:true,native_position:{X:20,Y:30,Z:110}}]}],
      observations:[{id:'sound-observation:1',emission_id:'sound-emission:1',observer_id:'1',observer_side:'T',tick:15,
        availability_tick:15,distance_3d_units:100,status:'estimated_possible_reception'}]};
    const onContext=vi.fn();
    try {
      render(<Replay2DViewerV2 initialPerspective="T" onPerspectiveContext={onContext}/>);
      expect(mocks.canvas.mock.calls.at(-1)[0].soundReviewMarkers[0]).toMatchObject({x:20,y:30,z:110});
      const raw=JSON.stringify(onContext.mock.calls.at(-1)[0]);
      expect(raw).not.toContain('demo_emitter_at_sound_tick');
      expect(raw).not.toContain('soundReviewMarkers');
      expect(screen.getByText(/Sonido · punto de la demo/)).toBeInTheDocument();
    } finally {mocks.sound=null;}
  });
  it("sends the same filtered context to canvas and caller and hides rival panels/annotations", () => {
    const onContext = vi.fn(), onPosition = vi.fn(); render(<Replay2DViewerV2 initialPerspective="T" onPerspectiveContext={onContext} onPositionChange={onPosition}
      scenarioContext={{ title: "Lectura con contexto ajeno", description: "Atacante secreto" }} />);
    expect(screen.queryByText("Rival oculto")).not.toBeInTheDocument();
    expect(screen.queryByText("Atacante secreto")).not.toBeInTheDocument();
    expect(screen.queryByText("Lectura con contexto ajeno")).not.toBeInTheDocument();
    expect(screen.getByText("CT muerto")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Contactos no disponibles");
    const canvas = mocks.canvas.mock.calls.at(-1)[0];
    expect(canvas.frame.shots).toEqual([]); expect(canvas.annotations).toEqual([]); expect(canvas.focusPlayerId).toBeNull();
    expect(canvas.frame.death_markers[0].position).toEqual({ x: 10, y: 11 });
    expect(onContext.mock.calls.at(-1)[0].frame).toEqual(canvas.frame);
    expect(screen.getByRole("button", { name: "Director" })).toBeDisabled();
    fireEvent.change(screen.getByRole("combobox", { name: "Perspectiva del replay" }), { target: { value: "observer" } });
    expect(onPosition).toHaveBeenLastCalledWith({ round: 1, tick: 20, side: "observer" });
  });
  it("lets the reviewer select CT and clears hidden focus/director", () => {
    render(<Replay2DViewerV2 initialPerspective="T" />);
    fireEvent.change(screen.getByRole("combobox", { name: "Perspectiva del replay" }), { target: { value: "CT" } });
    expect(screen.queryByText("Aliado")).not.toBeInTheDocument();
    expect(screen.getByText("Rival oculto")).toBeInTheDocument();
    expect(mocks.setFocus).toHaveBeenCalledWith(null);
    expect(mocks.setDirector).toHaveBeenCalledWith(false);
  });
  it("labels the visible map and roster without altering factual callbacks or revealing hidden opponents", () => {
    const onContext = vi.fn();
    render(<Replay2DViewerV2 initialPerspective="T" playerLabels={{1:"T1",2:"CT1",3:"CT2"}} onPerspectiveContext={onContext} />);
    const canvas = mocks.canvas.mock.calls.at(-1)[0];
    expect(canvas.frame.players.map(player=>player.name)).toEqual(["T1"]);
    expect(canvas.frame.death_markers[0].name).toBe("CT2");
    expect(screen.getByText("CT2")).toBeInTheDocument();
    expect(screen.queryByText("CT1")).not.toBeInTheDocument();
    expect(onContext.mock.calls.at(-1)[0].frame.players[0].name).toBe("Aliado");
  });
});
