// frontend/src/components/Layout/NavigationFrame.jsx
// Tactical HUD Navigation - Esports Control Panel Style
import React, { useState, useEffect, useRef } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useUser } from '../../context/UserContext';
import { 
  Home, BarChart2, Target, Brain, Upload, Settings, LogOut, User, Zap
} from 'lucide-react';
import '../../styles/Layout/navigationFrame.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

const NavigationFrame = ({ children }) => {
  const { user } = useUser();
  const location = useLocation();
  const [currentTime, setCurrentTime] = useState(new Date());
  const [showUserMenu, setShowUserMenu] = useState(false);
  const tabsRef = useRef(null);
  const [indicatorStyle, setIndicatorStyle] = useState({ left: 0, width: 0 });

  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  const navItems = [
    { path: '/dashboard', icon: Home, label: 'Dashboard' },
    { path: '/history-games', icon: BarChart2, label: 'Partidas' },
    { path: '/personal-performance', icon: Target, label: 'Análisis' },
    { path: '/map-performance', icon: Zap, label: 'Mapas' },
    { path: '/improvements', icon: Brain, label: 'Coach IA' },
    { path: '/analyze-demos', icon: Upload, label: 'Subir Demo' },
  ];

  // Update indicator position when route changes
  useEffect(() => {
    if (tabsRef.current) {
      const activeTab = tabsRef.current.querySelector('.nav-tab.active');
      if (activeTab) {
        setIndicatorStyle({
          left: activeTab.offsetLeft,
          width: activeTab.offsetWidth,
        });
      }
    }
  }, [location.pathname]);

  useEffect(() => {
    const handleClickOutside = () => setShowUserMenu(false);
    if (showUserMenu) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [showUserMenu]);

  const handleLogout = async (e) => {
    e.stopPropagation();
    try {
      await fetch(`${API_URL}/auth/steam/logout`, {
        method: 'POST',
        credentials: 'include',
      });
      window.location.href = '/';
    } catch (error) {
      console.error("Error al cerrar sesión", error);
    }
  };

  return (
    <div className="hud-layout">
      {/* Top Navigation Bar */}
      <header className="hud-header">
        {/* Logo */}
        <Link to="/dashboard" className="hud-logo">
          <div className="logo-mark">
            <svg viewBox="0 0 24 24" fill="none">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" fill="currentColor" fillOpacity="0.4"/>
              <path d="M2 17L12 22L22 17V12L12 17L2 12V17Z" fill="currentColor"/>
            </svg>
          </div>
          <span className="logo-name">STRAT<span className="logo-accent">AI</span></span>
        </Link>

        {/* Navigation Tabs */}
        <nav className="hud-nav" ref={tabsRef}>
          <div 
            className="nav-indicator" 
            style={{ 
              left: `${indicatorStyle.left}px`, 
              width: `${indicatorStyle.width}px` 
            }}
          />
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`nav-tab ${isActive ? 'active' : ''}`}
              >
                <Icon size={16} />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        {/* Right Side */}
        <div className="hud-right">
          <div 
            className="hud-user" 
            onClick={(e) => { e.stopPropagation(); setShowUserMenu(!showUserMenu); }}
          >
            <span className="user-name">{user?.username || 'Usuario'}</span>
            <img 
              src={user?.avatar || '/default-avatar.png'} 
              alt="Avatar" 
              className="user-avatar"
            />
            
            {showUserMenu && (
              <div className="user-dropdown" onClick={(e) => e.stopPropagation()}>
                <div className="dropdown-header">
                  <img src={user?.avatar || '/default-avatar.png'} alt="" />
                  <div>
                    <span className="dropdown-name">{user?.username}</span>
                    <span className="dropdown-status">En línea</span>
                  </div>
                </div>
                <div className="dropdown-divider" />
                <Link to="/profile" className="dropdown-item">
                  <User size={14} /> Perfil
                </Link>
                <Link to="/settings" className="dropdown-item">
                  <Settings size={14} /> Configuración
                </Link>
                <div className="dropdown-divider" />
                <button onClick={handleLogout} className="dropdown-item logout">
                  <LogOut size={14} /> Cerrar sesión
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="hud-content">
        {children}
      </main>

      {/* HUD Corners */}
      <div className="hud-corners">
        <div className="corner corner-bl">
          <span className="corner-label">VERSION</span>
          <span className="corner-value">v1.0.0</span>
        </div>
        <div className="corner corner-br">
          <div className="corner-ping">
            <span className="ping-dot"></span>
            <span className="ping-value">12ms</span>
          </div>
          <div className="corner-status">
            <span className="status-dot"></span>
            <span className="status-text">ONLINE</span>
          </div>
          <span className="corner-time">
            {currentTime.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </span>
        </div>
      </div>
    </div>
  );
};

export default NavigationFrame;
