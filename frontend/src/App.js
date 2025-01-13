import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import SteamLoginButton from './auth/SteamLoginButton';
import SteamLoginSuccess from './auth/SteamLoginSuccess';
import Dashboard from './components/Dashboard';
import UploadDemo from './components/UploadDemo';
//import MatchSummary from './components/Stats/MatchSummary';
import HistoryCodeForm from './auth/HistoryCodeForm'; // ⬅️ Nueva importación
import HistoryGames from './components/Stats/HistoryGames'; // ⬅️ Nueva importación



import './styles/common.css';

function App() {
  return (
    <Router>
     <div className="app-container">
      <Routes>
        <Route path="/HistoryCodeForm" element={<HistoryCodeForm />} /> {/* ⬅️ Nueva ruta */}
        <Route path="/" element={<SteamLoginButton />} />
        <Route path="/steam-login-success" element={<SteamLoginSuccess />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/UploadDemo" element={<UploadDemo />} />
        <Route path="/HistoryGames" element={<HistoryGames />} />
      </Routes>
     </div>
    </Router>
  );
}

export default App;