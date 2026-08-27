# Auditoría cuantitativa AI Coach — Mirage 44

Estado: evidencia cerrada e inmutable de Gate 1A. Fecha: 2026-08-27. Resultado: **apto para diseñar y probar contratos; no apto todavía para autorizar entrenamiento**. No mantiene el progreso ni la siguiente acción.

Evidencia reproducible: [reporte JSON](evidence/AI_COACH_MIRAGE_44_AUDIT_2026-08-27.json). Herramienta: [`ai_coach.audits.bundle_audit`](../ai_coach/audits/bundle_audit.py). Corpus: [aceptación Mirage smoke 50](../../Faceit-Demos/docs/MIRAGE_SMOKE_50_ACCEPTANCE_2026-08-26.md).

## Alcance y método

- Se auditaron exactamente 44 bundles aprobados, en orden lexicográfico; 6 demos quedan fuera según la aceptación.
- Los 44 manifests declaran formato `3.8.0` y parser `v16`. El SHA-256 determinista del conjunto de manifests es `9c9b5b1df3542c4120174b78fcad727765fbf641cba5d004e1dc69c478c0167e`.
- Los archivos JSONL se procesaron en streaming y los `.gz` sin descompresión persistente. Se mantuvo un bundle a la vez y se limitaron tamaño de JSON/línea.
- Los conteos de registros tácticos son exactos desde `sampling.json`/manifest. Para perfiles de campos se tomó el prefijo determinista de hasta 4.096 filas por bundle y partición: 180.224 observed y 180.224 oracle. No es una muestra aleatoria ni debe usarse para inferir prevalencias globales finas.
- El resto se recorrió completo: 2.265.375 registros. El reporte contiene 1.907 paths con presencia, nulls, tipos, rangos, frecuencias acotadas y ejemplos. Omite valores de IDs/refs de join; conserva el ID de bundle como referencia de procedencia.
- La herramienta es sólo lectura y falla con ruta/línea ante JSONL corrupto. Sus tests cubren gzip/nulls, corrupción y número inesperado de bundles.

## Presencia e inventario

Los 27 artefactos lógicos auditados aparecen en 44/44 bundles. Los conteos leídos coinciden con los declarados para las familias inspeccionadas.

| Artefacto | Archivos | Registros | Escaneo de campos | Uso AI Coach |
|---|---:|---:|---:|---|
| match | 44 | 44 | completo | contexto/split |
| participants | 44 | 440 | completo | joins de jugador/equipo |
| rounds | 44 | 934 | completo | ronda, side, winner |
| player_states 2 Hz | 934 | 1.203.728 | completo | estado físico y clocks |
| tactical observed 16 Hz | 44 | 59.909.699 | muestra 180.224 | hechos legales T0 |
| tactical oracle 16 Hz | 44 | 10.627.998 | muestra 180.224 | etiquetas/evaluación |
| tactical gaps | 44 | 363.155 | completo | disponibilidad/abstención |
| combat events | 44 | 466.754 | completo | disparos, daño, kills, reload |
| utility events | 44 | 14.759 | completo | lanzamientos/trayectorias |
| objective events | 44 | 5.117 | completo | bomba/plant/defuse |
| engagements | 44 | 9.646 | completo | episodios de contacto |
| trade candidates / completions | 44 / 44 | 6.662 / 344 | completo | equipo/trade |
| economy players / rounds | 44 / 44 | 9.308 / 1.868 | completo | dinero/inventario |
| clutch events | 44 | 1.051 | completo | episodios raros |
| decisions/features/oracle/outcomes/masks | 220 | 36.181 cada uno | completo | causal actual |
| quality reports | 44 | 44 | completo salvo arrays resumidos | gate/limitaciones |

Otros artefactos presentes: canonical/root manifests, player stats, tactical sampling y replay index. Los 44 bundles están en `warning`, no en `pass`; la aceptación agrega 880 estados de dominio: 562 `pass`, 318 `warning`, 0 `fail`.

## Decisiones causales ya materializadas

Hay 36.181 decisiones en todos los bundles:

| Tipo | Total | Acción observada | Outcome |
|---|---:|---|---|
| spacing/trade connection | 26.648 | disconnected 25.195; connected 1.453 | not_tradeable 21.760; not_attempted 2.844; completed 1.376; failed 668 |
| peek/hold/reposition | 9.533 | hold 5.153; peek 4.363; engage 17 | kill 6.511; disengaged 3.022 |

