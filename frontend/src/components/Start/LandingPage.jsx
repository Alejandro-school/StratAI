import React, { useEffect, useState } from 'react';
import BodyVideo from '../Layout/BodyVideo.jsx';
import '../../styles/Start/landing.css';

import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { FaSteam, FaChartLine, FaTrophy, FaUsers } from 'react-icons/fa';

import { Swiper, SwiperSlide } from 'swiper/react';
import { Pagination, Autoplay } from 'swiper/modules';
import 'swiper/css';
import 'swiper/css/pagination';

/* ---------- datos ---------- */
const features = [
  {
    Icon: FaChartLine,
    title: 'Estadísticas en tiempo real',
    text: 'Datos de rendimiento en vivo para que ajustes tu estrategia sobre la marcha.',
  },
  {
    Icon: FaTrophy,
    title: 'Comparativa con profesionales',
    text: 'Mide tu progreso frente a jugadores de élite y descubre dónde mejorar.',
  },
  {
    Icon: FaUsers,
    title: 'Coaching personalizado',
    text: 'Recibe recomendaciones diarias basadas en tu estilo de juego y objetivos.',
  },
];
// Barra de tareas (taskbar) simple para la landing page

const taskbarItems = [
  { id: 'inicio', label: 'Inicio', icon: <FaChartLine /> },
  { id: 'features', label: 'Características', icon: <FaTrophy /> },
  { id: 'demo', label: 'Demo', icon: <FaUsers /> },
  { id: 'pricing', label: 'Planes', icon: <FaSteam /> },
  { id: 'contact', label: 'Contacto', icon: <FaChartLine /> },
];

const Taskbar = () => (
  <nav className="taskbar">
    <ul>
      {taskbarItems.map((item) => (
        <li key={item.id}>
          <a href={`#${item.id}`}>
            {item.icon}
            <span>{item.label}</span>
          </a>
        </li>
      ))}
    </ul>
  </nav>
);

const navItems = [
  { id: 'hero', label: 'Inicio' },
  { id: 'features', label: 'Características' },
  { id: 'demo', label: 'Demo' },
  { id: 'pricing', label: 'Planes' },
  { id: 'contact', label: 'Contacto' },
];

const LandingPage = () => {
  /* ---------- barra de progreso scroll ---------- */
  const [progress, setProgress] = useState(0);
  const [active, setActive] = useState(0);

  /* ---------- registra animaciones (solo efectos, no visibilidad) ---------- */
  useEffect(() => {
    gsap.registerPlugin(ScrollTrigger);

    gsap.utils.toArray('.snap-section').forEach((section, i) => {
      ScrollTrigger.create({
        trigger: section,
        start: 'top center',
        end: 'bottom center',
        onEnter: () => setActive(i),
        onEnterBack: () => setActive(i),
      });
    });
  }, []);

  /* ---------- handler barra progreso ---------- */
  useEffect(() => {
    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = document.documentElement;
      setProgress((scrollTop / (scrollHeight - clientHeight)) * 100);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return (
    <div className="landing-container">
      {/* barra de progreso */}
      <span
        className="progress-bar"
        style={{ transform: `scaleX(${progress / 100})` }}
      />

      {/* navegación lateral */}
      <ul className="side-nav">
        {navItems.map((item, idx) => (
          <li key={item.id} className={active === idx ? 'active' : ''}>
            <a href={`#${item.id}`}>{item.label}</a>
          </li>
        ))}
      </ul>

      {/* vídeo de fondo + overlay */}
      <BodyVideo />

      {/* ---------- HERO ---------- */}
      <section id="hero" className="hero snap-section">
        <div className="glass-card">
          <h1 className="landing-title">STRATAI</h1>
          <p className="landing-subtitle">
            Domina tus partidas con estadísticas avanzadas y análisis detallados
          </p>
          <a href="/auth/steam" className="btn-steam">
            <FaSteam className="steam-icon" />
            Inicia sesión con Steam
          </a>
          <span className="scroll-arrow">&#x25BC;</span>
        </div>
      </section>

      {/* ---------- FEATURES ---------- */}
      <section id="features" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">Funciones clave</h2>
          <Swiper
            modules={[Pagination, Autoplay]}
            spaceBetween={30}
            pagination={{ clickable: true }}
            autoplay={{ delay: 4000, disableOnInteraction: false }}
            breakpoints={{
              0: { slidesPerView: 1 },
              640: { slidesPerView: 2 },
              1024: { slidesPerView: 3 },
            }}
            className="features-swiper"
          >
            {features.map(({ Icon, title, text }) => (
              <SwiperSlide key={title}>
                <div className="feature-card">
                  <Icon className="feature-icon" />
                  <h3 className="feature-title">{title}</h3>
                  <p className="feature-text">{text}</p>
                </div>
              </SwiperSlide>
            ))}
          </Swiper>
        </div>
      </section>

      {/* ---------- DEMO ---------- */}
      <section id="demo" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">Mira STRATAI en acción</h2>
          <div className="demo-wrapper">
            <video
              src="/videos/demo.mp4"
              className="demo-video"
              controls
              preload="metadata"
            />
          </div>
        </div>
      </section>

      {/* ---------- PRICING ---------- */}
      <section id="pricing" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">Planes</h2>
          <div className="pricing-grid">
            {['Free', 'Pro'].map((plan) => (
              <div
                key={plan}
                className={`price-card${plan === 'Pro' ? ' recommended' : ''}`}
              >
                <h3>{plan}</h3>
                <p>
                  {plan === 'Free'
                    ? 'Estadísticas básicas para iniciarte.'
                    : 'Coaching avanzado y datos en tiempo real.'}
                </p>
                <button className="btn-primary">
                  {plan === 'Free' ? 'Empezar' : 'Probar 7 días'}
                </button>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ---------- CONTACT ---------- */}
      <section id="contact" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">¿Preparado para la victoria?</h2>
          <p className="contact-text">
            Escríbenos y descubre cómo podemos impulsar tu rendimiento.
          </p>
          <a href="mailto:contacto@stratai.gg" className="btn-primary">
            Contactar
          </a>
        </div>
      </section>

      {/* ---------- FOOTER ---------- */}
      <footer className="footer snap-section">
        &copy; {new Date().getFullYear()} STRATAI. Todos los derechos reservados.
      </footer>
    </div>
  );
};

export default LandingPage;
