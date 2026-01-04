import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

// ───────────── Rutas públicas ──────────────
import LandingPage from "./components/Landing/LandingPage";
import SteamLoginSuccess from "./auth/SteamLoginSuccess";

// ───────────── Rutas privadas ──────────────
import Dashboard from "./components/Dashboard/DashboardMapV2";
import BotInstructions from "./auth/BotInstructions";
import HistoryCodeForm from "./auth/HistoryCodeForm";
import HistoryGames from "./components/Stats/HistoryGames";
import MatchDetails from "./components/Match/MatchDetails";

// ───────────── Nuevos componentes Stats ──────────────
import PersonalPerformance from "./components/Stats/PersonalPerformance";
import MapPerformance from "./components/Stats/MapPerformance";
import Replays2D from "./components/Stats/Replays2D";
import AnalyzeDemos from "./components/Stats/AnalyzeDemos";
import Progress from "./components/Stats/Progress";
import Improvements from "./components/Stats/Improvements";

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
        <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
        
        {/* Stats Section */}
        <Route path="/personal-performance" element={<PersonalPerformance />} />
        <Route path="/map-performance" element={<MapPerformance />} />
        <Route path="/replays-2d" element={<Replays2D />} />
        
        {/* Learn Section */}
        <Route path="/analyze-demos" element={<AnalyzeDemos />} />
        <Route path="/progress" element={<Progress />} />
        <Route path="/improvements" element={<Improvements />} />
      </Route>
    </Routes>
  </BrowserRouter>
);

export default App;
