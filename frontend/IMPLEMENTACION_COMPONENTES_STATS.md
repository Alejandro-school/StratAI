# Guía de Implementación - Componentes Stats con Datos Reales

## ✅ Componentes YA Implementados

### 1. PersonalPerformance ✅
- Hook: `usePersonalPerformance.js`
- Componente: Actualizado con Header + Sidebar + datos reales
- Muestra: KPIs detallados, tendencias semanales, top 5 armas con stats reales

### 2. MapPerformance (En proceso)
- Hook: `useMapPerformance.js` creado
- Componente: Pendiente de actualizar

## 📋 Patrón a Seguir para TODOS los Componentes

### Estructura del Hook Personalizado

```javascript
// frontend/src/hooks/use[NombreComponente].js
import { useState, useEffect } from 'react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export const use[NombreComponente] = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);

  useEffect(() => {
    const steamId = user?.steam_id || user?.steamid || user?.steamID;
    
    if (!steamId) {
      setLoading(false);
      return;
    }

    const fetchData = async () => {
      try {
        setLoading(true);
        
        // Llamar al endpoint real
        const response = await fetch(`${API_URL}/get-processed-demos?steam_id=${steamId}`);
        
        if (!response.ok) {
          throw new Error('No se pudieron obtener las demos');
        }

        const result = await response.json();
        const demos = result.demos || [];

        if (demos.length === 0) {
          setData(null);
          setLoading(false);
          return;
        }

        // Procesar datos específicos para este componente
        const processedData = process[Nombre]Data(demos, steamId);
        setData(processedData);
        setError(null);
      } catch (err) {
        console.error('Error:', err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [user]);

  return { data, loading, error };
};

// Función para procesar los datos según el componente
const process[Nombre]Data = (demos, userSteamId) => {
  // Extraer datos del jugador de cada demo
  const userMatches = demos
    .filter(demo => demo.players && demo.players.length > 0)
    .map(demo => {
      const player = demo.players.find(p => 
        String(p.steamID || p.steam_id || p.steamid).trim() === String(userSteamId).trim()
      );
      
      if (!player) return null;

      return {
        // Datos relevantes para el componente
        matchId: demo.match_id,
        map: demo.map_name,
        date: demo.date,
        kills: player.kills || 0,
        deaths: player.deaths || 0,
        // ... más datos
      };
    })
    .filter(Boolean);

  // Procesar y estructurar datos según necesite el componente
  return {
    // ... estructura de datos para el componente
  };
};
```

### Estructura del Componente JSX

```javascript
// frontend/src/components/Stats/[NombreComponente].jsx
import React, { useState } from "react";
import Header from "../Layout/Header";
import SidebarComponent from "../Layout/Sidebar";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { useUser } from "../../context/UserContext";
import { use[NombreComponente] } from "../../hooks/use[NombreComponente]";
import "../../styles/Stats/[nombreComponente].css";

export default function [NombreComponente]() {
  const { user } = useUser();
  const { data, loading, error } = use[NombreComponente](user);
  const [filters, setFilters] = useState({});

  // Estado de carga
  if (loading) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="[prefijo]-container">
                <div className="[prefijo]-loading">
                  <div className="[prefijo]-spinner"></div>
                  <p>Cargando...</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Estado de error
  if (error) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="[prefijo]-container">
                <Card>
                  <ErrorState 
                    message={error} 
                    onRetry={() => window.location.reload()} 
                  />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Estado vacío
  if (!data) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="[prefijo]-container">
                <Card>
                  <EmptyState 
                    icon={IconoApropiado}
                    title="No hay datos"
                    description="Mensaje apropiado"
                  />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Contenido principal
  return (
    <>
      <Header />
      <div className="dashboard-layout">
        <SidebarComponent />
        <div className="main-content">
          <div className="main-scroll">
            <div className="[prefijo]-container">
              {/* Header del componente */}
              <div className="dashboard-header">
                <div className="header-left">
                  <h1 className="dashboard-title">
                    Título <span className="highlight">Destacado</span>
                  </h1>
                  <p className="dashboard-subtitle">
                    Descripción del componente
                  </p>
                </div>
                <div className="header-actions">
                  {/* Filtros y acciones */}
                </div>
              </div>

              {/* Contenido específico del componente */}
              {/* Usar Cards, grids, etc. según la estética del Dashboard */}
              
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
```

## 🔧 Componentes Pendientes de Actualizar

### 1. MapPerformance
**Hook creado:** ✅ `useMapPerformance.js`
**Componente:** ❌ Pendiente

**Datos a mostrar (MÁS detallados que Dashboard):**
- Tabla completa de TODOS los mapas (no solo top 4)
- Win rate, ADR, K/D, Entry success rate, Clutch rate POR MAPA
- Heatmap de rendimiento por zonas (si hay datos disponibles)
- Comparativa entre lado T y CT por mapa
- Histórico de rendimiento en cada mapa

**Actualizar:** `frontend/src/components/Stats/MapPerformance.jsx`

### 2. Replays2D
**Hook a crear:** `useReplays2D.js`

**Endpoint:** Obtener lista de demos y sus eventos
```javascript
const response = await fetch(`${API_URL}/get-processed-demos?steam_id=${steamId}`);
// Acceder a demo.event_logs para los eventos
```

