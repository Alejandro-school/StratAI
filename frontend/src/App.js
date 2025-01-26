import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import LandingPage from './components/Start/LandingPage';
import SteamLoginSuccess from './auth/SteamLoginSuccess';
import BotInstructions from './auth/BotInstructions';
import Dashboard from './components/Start/Dashboard';
import HistoryCodeForm from './auth/HistoryCodeForm';
import SessionHandler from './auth/SessionHandler';
import HistoryGames from './components/Stats/HistoryGames';


function App() {
  return (
<BrowserRouter>
  <Routes>
    {/* Rutas públicas */}
    <Route path="/" element={<LandingPage />} />


    {/* Rutas privadas, con SessionHandler como wrapper */}
    <Route element={<SessionHandler />}>
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/bot-instructions" element={<BotInstructions />} />
      <Route path="/HistoryCodeForm" element={<HistoryCodeForm />} />
      <Route path="/steam-login-success" element={<SteamLoginSuccess />} />
      <Route path="/HistoryGames" element={<HistoryGames />} />

    </Route>

  </Routes>
</BrowserRouter>

  );
}

export default App;