Todas las filas tienen `state_availability_status=observed` y un `observed_state_ref`. Sin embargo, elegibilidad no equivale a presencia de la fila:

- `trade_possible`, `nearest_connection_time_ms`, LOS y facing están derivados en 26.648 trades y no disponibles en las 9.533 decisiones peek.
- `trade_possible=true` sólo aparece 1.453 veces; 25.195 son false. El tiempo de conexión derivado va de 23,56 a 17.548,75 ms y facing de 0,0018° a 179,996°.
- `initial_distance` y `enemies_exposed` están no disponibles en las 36.181; `economy_context` también.
- `round_clock_remaining_ms` está null en 27.773/36.181 (76,76 %); `bomb_time_remaining_ms`, correctamente condicionado a plant, está null en 35.056 (96,89 %).
- La visibilidad de decisión se declara `observed_physical_proxy`/`observable_proxy`: es proxy, no prueba de percepción.

Cobertura de capacidades: trade/equipo tiene un punto de partida factual; peek sólo tiene acción/outcome y carece de features espaciales esenciales. El corpus no contiene aún decisiones explícitas para economía, utilidad, rotación, objetivo, clutch, combate granular ni perfil.

## Estado físico y combate

### Player states 2 Hz — escaneo completo

| Campo | Presencia | Null | Rango/unidad | Ejemplo/interpretación |
|---|---:|---:|---|---|
| `position.x` | 1.203.728/1.203.728 | 0 % | -2.655,97 a 1.455,33 world units | posición Mirage |
| `view_yaw_deg` | 100 % | 0 % | -180° a 179,999° | orientación horizontal |
| horizontal velocity | 100 % | 29,40 % | 0 a 970,47 world units/s | disponible en 849.856 estados |
| round clock | 100 % | 14,45 % | 500 a 115.000 ms | null fuera de fase aplicable |
| bomb clock | 100 % | 85,55 % | 0 a 40.000 ms | 173.964 estados post-plant |
| objective phase | 100 % | 0 % | preplant 1.004.529; planted 160.834; planting 25.235; defusing 13.130 | etiqueta de fase |
| defuse kit | 100 % | 0 % | true 132.775; false 1.070.953 | booleano |

La velocidad máxima observada necesita tests de semántica —puede reflejar desplazamientos especiales, errores o derivación— antes de usarla como señal de mecánica.

### Tactical observed/oracle — muestra acotada

En 180.224 filas observed: team 133.080, self 33.270 y `enemy_los` 13.874. En la muestra oracle, las 180.224 filas son `visibility_scope=oracle`.

- La velocidad horizontal aparece en 164.150 observed (91,08 %) y queda no disponible en 16.074; rango muestral 0–307,35 world units/s.
- Ammo in magazine, ammo reserve e `is_reloading` son null/no disponibles en 180.224/180.224 observed muestreados. Esto bloquea estado de disparo/recarga a 16 Hz.
- `enemy_los` es raycast geométrico. La traza de código muestra que el export causal actual no aplica de forma demostrada FOV ni smoke; no debe llamarse “enemigo visible”.

### Combat ledger — escaneo completo

466.754 eventos: weapon_equip 297.833, weapon_fire 132.482, player_hurt 23.678, kill 6.805 y weapon_reload 5.956.

- Ammo existe como campo en todos los eventos, pero es null/no disponible en 43.943 (9,41 %). `ammo_in_magazine` alcanza `4294967295`, un sentinel/valor imposible que debe normalizarse antes de modelar; reserve va 0–32.
- `view_yaw` sólo tiene valor en 132.482 eventos, por eso su null global es 71,62 %: corresponde a los disparos, no a todos los tipos de evento.
- `reaction_time_ms` existe sólo en 3.971 eventos (0,85 %), rango 15–2.500 ms. No puede tratarse como una feature general de combate.
- Vida/armor antes/después sólo aplican a 23.678 `player_hurt`; el null global 94,93 % es estructural.

## Utilidad, objetivo y economía

### Utilidad

