// frontend/src/components/Stats/Replays2D.jsx
import React, { useState } from "react";
import SectionHeader from "./common/SectionHeader";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { Play, Pause, RotateCcw, Eye, Clock, Target } from "lucide-react";
import "../../styles/Stats/replays2D.css";

/**
 * MOCK DATA - Ejemplo de estructura de datos esperada
 */
const MOCK_DATA = {
  matches: [
    { id: "match_1", map: "Mirage", date: "2025-10-14", rounds: 24 },
    { id: "match_2", map: "Inferno", date: "2025-10-13", rounds: 22 },
    { id: "match_3", map: "Dust2", date: "2025-10-12", rounds: 26 }
  ],
  events: [
    { tick: 0, time: "0:00", type: "round_start", description: "Inicio de ronda" },
    { tick: 1200, time: "0:12", type: "smoke", description: "Smoke CT en A Main" },
    { tick: 2400, time: "0:24", type: "flash", description: "Flash sobre A Site" },
    { tick: 3600, time: "0:36", type: "kill", description: "Kill con AK-47 en A Ramp" },
    { tick: 4800, time: "0:48", description: "Movimiento hacia B Site" },
    { tick: 6000, time: "1:00", type: "plant", description: "Bomba plantada en B" },
    { tick: 7200, time: "1:12", type: "kill", description: "Kill con AWP en B Window" },
    { tick: 8400, time: "1:24", type: "defuse", description: "Defuse iniciado" }
  ]
};

/**
 * Replays2D - Componente visor 2D de repeticiones de partidas
 * 
 * @param {Object} data - Datos de repeticiones { matches, events }
 * @param {boolean} loading - Estado de carga
 * @param {string} error - Mensaje de error si existe
 * @param {Function} onRetry - Función para reintentar carga
 */
