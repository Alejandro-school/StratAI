# Auditoría del ledger de combate — 2026-08-15

## Alcance y baseline

Se auditó el código vigente y exclusivamente estos dos exports canónicos:

- `match_3GJOGmqKyZhYTtJrEmMvfNLsE` (Nuke).
- `match_7D4W7HC7RsQ7ZsCj3UVKmS5KQ` (Cache).

Ambos bundles declaran parser `v12`, export `3.3.0` y
`stratai.combat_event@1`. El validador oficial vigente devuelve `OK` para los
dos. Antes de modificar código también pasaban `go test ./...`, `go vet ./...`
y las 77 pruebas Python de `test_canonical_export_validator.py`.

Ese baseline no demuestra que combate sea correcto: las comprobaciones
actuales no cubren su semántica atómica.

## Evidencia cuantitativa de los exports

| Evidencia | Nuke | Cache |
| --- | ---: | ---: |
| Filas de `combat_events.jsonl` | 533 | 728 |
| Tipos presentes | 398 `damage`, 135 `kill` | 568 `damage`, 160 `kill` |
| `weapon_fire` en el ledger | 0 | 0 |
| Impactos en el ledger | 0 | 0 |
| Assists en el ledger | 0 | 0 |
| Filas sin `source_event_ids` | 533 | 728 |
| Filas sin daño de armadura | 533 | 728 |
| Daño de vida sumado por el ledger | 14.664 | 17.792 |
| Daño total nativo del scoreboard | 14.795 | 17.944 |
| Delta ledger − scoreboard | -131 | -152 |
| Kills del ledger / scoreboard | 135 / 135 | 160 / 160 |
| Deaths del scoreboard | 135 | 161 |
| Assists del scoreboard | 42 | 51 |
| Flash assists derivados actuales | 2 | 1 |
| Friendly damage derivado actual | 77 | 112 |
| Self damage derivado actual | 7 | 2 |

El replay, generado por callbacks independientes, contiene más hechos que el
ledger:

| Hecho del replay | Nuke | Cache |
| --- | ---: | ---: |
| `player_hurt` | 550 | 748 |
| Hurt enemy / friendly / self / world | 531 / 5 / 3 / 11 | 726 / 15 / 2 / 5 |
| Daño enemy / friendly / self / world | 14.896 / 77 / 7 / 156 | 17.814 / 112 / 2 / 175 |
| Kills enemy / friendly | 135 / 0 | 160 / 1 |
| Kills con assister observado | 42 | 53 |
| Eventos de replay con provenance de combate | 0 | 0 |

El replay repite cada disparo durante varias muestras: 2.185 disparos únicos
producen 18.006 apariciones en Nuke y 3.483 producen 28.708 en Cache. Solo 92 y
110 disparos, respectivamente, llegan a marcarse como hit, porque el flag se
aplica únicamente al último disparo reciente al recibir un callback de kill.
Las estadísticas declaran 1.972/414 y 3.090/588 disparos/aciertos; por tanto el
replay no es una proyección reconciliada de esas estadísticas.

## Fallos demostrados

### P0 — No existe un ledger atómico de combate

`handlers/combat.go` captura `PlayerHurt`, `Kill`, `WeaponFire`,
`WeaponReload` y `BulletDamage` en estructuras distintas. El exportador solo
transforma `AI_CombatEvents`, que contiene daño enemigo no fatal y kills
enemigas. `weapon_fire`, impactos, misses, assists, flash assists y lifecycle de
arma no se publican como hechos atómicos.

### P0 — Hay eventos omitidos silenciosamente

`captureRawDamageEvent` y `captureRawKillEvent` descartan attacker/victim nulos,
self damage, friendly fire y team kills. También descartan el `PlayerHurt`
fatal y lo sustituyen por una fila de kill. En Cache esto elimina del ledger un
team kill real; el scoreboard conserva 161 deaths frente a 160 kills enemigas.
Los 16 hurts friendly/self y 11 world de Nuke, y los 17 friendly/self y 5 world
de Cache, tampoco tienen representación canónica de combate.

### P0 — Stats, replay y export interpretan callbacks por separado

- `PlayerStatsHandler` vuelve a contar fire/hurt/kill y después reemplaza
  K/D/A/daño por el scoreboard nativo.
