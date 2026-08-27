# Auditoría de engagements y trades — 2026-08-18

## Alcance y prerrequisito

La auditoría fue read-only y se realizó sobre el código vigente y los dos únicos
exports canónicos:

- `match_3GJOGmqKyZhYTtJrEmMvfNLsE` (`de_nuke`).
- `match_7D4W7HC7RsQ7ZsCj3UVKmS5KQ` (`de_cache`).

El Bloque 4 está cerrado. Ambos bundles declaran export `3.4.0`, parser `v13`,
`stratai.combat_event@2` y quality schema `9`; los dos pasan
`canonical_export_validator.py`. Todos los gates duros `combat_*` están en
`pass`. Los 52 ficheros del subárbol canónico de Nuke y los 56 de Cache
coinciden por SHA-256 con sus snapshots deterministas del cierre. El único
fichero distinto fuera de esos subárboles es el `manifest.json` transaccional,
que contiene `committed_at`.

## Implementación auditada

`handlers/combat_duels.go` continúa creando `RawCombatEvent` directamente
desde `PlayerHurt` y `Kill`. Esa ruta:

- descarta world, self, friendly y daño fatal;
- vuelve a reconstruir daño fatal, hitgroup y armor;
- agrupa pares con `DuelTimeoutTicks=640`;
- agrupa granadas con 30 ticks y fuego con 500 ticks;
- elige como `Attacker` al jugador que mata o, si no hay kill, al que causa más
  daño;
- usa maps de Go para elegir participantes y recorrer grupos;
- deja `IsTrade=false` en todos los eventos.

Después, `buildCanonicalEngagements` remapea por ronda, ventana, actor y target
algunos `player_hurt`/`kill` del ledger. No construye el engagement desde el
ledger ni conserva una correspondencia 1:1 entre exchange y evento atómico.

Los trades se cuentan por separado en `PlayerStatsHandler.HandleKill` mediante
una ventana fija de 320 ticks, descrita como cinco segundos a 64 Hz. No existe
un artefacto de trade, IDs, provenance, candidatos, intentos ni estado no
evaluable.

## Fallos demostrados

### P0 — Iniciador y ganador están conflados

El campo `attacker_player_id` representa al ganador observado, no al iniciador.
En 25 engagements de Nuke y 35 de Cache el primer exchange pertenece a otro
jugador.

Ejemplos:

- Nuke, ronda 1, ticks `3047–3102`, engagement `000006`: el campo attacker es
  `steam:76561198079080929`, pero el primer daño lo produce
  `steam:76561197991999414` en tick `3047`. El primero termina perdiendo y el
  segundo queda oculto como iniciador.
- Cache, ronda 1, ticks `2920–2944`, engagement `000004`: el campo attacker es
  `steam:76561198065602953`, pero el primer daño lo produce
  `steam:76561198400546718` en tick `2920`.

La causa raíz está en `buildMultiVictimDuel`: primero busca quién obtuvo la
kill o causó más daño y luego lo publica como atacante.

### P0 — Provenance incompleta y reutilizada

Los exchanges bidireccionales no conservan sus IDs. En Nuke `000006`, los
eventos atómicos `combat:001:000003047:003`, `...3049:001`, `...3062:001` y
`...3096:003` aparecen en `details.exchanges`, pero faltan en
`source_event_ids` porque su actor no es el ganador.

Además, cinco IDs atómicos de Nuke y uno de Cache se reutilizan en dos
engagements incompatibles. Ejemplos:

- Nuke, ronda 8, `combat:008:000044251:003` se asigna al engagement de granada
  `000086` y al duelo `000087`.
- Cache, ronda 20, `combat:020:000118085:001` se asigna al duelo `000213` y al
  engagement de granada `000214`.

El remapeo por ventana no compara el evento atómico exacto, arma, daño ni
shot ID. Por ello una ventana solapada puede reclamar eventos ajenos.

### P0 — No existe contrato de trades