export default function Replays2D({ data, loading = false, error = null, onRetry }) {
  const [selectedMatch, setSelectedMatch] = useState(null);
  const [selectedRound, setSelectedRound] = useState(1);
  const [isPlaying, setIsPlaying] = useState(false);
  const [showGrenades, setShowGrenades] = useState(true);
  
  // Usar mock data si no se proveen datos
  const statsData = data || MOCK_DATA;

  // Estado de carga
  if (loading) {
    return (
      <div className="r2d-container">
        <SectionHeader 
          title="Repeticiones 2D" 
          description="Visualiza rondas y jugadas en vista 2D"
        />
        <Card>
          <div className="r2d-loading">
            <div className="r2d-spinner"></div>
            <p>Cargando repeticiones...</p>
          </div>
        </Card>
      </div>
    );
  }

  // Estado de error
  if (error) {
    return (
      <div className="r2d-container">
        <SectionHeader 
          title="Repeticiones 2D" 
          description="Visualiza rondas y jugadas en vista 2D"
        />
        <Card>
          <ErrorState message={error} onRetry={onRetry} />
        </Card>
      </div>
    );
  }

  // Estado vacío
  if (!statsData.matches || statsData.matches.length === 0) {
    return (
      <div className="r2d-container">
        <SectionHeader 
          title="Repeticiones 2D" 
          description="Visualiza rondas y jugadas en vista 2D"
        />
        <Card>
          <EmptyState 
            icon={Play}
            title="No hay repeticiones disponibles"
            description="Sube demos para poder visualizar repeticiones 2D de tus partidas"
          />
        </Card>
      </div>
    );
  }

  const { matches, events } = statsData;

  // Handlers
  const handlePlayPause = () => {
    setIsPlaying(!isPlaying);
    // Aquí iría la lógica de reproducción del replay
  };

  const handleReset = () => {
    setIsPlaying(false);
    // Reiniciar el replay (currentTick se usaría cuando se implemente el replay real)
  };

  const handleMatchChange = (matchId) => {
    setSelectedMatch(matchId);
    setSelectedRound(1);
    setIsPlaying(false);
  };

  return (
    <div className="r2d-container">
      <SectionHeader 
        title="Repeticiones 2D" 
        description="Visualiza rondas y jugadas en vista 2D"
      />

      {/* Controls Top Bar */}
      <Card className="r2d-controls-card">
        <div className="r2d-controls-top">
          <div className="r2d-selectors">
            <div className="r2d-selector-group">
              <label className="r2d-selector-label">Partida:</label>
              <select 
                className="r2d-select" 
                value={selectedMatch || ""}
                onChange={(e) => handleMatchChange(e.target.value)}
              >
                <option value="">Seleccionar partida...</option>
                {matches.map(match => (
                  <option key={match.id} value={match.id}>
                    {match.map} - {match.date} ({match.rounds} rounds)
                  </option>
                ))}
              </select>
            </div>

            {selectedMatch && (
              <div className="r2d-selector-group">
                <label className="r2d-selector-label">Ronda:</label>
                <select 
                  className="r2d-select" 
                  value={selectedRound}
                  onChange={(e) => setSelectedRound(Number(e.target.value))}
                >
                  {Array.from({ length: 30 }, (_, i) => i + 1).map(round => (
                    <option key={round} value={round}>
                      Ronda {round}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          <div className="r2d-controls-actions">
            <button 
              className="r2d-control-btn r2d-btn-secondary"
              onClick={handleReset}
              disabled={!selectedMatch}
              title="Reset"
            >
              <RotateCcw size={18} />
            </button>
            <button 
              className={`r2d-control-btn ${isPlaying ? 'r2d-btn-danger' : 'r2d-btn-primary'}`}
              onClick={handlePlayPause}
              disabled={!selectedMatch}
            >
              {isPlaying ? <Pause size={18} /> : <Play size={18} />}
              <span>{isPlaying ? 'Pausar' : 'Reproducir'}</span>
            </button>
            <label className="r2d-toggle-label">
              <input 
                type="checkbox" 
                checked={showGrenades}
                onChange={(e) => setShowGrenades(e.target.checked)}
                className="r2d-checkbox"
              />
              <Eye size={18} />
              <span>Mostrar granadas</span>
            </label>
          </div>
        </div>
      </Card>

      {/* Main Content */}
      <div className="r2d-content-grid">
        {/* Canvas Area */}
        <Card className="r2d-canvas-card">
          <div className="r2d-canvas-placeholder">
            {selectedMatch ? (
              <>
                <div className="r2d-canvas-overlay">
                  <Target size={64} className="r2d-canvas-icon" />
                  <p className="r2d-canvas-text">Visor 2D</p>
                  <p className="r2d-canvas-subtext">
                    {isPlaying ? 'Reproduciendo...' : 'Presiona Play para comenzar'}
                  </p>
                </div>
                {/* Aquí iría el canvas real del mapa 2D */}
              </>
            ) : (
              <div className="r2d-canvas-empty">
                <Play size={64} className="r2d-canvas-icon" />
                <p className="r2d-canvas-text">Selecciona una partida</p>
                <p className="r2d-canvas-subtext">Elige una partida y ronda para comenzar</p>
              </div>
            )}
          </div>
        </Card>

        {/* Events Timeline */}
        <Card className="r2d-events-card">
          <h3 className="r2d-events-title">Eventos de la ronda</h3>
          <p className="r2d-events-subtitle">
            {selectedMatch ? `Ronda ${selectedRound}` : 'Sin ronda seleccionada'}
          </p>
          
          <div className="r2d-events-list">
            {selectedMatch && events ? (
              events.map((event, idx) => (
                <div key={idx} className="r2d-event-item">
                  <div className={`r2d-event-icon r2d-event-${event.type || 'default'}`}>
                    <Clock size={14} />
                  </div>
                  <div className="r2d-event-content">
                    <span className="r2d-event-time">{event.time}</span>
                    <span className="r2d-event-description">{event.description}</span>
                  </div>
                </div>
              ))
            ) : (
              <div className="r2d-events-empty">
                <Clock size={32} opacity={0.3} />
                <p>No hay eventos para mostrar</p>
              </div>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}

/**
 * EJEMPLO DE USO EN ROUTER:
 * 
 * import Replays2D from './components/Stats/Replays2D';
 * 
 * const replayData = {
 *   matches: [...],
 *   events: [...]
 * };
 * 
 * <Replays2D 
 *   data={replayData} 
 *   loading={false}
 *   error={null}
 *   onRetry={() => fetchData()}
 * />
 */




