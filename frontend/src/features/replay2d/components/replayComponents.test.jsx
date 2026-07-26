import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { normalizeReplayEvent } from "../domain/replayModel";
import { ReplayRosterPanel } from "./ReplayRosterPanel";
import { ReplayTimeline } from "./ReplayTimeline";

describe("ReplayTimeline", () => {
  it("renders semantic lanes and seeks to an exact event", () => {
    const onSeekEvent = vi.fn();
    const kill = normalizeReplayEvent({
      type: "kill",
      tick: 140,
      killer_name: "Coach",
      victim_name: "Rival",
      weapon: "AK-47",
    });
    render(
      <ReplayTimeline
        events={[kill]}
        frames={[{ tick: 100, time_remaining: 90 }, { tick: 200, time_remaining: 88 }]}
        startTick={100}
        endTick={200}
        tick={120}
        progress={0.2}
        activeClip={null}
        onSeekProgress={vi.fn()}
        onSeekEvent={onSeekEvent}
      />,
    );
    expect(screen.getByText("Combate")).toBeInTheDocument();
    expect(screen.getByText("Utilidad")).toBeInTheDocument();
    expect(screen.getByText("Objetivo")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Ir a Coach → Rival/ }));
    expect(onSeekEvent).toHaveBeenCalledWith(kill);
  });
});

describe("ReplayRosterPanel", () => {
  it("keeps both teams visible and exposes player selection", () => {
    const onSelect = vi.fn();
    const players = [
      { steam_id: "1", name: "Analista", team: "CT", alive: true, health: 87, armor: 50, weapon: "M4A1-S", money: 3200 },
      { steam_id: "2", name: "Rival", team: "T", alive: false, health: 0, armor: 0, weapon: "AK-47", money: 900 },
    ];
    render(<ReplayRosterPanel players={players} ctScore={5} tScore={4} focusPlayerId={null} onSelect={onSelect} />);
    fireEvent.click(screen.getByRole("button", { name: /Analista, 87 de vida/ }));
    expect(onSelect).toHaveBeenCalledWith("1");
    expect(screen.getByText("Counter-Terrorists")).toBeInTheDocument();
    expect(screen.getByText("Terrorists")).toBeInTheDocument();
  });

  it("renders complete official equipment, repeated utility, money, C4 and kit", () => {
    const player = {
      steam_id: "5",
      name: "Kerchak",
      team: "CT",
      alive: true,
      health: 100,
      armor: 100,
      weapon: "MP7",
      weapons: ["MP7", "Five-SeveN", "Flashbang", "Flashbang", "Smoke Grenade", "HE Grenade", "Knife"],
      money: 5300,
      has_c4: true,
      has_defuse_kit: true,
      has_helmet: true,
    };
    const { container } = render(
      <ReplayRosterPanel players={[player]} ctScore={3} tScore={1} focusPlayerId={null} onSelect={vi.fn()} />,
    );

    expect(screen.getByText("$5300")).toBeInTheDocument();
    expect(screen.getByText("KIT")).toBeInTheDocument();
    expect(container.querySelectorAll('img[src="/images/cs2/equipment/flashbang.svg"]')).toHaveLength(2);
    expect(container.querySelector('img[src="/images/cs2/equipment/c4.svg"]')).toBeInTheDocument();
    expect(container.querySelector('img[src="/images/cs2/equipment/defuser.svg"]')).toBeInTheDocument();
    expect(container.querySelector('img[src="/images/cs2/equipment/mp7.svg"]')).toHaveClass("active");
    expect(container.querySelector('img[src="/images/cs2/equipment/fiveseven.svg"]')).not.toHaveClass("active");
  });
});
