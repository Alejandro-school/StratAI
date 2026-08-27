import React, { Suspense, lazy } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

import { AuthProvider } from "./auth/useAuth";

const LandingPage = lazy(() => import('./components/Landing/LandingPage').then((module) => ({
  default: module.LandingPage,
})));
const ProtectedAppLayout = lazy(() => import('./components/Layout/ProtectedAppLayout'));
const SteamLoginSuccess = lazy(() => import("./auth/SteamLoginSuccess"));
const TacticalMap = lazy(() => import("./pages/TacticalMap"));
const CoachDashboard = lazy(() => import("./pages/CoachDashboard"));
const BotInstructions = lazy(() => import("./auth/BotInstructions"));
const HistoryCodeForm = lazy(() => import("./auth/HistoryCodeForm"));
const MatchHistoryPage = lazy(() => import("./pages/MatchHistoryPage"));
const MatchDetails = lazy(() => import("./pages/MatchDetails"));
const Performance = lazy(() => import("./pages/Performance"));
const Progress = lazy(() => import("./pages/Progress"));
const Feedback = lazy(() => import("./pages/Feedback"));

const RouteFallback = () => (
  <div role="status" aria-live="polite" className="route-loading">
    Cargando…
  </div>
);

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
    <Suspense fallback={<RouteFallback />}>
      <Routes>
        {/* -------------- PÚBLICAS -------------- */}
        <Route path="/" element={<LandingPage />} />

        {/* Callback post‑Steam OAuth: necesita contexto pero no auth previa */}
        <Route
          path="/steam-login-success"
          element={
            <AuthProvider>
              <SteamLoginSuccess />
            </AuthProvider>
          }
        />

        {/* -------------- PRIVADAS -------------- */}
        <Route
          element={<ProtectedAppLayout />}
        >
          {/* Navegación principal */}
          <Route path="/dashboard" element={<CoachDashboard />} />
          <Route path="/tactical-map" element={<TacticalMap />} />
          <Route path="/history-games" element={<MatchHistoryPage />} />
          <Route path="/performance" element={<Performance />} />
          <Route path="/progress" element={<Progress />} />
          <Route path="/feedback" element={<Feedback />} />

          {/* Rutas de detalle y utilidades */}
          <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
          <Route path="/bot-instructions" element={<BotInstructions />} />
          <Route path="/history-code" element={<HistoryCodeForm />} />
        </Route>
      </Routes>
    </Suspense>
  </BrowserRouter>
);

export default App;
