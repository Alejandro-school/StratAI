import React, { useEffect, useState } from 'react';
import Header from '../Layout/Header';
import BodyVideo from '../Layout/BodyVideo';
import '../../styles/Start/dashboard.css';
import SidebarComponent from '../Layout/Sidebar';  // ✅ Corrección
import { useLocation } from 'react-router-dom';




const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

function Dashboard() {
  const [user, setUser] = useState(null);
  const [error, setError] = useState(false);
  const location = useLocation();
console.log('Ubicación actual:', location.pathname);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: 'include',
        });

        if (!response.ok) {
          setError(true);
          return;
        }
        const data = await response.json();
        setUser(data);
      } catch (err) {
        setError(true);
      }
    };

    fetchUser();
  }, []);

  return (
    <div className="container">
      {/* Sidebar */}
      <SidebarComponent user={user} />
      <div className="main-content">
        {/* Header */}
        <Header user={user} />

        {/* Video de fondo */}
        <BodyVideo />
      </div>
    </div>
  );
}

export default Dashboard;
