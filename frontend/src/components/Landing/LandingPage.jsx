// ============================================
// 📁 ARCHIVO: src/components/Landing/LandingPage.jsx
// 🔥 StratAI Landing — 2025 Ultra Conversion Edition
// ============================================

import React, { useEffect, useRef, useState } from "react";
import "../../styles/Landing/landing.css";
import { steamLogin} from "../../auth/SteamLoginButton";


/**
 * Util: detecta si el usuario prefiere reducir animaciones
 */
const useReducedMotion = () => {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const onChange = () => setReduced(mq.matches);
    onChange();
    mq.addEventListener?.("change", onChange);
    return () => mq.removeEventListener?.("change", onChange);
  }, []);
  return reduced;
};

/**
 * Util: añade clase 'in-view' cuando un bloque entra en viewport
 */
const useInView = (ref, opts = { threshold: 0.2, rootMargin: "0px" }) => {
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) el.classList.add("in-view");
    }, opts);
    io.observe(el);
    return () => io.disconnect();
  }, [ref, opts]);
};

/**
 * Util: pequeño efecto tilt con el puntero (sin dependencias)
 */
const useTilt = (ref, strength = 10) => {
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const onMove = (e) => {
      const rect = el.getBoundingClientRect();
      const x = e.clientX - rect.left - rect.width / 2;
      const y = e.clientY - rect.top - rect.height / 2;
      const rx = (y / rect.height) * -strength;
      const ry = (x / rect.width) * strength;
      el.style.setProperty("--rx", `${rx}deg`);
      el.style.setProperty("--ry", `${ry}deg`);
    };
    const reset = () => {
      el.style.setProperty("--rx", `0deg`);
      el.style.setProperty("--ry", `0deg`);
    };
    el.addEventListener("pointermove", onMove);
    el.addEventListener("pointerleave", reset);
    return () => {
      el.removeEventListener("pointermove", onMove);
      el.removeEventListener("pointerleave", reset);
    };
  }, [ref, strength]);
};

// ============================================
// 🎯 HERO
// ============================================
const Hero = () => {
  const reduced = useReducedMotion();
  const heroRef = useRef(null);
  useInView(heroRef, { threshold: 0.01 });

  // Partículas “baratas” sólo si no hay reduce motion
  useEffect(() => {
    if (reduced) return;
    const nodes = heroRef.current?.querySelectorAll?.(".particle") ?? [];
    nodes.forEach((p, i) => {
      const delay = i * 0.08;
      const duration = 3 + Math.random() * 2;
      p.style.animation = `float ${duration}s ease-in-out ${delay}s infinite`;
    });
  }, [reduced]);



  return (
    <section className="hero-cinema reveal-once" ref={heroRef}>
      <video
        className="hero-video"
        autoPlay
        loop
        muted
        playsInline
        preload="metadata"
      >
        <source src="/videos/hero-bg.mp4" type="video/mp4" />
      </video>

      <div className="hero-overlay"></div>

      {!reduced && (
        <div aria-hidden className="particles">
          {[...Array(20)].map((_, i) => (
            <div
              key={i}
              className="particle"
              style={{
                left: `${Math.random() * 100}%`,
                top: `${Math.random() * 100}%`,
                width: `${2 + Math.random() * 4}px`,
                height: `${2 + Math.random() * 4}px`,
              }}
            />
          ))}
        </div>
      )}

      <div className="hero-content">
        <div className="hero-badge">
          <span className="pulse-dot"></span>
          <span>AI-Powered CS2 Coach</span>
        </div>

        <h1 className="hero-title">
          Domina cada ronda
          <br />
          con <span className="gradient-text">StratAI</span>
        </h1>

        <p className="hero-subtitle">
          Análisis táctico por IA que identifica errores, optimiza tu economía y
          te convierte en un jugador imparable.
        </p>

        <div className="hero-ctas">
          <button
            className="btn-primary btn-magnetic"
            onClick={() => steamLogin()}
          >
            <span>Empieza ahora</span>
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path
                d="M4 10h12m-6-6l6 6-6 6"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </button>

          <a className="btn-secondary" href="#demo">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M6.3 2.84A10 10 0 0117 10c0 5.52-4.48 10-10 10S-3 15.52-3 10 1.48 0 7 0v2c-3.87 0-7 3.13-7 7s3.13 7 7 7 7-3.13 7-7c0-1.93-.78-3.68-2.05-4.95L10 10V2.84z" />
            </svg>
            <span>Ver Demo</span>
          </a>
        </div>

        {/* Trust row para reducir fricción de compra */}
        <div className="trust-row" role="list" aria-label="Confianza">
          <div className="trust-item" role="listitem">RGPD Ready</div>
          <div className="trust-dot" aria-hidden></div>
          <div className="trust-item" role="listitem">AES-256</div>
          <div className="trust-dot" aria-hidden></div>
          <div className="trust-item" role="listitem">Faceit Friendly</div>
          <div className="trust-dot" aria-hidden></div>
          <div className="trust-item" role="listitem">Cancelación 1-click</div>
        </div>

        <div className="hero-stats">
          <div className="stat">
            <div className="stat-value">22K+</div>
            <div className="stat-label">Rondas analizadas</div>
          </div>
          <div className="stat">
            <div className="stat-value">3.2K+</div>
            <div className="stat-label">Jugadores activos</div>
          </div>
          <div className="stat">
            <div className="stat-value">+18%</div>
            <div className="stat-label">Win-rate medio</div>
          </div>
        </div>
      </div>

      <div className="hero-scroll-hint" aria-hidden>
        <span>Scroll para explorar</span>
        <div className="scroll-arrow"></div>
      </div>
    </section>
  );
};

