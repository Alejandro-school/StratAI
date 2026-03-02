import { createContext, useContext, useState, useEffect } from "react";
import { API_URL } from "../utils/api";

const UserContext = createContext(null);            // ① valor inicial null

export const UserProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const res = await fetch(`${API_URL}/auth/steam/status`, {
          credentials: "include",
        });
        if (res.ok) {
          const data = await res.json();
          setUser(data);
        }
      } catch (err) {
        console.error("Error al obtener usuario:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchUser();
  }, []);

  return (
    <UserContext.Provider value={{ user, setUser, loading }}>
      {children}
    </UserContext.Provider>
  );
};

/** Hook de consumo; avisa si se usa fuera del provider */
export const useUser = () => {
  const ctx = useContext(UserContext);
  if (ctx === null)
    throw new Error("useUser must be used inside <UserProvider>"); // ②
  return ctx;                                                     // ③ devuelve { user, setUser, loading }
};
