import { useState, useEffect, createContext, useContext } from "react";
import { API_URL } from "../utils/api";

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [checking, setChecking] = useState(true);
  const [authError, setAuthError] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    let isActive = true;

    const checkSession = async () => {
      try {
        const response = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
          signal: controller.signal,
        });
        if (!response.ok) throw new Error(`Auth status failed: ${response.status}`);

        const data = await response.json();
        if (isActive) setUser(data.authenticated ? data : null);
      } catch (error) {
        if (isActive && error.name !== "AbortError") setAuthError(error);
      } finally {
        if (isActive) setChecking(false);
      }
    };

    checkSession();
    return () => {
      isActive = false;
      controller.abort();
    };
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
