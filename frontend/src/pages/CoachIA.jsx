// frontend/src/pages/CoachIA.jsx
// AI Coach Hub - Chat inteligente y análisis de partidas
import React, { useState } from 'react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { 
  Brain, MessageCircle, Play, Sparkles,
  Lightbulb, Target, TrendingUp, Send,
  ChevronRight, Zap, Map
} from 'lucide-react';
import '../styles/pages/coachIA.css';

const CoachIA = () => {
  const [chatInput, setChatInput] = useState('');

  const quickPrompts = [
    { icon: Target, text: "¿Cómo mejoro mi aim?" },
    { icon: Map, text: "Consejos para Mirage" },
    { icon: Zap, text: "¿Por qué pierdo clutches?" },
    { icon: TrendingUp, text: "Analiza mi última partida" },
  ];

  return (
    <NavigationFrame>
      <div className="coach-container">
        {/* Hero Section */}
        <header className="coach-hero">
          <div className="hero-glow" />
          <div className="hero-content">
            <div className="hero-badge">
              <Sparkles size={14} />
              <span>Powered by AI</span>
            </div>
            <h1>
              <Brain className="hero-icon" />
              Tu Coach IA Personal
            </h1>
            <p>
              Análisis inteligente de tu gameplay y sugerencias personalizadas para mejorar
            </p>
          </div>
        </header>

        {/* Main Content */}
        <main className="coach-content">
          {/* Feature Cards */}
          <section className="coach-features">
            {/* Match Analysis Card */}
            <div className="feature-card primary">
              <div className="feature-icon">
                <Play size={28} />
              </div>
              <div className="feature-info">
                <h2>Análisis de Partida</h2>
                <p>
                  Replay 2D interactivo con insights en tiempo real. 
                  La IA comenta tus jugadas mientras revés la partida.
                </p>
                <ul className="feature-list">
                  <li><Lightbulb size={14} /> Insights contextuales</li>
                  <li><Target size={14} /> Errores de posicionamiento</li>
                  <li><TrendingUp size={14} /> Oportunidades perdidas</li>
                </ul>
              </div>
              <button className="feature-cta">
                Seleccionar partida
                <ChevronRight size={16} />
              </button>
            </div>

            {/* Chat Card */}
            <div className="feature-card secondary">
              <div className="feature-icon">
                <MessageCircle size={28} />
              </div>
              <div className="feature-info">
                <h2>Chat con Coach</h2>
                <p>
                  Pregunta cualquier cosa sobre tu gameplay. 
                  El coach conoce todas tus estadísticas.
                </p>
              </div>

              {/* Mini Chat Preview */}
              <div className="mini-chat">
                <div className="chat-messages">
                  <div className="chat-message ai">
                    <Brain size={16} />
                    <span>¡Hola! Soy tu coach IA. ¿En qué puedo ayudarte hoy?</span>
                  </div>
                </div>

                <div className="quick-prompts">
                  {quickPrompts.map((prompt, i) => {
                    const Icon = prompt.icon;
                    return (
                      <button key={i} className="quick-prompt">
                        <Icon size={14} />
                        <span>{prompt.text}</span>
                      </button>
                    );
                  })}
                </div>

                <div className="chat-input-container">
                  <input
                    type="text"
                    placeholder="Escribe tu pregunta..."
                    value={chatInput}
                    onChange={(e) => setChatInput(e.target.value)}
                    className="chat-input"
                  />
                  <button className="chat-send" disabled={!chatInput.trim()}>
                    <Send size={18} />
                  </button>
                </div>
              </div>
            </div>
          </section>

          {/* Coming Soon Info */}
          <section className="coach-info">
            <div className="info-card">
              <Sparkles className="info-icon" />
              <div>
                <h3>Próximamente</h3>
                <p>
                  El coach IA está en desarrollo activo. Pronto podrás tener 
                  conversaciones detalladas sobre tu gameplay y recibir 
                  planes de entrenamiento personalizados.
                </p>
              </div>
            </div>
          </section>
        </main>
      </div>
    </NavigationFrame>
  );
};

export default CoachIA;