**Datos a mostrar:**
- Lista de partidas con selector
- Selector de rondas
- Timeline de eventos reales de la ronda seleccionada
- Canvas placeholder para el visor 2D
- Panel de eventos con timestamps reales de la demo

### 3. AnalyzeDemos
**Hook a crear:** `useAnalyzeDemos.js`

**Endpoints:**
- GET: Obtener lista de demos subidas
- POST: Subir nueva demo
- GET: Estado de procesamiento

**Datos a mostrar:**
- Lista REAL de demos subidas con su estado
- Drag & drop funcional para subir
- Estado real del procesamiento (queued, processing, completed, error)
- Botón para procesar demos

### 4. Progress
**Hook a crear:** `useProgress.js`

**Datos a mostrar (MÁS detallados):**
- Gráficos de evolución de TODAS las métricas a lo largo del tiempo
- Comparativa mes a mes, semana a semana
- Hitos reales (primera ACE, mejor K/D, mejor partida, etc.)
- Tabla de TODAS las partidas recientes (no solo 5)
- Análisis de racha actual (winning streak, losing streak)
- Mejora porcentual en cada métrica

### 5. Improvements
**Hook a crear:** `useImprovements.js`

**Datos a mostrar (REALES y específicos):**
- Análisis automático de debilidades basado en datos reales:
  * Armas con peor rendimiento
  * Mapas con peor win rate
  * Zonas donde mueres más
  * Errores tácticos detectados en los eventos
- Sugerencias específicas basadas en los datos
- Contadores reales de errores extraídos de event_logs:
  * Team kills
  * Flashes que cegaron aliados
  * Granadas desperdiciadas
  * Muertes por ruido excesivo
- Recomendaciones priorizadas por impacto

## 📊 Estructura de Datos Disponibles

### Demo Object
```javascript
{
  match_id: "string",
  map_name: "de_mirage",
  date: "2025-10-17",
  duration: "00:45:30",
  team_score: 16,
  opponent_score: 12,
  players: [
    {
      steamID: "string",
      name: "string",
      team: "Terrorist" | "CounterTerrorist",
      kills: number,
      deaths: number,
      assists: number,
      kd_ratio: number,
      adr: number,
      hs_percentage: number,
      mvp: number,
      clutch_wins: number,
      entry_kills: number,
      trade_kills: number,
      utility_damage: number,
      flash_assists: number,
      double_kills: number,
      triple_kills: number,
      quad_kills: number,
      ace: number
    }
  ],
  event_logs: {
    events_by_round: [
      {
        round_number: number,
        events: [
          {
            event_type: "Kill" | "Death" | "BombPlanted" | "BombDefused" | ...,
            tick: number,
            time: "MM:SS",
            details: "string con info parseada"
          }
        ]
      }
    ]
  }
}
```

## 🎨 Mantener Estética del Dashboard

### Clases CSS a reutilizar:
```css
/* Layout */
.dashboard-layout
.main-content
.main-scroll

/* Headers */
.dashboard-header
.header-left
.dashboard-title
.dashboard-subtitle
.highlight
.header-actions

/* Cards */
Usar el componente <Card> de common/

/* KPIs */
Seguir patrón de .kpi-card, .kpi-icon, .kpi-value, etc.

/* Grids */
.dashboard-grid
.grid-item
.grid-large
.grid-medium
.grid-full

/* Tablas */
Seguir estructura de matches-table del Dashboard
```

### Colores a usar:
```css
--dashboard-primary: #8b5cf6
--dashboard-success: #10b981
--dashboard-danger: #ef4444
--dashboard-warning: #f59e0b
--dashboard-text: #f1f5f9
--dashboard-text-secondary: #94a3b8
```

## ✅ Checklist de Implementación

Para cada componente:

- [ ] Crear hook en `frontend/src/hooks/use[Nombre].js`
- [ ] Hook obtiene datos reales del backend
- [ ] Hook procesa datos específicos para el componente
- [ ] Actualizar componente JSX con estructura completa (Header + Sidebar)
- [ ] Usar `useUser` para obtener steam_id
- [ ] Usar hook personalizado para datos
- [ ] Manejar estados: loading, error, vacío
- [ ] Mostrar datos MÁS detallados que el Dashboard
- [ ] Mantener estética exacta del proyecto
- [ ] Verificar que no hay errores de linting
- [ ] Probar con datos reales del backend

## 🚀 Orden Recomendado de Implementación

1. ✅ PersonalPerformance (COMPLETO)
2. ⏳ MapPerformance (Hook listo, falta actualizar componente)
3. Progress (Similar a PersonalPerformance)
4. Improvements (Análisis de datos)
5. Replays2D (Más complejo, necesita eventos)
6. AnalyzeDemos (Necesita endpoints de upload)

## 📝 Notas Importantes

- **NO** usar mock data, SOLO datos reales del backend
- **SIEMPRE** incluir Header y Sidebar en cada componente
- Mostrar información **MÁS DETALLADA** que el Dashboard
- Mantener la **estética exacta** del proyecto actual
- Usar los **mismos patrones CSS** del Dashboard
- Verificar que no hay **errores de linting** antes de finalizar

---

**Última actualización:** Octubre 2025
**Estado:** PersonalPerformance completo, MapPerformance en proceso

