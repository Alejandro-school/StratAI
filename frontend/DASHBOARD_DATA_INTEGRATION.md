# 📊 Integración de Datos Reales - Dashboard

## ✅ **INTEGRACIÓN COMPLETADA**

El Dashboard ahora usa **datos reales** de tus partidas procesadas en lugar de datos simulados.

---

## 🔄 **CÓMO FUNCIONA**

### 1. **Flujo de Datos**

```
Steam API → Sharecodes → Node.js Bot → Descarga Demos
                                              ↓
                                     Go Service (Parsea .dem)
                                              ↓
                                        Redis Storage
                                              ↓
                                    Python FastAPI Backend
                                              ↓
                                      React Frontend
```

### 2. **Endpoint Principal**
```javascript
GET /steam/get-processed-demos
// Requiere: sesión autenticada
// Retorna: Array de demos con estadísticas completas
```

### 3. **Hook Personalizado: `useDashboardData`**

Creado en `frontend/src/hooks/useDashboardData.js`, este hook:
- ✅ Obtiene demos procesadas desde el backend
- ✅ Filtra datos del jugador actual
- ✅ Calcula estadísticas agregadas
- ✅ Genera tendencias (comparando partidas recientes vs antiguas)
- ✅ Formatea datos para gráficos y tablas

---

## 📊 **ESTADÍSTICAS CALCULADAS**

### **KPIs Principales**
Todas calculadas automáticamente:

#### 1. **Headshot %**
- **Valor**: Promedio de `hs_percentage` de todas las partidas
- **Tendencia**: Compara últimas 5 partidas vs 5 anteriores
- **Cambio**: Diferencia porcentual

#### 2. **ADR (Average Damage per Round)**
- **Valor**: Promedio de `adr` de todas las partidas
- **Tendencia**: Compara últimas 5 partidas vs 5 anteriores
- **Cambio**: Diferencia absoluta

#### 3. **K/D Ratio**
- **Valor**: Promedio de `kd_ratio` de todas las partidas
- **Tendencia**: Compara últimas 5 partidas vs 5 anteriores
- **Cambio**: Diferencia decimal

#### 4. **Win Rate**
- **Valor**: `(Victorias / Total partidas) * 100`
- **Tendencia**: Compara últimas 5 partidas vs 5 anteriores
- **Cambio**: Diferencia porcentual

---

## 📈 **GRÁFICO DE RENDIMIENTO**

### **Datos Mostrados**
- **Últimas 7 partidas** (o menos si no hay suficientes)
- **Eje Y**: K/D Ratio y ADR
- **Eje X**: Número de partida

### **Origen de Datos**
```javascript
{
  match: 1-7,          // Índice de la partida
  kd: player.kd_ratio,  // Del campo kd_ratio
  adr: player.adr,      // Del campo adr
  hs: player.hs_percentage // Del campo hs_percentage
}
```

---

## 🎮 **TABLA DE PARTIDAS RECIENTES**

### **Columnas Mostradas**
| Columna | Origen | Formato |
|---------|--------|---------|
| **Resultado** | `team_score vs opponent_score` | W/L badge |
| **Mapa** | `map_name` | Texto |
| **Score** | `team_score-opponent_score` | "16-12" |
| **K/D** | `kd_ratio` | 2 decimales |
| **ADR** | `adr` | Redondeado |
| **HS%** | `hs_percentage` | Redondeado |
| **Fecha** | `date` | Relativa ("Hace 2 horas") |

### **Ordenamiento**
- Por fecha descendente (más reciente primero)
- Máximo 4 partidas mostradas

---

## 🗺️ **ESTADÍSTICAS POR MAPA**

### **Cálculo Automático**
```javascript
// Para cada mapa:
winRate = (wins / (wins + losses)) * 100

// Top 4 mapas más jugados
sortBy: total_matches DESC
```

### **Datos Mostrados**
- Nombre del mapa
- Récord (W-L)
- Win rate con código de colores:
  - 🟢 Verde: ≥ 55%
  - 🟡 Amarillo: 50-54%
  - 🔴 Rojo: < 50%

---

## 🔫 **ESTADÍSTICAS DE ARMAS**

### **Nota Actual**
Las estadísticas de armas están **estimadas** basadas en kills totales.

### **Futuro**
Cuando se implemente el tracking detallado de armas en el backend Go:
```go
type WeaponStats struct {
    Weapon     string
    Kills      int
    Shots      int
    Hits       int
    Accuracy   float64
}
```

---

## 🎯 **FORMATO DE DATOS DEL BACKEND**

### **Estructura de Demo Procesada**
```json
{
  "match_id": "ABC123",
  "map_name": "de_mirage",
  "date": "2024-10-15T10:30:00Z",
  "duration": "00:45:23",
  "team_score": 16,
  "opponent_score": 12,
  "players": [
    {
      "steamID": "76561198000000000",
      "name": "PlayerName",
      "team": "CT",
      "kills": 24,
      "assists": 5,
      "deaths": 17,
      "adr": 89.5,
      "hs_percentage": 65.2,
      "kd_ratio": 1.41,
      "flash_assists": 3,
      "accuracy": 24.5,
      ...
    }
  ]
}
```

