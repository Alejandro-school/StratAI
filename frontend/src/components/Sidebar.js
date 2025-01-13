// Sidebar.js
import React, { useState } from 'react';
import { Sidebar, Menu, MenuItem } from 'react-pro-sidebar';
import {
        Home, BarChart, Timeline,
        Build, FileUploadOutlined,AirlineStopsOutlined,
        LockOpenOutlined,MapOutlined, MissedVideoCall, 
        EmojiPeopleOutlined,FlightTakeoffOutlined,DesktopWindowsOutlined
      
      } from '@mui/icons-material';
import { Link } from 'react-router-dom';
import Grow from '@mui/material/Grow';

import '../styles/sidebar.css';
import Avatar from '@mui/material/Avatar';
import { Box } from '@mui/material';


const SidebarComponent = ({ user }) => {
  const [collapsed, setCollapsed] = useState(true);
  const [hovered, setHovered] = useState(false);

  const handleMouseEnter = () => {
    setHovered(true);
    setCollapsed(false);
  };

  const handleMouseLeave = () => {
    setHovered(false);
    setCollapsed(true);
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

        <Grow in={user} timeout={500}>

            <Avatar alt="Avatar" src={user.avatar} sx={{ width: 56, height: 56}} />

        </Grow>

      </Box>

      <Menu className="menu">
        <MenuItem icon={<Home />} className="menu-item">
          <Link to="/dashboard">Inicio</Link>
        </MenuItem>

        {/* Análisis de Partidas */}
        <div className="menu-section-title">
          Stats
        </div>
        <MenuItem icon={<BarChart />} className="menu-item">
          <Link to="/HistoryGames">Partidas</Link>
        </MenuItem>
        <MenuItem icon={<EmojiPeopleOutlined />} className="menu-item">
          <Link to="/RecentShareCodes">Desempeño personal</Link>
        </MenuItem>
        <MenuItem icon={<MapOutlined />} className="menu-item">
          <Link to="/analysis/movements">Rendimiento mapas</Link>
        </MenuItem>
        <MenuItem icon={<MissedVideoCall />} className="menu-item">
          <Link to="/analysis/events">Repeticiones 2D</Link>
        </MenuItem>

        {/* Entrenamiento de IA */}
        <div className="menu-section-title">
          Learn
        </div>
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

        {/* Datos en Tiempo Real */}
        <div className="menu-section-title">
         Live
        </div>
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
