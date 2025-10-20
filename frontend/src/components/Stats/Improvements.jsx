// frontend/src/components/Stats/Improvements.jsx
import React, { useState } from "react";
import SectionHeader from "./common/SectionHeader";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { Lightbulb, AlertTriangle, TrendingUp, MapPin, Crosshair, ChevronRight } from "lucide-react";
import "../../styles/Stats/improvements.css";

/**
 * MOCK DATA - Ejemplo de estructura de datos esperada
 */
const MOCK_DATA = {
  suggestions: [
    {
      id: 1,
      title: "Mejora tu control de retroceso con el AK-47",
      summary: "Tu precisión con el AK-47 está por debajo del promedio (42% vs 48% esperado). Practica el patrón de spray en aim maps.",
      map: "N/A",
      weapon: "AK-47",
      impact: "high",
      difficulty: "medium",
      category: "aim"
    },
    {
      id: 2,
      title: "Optimiza el uso de smokes en Mirage",
      summary: "Solo usas el 45% de tus smokes en Mirage. Aprende lineups clave para A y B site.",
      map: "Mirage",
      weapon: "N/A",
      impact: "high",
      difficulty: "easy",
      category: "utility"
    },
    {
      id: 3,
      title: "Mejora tu posicionamiento en Banana (Inferno)",
      summary: "Tienes un ratio de muertes alto en Banana. Practica posiciones off-angles y usa más mollys.",
      map: "Inferno",
      weapon: "N/A",
      impact: "medium",
      difficulty: "medium",
      category: "positioning"
    },
    {
      id: 4,
      title: "Incrementa tu impacto en rondas eco",
      summary: "Tu K/D en rondas eco es 0.68. Busca picks con pistolas y evita duelos desfavorables.",
      map: "General",
      weapon: "USP-S / Glock",
      impact: "medium",
      difficulty: "hard",
      category: "game-sense"
    },
    {
      id: 5,
      title: "Timing de rotaciones en CT",
      summary: "Rotas demasiado pronto en el 38% de las rondas. Espera más información antes de rotar.",
      map: "General",
      weapon: "N/A",
      impact: "high",
      difficulty: "medium",
      category: "game-sense"
    }
  ],
  mistakes: {
    smokes: { count: 12, description: "Smokes mal lanzados o sin impacto" },
    flashes: { count: 18, description: "Flashes que cegaron compañeros" },
    molotovs: { count: 8, description: "Mollys sin valor táctico" },
    noise: { count: 15, description: "Ruido innecesario revelando posición" },
    economy: { count: 6, description: "Compras sub-óptimas" }
  }
};

/**
 * Improvements - Componente de sugerencias y recomendaciones
 * 
 * @param {Object} data - Datos de mejoras { suggestions, mistakes }
 * @param {boolean} loading - Estado de carga
 * @param {string} error - Mensaje de error si existe
 * @param {Function} onRetry - Función para reintentar carga
 * @param {Function} onViewDetails - Función para ver detalles de sugerencia
 */
