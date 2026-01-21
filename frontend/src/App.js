import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

// ───────────── Rutas públicas ──────────────
import { LandingPage } from "./components/Landing/LandingPage";
import SteamLoginSuccess from "./auth/SteamLoginSuccess";

// ───────────── Rutas privadas ──────────────
import Dashboard from "./components/Dashboard/DashboardMapV2";
import BotInstructions from "./auth/BotInstructions";
import HistoryCodeForm from "./auth/HistoryCodeForm";
import HistoryGames from "./components/Stats/HistoryGames";
import MatchDetails from "./components/Match/MatchDetails";

// ───────────── Nuevas páginas principales ──────────────
import Performance from "./pages/Performance";
import CoachIA from "./pages/CoachIA";
import Progreso from "./pages/Progreso";

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
 * 1. Dashboard - Mapa interactivo con estadísticas
 * 2. Partidas - Historial de partidas + Match Details
 * 3. Performance - Estadísticas personales detalladas
 * 4. Coach IA - Chat inteligente + análisis de partidas
 * 5. Progreso - Misiones, logros, evolución
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
        {/* Navegación principal (5 secciones) */}
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/history-games" element={<HistoryGames />} />
        <Route path="/performance" element={<Performance />} />
        <Route path="/coach" element={<CoachIA />} />
        <Route path="/progress" element={<Progreso />} />
        
        {/* Rutas de detalle y utilidades */}
        <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
        <Route path="/bot-instructions" element={<BotInstructions />} />
        <Route path="/history-code" element={<HistoryCodeForm />} />
      </Route>
    </Routes>
  </BrowserRouter>
);

export default App;
