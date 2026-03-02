import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

// ───────────── Rutas públicas ──────────────
import { LandingPage } from "./components/Landing/LandingPage";
import SteamLoginSuccess from "./auth/SteamLoginSuccess";

// ───────────── Rutas privadas ──────────────
import TacticalMap from "./pages/TacticalMap";
import CoachDashboard from "./pages/CoachDashboard";
import BotInstructions from "./auth/BotInstructions";
import HistoryCodeForm from "./auth/HistoryCodeForm";
import HistoryGames from "./pages/HistoryGames";
import MatchDetails from "./pages/MatchDetails";

// ───────────── Nuevas páginas principales ──────────────
import Performance from "./pages/Performance";
import Progress from "./pages/Progress";

// ───────────── Contextos & Auth ──────────────
import { AuthProvider } from "./auth/useAuth";
import { UserProvider } from "./context/UserContext";
import RequireAuth from "./auth/RequireAuth";

/**
 * ▸ En la landing ("/"), la app se renderiza **sin** comprobar sesión
 *   → no dispara /auth/steam/status y evita los Failed‑to‑fetch.
 * ▸ Las rutas protegidas se agrupan bajo un layout que **sí** envuelve
 *   de AuthProvider + UserProvider + RequireAuth.
 * 
 * ESTRUCTURA DE NAVEGACIÓN:
 * 1. Dashboard (Coach Center) - Command Center con IA y Resumen
 * 2. Tactical Map (Old Dashboard) - Mapa interactivo 2D
 * 3. Matches - Match history + Match Details
 * 4. Performance - Estadísticas personales detalladas
 * 5. Progress - Missions, achievements, evolution
 */
const App = () => (
  <BrowserRouter>
    <Routes>
      {/* -------------- PÚBLICAS -------------- */}
      <Route path="/" element={<LandingPage />} />

      {/* Callback post‑Steam OAuth: necesita contexto pero no auth previa */}
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

      {/* -------------- PRIVADAS -------------- */}
      <Route
        element={
          <AuthProvider>
            <UserProvider>
              <RequireAuth />
            </UserProvider>
          </AuthProvider>
        }
      >
        {/* Navegación principal */}
        <Route path="/dashboard" element={<CoachDashboard />} />
        <Route path="/tactical-map" element={<TacticalMap />} />
        <Route path="/history-games" element={<HistoryGames />} />
        <Route path="/performance" element={<Performance />} />
        <Route path="/progress" element={<Progress />} />
        
        {/* Rutas de detalle y utilidades */}
        <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
        <Route path="/bot-instructions" element={<BotInstructions />} />
        <Route path="/history-code" element={<HistoryCodeForm />} />
      </Route>
    </Routes>
  </BrowserRouter>
);

export default App;
