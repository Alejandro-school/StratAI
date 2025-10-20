# Componentes Stats - StratAI

Documentación completa de los nuevos componentes React para las secciones de estadísticas del proyecto StratAI.

## 📁 Estructura de Archivos

```
src/
├─ components/Stats/
│   ├─ PersonalPerformance.jsx       # Resumen de desempeño personal
│   ├─ MapPerformance.jsx            # Comparación por mapas
│   ├─ Replays2D.jsx                 # Visor 2D de repeticiones
│   ├─ AnalyzeDemos.jsx              # Subida y análisis de demos
│   ├─ Progress.jsx                  # Evolución temporal
│   ├─ Improvements.jsx              # Recomendaciones
│   ├─ index.js                      # Exportaciones centralizadas
│   └─ common/
│       ├─ SectionHeader.jsx         # Header reutilizable
│       ├─ Card.jsx                  # Contenedor con estilo glass
│       ├─ EmptyState.jsx            # Estado vacío
│       ├─ ErrorState.jsx            # Estado de error
│       └─ index.js                  # Exportaciones comunes
├─ styles/Stats/
│   ├─ personalPerformance.css       # Prefijo: pp-*
│   ├─ mapPerformance.css            # Prefijo: mp-*
│   ├─ replays2D.css                 # Prefijo: r2d-*
│   ├─ analyzeDemos.css              # Prefijo: ad-*
│   ├─ progress.css                  # Prefijo: pr-*
│   ├─ improvements.css              # Prefijo: im-*
│   └─ common.css                    # Prefijo: cm-*
```

## 🎨 Estética y Diseño

Todos los componentes mantienen la estética actual del dashboard:

### Paleta de colores
- **Background oscuro**: `#0f172a`, `#1e293b`
- **Texto**: `#f1f5f9`, `#cbd5e1`, `#94a3b8`
- **Primary (púrpura)**: `#8b5cf6`, `#a78bfa`
- **Success (verde)**: `#10b981`
- **Danger (rojo)**: `#ef4444`
- **Warning (amarillo)**: `#f59e0b`
- **Info (azul)**: `#3b82f6`

### Características de diseño
- Glass morphism con `backdrop-filter: blur(20px)`
- Bordes sutiles con transparencia `rgba(255, 255, 255, 0.1)`
- Hover suave con `transform: translateY(-2px)`
- Border radius: 12px-20px
- Responsive con Flexbox y CSS Grid

## 📦 Importación

### Opción 1: Importación individual
```javascript
import PersonalPerformance from './components/Stats/PersonalPerformance';
import MapPerformance from './components/Stats/MapPerformance';
```

### Opción 2: Importación centralizada
```javascript
import { 
  PersonalPerformance, 
  MapPerformance, 
  Replays2D,
  AnalyzeDemos,
  Progress,
  Improvements,
  // Componentes comunes
  SectionHeader,
  Card,
  EmptyState,
  ErrorState
} from './components/Stats';
```

## 🔧 Uso de Componentes

### 1. PersonalPerformance

**Objetivo**: Mostrar resumen general del jugador con KPIs, tendencias y top armas.

**Props**:
```javascript
{
  data: {
    summary: {
      kd: { value: 1.25, change: 0.15, trend: "up" },
      adr: { value: 82.4, change: 3.2, trend: "up" },
      hs: { value: 48.5, change: -1.2, trend: "down" },
      impact: { value: 1.18, change: 0.08, trend: "up" },
      winRate: { value: 58.3, change: 2.5, trend: "up" },
      clutches: { value: 12, change: 3, trend: "up" }
    },
    trends: [
      { period: "Sem 1", kd: 1.15, adr: 78, hs: 45 },
      // ...
    ],
    topWeapons: [
      { weapon: "AK-47", accuracy: 42.5, hs: 52.3, avgDamage: 95.2, kills: 342 },
      // ...
    ]
  },
  loading: false,
  error: null,
  onRetry: () => fetchData()
}
```

**Ejemplo**:
```javascript
<PersonalPerformance 
  data={statsData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
/>
```

---

### 2. MapPerformance

**Objetivo**: Comparar rendimiento en cada mapa con métricas clave.