// ============================================
// ⚡ BENTO FEATURES con Tilt 3D
// ============================================
// =====================
// ⚡ BENTO FEATURES FIX
// =====================

const FeatureCard = ({ size, title, desc, icon }) => {
  const cardRef = useRef(null);
  useTilt(cardRef, 8);
  return (
    <article
      ref={cardRef}
      className={`bento-card bento-${size}`}
      aria-label={title}
    >
      <div className="bento-glow" />
      <div className="bento-icon" aria-hidden>
        {icon}
      </div>
      <h3 className="bento-title">{title}</h3>
      <p className="bento-desc">{desc}</p>
    </article>
  );
};

const PowerFeatures = () => {
  const items = [
    {
      title: "Análisis Táctico IA",
      desc: "Detecta errores de posicionamiento, utilidad y timings en cada ronda",
      icon: "🎯",
      size: "large",
    },
    {
      title: "Demo Parser",
      desc: "Sube tus demos o ShareCode automáticamente",
      icon: "📊",
      size: "small",
    },
    {
      title: "Recomendaciones",
      desc: "Consejos personalizados para mejorar tu gameplay",
      icon: "💡",
      size: "small",
    },
    {
      title: "Economía Optimizada",
      desc: "Gestión inteligente de compras y calls óptimos",
      icon: "💰",
      size: "medium",
    },
    {
      title: "Heatmaps",
      desc: "Visualiza tus patrones de movimiento y zonas calientes",
      icon: "🗺️",
      size: "medium",
    },
    {
      title: "Historial Completo",
      desc: "Accede a todas tus partidas y trackea tu evolución",
      icon: "📈",
      size: "small",
    },
  ];

  return (
    <section className="power-features reveal-once">
      <div className="container">
        <div className="section-header">
          <span className="section-tag">CARACTERÍSTICAS</span>
          <h2 className="section-title">
            Todo lo que necesitas para <br />
            <span className="gradient-text">dominar CS2</span>
          </h2>
        </div>

        <div className="bento-grid">
          {items.map((f, idx) => (
            <FeatureCard
              key={`${f.title}-${idx}`}
              size={f.size}
              title={f.title}
              desc={f.desc}
              icon={f.icon}
            />
          ))}
        </div>
      </div>
    </section>
  );
};