- `ReplayHandler` vuelve a capturar hurt/kill/fire y fabrica una línea de 1.500
  unidades desde el view angle para cada disparo.
- `buildCanonicalCombatEvents` exporta una tercera interpretación reducida.

Los deltas de daño y las diferencias de conteos anteriores son consecuencia
observable de esas tres fuentes.

### P0 — No hay provenance ni correlación causal

`combat_event@1` no tiene `source_event_ids`, `sequence_in_tick`, shot ID,
impact ID ni enlaces fire → impact/damage → kill. `BulletDamage` correlaciona
441/441 callbacks en Nuke y 594/594 en Cache dentro de memoria, pero el export
descarta su dirección, distancia específica, penetraciones, no-scope e in-air.

### P0 — Los gates aceptan el defecto

Go solo comprueba secuencia de rondas, rangos de accuracy, scoreboard final y
la cuenta de `BulletDamage`. El validador Python exige para combate únicamente
ID, ronda, tick, tipo, attacker, victim, damage e `is_kill`; no ejecuta ninguna
validación semántica de combate. Por eso ambos bundles incompletos pasan como
válidos.

### P1 — El daño fatal y su hitgroup no son factuales

La fila de kill calcula `Damage` con `remainingKillDamage()` y asigna hitgroup
`head` si es headshot o `chest` en cualquier otro caso. También fuerza armor
after a cero. Es una reconstrucción, no el `PlayerHurt` factual, y explica que
la suma del ledger no reconcilie con el scoreboard.

### P1 — Faltan precisión, disponibilidad y actores nulos explícitos

Los IDs de attacker/victim y arma son strings no nulables. No existen campos de
availability/source para actor, arma, impacto, posición o precisión temporal.
World damage y callbacks corruptos desaparecen en vez de conservarse como
`null/unavailable`.

### P1 — Assists y flash assists quedan fuera del hecho kill

`events.Kill` aporta `Assister` y `AssistedFlash`. El replay conserva el ID del
assister, pero no `AssistedFlash`; el ledger no conserva ninguno. En Cache se
observan 53 kills con assister mientras el scoreboard acredita 51 assists, una
diferencia que hoy no puede auditarse desde el contrato canónico.

### P1 — Lifecycle de arma capturado pero perdido

`WeaponReload` solo se guarda en `MatchData.Reloads`, fuera del bundle
canonical-only. `ItemEquip` se usa únicamente para terminar sprays. No existen
eventos canónicos de reload start, equip o weapon switch, ni disponibilidad que
indique cuándo el callback no es fiable.

### P1 — No hay impactos espaciales observables en estas fuentes

`demoinfocs v5.2.0` expone `BulletDamage`, no un evento tipado
`BulletImpact`. Los dos quality reports tienen cero parser warnings, por lo que
estas demos tampoco anuncian un `bullet_impact` desconocido mediante
`GenericGameEvent`. No se deben inventar coordenadas de impacto: la cobertura
de impacto debe declararse `unavailable`, salvo la evidencia de impacto dañino
que aporte `BulletDamage`/`PlayerHurt`.

## Condición para empezar la implementación

La implementación debe sustituir estas interpretaciones por un tracker causal
único, conservar cada callback observado aunque falten actores, asignar IDs
solo tras un orden total determinista y derivar desde su snapshot el export,
las estadísticas de combate y el replay. El contrato y sus invariantes se
definen antes de cambiar handlers productivos.

## Resolución y cierre

Los fallos anteriores se resolvieron con `combat_event@2`, el tracker único de
`pkg/combat`, replay `@5`, gates Go y validación Python. Nuke publica 6.567
filas y Cache 12.572; los fires únicos son 2.185 y 3.485 y reconcilian uno a
uno con `combat_shots`. Stats, scoreboard, daño, markers, callbacks y
provenance pasan la auditoría independiente sin errores.

Dos reprocesados consecutivos generaron árboles canónicos idénticos: Nuke
`2f6ff5e728373b83bf65004fd3ac5fb31114448e1001370034e939d6167fa934` y Cache
`bb803fe56cd9b708c53f20562dd2c9cf03ec1778b17a52c602492ec78b26197b`.
Los missing values observables siguen explícitos en `quality_report`; no quedan
P0/P1 conocidos del Bloque 4.
