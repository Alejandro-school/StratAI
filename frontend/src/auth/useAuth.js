import { useState, useEffect, createContext, useContext } from "react";

const AuthContext = createContext(null);

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    fetch(`${API_URL}/auth/steam/status`, { credentials: "include" })
      .then(r => r.json())
      .then(data => {
        if (data.authenticated) setUser(data);
      })
      .finally(() => setChecking(false));
  }, []);

  if (checking) return null;          // o un spinner

  return (
    <AuthContext.Provider value={{ user, setUser }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);