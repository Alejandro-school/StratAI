import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

// ───────────── Rutas públicas ──────────────
import LandingPage from "./components/Landing/LandingPage";
import SteamLoginSuccess from "./auth/SteamLoginSuccess";

// ───────────── Rutas privadas ──────────────
import Dashboard from "./components/Start/Dashboard";
import BotInstructions from "./auth/BotInstructions";
import HistoryCodeForm from "./auth/HistoryCodeForm";
import HistoryGames from "./components/Stats/HistoryGames";
import PersonalPerformance from "./components/Stats/PersonalPerformance";
import MatchDetails from "./components/Stats/MatchDetails";

// ───────────── Contextos & Auth ──────────────
import { AuthProvider } from "./auth/useAuth";
import { UserProvider } from "./context/UserContext";
import RequireAuth from "./auth/RequireAuth";

/**
 * ▸ En la landing ("/"), la app se renderiza **sin** comprobar sesión
 *   → no dispara /auth/steam/status y evita los Failed‑to‑fetch.
 * ▸ Las rutas protegidas se agrupan bajo un layout que **sí** envuelve
 *   de AuthProvider + UserProvider + RequireAuth.
 */
const App = () => (
  <BrowserRouter>
    <Routes>
      {/* -------------- PÚBLICAS -------------- */}
      <Route path="/" element={<LandingPage />} />

      {/* Callback post‑Steam OAuth: necesita contexto pero no auth previa */}
      <Route
        path="/steam-login-success"
        element={
          <AuthProvider>
            <UserProvider>
              <SteamLoginSuccess />
            </UserProvider>
          </AuthProvider>
        }
      />

      {/* -------------- PRIVADAS -------------- */}
      <Route
        element={
          <AuthProvider>
            <UserProvider>
              <RequireAuth />
            </UserProvider>
          </AuthProvider>
        }
      >
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/bot-instructions" element={<BotInstructions />} />
        <Route path="/history-code" element={<HistoryCodeForm />} />
        <Route path="/history-games" element={<HistoryGames />} />
        <Route path="/personal-performance" element={<PersonalPerformance />} />
        <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
      </Route>
    </Routes>
  </BrowserRouter>
);

export default App;
