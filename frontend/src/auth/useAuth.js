import { useState, useEffect, createContext, useContext } from "react";
import { API_URL } from "../utils/api";

const AuthContext = createContext(null);

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