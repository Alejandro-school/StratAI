import { useState, useEffect, createContext, useContext } from "react";
import { API_URL } from "../utils/api";

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [checking, setChecking] = useState(true);
  const [authError, setAuthError] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    fetch(`${API_URL}/auth/steam/status`, {
      credentials: "include",
      signal: controller.signal,
    })
      .then((response) => {
        if (!response.ok) throw new Error(`Auth status failed: ${response.status}`);
        return response.json();
      })
      .then(data => {
        setUser(data.authenticated ? data : null);
      })
      .catch((error) => {
        if (error.name !== "AbortError") setAuthError(error);
      })
      .finally(() => setChecking(false));
    return () => controller.abort();
  }, []);

  if (checking) {
    return <div role="status" aria-live="polite">Comprobando sesión…</div>;
  }

  return (
    <AuthContext.Provider value={{ user, setUser, checking, authError }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