**Props**:
```javascript
{
  data: {
    maps: [
      { 
        name: "Mirage", 
        matches: 45, 
        winRate: 62.2, 
        adr: 85.3, 
        kd: 1.32, 
        entrySuccess: 58.5, 
        clutchRate: 42.1 
      },
      // ...
    ],
    hotspots: [
      { 
        map: "Mirage", 
        position: "A Site - Default plant", 
        risk: "high", 
        deaths: 15, 
        description: "Frecuentes muertes desde Palace" 
      },
      // ...
    ]
  },
  loading: false,
  error: null,
  onRetry: () => fetchData()
}
```

**Ejemplo**:
```javascript
<MapPerformance 
  data={mapsData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
/>
```

---

### 3. Replays2D

**Objetivo**: Visor 2D de repeticiones con controles de reproducción.

**Props**:
```javascript
{
  data: {
    matches: [
      { id: "match_1", map: "Mirage", date: "2025-10-14", rounds: 24 },
      // ...
    ],
    events: [
      { tick: 0, time: "0:00", type: "round_start", description: "Inicio de ronda" },
      { tick: 1200, time: "0:12", type: "smoke", description: "Smoke CT en A Main" },
      // ...
    ]
  },
  loading: false,
  error: null,
  onRetry: () => fetchData()
}
```

**Ejemplo**:
```javascript
<Replays2D 
  data={replayData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
/>
```

---

### 4. AnalyzeDemos

**Objetivo**: Subir y procesar demos con drag & drop.

**Props**:
```javascript
{
  data: {
    queue: [
      { 
        id: "demo_1", 
        filename: "match_mirage.dem", 
        size: "45.2 MB",
        status: "completed", // "processing", "queued", "error"
        uploadDate: "2025-10-14 15:30",
        map: "Mirage",
        rounds: 24
      },
      // ...
    ]
  },
  loading: false,
  error: null,
  onRetry: () => fetchData(),
  onProcess: (demoId) => processDemo(demoId),
  onViewResults: (demoId) => viewResults(demoId),
  onUpload: (files) => uploadFiles(files),
  onDelete: (demoId) => deleteDemo(demoId)
}
```

**Ejemplo**:
```javascript
<AnalyzeDemos 
  data={demosData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
  onUpload={handleUpload}
  onProcess={handleProcess}
  onViewResults={handleViewResults}
  onDelete={handleDelete}
/>
```

---

### 5. Progress

**Objetivo**: Mostrar evolución temporal con métricas, hitos y partidas recientes.

**Props**:
```javascript
{
  data: {
    series: {
      kd: { current: 1.25, previous: 1.18, change: 5.9, trend: "up" },
      adr: { current: 82.4, previous: 78.2, change: 5.4, trend: "up" },
      hs: { current: 48.5, previous: 49.2, change: -1.4, trend: "down" },
      impact: { current: 1.18, previous: 1.12, change: 5.4, trend: "up" }
    },
    milestones: [
      { 
        date: "2025-10-10", 
        title: "Primera ACE", 
        description: "5K en Mirage",
        type: "achievement" // "stat", "milestone"
      },
      // ...
    ],
    recent: [
      { 
        date: "2025-10-14", 
        map: "Mirage", 
        result: "Victoria", 
        score: "16-12",
        adr: 88.5, 
        kd: 1.42 
      },
      // ...
    ]
  },
  loading: false,
  error: null,
  onRetry: () => fetchData()
}
```

**Ejemplo**:
```javascript
<Progress 
  data={progressData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
/>
```

---

### 6. Improvements

**Objetivo**: Mostrar recomendaciones accionables y errores frecuentes.

**Props**:
```javascript
{
  data: {
    suggestions: [
      {
        id: 1,
        title: "Mejora tu control de retroceso con el AK-47",
        summary: "Tu precisión está por debajo del promedio...",
        map: "N/A",
        weapon: "AK-47",
        impact: "high", // "medium", "low"
        difficulty: "medium", // "easy", "hard"
        category: "aim" // "utility", "positioning", "game-sense"
      },
      // ...
    ],
    mistakes: {
      smokes: { count: 12, description: "Smokes mal lanzados" },
      flashes: { count: 18, description: "Flashes que cegaron compañeros" },
      // ...
    }
  },
  loading: false,
  error: null,
  onRetry: () => fetchData(),
  onViewDetails: (suggestionId) => viewDetails(suggestionId)
}
```

