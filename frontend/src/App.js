import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import LandingPage from './components/Landing/LandingPage';
import SteamLoginSuccess from './auth/SteamLoginSuccess';
import BotInstructions from './auth/BotInstructions';
import Dashboard from './components/Start/Dashboard';
import HistoryCodeForm from './auth/HistoryCodeForm';
import SessionHandler from './auth/SessionHandler';
import HistoryGames from './components/Stats/HistoryGames';
import PersonalPerformance from './components/Stats/PersonalPerformance';
import MatchDetails from './components/Stats/MatchDetails';
import { UserProvider } from './context/UserContext';

function App() {
  return (
    <UserProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<LandingPage />} />

          <Route element={<SessionHandler />}>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/bot-instructions" element={<BotInstructions />} />
            <Route path="/HistoryCodeForm" element={<HistoryCodeForm />} />
            <Route path="/steam-login-success" element={<SteamLoginSuccess />} />
            <Route path="/HistoryGames" element={<HistoryGames />} />
            <Route path="/PersonalPerformance" element={<PersonalPerformance />} />
            <Route path="/match/:steamID/:matchID" element={<MatchDetails />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </UserProvider>
  );
}

export default App;
