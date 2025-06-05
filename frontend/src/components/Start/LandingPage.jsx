import React, { useEffect, useState } from 'react';
import BodyVideo from '../Layout/BodyVideo.jsx';
import '../../styles/Start/landing.css';

import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { motion } from 'framer-motion';
import {
  FaSteam,
  FaChartLine,
  FaTrophy,
  FaUsers,
  FaShieldAlt,
  FaUserFriends,
  FaDiscord,
} from 'react-icons/fa';

import { Swiper, SwiperSlide } from 'swiper/react';
import { Pagination, Autoplay } from 'swiper/modules';
import 'swiper/css';
import 'swiper/css/pagination';

/* ---------- datos ---------- */
const features = [
  {
    Icon: FaChartLine,
    title: 'Estadísticas instantáneas',
    text: 'Consulta tu rendimiento en tiempo real durante cada partida.',
  },
  {
    Icon: FaTrophy,
    title: 'Desafíos de entrenamiento',
    text: 'Supera retos diarios para perfeccionar tus habilidades.',
  },
  {
    Icon: FaUsers,
    title: 'Coach IA',
    text: 'Recibe consejos personalizados gracias a nuestra inteligencia artificial.',
  },
];

const plans = [
  {
    name: 'Free',
    desc: 'Estadísticas básicas para iniciarte.',
    features: [
      'Análisis limitado de partidas',
      'Acceso a la comunidad',
      '5 partidas semanales',
    ],
  },
  {
    name: 'Pro',
    desc: 'Coaching avanzado y datos en tiempo real.',
    features: [
      'Estadísticas ilimitadas',
      'Asesoramiento personalizado',
      'Retos exclusivos y rankings',
    ],
  },
];

const testimonials = [
  {
    text: 'STRATAI llev\u00f3 mi juego al siguiente nivel.',
    author: 'Jugador profesional',
  },
  {
    text: 'Las estad\u00edsticas en vivo son simplemente geniales.',
    author: 'Usuario beta',
  },
  {
    text: 'El coach de IA me ayuda a mejorar d\u00eda a d\u00eda.',
    author: 'Jugador competitivo',
  },
];
// Barra de tareas (taskbar) simple para la landing page

const taskbarItems = [
  { id: 'inicio', label: 'Inicio', icon: <FaChartLine /> },
  { id: 'features', label: 'Características', icon: <FaTrophy /> },
  { id: 'demo', label: 'Demo', icon: <FaUsers /> },
  { id: 'testimonials', label: 'Testimonios', icon: <FaUserFriends /> },
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
  { id: 'testimonials', label: 'Testimonios' },
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
          <h1 className="landing-title">
            Domina cada ronda de CS2 con análisis de élite
          </h1>
          <p className="landing-subtitle">
            Retos, estadísticas en tiempo real y coaching impulsado por IA
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
            {features.map(({ Icon, title, text }, idx) => (
              <SwiperSlide key={title}>
                <motion.div
                  className="feature-card"
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6, delay: idx * 0.2 }}
                  viewport={{ once: true }}
                >
                  <Icon className="feature-icon" />
                  <h3 className="feature-title">{title}</h3>
                  <p className="feature-text">{text}</p>
                </motion.div>
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
          <ul className="demo-list">
            <li>Visualiza cómo se registran tus partidas en segundos.</li>
            <li>Comprueba la interfaz intuitiva y fácil de usar.</li>
            <li>Descubre consejos rápidos para mejorar tu rendimiento.</li>
          </ul>
        </div>
      </section>

      {/* ---------- TESTIMONIOS ---------- */}
      <section id="testimonials" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">Lo que dicen los jugadores</h2>
          <Swiper
            modules={[Pagination, Autoplay]}
            spaceBetween={30}
            pagination={{ clickable: true }}
            autoplay={{ delay: 5000, disableOnInteraction: false }}
            className="features-swiper"
          >
            {testimonials.map((t, idx) => (
              <SwiperSlide key={idx}>
                <motion.div
                  className="testimonial-card"
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6, delay: idx * 0.2 }}
                  viewport={{ once: true }}
                >
                  <p className="testimonial-text">"{t.text}"</p>
                  <p className="testimonial-author">- {t.author}</p>
                </motion.div>
              </SwiperSlide>
            ))}
          </Swiper>
        </div>
      </section>

      {/* ---------- PRICING ---------- */}
      <section id="pricing" className="scroll-section snap-section">
        <div className="section-content">
          <h2 className="section-title">Planes</h2>
          <div className="pricing-grid">
            {plans.map((plan, idx) => (
              <motion.div
                key={plan.name}
                className={`price-card${plan.name === 'Pro' ? ' recommended' : ''}`}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: idx * 0.2 }}
                viewport={{ once: true }}
              >
                <h3>{plan.name}</h3>
                <p>{plan.desc}</p>
                <ul className="plan-features">
                  {plan.features.map((f) => (
                    <li key={f}>{f}</li>
                  ))}
                </ul>
                <button className="btn-primary">
                  {plan.name === 'Free' ? 'Empezar' : 'Probar 7 días'}
                </button>
              </motion.div>
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
            Únete a nuestra comunidad para estar al día de las novedades.
          </p>
          <a href="mailto:contacto@stratai.gg" className="btn-primary">
            Contactar
          </a>
          <a
            href="https://discord.gg/stratai"
            className="btn-primary discord-link"
            target="_blank"
            rel="noopener noreferrer"
          >
            <FaDiscord style={{ marginRight: '0.4rem' }} />
            Únete a Discord
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
