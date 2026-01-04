# Phase 1 Integration Rules - StratAI

## ❌ NO Crear Archivos JSON Separados

Los datos de Phase 1 deben **integrarse** en los archivos JSON existentes para proporcionar contexto completo, NO crear archivos separados.

---

## 1. Weapon States → `combat.json`

### Concepto
Registrar el estado del arma **antes y después** de cada evento de combate (disparo, kill, damage) para entender la gestión de munición durante duelos.

### Casos de Uso
- **Inicio de spray**: Jugador empieza a disparar con 30 balas en AK-47
- **Fin de spray**: Jugador deja de disparar, quedan 20 balas
- **Recarga durante duelo**: Detectar si recarga con 20 balas restantes (mala gestión) vs esperar a tener 5
- **Análisis de duelo**: ¿Tenía suficientes balas? ¿Recargó en mal momento? ¿Se quedó sin munición?

### Implementación
```go
type WeaponStateSnapshot struct {
    AmmoInMag   int    `json:"ammo_in_mag"`
    AmmoReserve int    `json:"ammo_reserve"`
    IsReloading bool   `json:"is_reloading"`
    ZoomLevel   int    `json:"zoom_level"`
}

type KillEvent struct {
    // ... campos existentes ...
    
    // NUEVO: Estado del arma del killer
    WeaponStateBefore *WeaponStateSnapshot `json:"weapon_state_before,omitempty"`
    WeaponStateAfter  *WeaponStateSnapshot `json:"weapon_state_after,omitempty"`
}

type DamageEvent struct {
    // ... campos existentes ...
    
    // NUEVO: Estado del arma del attacker
    WeaponState *WeaponStateSnapshot `json:"weapon_state,omitempty"`
}
```

### Lógica de Captura
1. **En WeaponFire**: Capturar estado ANTES del disparo
2. **En PlayerHurt/Kill**: Asociar el último estado capturado
3. **En cada tick de combate**: Actualizar estado del arma activa
4. **En Reload**: Registrar inicio/fin de recarga

---

## 2. Spotting Events → `movement.json`

### Concepto
Registrar cuándo un jugador ve a un enemigo en el radar/minimapa para correlacionar con su posición exacta en ese momento.

### Casos de Uso
- **Awareness**: ¿El jugador vio al enemigo en el radar antes del duelo?
- **Información**: ¿Dónde estaba cuando lo spotteó?
- **Rotaciones**: ¿Vio enemigos en el radar y rotó correctamente?

### Implementación
```go
type TickMovement struct {
    Tick    int         `json:"tick"`
    Players []PlayerPos `json:"players"`
    
    // NUEVO: Spotting events en este tick
    SpottingEvents []SpottingEvent `json:"spotting_events,omitempty"`
}

type SpottingEvent struct {
    SpotterSteamID uint64  `json:"spotter_steam_id"`
    SpotterName    string  `json:"spotter_name"`
    SpottedSteamID uint64  `json:"spotted_steam_id"`
    SpottedName    string  `json:"spotted_name"`
    Distance       float64 `json:"distance"`
    
    // NUEVO: Posiciones en el momento del spotting
    SpotterPosition Position `json:"spotter_position"`
    SpottedPosition Position `json:"spotted_position"`
}
```

### Lógica de Captura
1. **En PlayerSpottersChanged**: Detectar nuevo spotting
2. **Capturar posiciones**: De ambos jugadores en ese instante
3. **Asociar con movement tick**: Añadir al tick más cercano en movement.json

---

## 3. Defuse Kit → Solo Registro de Pickup

### Concepto
El kit de desactivación **siempre se usa** si lo tienes al desactivar. Solo necesitamos saber **cuándo lo recoge** del suelo.

### ❌ NO Registrar
- Uso del kit durante defuse (obvio si lo tiene)
- Compra del kit (ya está en `economy.json`)

### ✅ SÍ Registrar
- **Pickup del kit** cuando lo recoge de un compañero muerto

### Implementación
```go
type DefuseKitPickup struct {
    Tick          int    `json:"tick"`
    Round         int    `json:"round"`
    PlayerSteamID uint64 `json:"player_steam_id"`
    PlayerName    string `json:"player_name"`
    PickedFrom    string `json:"picked_from"` // Nombre del jugador muerto
    X             float64 `json:"x"`
    Y             float64 `json:"y"`
    Z             float64 `json:"z"`
}

// Añadir a economy.json o crear events específicos
```

### Lógica de Captura
1. **En ItemPickup**: Detectar cuando `item.Type == EqDefuseKit`
2. **Verificar que NO sea compra**: Comprobar que no está en buy time
3. **Registrar pickup**: Guardar quién lo recoge y de dónde

---

## 4. Round Number Correcto

### Bug Actual
Todos los eventos aparecen como `"round": 0`

### Fix
Usar `ctx.ActualRoundNumber` o `ctx.RoundTimelines[currentRound].RoundNumber` en lugar de hardcodear 0.

```go
// ❌ INCORRECTO
Round: 0

// ✅ CORRECTO
Round: ctx.ActualRoundNumber
```

---

## Resumen de Cambios Necesarios

### Archivos a Modificar

1. **models/events.go**
   - Añadir `WeaponStateSnapshot` struct
   - Añadir campos `WeaponStateBefore/After` a `KillEvent`
   - Añadir campo `WeaponState` a `DamageEvent`
   - Modificar `SpottingEvent` para incluir posiciones
   - Crear `DefuseKitPickup` struct

2. **handlers/combat.go**
   - Capturar weapon state en WeaponFire
   - Asociar weapon state a KillEvent
   - Asociar weapon state a DamageEvent

3. **handlers/player.go**
   - **ELIMINAR** weapon state sampling periódico
   - Integrar spotting events con posiciones en movement

4. **handlers/bomb.go**
   - **ELIMINAR** tracking de defuse kit usage
   - **AÑADIR** tracking de defuse kit pickup (ItemPickup event)

5. **parser/timeline_exporter.go**
   - **ELIMINAR** `exportPhase1Data()` function
   - Integrar weapon states en `exportCombat()`
   - Integrar spotting events en `exportMovement()`
   - Añadir defuse kit pickups en timeline events

---

## Filosofía de Integración

> **"Los datos deben vivir junto al contexto que les da significado"**

- Weapon states → Combat (contexto de duelo)
- Spotting → Movement (contexto de posición)
- Kit pickup → Timeline events (evento puntual)

**NO crear archivos separados que requieran correlación manual posterior.**
