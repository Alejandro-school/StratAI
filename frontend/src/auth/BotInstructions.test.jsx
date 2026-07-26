import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import BotInstructions from "./BotInstructions";

vi.mock("./useAuth", () => ({
  useAuth: () => ({
    user: { authenticated: true, steam_id: "76561198000000000" },
  }),
}));

const response = (body) => Promise.resolve({
  ok: true,
  json: () => Promise.resolve(body),
});

function renderFlow() {
  return render(
    <MemoryRouter initialEntries={["/bot-instructions"]}>
      <Routes>
        <Route path="/bot-instructions" element={<BotInstructions />} />
        <Route path="/dashboard" element={<div>Dashboard listo</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("Bot friendship flow", () => {
  it("redirects an existing friend to the dashboard", async () => {
    vi.stubGlobal("fetch", vi.fn(() => response({
      is_friend: true,
      status: "friend",
      bot_steam_id: "76561198000000099",
      service_down: false,
      source: "live",
    })));

    renderFlow();

    expect(await screen.findByText("Dashboard listo")).toBeInTheDocument();
  });

  it("sends an authenticated empty POST and exposes the bot Steam ID", async () => {
    const fetchMock = vi.fn((_url, options = {}) => {
      if (options.method === "POST") {
        return response({
          is_friend: false,
          status: "pending",
          bot_steam_id: "76561198000000099",
          message: "Solicitud de amistad enviada",
        });
      }
      return response({
        is_friend: false,
        status: "not_friend",
        bot_steam_id: "76561198000000099",
        service_down: false,
        source: "live",
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    renderFlow();

    const sendButton = await screen.findByRole("button", {
      name: "Enviar solicitud de amistad",
    });
    await waitFor(() => expect(sendButton).toBeEnabled());
    fireEvent.click(sendButton);

    await screen.findByText("Solicitud de amistad enviada");
    const postCall = fetchMock.mock.calls.find(([, options]) => options?.method === "POST");
    expect(postCall[1]).toEqual({
      method: "POST",
      credentials: "include",
    });
    expect(screen.getByText("76561198000000099")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Abrir en Steam" })).toBeEnabled();
  });
});
