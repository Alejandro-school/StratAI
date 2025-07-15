import React, { useEffect, useState } from "react";
import Header from "../Layout/Header";
import SidebarComponent from "../Layout/Sidebar";
import "../../styles/Start/dashboard.css";
import { useLocation } from "react-router-dom";
import { useUser } from "../../context/UserContext";          // ✅

const API_URL = process.env.REACT_APP_API_URL || "http://localhost:8000";

function Dashboard() {
  const { user, setUser } = useUser();                       // ✅
  const [error, setError] = useState(false);
  const location = useLocation();
  console.log("Ubicación actual:", location.pathname);

  /** Trae los datos del usuario sólo si todavía no los tenemos */
  useEffect(() => {
    if (user) return;                                        // ya tenemos datos

    const fetchUser = async () => {
      try {
        const res = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
        });
        if (!res.ok) throw new Error();
        const data = await res.json();
        setUser(data);                                      // guarda en contexto
      } catch {
        setError(true);
      }
    };

    fetchUser();
  }, [user, setUser]);

  if (error) return <p style={{ color: "red" }}>No estás autenticado.</p>;

  return (
    <div className="container">
      <SidebarComponent />     {/* ya no necesita prop user */}
      <div className="main-content">
        <Header />            {/* idem */}
      </div>
    </div>
  );
}

export default Dashboard;