export default function Improvements({ 
  data, 
  loading = false, 
  error = null, 
  onRetry,
  onViewDetails 
}) {
  const [filterCategory, setFilterCategory] = useState("all");
  
  // Usar mock data si no se proveen datos
  const statsData = data || MOCK_DATA;

  // Estado de carga
  if (loading) {
    return (
      <div className="im-container">
        <SectionHeader 
          title="Mejoras" 
          description="Sugerencias personalizadas para mejorar tu juego"
        />
        <Card>
          <div className="im-loading">
            <div className="im-spinner"></div>
            <p>Analizando tu rendimiento...</p>
          </div>
        </Card>
      </div>
    );
  }

  // Estado de error
  if (error) {
    return (
      <div className="im-container">
        <SectionHeader 
          title="Mejoras" 
          description="Sugerencias personalizadas para mejorar tu juego"
        />
        <Card>
          <ErrorState message={error} onRetry={onRetry} />
        </Card>
      </div>
    );
  }

  // Estado vacío
  if (!statsData.suggestions || statsData.suggestions.length === 0) {
    return (
      <div className="im-container">
        <SectionHeader 
          title="Mejoras" 
          description="Sugerencias personalizadas para mejorar tu juego"
        />
        <Card>
          <EmptyState 
            icon={Lightbulb}
            title="No hay sugerencias disponibles"
            description="Juega más partidas para recibir recomendaciones personalizadas"
          />
        </Card>
      </div>
    );
  }

  const { suggestions, mistakes } = statsData;

  // Filtrar sugerencias por categoría
  const filteredSuggestions = filterCategory === "all" 
    ? suggestions 
    : suggestions.filter(s => s.category === filterCategory);

  // Funciones helper
  const getImpactColor = (impact) => {
    switch (impact) {
      case "high": return "#ef4444";
      case "medium": return "#f59e0b";
      case "low": return "#10b981";
      default: return "#94a3b8";
    }
  };

  const getImpactLabel = (impact) => {
    switch (impact) {
      case "high": return "Alto impacto";
      case "medium": return "Impacto medio";
      case "low": return "Bajo impacto";
      default: return "Impacto";
    }
  };

  const getDifficultyLabel = (difficulty) => {
    switch (difficulty) {
      case "easy": return "Fácil";
      case "medium": return "Medio";
      case "hard": return "Difícil";
      default: return "Dificultad";
    }
  };

  const getCategoryIcon = (category) => {
    switch (category) {
      case "aim": return <Crosshair size={16} />;
      case "utility": return <Lightbulb size={16} />;
      case "positioning": return <MapPin size={16} />;
      case "game-sense": return <TrendingUp size={16} />;
      default: return <Lightbulb size={16} />;
    }
  };

  return (
    <div className="im-container">
      <SectionHeader 
        title="Mejoras" 
        description="Sugerencias personalizadas para mejorar tu juego"
        actions={
          <select 
            className="im-category-select" 
            value={filterCategory}
            onChange={(e) => setFilterCategory(e.target.value)}
          >
            <option value="all">Todas las categorías</option>
            <option value="aim">Puntería</option>
            <option value="utility">Utilidad</option>
            <option value="positioning">Posicionamiento</option>
            <option value="game-sense">Game Sense</option>
          </select>
        }
      />

      {/* Suggestions Grid */}
      <div className="im-suggestions-grid">
        {filteredSuggestions.map((suggestion) => (
          <Card key={suggestion.id} className="im-suggestion-card" hoverable>
            <div className="im-suggestion-header">
              <div className="im-suggestion-category">
                {getCategoryIcon(suggestion.category)}
                <span>{suggestion.category.replace('-', ' ')}</span>
              </div>
              <div 
                className="im-suggestion-impact" 
                style={{ color: getImpactColor(suggestion.impact) }}
              >
                {getImpactLabel(suggestion.impact)}
              </div>
            </div>

            <h3 className="im-suggestion-title">{suggestion.title}</h3>
            <p className="im-suggestion-summary">{suggestion.summary}</p>

            <div className="im-suggestion-meta">
              {suggestion.map !== "N/A" && (
                <span className="im-meta-tag im-meta-map">
                  <MapPin size={14} />
                  {suggestion.map}
                </span>
              )}
              {suggestion.weapon !== "N/A" && (
                <span className="im-meta-tag im-meta-weapon">
                  <Crosshair size={14} />
                  {suggestion.weapon}
                </span>
              )}
              <span className="im-meta-tag im-meta-difficulty">
                {getDifficultyLabel(suggestion.difficulty)}
              </span>
            </div>

            <button 
              className="im-suggestion-btn"
              onClick={() => onViewDetails && onViewDetails(suggestion.id)}
            >
              <span>Ver cómo mejorar</span>
              <ChevronRight size={16} />
            </button>
          </Card>
        ))}
      </div>

      {/* Mistakes Section */}
      {mistakes && (
        <Card className="im-mistakes-card">
          <div className="im-mistakes-header">
            <div>
              <h3 className="im-mistakes-title">Errores frecuentes</h3>
              <p className="im-mistakes-subtitle">
                Áreas que requieren atención inmediata
              </p>
            </div>
          </div>

          <div className="im-mistakes-grid">
            {Object.entries(mistakes).map(([key, mistake]) => (
              <div key={key} className="im-mistake-item">
                <div className="im-mistake-icon">
                  <AlertTriangle size={20} />
                </div>
                <div className="im-mistake-content">
                  <div className="im-mistake-count">{mistake.count}</div>
                  <div className="im-mistake-type">{key.toUpperCase()}</div>
                  <div className="im-mistake-description">{mistake.description}</div>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}

/**
 * EJEMPLO DE USO EN ROUTER:
 * 
 * import Improvements from './components/Stats/Improvements';
 * 
 * const improvementsData = {
 *   suggestions: [...],
 *   mistakes: {...}
 * };
 * 
 * const handleViewDetails = (suggestionId) => {
 *   console.log('Ver detalles de sugerencia:', suggestionId);
 * };
 * 
 * <Improvements 
 *   data={improvementsData} 
 *   loading={false}
 *   error={null}
 *   onRetry={() => fetchData()}
 *   onViewDetails={handleViewDetails}
 * />
 */




