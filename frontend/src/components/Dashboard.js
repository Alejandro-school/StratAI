import React, { useEffect, useState } from 'react';
import Sidebar from './Sidebar';
import Header from './Header';
import BodyVideo from './BodyVideo';
import '../styles/dashboard.css';
import { data } from 'react-router-dom';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

function Dashboard() {
  const [user, setUser] = useState(null);
  const [error, setError] = useState(false);

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
      <Sidebar user={data} />
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
