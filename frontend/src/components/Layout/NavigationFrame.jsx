// frontend/src/components/Layout/NavigationFrame.jsx
// Tactical HUD Navigation - Esports Control Panel Style
import React, { useState, useEffect, useRef } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useUser } from '../../context/UserContext';
import { API_URL } from '../../utils/api';
import {
  BarChart2, Target, Brain, TrendingUp, LogOut, Map, MessageSquare
} from 'lucide-react';

const NavigationFrame = ({ children }) => {
  const { user } = useUser();
  const location = useLocation();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const tabsRef = useRef(null);
  const [indicatorStyle, setIndicatorStyle] = useState({ left: 0, width: 0 });

  const navItems = [
    { path: '/dashboard', icon: Brain, label: 'Coach IA' },
    { path: '/tactical-map', icon: Map, label: 'Mapa Táctico' },
    { path: '/history-games', icon: BarChart2, label: 'Partidas' },
    { path: '/performance', icon: Target, label: 'Rendimiento' },
    { path: '/progress', icon: TrendingUp, label: 'Progreso' },
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
      <a className="skip-link" href="#main-content">Saltar al contenido</a>
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
          <Link
            to="/feedback"
            className={`hud-feedback-link ${location.pathname === '/feedback' ? 'active' : ''}`}
          >
            <MessageSquare size={15} />
            <span>Comentarios</span>
          </Link>

          <div className="hud-user">
            <button
              type="button"
              className="hud-user-trigger"
              aria-label="Abrir menú de usuario"
              aria-expanded={showUserMenu}
              onClick={(event) => {
                event.stopPropagation();
                setShowUserMenu((current) => !current);
              }}
            >
              <span className="user-name">{user?.username || 'Usuario'}</span>
              <img
                src={user?.avatar || '/default-avatar.png'}
                alt=""
                width="32"
                height="32"
                className="user-avatar"
              />
            </button>
            
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
                <Link to="/feedback" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                  <MessageSquare size={14} /> Comentarios
                </Link>
                <button onClick={handleLogout} className="dropdown-item logout">
                  <LogOut size={14} /> Cerrar sesión
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main id="main-content" className="hud-content" tabIndex="-1">
        {children}
      </main>
    </div>
  );
};

export default NavigationFrame;