// ============================================
// 🎮 DEMO INTERACTIVA (autoplay al entrar)
// ============================================
const InteractiveDemo = () => {
  const wrapRef = useRef(null);
  const videoRef = useRef(null);
  const [isPlaying, setIsPlaying] = useState(false);
  useInView(wrapRef, { threshold: 0.25 });

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    const io = new IntersectionObserver(
      ([e]) => {
        if (e.isIntersecting) {
          video.play().catch(() => {});
          setIsPlaying(true);
        } else {
          video.pause();
          setIsPlaying(false);
        }
      },
      { threshold: 0.3 }
    );
    io.observe(video);
    return () => io.disconnect();
  }, []);

  return (
    <section id="demo" className="demo-section reveal-once" ref={wrapRef}>
      <div className="container">
        <div className="demo-grid">
          <div className="demo-content">
            <span className="section-tag">DEMO EN VIVO</span>
            <h2 className="section-title">
              Mira cómo{" "}
              <span className="gradient-text">detectamos errores</span> en tiempo
              real
            </h2>
            <p className="demo-desc">
              Nuestro sistema de IA analiza cada frame, identifica
              posicionamientos subóptimos, uso ineficiente de utilidad y
              oportunidades perdidas.
            </p>

            <div className="demo-features">
              {[
                "Detección de errores en milisegundos",
                "Análisis de utilidad y economía",
                "Recomendaciones accionables",
              ].map((txt) => (
                <div key={txt} className="demo-feature">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                    <path
                      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                      stroke="#00ffc3"
                      strokeWidth="2"
                    />
                  </svg>
                  <span>{txt}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="demo-video-wrap">
            <div className="demo-video-frame">
              <video
                ref={videoRef}
                className="demo-video"
                src="/videos/demo.mp4"
                loop
                muted
                playsInline
                preload="metadata"
              />
              {!isPlaying && (
                <div className="video-play-overlay" aria-hidden>
                  <button
                    className="video-play-btn"
                    onClick={() => videoRef.current?.play()}
                    aria-label="Reproducir demo"
                  >
                    <svg width="60" height="60" viewBox="0 0 60 60" fill="none">
                      <circle
                        cx="30"
                        cy="30"
                        r="28"
                        stroke="white"
                        strokeWidth="2"
                      />
                      <path d="M24 20l16 10-16 10V20z" fill="white" />
                    </svg>
                  </button>
                </div>
              )}
            </div>

            <div className="demo-indicator" style={{ top: "15%", left: "10%" }}>
              <span className="indicator-dot"></span>
              <span className="indicator-text">Posición subóptima</span>
            </div>
            <div
              className="demo-indicator"
              style={{ top: "60%", right: "15%" }}
            >
              <span className="indicator-dot indicator-success"></span>
              <span className="indicator-text">Timing perfecto</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

// ============================================
// 🔥 TESTIMONIOS (marquee con pausa hover)
// ============================================
const SocialProof = () => {
  const wrapRef = useRef(null);
  useInView(wrapRef);

  const testimonials = [
    {
      name: "Álvaro G.",
      role: "IGL Amateur",
      text:
        "Pasé de perder 1v1 tontos a entender mis timings. +6% win-rate en 2 semanas.",
      avatar: "/images/avatars/1.png",
    },
    {
      name: "Lucía P.",
      role: "Entry Fragger",
      text:
        "Los drills personalizados son oro. Cada partida me enseña algo nuevo.",
      avatar: "/images/avatars/2.png",
    },
    {
      name: "Team FOCUS",
      role: "Mix de 5",
      text:
        "El análisis económico nos ordenó el mid-round. Mucha más claridad.",
      avatar: "/images/avatars/3.png",
    },
    {
      name: "Carlos M.",
      role: "AWPer",
      text: "Detectó errores que ni mi coach vio. Impresionante.",
      avatar: "/images/avatars/1.png",
    },
    {
      name: "Sara T.",
      role: "Support",
      text:
        "Las recomendaciones de utilidad cambiaron mi juego completamente.",
      avatar: "/images/avatars/2.png",
    },
  ];

  return (
    <section className="social-proof reveal-once" ref={wrapRef}>
      <div className="container">
        <div className="section-header centered">
          <span className="section-tag">TESTIMONIOS</span>
          <h2 className="section-title">
            Jugadores reales, <br />
            <span className="gradient-text">resultados reales</span>
          </h2>
        </div>

        <div className="testimonials-track" aria-live="polite">
          <div className="testimonials-scroll">
            {[...testimonials, ...testimonials].map((t, i) => (
              <figure key={i} className="testimonial-card">
                <blockquote className="testimonial-text">“{t.text}”</blockquote>
                <figcaption className="testimonial-author">
                  <img
                    src={t.avatar}
                    alt=""
                    className="testimonial-avatar"
                    loading="lazy"
                  />
                  <div>
                    <div className="testimonial-name">{t.name}</div>
                    <div className="testimonial-role">{t.role}</div>
                  </div>
                </figcaption>
              </figure>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};

// ============================================
// 💎 PRECIOS (toggle mensual/anual)
// ============================================
const Pricing = () => {
  const [yearly, setYearly] = useState(true);
  const prices = {
    Free: { m: 0, y: 0, period: "Gratis para siempre" },
    Pro: { m: 9, y: 84, period: "/mes ó /año" }, // 12x9 => 108, anual -22%
    Elite: { m: 19, y: 168, period: "/mes ó /año" }, // 12x19 => 228, anual -26%
  };

  const features = {
    Free: ["5 demos/mes", "Análisis básico", "Historial 30 días"],
    Pro: [
      "Demos ilimitadas",
      "IA avanzada",
      "Historial 180 días",
      "Integración Faceit",
    ],
    Elite: [
      "Todo de Pro",
      "Coach en tiempo real",
      "Análisis de equipo",
      "Soporte prioritario",
    ],
  };

  const plans = ["Free", "Pro", "Elite"];

  return (
    <section className="pricing-section reveal-once">
      <div className="container">
        <div className="section-header centered">
          <span className="section-tag">PRECIOS</span>
          <h2 className="section-title">
            Elige tu <span className="gradient-text">nivel</span>
          </h2>
          <p className="section-subtitle">Sin permanencia. Cancela cuando quieras.</p>

          <div className="billing-toggle" role="group" aria-label="Periodo de facturación">
            <button
              className={`toggle-btn ${!yearly ? "active" : ""}`}
              onClick={() => setYearly(false)}
              aria-pressed={!yearly}
            >
              Mensual
            </button>
            <button
              className={`toggle-btn ${yearly ? "active" : ""}`}
              onClick={() => setYearly(true)}
              aria-pressed={yearly}
            >
              Anual <span className="save-pill">-22%/-26%</span>
            </button>
          </div>
        </div>

        <div className="pricing-grid">
          {plans.map((p, i) => {
            const popular = p === "Pro";
            const amount = yearly ? prices[p].y : prices[p].m;
            const period = yearly ? "/año" : "/mes";
            return (
              <div key={p} className={`pricing-card ${popular ? "popular" : ""}`}>
                {popular && <div className="popular-badge">Más popular</div>}

                <div className="pricing-header">
                  <h3 className="pricing-name">{p}</h3>
                  <div className="pricing-price">
                    <span className="price-amount">
                      {amount}
                      {amount !== 0 ? "€" : ""}
                    </span>
                    <span className="price-period">
                      {amount === 0 ? prices[p].period : period}
                    </span>
                  </div>
                </div>

                <ul className="pricing-features">
                  {features[p].map((f) => (
                    <li key={f} className="pricing-feature">
                      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                        <path
                          d="M7 10l2 2 4-4m5 2a9 9 0 11-18 0 9 9 0 0118 0z"
                          stroke="#00ffc3"
                          strokeWidth="2"
                        />
                      </svg>
                      <span>{p === "Elite" && f === "Coach en tiempo real" ? "Coach IA en vivo (beta)" : f}</span>
                    </li>
                  ))}
                </ul>

                <button
                  className={`pricing-cta ${popular ? "primary" : "secondary"}`}
                  onClick={() =>
                    p === "Free" ? steamLogin() : steamLogin(p.toLowerCase())
                  }
                >
                  {p === "Free" ? "Empezar Gratis" : `Probar ${p}`}
                </button>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};

// ============================================
// 💬 FAQ
// ============================================
const FAQ = () => {
  const [openIndex, setOpenIndex] = useState(null);
  const faqs = [
    {
      q: "¿Necesito subir mis demos manualmente?",
      a:
        "No. Nuestro sistema extrae las demos automáticamente desde tu cuenta de Steam. También puedes subir demos específicas si lo prefieres.",
    },
    {
      q: "¿Funciona con CS:GO antiguo?",
      a: "Actualmente solo soportamos CS2. Las versiones anteriores no tienen soporte oficial.",
    },
    {
      q: "¿Puedo cancelar cuando quiera?",
      a:
        "Sí, sin permanencia. Tu plan se detendrá al final del ciclo de facturación sin cargos adicionales.",
    },
    {
      q: "¿Es segura mi información?",
      a:
        "Totalmente. Usamos cifrado AES-256 y cumplimos con RGPD. Las demos se eliminan tras 30-180 días según tu plan.",
    },
  ];

  return (
    <section className="faq-section reveal-once">
      <div className="container-narrow">
        <div className="section-header centered">
          <span className="section-tag">FAQ</span>
          <h2 className="section-title">
            Preguntas <span className="gradient-text">frecuentes</span>
          </h2>
        </div>

        <div className="faq-list">
          {faqs.map((faq, i) => (
            <div key={i} className={`faq-item ${openIndex === i ? "open" : ""}`}>
              <button
                className="faq-question"
                onClick={() => setOpenIndex(openIndex === i ? null : i)}
                aria-expanded={openIndex === i}
              >
                <span>{faq.q}</span>
                <svg className="faq-icon" width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M19 9l-7 7-7-7"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                  />
                </svg>
              </button>
              <div className="faq-answer" role="region">
                <p>{faq.a}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

// ============================================
// 🎯 CTA FINAL
// ============================================
const FinalCTA = () => {
  const ctaRef = useRef(null);
  useInView(ctaRef);
  return (
    <section className="final-cta reveal-once" ref={ctaRef}>
      <div className="cta-glow"></div>
      <div className="container">
        <div className="cta-content">
          <h2 className="cta-title">
            ¿Listo para <span className="gradient-text">subir de nivel?</span>
          </h2>
          <p className="cta-subtitle">
            Únete a más de 3,200 jugadores que ya están mejorando cada día
          </p>
          <a href="/auth/steam" className="cta-button-large">
            <span>Empezar ahora — Gratis</span>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M5 12h14m-7-7l7 7-7 7" stroke="currentColor" strokeWidth="2" />
            </svg>
          </a>
          <p className="cta-note">Sin tarjeta • Cancela cuando quieras</p>
        </div>
      </div>
    </section>
  );
};

// ============================================
// 🎨 FOOTER
// ============================================
const Footer = () => {
  return (
    <footer className="footer reveal-once">
      <div className="container">
        <div className="footer-content">
          <div className="footer-brand">
            <span className="brand-logo">StratAI</span>
            <p>AI-powered CS2 coaching</p>
          </div>
          <nav className="footer-links" aria-label="Legal">
            <a href="#">Términos</a>
            <a href="#">Privacidad</a>
            <a href="#">Contacto</a>
          </nav>
        </div>
        <div className="footer-copy">
          © {new Date().getFullYear()} StratAI. Todos los derechos reservados.
        </div>
      </div>
    </footer>
  );
};

// ============================================
// 🚀 LANDING (Export)
// ============================================
export default function LandingPage() {
   // fuerza smooth scroll sólo en esta vista + añade clases de scope
   useEffect(() => {
    const root = document.documentElement;
    const prev = root.style.scrollBehavior;

    // 🟢 AÑADIMOS estas dos líneas:
    document.documentElement.classList.add("landing-html");
    document.body.classList.add("landing-page");

    root.style.scrollBehavior = "smooth";

    // 🔴 y cuando sales del componente (p. ej. entras al Dashboard), se quitan:
    return () => {
      root.style.scrollBehavior = prev || "auto";
      document.documentElement.classList.remove("landing-html");
      document.body.classList.remove("landing-page");
    };
  }, []);


  // revelar on-scroll: activa 'reveal-once' -> 'in-view'
  useEffect(() => {
    const sections = document.querySelectorAll(".reveal-once");
    const io = new IntersectionObserver(
      (entries) =>
        entries.forEach((e) => {
          if (e.isIntersecting) {
            e.target.classList.add("in-view");
          }
        }),
      { threshold: 0.2 }
    );
    sections.forEach((s) => io.observe(s));
    return () => io.disconnect();
  }, []);

  return (
    <div className="landing-wrapper">
      <Hero />
      <PowerFeatures />
      <InteractiveDemo />
      <SocialProof />
      <Pricing />
      <FAQ />
      <FinalCTA />
      <Footer />
    </div>
  );
}
