import React from "react";
import { Avatar } from "@mui/material";
import { useUser } from "../../context/UserContext";
import "../../styles/Layout/header.css";

const Header = () => {
  const { user } = useUser();

  return (
    <header className="header">
      <div className="header-inner">
        <div /> {/* columna izquierda vacía para centrar el título */}
        <h3 className="header-title">StratAI</h3>
        <div className="header-right">
          <span className="header-name">{user?.username ?? "Invitado"}</span>
          <Avatar
            alt="Avatar"
            src={user?.avatar || "/default-avatar.png"}
            sx={{ width: 36, height: 36 }}
          />
        </div>
      </div>
    </header>
  );
};

export default Header;
