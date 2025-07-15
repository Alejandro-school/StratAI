import React, { useState } from 'react';
import { Sidebar, Menu, MenuItem } from 'react-pro-sidebar';
import {
  Home, BarChart,
  Build, FileUploadOutlined, AirlineStopsOutlined, MapOutlined, MissedVideoCall,
  EmojiPeopleOutlined, FlightTakeoffOutlined, DesktopWindowsOutlined
} from '@mui/icons-material';
import { Link } from 'react-router-dom';
import Avatar from '@mui/material/Avatar';
import { Box, Menu as DropdownMenu, MenuItem as DropdownItem, IconButton } from '@mui/material';
import '../../styles/Layout/sidebar.css';
import { useUser } from '../../context/UserContext';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

const SidebarComponent = () => {
  const [collapsed, setCollapsed] = useState(true);
  const [hovered, setHovered] = useState(false);
  const [anchorEl, setAnchorEl] = useState(null);
  const { user } = useUser();

  const handleMouseEnter = () => {
    setHovered(true);
    setCollapsed(false);
  };

  const handleMouseLeave = () => {
    setHovered(false);
    setCollapsed(true);
  };

  const handleAvatarClick = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleCloseMenu = () => {
    setAnchorEl(null);
  };

  const handleLogout = async () => {
    try {
      const response = await fetch(`${API_URL}/auth/steam/logout`, {
        method: 'POST',
        credentials: 'include',
      });

      if (response.ok) {
        window.location.href = '/';  // Redirigir a la página de inicio
      }
    } catch (error) {
      console.error("Error al cerrar sesión", error);
    }
  };

  return (
    <Sidebar
      collapsed={collapsed && !hovered}
      className="sidebar"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Header del Sidebar con Avatar y Nombre de Usuario */}
      <Box className="sidebar-header">
        <IconButton onClick={handleAvatarClick}>
          {user && user.avatar ? (
            <Avatar alt="Avatar" src={user?.avatar} sx={{ width: 56, height: 56 }} />
          ) : (
            <Avatar alt="Avatar" src="/default-avatar.png" sx={{ width: 56, height: 56 }} />
          )}
        </IconButton>

        <DropdownMenu
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={handleCloseMenu}
        >
          <DropdownItem onClick={handleCloseMenu}>Perfil</DropdownItem>
          <DropdownItem onClick={handleCloseMenu}>Configuración</DropdownItem>
          <DropdownItem onClick={handleLogout}>Cerrar sesión</DropdownItem>
        </DropdownMenu>
      </Box>

      <Menu className="menu">
        <MenuItem icon={<Home />} className="menu-item">
          <Link to="/dashboard">Inicio</Link>
        </MenuItem>

        <div className="menu-section-title">Stats</div>
        <MenuItem icon={<BarChart />} className="menu-item">
          <Link to="/HistoryGames">Partidas</Link>
        </MenuItem>
        <MenuItem icon={<EmojiPeopleOutlined />} className="menu-item">
          <Link to="/PersonalPerformance">Desempeño personal</Link>
        </MenuItem>
        <MenuItem icon={<MapOutlined />} className="menu-item">
          <Link to="/analysis/movements">Rendimiento mapas</Link>
        </MenuItem>
        <MenuItem icon={<MissedVideoCall />} className="menu-item">
          <Link to="/analysis/events">Repeticiones 2D</Link>
        </MenuItem>

        <div className="menu-section-title">Learn</div>
        <MenuItem icon={<FileUploadOutlined />} className="menu-item">
          <Link to="/MatchSummary">Analizar Demos</Link>
        </MenuItem>
        <MenuItem icon={<AirlineStopsOutlined />} className="menu-item">
          <Link to="/training/recommendations">Progreso</Link>
        </MenuItem>
        <MenuItem icon={<FlightTakeoffOutlined />} className="menu-item">
          <Link to="/training/progress">Mejoras</Link>
        </MenuItem>
        <MenuItem icon={<DesktopWindowsOutlined />} className="menu-item">
          <Link to="/training/progress">Simulaciones</Link>
        </MenuItem>

        <div className="menu-section-title">Live</div>
        <MenuItem icon={<Build />} className="menu-item">
          <Link to="/live/monitor">Monitor de Partidas</Link>
        </MenuItem>
        <MenuItem icon={<Build />} className="menu-item">
          <Link to="/live/alerts">Alertas</Link>
        </MenuItem>
        <MenuItem icon={<Build />} className="menu-item">
          <Link to="/live/visualization">Visualización Dinámica</Link>
        </MenuItem>
      </Menu>
    </Sidebar>
  );
};

export default SidebarComponent;