El ledger permite observar, con ventana de 5.000 ms y tick rate real, 21
respuestas de trade en Nuke y 24 en Cache antes de aplicar asignación 1:1. Sin
embargo, los 454 engagements publican `details.context.is_trade=false`.

Ejemplos:

- Nuke, ronda 1: muerte `combat:001:000003054:004` en tick `3054`; respuesta
  `combat:001:000003249:004` por `steam:76561198170650958` en tick `3249`,
  `3046.875 ms` después.
- Cache, ronda 3: muerte `combat:003:000014798:004`; respuesta
  `combat:003:000014862:004` por `steam:76561199161958820`, exactamente
  `1000 ms` después.

Las estadísticas legacy declaran 19/19 trade kills/traded deaths en Nuke y
23/23 en Cache, pero no pueden reconciliarse con filas trazables ni explicar
colisiones entre varios candidatos.

### P0 — Ventanas temporales dependientes de ticks hardcodeados

`DuelTimeoutTicks=640`, `tradeWindowTicks=320`, granadas `30` y fuego `500`
asumen 64 Hz. Aunque ambas demos actuales son de 64 Hz, el contrato es
incorrecto para cualquier otro tick rate.

La ventana de duelo permite agrupaciones excesivas:

- Cache, ronda 3, engagement `000028`: `9671.875 ms`.
- Cache, ronda 1, engagement `000011`: `9062.500 ms`.
- Nuke, ronda 11, engagement `000113`: `7453.125 ms`.

### P1 — Collateral y multi-target no se preservan

El ledger contiene tres shots con múltiples targets en Nuke y uno en Cache,
pero el export publica cero engagements `collateral`.

- Nuke, ronda 10, shot `shot:010:000028`, tick `55838`, actor
  `steam:76561198116485358`, targets `steam:76561198011519274` y
  `steam:76561198035010243`.
- Cache, ronda 19, shot `shot:019:000053`, tick `113066`, actor
  `steam:76561198119025520`, targets `steam:76561197991999414` y
  `steam:76561198065602953`.

La causa es que el camino no-granada agrupa siempre por par antes de construir
el derivado.

### P1 — Exchanges sin identidad exacta

Un exchange solo contiene tick, nombre del atacante, arma, daño e información
parcial. No tiene actor ID, target ID, event ID, armor damage ni provenance.
En shotgun hits hay filas indistinguibles con el mismo payload; sin ID no es
posible distinguir impactos legítimos de duplicados.

### P1 — Future leakage y missing fabricado

`details.context` toma el último evento o la kill. En 108 engagements de Nuke
y 137 de Cache el final es posterior al inicio, pero bomba, vivos, distancia y
tiempo de ronda se publican junto al supuesto contexto de decisión. No existe
separación física entre `causal_context` y `outcome_context`.

Los valores desconocidos se representan con frecuencia como `0`, `false` o
string vacío: arma del perdedor, reacción, first damage, visibilidad,
`is_trade`, economía y exposición simultánea.

## Campos correctos que deben preservarse

La auditoría también verificó estos aspectos y no deben reimplementarse:

- `tick_rate_hz=64` y `duration_ms=(end_tick-start_tick)/tick_rate_hz` coinciden
  en los 454 registros.
- Cero ticks fuera de límites de ronda y cero finales anteriores al inicio.
- Cero participantes ajenos al roster.
- Cero `source_event_ids` inexistentes o fuera de la ronda/ventana declarada.
- Cero observaciones de movimiento o arma con tick posterior al final.
- La semántica `null/unavailable` de velocidad y arma introducida en el
  Bloque 1 es válida y debe reutilizarse.
- Los IDs actuales se asignan después de ordenar en el exportador. El nuevo
  contrato conserva esa propiedad, pero debe ordenar también el contenido
  antes de crear IDs.

## Decisión

`stratai.engagements@1`, `duel_consolidation@1` y el contador de trades de
`PlayerStatsHandler` no son aptos para Bloque 5. Se sustituyen por derivados
puros de `combat_event@2`; el ledger no se modifica ni se compensa dentro del
algoritmo nuevo.