**Ejemplo**:
```javascript
<Improvements 
  data={improvementsData}
  loading={isLoading}
  error={error}
  onRetry={handleRetry}
  onViewDetails={handleViewDetails}
/>
```

---

## 🔀 Integración con Router

### Ejemplo con React Router v6

```javascript
// App.js o router principal
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import {
  PersonalPerformance,
  MapPerformance,
  Replays2D,
  AnalyzeDemos,
  Progress,
  Improvements
} from './components/Stats';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/stats/personal" element={<PersonalPerformance />} />
        <Route path="/stats/maps" element={<MapPerformance />} />
        <Route path="/stats/replays" element={<Replays2D />} />
        <Route path="/stats/demos" element={<AnalyzeDemos />} />
        <Route path="/stats/progress" element={<Progress />} />
        <Route path="/stats/improvements" element={<Improvements />} />
      </Routes>
    </BrowserRouter>
  );
}
```

### Integración con Sidebar existente

```javascript
// Sidebar.js
const menuItems = [
  // ... items existentes
  { path: '/stats/personal', label: 'Desempeño Personal', icon: Target },
  { path: '/stats/maps', label: 'Rendimiento por Mapa', icon: MapPin },
  { path: '/stats/replays', label: 'Repeticiones 2D', icon: Play },
  { path: '/stats/demos', label: 'Analizar Demos', icon: Upload },
  { path: '/stats/progress', label: 'Progreso', icon: TrendingUp },
  { path: '/stats/improvements', label: 'Mejoras', icon: Lightbulb }
];
```

## 🛠️ Componentes Comunes

### SectionHeader
```javascript
<SectionHeader 
  title="Título de la sección" 
  description="Descripción breve"
  actions={
    <button className="action-btn">Acción</button>
  }
/>
```

### Card
```javascript
<Card hoverable>
  <h3>Contenido</h3>
  <p>Descripción</p>
</Card>
```

### EmptyState
```javascript
<EmptyState 
  icon={FileQuestion}
  title="No hay datos"
  description="Mensaje explicativo"
  action={<button>Acción</button>}
/>
```

### ErrorState
```javascript
<ErrorState 
  message="Error al cargar datos"
  onRetry={handleRetry}
  retryLabel="Reintentar"
/>
```

## 📱 Responsive Design

Todos los componentes son completamente responsive:

- **Desktop** (>1024px): Grid completo con todas las columnas
- **Tablet** (768px-1024px): Grid adaptado a 2 columnas o columna única
- **Mobile** (<768px): Diseño de columna única, controles apilados

## 🎯 Estados Soportados

Todos los componentes principales manejan 3 estados:

1. **Loading**: Spinner con mensaje de carga
2. **Error**: Mensaje de error con botón "Reintentar"
3. **Empty**: Estado vacío con icono y mensaje explicativo
4. **Success**: Contenido completo con datos

## 🔍 Mock Data

Cada componente incluye datos de ejemplo (mock data) en el propio archivo para facilitar el desarrollo y testing sin necesidad de backend.

Para usar mock data durante desarrollo:
```javascript
// El componente usa mock data si no se pasa la prop "data"
<PersonalPerformance />
```

Para usar datos reales:
```javascript
<PersonalPerformance data={realData} />
```

## ✅ Checklist de Implementación

- [x] Componentes JSX creados
- [x] Estilos CSS dedicados
- [x] Componentes comunes reutilizables
- [x] Mock data incluido
- [x] Estados de carga/error/vacío
- [x] Diseño responsive
- [x] Prefijos de clase únicos
- [x] Comentarios explicativos
- [x] Ejemplos de uso documentados
- [x] Sin errores de linting
- [x] Sin dependencias externas nuevas

## 🚀 Próximos Pasos

1. **Integrar con el Router**: Añadir las rutas en tu archivo principal de rutas
2. **Conectar con API**: Reemplazar mock data con llamadas reales al backend
3. **Testing**: Probar cada componente en diferentes dispositivos
4. **Personalizar**: Ajustar colores, textos o funcionalidades según necesites

## 📞 Soporte

Para cualquier duda sobre la implementación o personalización de estos componentes, revisa:
- Los comentarios JSDoc en cada archivo
- Los ejemplos de uso al final de cada componente
- El mock data incluido como referencia de estructura

---

**Creado por**: Sistema de generación de componentes StratAI  
**Fecha**: Octubre 2025  
**Versión**: 1.0.0




