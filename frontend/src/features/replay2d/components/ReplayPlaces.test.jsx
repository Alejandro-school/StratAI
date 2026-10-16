import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { ReplayRosterPanel } from "./ReplayRosterPanel";

it("exposes the qualified place in visible text and accessible name, but not for a corpse", () => {
  const player = { steam_id: "1", team: "T", name: "T1", alive: true, health: 100,
    place: { status: "mapped", zone: "Apartments", candidates: [] } };
  const { rerender } = render(<ReplayRosterPanel players={[player]} onSelect={() => {}} />);
  expect(screen.getByText("Apartamentos B · mapa de referencia")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /T1.*Apartamentos B · mapa de referencia/ })).toBeInTheDocument();
  rerender(<ReplayRosterPanel players={[{ ...player, alive: false }]} onSelect={() => {}} />);
  expect(screen.queryByText(/Apartamentos/)).not.toBeInTheDocument();
});
