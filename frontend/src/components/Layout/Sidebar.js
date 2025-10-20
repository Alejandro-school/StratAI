import React, { useState } from 'react';
import { Sidebar, Menu, MenuItem } from 'react-pro-sidebar';
import {
  Home, BarChart, FileUploadOutlined, AirlineStopsOutlined, 
  MapOutlined, MissedVideoCall, EmojiPeopleOutlined, FlightTakeoffOutlined, 
  LogoutOutlined, SettingsOutlined, PersonOutlined, ChevronRight
} from '@mui/icons-material';
import { Link, useLocation } from 'react-router-dom';
import Avatar from '@mui/material/Avatar';
import { Box, Menu as DropdownMenu, MenuItem as DropdownItem, IconButton } from '@mui/material';
import '../../styles/Layout/sidebar.css';
import { useUser } from '../../context/UserContext';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

const SidebarComponent = () => {
  const [anchorEl, setAnchorEl] = useState(null);
  const { user } = useUser();
  const location = useLocation();

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
        window.location.href = '/';
      }
    } catch (error) {
      console.error("Error al cerrar sesión", error);
    }
  };

  const isActive = (path) => location.pathname === path;

  const menuSections = [
    {
      title: 'STATS',
      items: [
        { path: '/dashboard', icon: <Home />, label: 'Inicio' },
        { path: '/history-games', icon: <BarChart />, label: 'Partidas' },
        { path: '/personal-performance', icon: <EmojiPeopleOutlined />, label: 'Desempeño personal' },
        { path: '/map-performance', icon: <MapOutlined />, label: 'Rendimiento mapas' },
        { path: '/replays-2d', icon: <MissedVideoCall />, label: 'Repeticiones 2D' },
      ]
    },
    {
      title: 'LEARN',
      items: [
        { path: '/analyze-demos', icon: <FileUploadOutlined />, label: 'Analizar Demos' },
        { path: '/progress', icon: <AirlineStopsOutlined />, label: 'Progreso' },
        { path: '/improvements', icon: <FlightTakeoffOutlined />, label: 'Mejoras' },
      ]
    }
  ];

  return (
    <div className="sidebar-wrapper">
      <Sidebar 
        collapsed={true}
        className="modern-sidebar"
      >
        {/* Branding con logo moderno */}
        <Box className="sidebar-brand">
          <Link to="/dashboard" style={{ textDecoration: 'none' }}>
            <IconButton className="brand-logo-btn">
              <div className="brand-logo">
                <svg className="brand-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M12 2L2 7L12 12L22 7L12 2Z" fill="currentColor" fillOpacity="0.3"/>
                  <path d="M2 17L12 22L22 17V12L12 17L2 12V17Z" fill="currentColor"/>
                  <path d="M2 12L12 17L22 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
            </IconButton>
          </Link>
          <span className="brand-name">StratAI</span>
        </Box>

        {/* User Account con nombre */}
        <Box className="sidebar-user">
          <IconButton onClick={handleAvatarClick} className="user-avatar-btn">
            <Avatar 
              alt="Avatar" 
              src={user?.avatar || "/default-avatar.png"} 
              sx={{ width: 40, height: 40 }} 
            />
            <span className="status-indicator"></span>
          </IconButton>
          
          <div className="user-info">
            <span className="user-name">{user?.username || 'Usuario'}</span>
            <span className="user-status">En línea</span>
          </div>

          <DropdownMenu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={handleCloseMenu}
            className="user-dropdown"
            PaperProps={{
              sx: {
                mt: 1,
                background: 'rgba(0, 0, 0, 0.9)',
                backdropFilter: 'blur(20px)',
                border: '1px solid rgba(255, 255, 255, 0.1)',
                borderRadius: '12px',
                boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
                minWidth: '220px',
              }
            }}
          >
            <Box sx={{ p: 2, borderBottom: '1px solid rgba(255, 255, 255, 0.05)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <Avatar 
                  alt="Avatar" 
                  src={user?.avatar || "/default-avatar.png"} 
                  sx={{ width: 40, height: 40 }} 
                />
                <div>
                  <div style={{ color: '#fff', fontSize: '14px', fontWeight: 600 }}>
                    {user?.username || 'Usuario'}
                  </div>
                  <div style={{ color: '#888', fontSize: '12px' }}>En línea</div>
                </div>
              </div>
            </Box>
            <DropdownItem onClick={handleCloseMenu} sx={{ py: 1.5, px: 2, color: '#e0e0e0' }}>
              <PersonOutlined sx={{ mr: 2, fontSize: 20 }} />
              Perfil
            </DropdownItem>
            <DropdownItem onClick={handleCloseMenu} sx={{ py: 1.5, px: 2, color: '#e0e0e0' }}>
              <SettingsOutlined sx={{ mr: 2, fontSize: 20 }} />
              Configuración
            </DropdownItem>
            <Box sx={{ borderTop: '1px solid rgba(255, 255, 255, 0.05)', mt: 0.5 }}>
              <DropdownItem 
                onClick={handleLogout} 
                sx={{ py: 1.5, px: 2, color: '#ef4444', '&:hover': { backgroundColor: 'rgba(239, 68, 68, 0.1)' } }}
              >
                <LogoutOutlined sx={{ mr: 2, fontSize: 20 }} />
                Cerrar sesión
              </DropdownItem>
            </Box>
          </DropdownMenu>
        </Box>

        {/* Menu */}
        <Menu className="sidebar-menu">
          {menuSections.map((section, sectionIdx) => (
            <React.Fragment key={section.title}>
              <div className="menu-section-title">{section.title}</div>
              {section.items.map((item) => (
                <MenuItem 
                  key={item.path}
                  icon={item.icon}
                  className={`menu-item ${isActive(item.path) ? 'active' : ''}`}
                  component={<Link to={item.path} />}
                >
                  <Link to={item.path} style={{ display: 'flex', alignItems: 'center', width: '100%', textDecoration: 'none' }}>
                    <span style={{ flex: 1 }}>{item.label}</span>
                    <ChevronRight className="menu-arrow" />
                  </Link>
                </MenuItem>
              ))}
            </React.Fragment>
          ))}
        </Menu>
      </Sidebar>
    </div>
  );
};

export default SidebarComponent;