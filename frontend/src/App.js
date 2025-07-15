import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import LandingPage from "./components/Landing/LandingPage";
import Dashboard from "./components/Start/Dashboard";
import BotInstructions from "./auth/BotInstructions";
import HistoryCodeForm from "./auth/HistoryCodeForm";
import HistoryGames from "./components/Stats/HistoryGames";
import PersonalPerformance from "./components/Stats/PersonalPerformance";
import MatchDetails from "./components/Stats/MatchDetails";
import SteamLoginSuccess from "./auth/SteamLoginSuccess";

import { AuthProvider } from "./auth/useAuth";
import RequireAuth from "./auth/RequireAuth";
import { UserProvider } from "./context/UserContext";   //  ✅  nuevo

function App() {
  return (
    <UserProvider>        {/*  ← primer wrapper */}
      <AuthProvider>      {/*  ← tu wrapper existente */}
        <BrowserRouter>
          <Routes>
            {/* Público */}
            <Route path="/" element={<LandingPage />} />
            <Route path="/steam-login-success" element={<SteamLoginSuccess />} />

            {/* Privado */}
            <Route element={<RequireAuth />}>
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/bot-instructions" element={<BotInstructions />} />
              <Route path="/history-code" element={<HistoryCodeForm />} />
              <Route path="/history-games" element={<HistoryGames />} />
              <Route path="/personal-performance" element={<PersonalPerformance />} />
              <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </UserProvider>
  );
}

export default App;
