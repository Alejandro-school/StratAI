import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReplayCoachCase } from "./ReplayCoachCase";

const canvas = vi.hoisted(() => vi.fn());
vi.mock("./ReplayCanvas", () => ({ ReplayCanvas: (props) => { canvas(props); return <div>Mapa</div>; } }));
vi.mock("../hooks/useReplayAnnotations", () => ({ useReplayAnnotations: () => ({ annotations: [], create: vi.fn(), update: vi.fn(), remove: vi.fn() }) }));
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); window.history.replaceState(null,"","/"); });

it("plays through the analysis cut without reloading on position updates, pauses, seeks and restarts at the end", async () => {
  let scheduled=null;
  vi.stubGlobal("requestAnimationFrame",vi.fn(callback=>{scheduled=callback;return 1;}));
  vi.stubGlobal("cancelAnimationFrame",vi.fn(()=>{scheduled=null;}));
  vi.spyOn(performance,"now").mockReturnValue(0);
  const frames=[100,164,228,292].map(tick=>({tick,time_remaining:115-(tick-100)/64,
    players:[{steam_id:"1",name:"P01",team:"T",alive:true,health:100,x:tick,y:10,z:0,yaw:0},
      {steam_id:"2",name:"P02",team:"CT",alive:true,health:100,x:800,y:800,z:0}]}));
  const data={metadata:{map_name:"de_mirage",tick_rate:64},rounds:[{round:1,start_tick:100,end_tick:292,frames,events:[]}]};
  const bundle={case_id:"N01",round:1,start_tick:100,replay_cut_tick:164,cut_seconds:1,input_mode:"enriched_replay",default_variant:"C",
    context_sha256:"a".repeat(64),source_html_sha256:"b".repeat(64),variants:[{code:"C",reading:{situation:[{text:"P01 sigue vivo.",kind:"observed"}]}}]};
  render(<QueryClientProvider client={new QueryClient()}><ReplayCoachCase data={data} bundle={bundle}/></QueryClientProvider>);
  await waitFor(()=>expect(canvas.mock.calls.at(-1)[0].tick).toBe(164));
  fireEvent.click(screen.getByRole("button",{name:"Reproducir ronda desde el inicio"}));
  expect(canvas.mock.calls.at(-1)[0].tick).toBe(100);
  for(let time=250;time<=1500;time+=250)act(()=>{const next=scheduled;scheduled=null;next(time);});
  expect(canvas.mock.calls.at(-1)[0].tick).toBe(196);
  expect(screen.getByRole("button",{name:"Pausar"})).toBeInTheDocument();
  expect(screen.getByText(/La IA no dispuso de ella/)).toBeInTheDocument();
  expect(canvas.mock.calls.at(-1)[0].frame.players.map(player=>player.name)).toEqual(["T1"]);
  fireEvent.click(screen.getByRole("button",{name:"Pausar"}));
  expect(scheduled).toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"Ir al momento analizado"}));
  expect(canvas.mock.calls.at(-1)[0].tick).toBe(164);
  fireEvent.click(screen.getByRole("button",{name:"Reproducir"}));
  for(let time=250;time<=2000;time+=250)act(()=>{const next=scheduled;scheduled=null;next(time);});
  expect(canvas.mock.calls.at(-1)[0].tick).toBe(292);
  expect(screen.getByRole("button",{name:"Reproducir"})).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:"Reproducir"}));
  expect(canvas.mock.calls.at(-1)[0].tick).toBe(100);
});