---

## 💡 **FUNCIONALIDADES INTELIGENTES**

### **1. Manejo de Estados**

#### **Loading** 🔄
```
"Cargando datos..."
"Obteniendo tus estadísticas"
```

#### **Error** ❌
```
"No se pudieron cargar las estadísticas"
+ Botón para analizar demos
```

#### **Sin Datos** 📭
```
"No tienes partidas analizadas todavía"
+ Botón para subir primera demo
```

#### **Con Datos** ✅
Dashboard completo con todas las estadísticas

### **2. Cálculo de Tendencias**

```javascript
// Compara partidas recientes vs antiguas
recientes = últimas 5 partidas
antiguas = partidas 6-10

tendencia = "up" si recientes > antiguas
           "down" si recientes < antiguas
```

### **3. Fechas Relativas**

```javascript
< 1 hora  → "Hace menos de 1 hora"
< 24h     → "Hace X horas"
1 día     → "Ayer"
< 7 días  → "Hace X días"
< 30 días → "Hace X semanas"
> 30 días → "Hace X meses"
```

---

## 🚀 **CÓMO USAR**

### **Paso 1: Autenticarse**
```
1. Login con Steam
2. Sesión almacenada en cookies
```

### **Paso 2: Subir Demos**
```
1. Obtener sharecodes de Steam
2. Bot descarga demos automáticamente
3. Go service procesa las demos
4. Datos guardados en Redis
```

### **Paso 3: Ver Dashboard**
```
1. Navegar a /dashboard
2. Hook useDashboardData se ejecuta automáticamente
3. Datos cargados y mostrados
```

---

## 🔧 **PERSONALIZACIÓN**

### **Cambiar Número de Partidas en Gráfico**
```javascript
// En useDashboardData.js, línea ~70
const performanceData = userMatches.slice(0, 7) // Cambia 7 por el número deseado
```

### **Cambiar Partidas en Tabla**
```javascript
// En useDashboardData.js, línea ~80
const recentMatches = userMatches.slice(0, 4) // Cambia 4 por el número deseado
```

### **Ajustar Umbral de Tendencia**
```javascript
// En useDashboardData.js, línea ~130
const recent = matches.slice(0, 5) // Primeras 5
const older = matches.slice(5, 10) // Siguientes 5
```

---

## 📊 **VENTAJAS DE LA INTEGRACIÓN**

### ✅ **Datos Reales**
- No más simulaciones
- Estadísticas precisas
- Actualizaciones automáticas

### ✅ **Cálculos Inteligentes**
- Tendencias automáticas
- Comparación temporal
- Agregación de datos

### ✅ **Performance Optimizado**
- Un solo endpoint
- Cálculos en cliente
- Caché eficiente

### ✅ **UX Mejorada**
- Estados de carga claros
- Manejo de errores
- Feedback visual

---

## 🎯 **PRÓXIMAS MEJORAS**

### **1. Estadísticas de Armas Reales**
Cuando se implemente tracking detallado:
- Kills por arma
- Precisión real por arma
- Daño por arma

### **2. Caché Inteligente**
```javascript
// Cachear datos en localStorage
const cachedData = localStorage.getItem('dashboard_cache')
const cacheAge = Date.now() - cachedData.timestamp

if (cacheAge < 5 * 60 * 1000) { // 5 minutos
  return cachedData
}
```

### **3. Filtros Temporales**
```javascript
// Filtrar por período
const filters = {
  last7days: true,
  last30days: false,
  all: false
}
```

### **4. Comparación con Amigos**
```javascript
// Mostrar ranking entre amigos
const friendsRanking = [
  { username: "Tu", kd: 1.23, rank: 2 },
  { username: "Amigo1", kd: 1.45, rank: 1 },
  ...
]
```

---

## 📞 **TROUBLESHOOTING**

### **Problema: No aparecen estadísticas**

**Solución**:
1. Verificar que haya demos procesadas en Redis:
   ```bash
   redis-cli
   LRANGE processed_demos:YOUR_STEAM_ID 0 -1
   ```
2. Verificar endpoint:
   ```bash
   curl http://localhost:8000/steam/get-processed-demos \
     --cookie "session_cookie"
   ```

### **Problema: Datos desactualizados**

**Solución**:
1. Procesar nuevas demos
2. Refrescar la página (F5)
3. El hook se ejecuta automáticamente en cada mount

### **Problema: Tendencias incorrectas**

**Solución**:
Verificar que haya al menos 5 partidas:
```javascript
if (matches.length < 5) {
  // Tendencia se calcula con datos disponibles
}
```

---

## 🎉 **RESULTADO FINAL**

Ahora tienes un dashboard **profesional y dinámico** que:

✅ Muestra datos reales de tus partidas  
✅ Calcula estadísticas automáticamente  
✅ Genera tendencias inteligentes  
✅ Se actualiza con cada nueva demo  
✅ Maneja todos los estados correctamente  
✅ Ofrece una UX excepcional  

¡Tu dashboard ahora es una herramienta profesional de análisis! 🚀