Hay 14.759 lanzamientos. La trayectoria está observed en 14.648 (99,25 %) y unavailable en 111. El número de rebotes existe en 11.076; 3.683 (24,95 %) no lo tienen, con rango observado 1–9. La velocidad del lanzador existe en 14.639 (99,19 %), 0–494,71 world units/s.

Limitación importante: `projectile_initial_velocity` está unavailable en 14.759/14.759. Hay samples de trayectoria, detonation/effect y rebotes, pero no el vector inicial canónico necesario para reconstruir de forma uniforme candidatos de lineup.

### Objetivo

5.117 eventos: bomb_drop 2.037, pickup 1.562, plant_start 453, plant 379, carrier_snapshot 349, defuse_start 105, explosion 76, defuse 74, plant_abort 55 y defuse_abort 27. `has_defuse_kit` sólo aplica a 206 filas (66 true, 140 false); su null global 95,97 % es estructural.

Los eventos permiten describir lifecycle. Para consejo de plant/defuse/retake aún hacen falta sitio/posición clasificada, reachability, ETA y deadline comprobados en T0.

### Economía

Hay 9.308 player-round y 1.868 team-round. El dinero observado al final de freeze aparece en 9.308/9.308, rango 0–16.000, y el inventario de freeze declara `observed_with_calculated_valuation` en 9.308/9.308. Los quality flags de los 44 bundles avisan sobre cobertura/reconciliación de economía e itemización; por ello el contrato mantiene contexto económico de decisión como parcial hasta producir un snapshot causal explícito y pasar reconciliación.

## Cobertura por capacidad

| Capacidad | Evidencia existente | Brecha que impide consejo prescriptivo |
|---|---|---|
| combate/crosshair/movement | combat ledger, shot angles, states 2 Hz | ammo/reload 16 Hz, normalizar sentinel, reaction/crosshair sparse |
| peek/posición | 9.533 decisiones, position, yaw, raycast proxy | FOV/smoke, exposure, zones, cover, routes |
| equipo/trade/entry | 26.648 decisiones, 6.662 candidates, 344 completions | candidatos espaciales/entry, support desbalanceado |
| utilidad | 14.759 throws, 14.648 trajectories | initial velocity, intención/objetivo táctico, candidates |
| economía | dinero e inventario por ronda | snapshot T0 de freeze reconciliado y acciones de compra |
| información/rotación | observed/team y clocks | contact ledger, last-known/age, path/ETA |
| objetivo/tiempo | events, phase, clocks, kit | sites/routes/deadlines/candidatos |
| ventaja/riesgo | team state y outcomes | definición causal común y ranking calibrado |
| clutch | 1.051 eventos | soporte por acción, contact/path, protocolo raro |
| perfil | participantes/matches | store de matches previos, identidad y privacidad |

## Ejemplos representativos y límites

- Ejemplo soportado: “en T0 el trade era físicamente conectable, con connection time y facing derivados; después se completó/no se completó”. Aún no prueba que conectar fuera causalmente mejor.
- Ejemplo sólo descriptivo: “el jugador disparó con yaw X y hubo daño Y”. Si ammo o reaction faltan, se muestran como no disponibles.
- Ejemplo bloqueado: “debía rotar por connector y llegaba antes de la bomba”. No hay grafo/ruta/ETA tácticos versionados.
- Ejemplo prohibido: “sabía que el rival estaba ahí por sonido/voz”. La demo no prueba percepción subjetiva.

Mirage y esta selección de 44 partidas limitan generalización. Las filas están correlacionadas dentro de match/jugador/episodio; los enormes streams tácticos no equivalen a soporte independiente. Los resultados futuros describen la política observada y contienen confusores. No existe ground truth directo para una alternativa no ejecutada.

## Conclusión y gates

El audit confirma integridad estructural y mucha telemetría útil, pero también brechas P0 verificables: semántica de visibilidad, estado de arma a 16 Hz, contexto causal en decisiones y capa espacial táctica/pathfinding. Antes de entrenar deben existir dataset por decisión versionado, task eligibility, aislamiento oracle, splits agrupados, soporte por acción, candidatos físicos, challenge set y evaluación humana. Las [dependencias técnicas del plan](AI_COACH_IMPLEMENTATION_PLAN.md#dependencias-técnicas-verificadas-en-gate-1a) ordenan ese trabajo.
