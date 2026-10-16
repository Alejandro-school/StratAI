import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReplayHeader } from "./ReplayHeader";
import { drawBomb } from "../renderer/renderReplayCombat";
import { createViewport } from "../renderer/replayViewport";

describe("C4 presentation", () => {
  it("reports known state without requiring a map position", () => {
    render(<ReplayHeader roundIndex={3} rounds={[1,2,3]} ctScore={0} tScore={0} onRoundChange={() => {}}
      frame={{ bomb: { state:"dropped", position_status:"unavailable" }, time_remaining:70 }} />);
    expect(screen.getByText(/C4 caído/)).toHaveTextContent("Ubicación desconocida");
  });
  it("does not call a resolved planted round live or postplant", () => {
    render(<ReplayHeader roundIndex={3} rounds={[1,2,3]} ctScore={0} tScore={0} onRoundChange={() => {}}
      frame={{ bomb: { state:"resolved", objective_phase:"resolved", was_planted_this_round:true }, time_remaining:0 }} />);
    expect(screen.getByText("FINALIZADA")).toBeInTheDocument();
    expect(screen.getByText(/C4.*resuelto/)).toHaveTextContent("Hubo plantado");
  });
  it("labels a between-sample marker as a dated observation rather than current location", () => {
    const context=new Proxy({}, {get:(target,key) => target[key] ||= vi.fn()});
    drawBomb(context,{state:"dropped",last_observed_position:{x:100,y:200,observation_tick:100}},
      {pos_x:0,pos_y:0,scale:1},createViewport(500,500),102,64);
    expect(context.fillText).toHaveBeenCalledWith("C4 · hace 0.03 s",expect.any(Number),expect.any(Number));
  });
  it("does not plot producer placeholder zeros when the current position is unavailable", () => {
    const context={beginPath:vi.fn()};
    drawBomb(context,{state:"dropped",position_status:"unavailable",x:0,y:0},
      {pos_x:0,pos_y:0,scale:1},createViewport(500,500),102,64);
    expect(context.beginPath).not.toHaveBeenCalled();
  });
});
