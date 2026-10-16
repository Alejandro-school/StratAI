# Cierre de reachability, path distance y travel time causales

Fecha: 28 de agosto de 2026  
Fase: 1B.3  
Estado: cerrada en código; no materializada en el corpus canónico

> Nota de vigencia: este informe conserva el cierre histórico del primer candidato. `stratai.observed_navigation_path@1` fue sustituido antes de materializarse por `stratai.observed_navigation_path@2`; la semántica vigente está en [la corrección de ETA contextual y smoke real](AI_COACH_ETA_CONTEXT_AND_SINGLE_DEMO_SMOKE_PHASE_2026-08-28.md). El supuesto único `path/250` ya no es el contrato activo.

## Resultado

Se implementó `stratai.observed_navigation_path@1`, un contrato candidato factual para consultar dos posiciones disponibles en `T0`. Devuelve por separado:

1. distancia euclídea directa;
2. existencia de conexión dirigida sobre el grafo NAV;
3. distancia mínima del camino navegable;
4. tiempo de viaje aproximado por ese camino.

No decide si una ruta es buena, no nombra rutas, no clasifica zonas, no infiere intención y no usa outcomes, oracle ni futuro. Tampoco modifica el contrato canónico `3.8.0`.

## Contrato físico explícito

| Elemento | Versión o unidad |
|---|---|
| Schema | `stratai.observed_navigation_path@1`, contrato `1.0.0` |
| Grafo | `nav_area_connections@1`, dirigido |
| Camino mínimo | `directed_dijkstra_stable_ties@1` |
| Asociación de extremos | centro de área NAV 3D más cercano, tolerancia de 96 world units |
| Peso de arista | distancia euclídea entre centros de áreas, en world units |
| ETA | distancia del camino / 250 world units por segundo |
| Velocidad | supuesto físico fijo y declarado; no es velocidad real observada |
| Identidades | sólo dentro de `join_keys`; los IDs de nodos NAV no se publican |

La ETA no modela aceleración, stance, arma, paradas ni intención. Es una aproximación cinemática reproducible. `reachability=true` sólo afirma conexión entre áreas; no afirma que el jugador llegue antes de un deadline.

## Implementación

- El parser NAV conserva ahora conexiones normales y de escalera que antes leía y descartaba.
- `MapManager.NavigationGraphSnapshot` expone una copia defensiva y ordenada de áreas y conexiones del NAV cargado. Un physics mesh/raycast no se acepta como sustituto del NAV.
- `pkg/navigation.Evaluate` valida nodos, aristas, pesos y configuración, asocia extremos con tolerancia y ejecuta Dijkstra con desempates estables.
- `parser.BuildObservedNavigationPath` valida `availability_tick <= T0`, separa los cuatro hechos, añade procedencia y obliga a usar `null/unavailable` cuando corresponde.
- El schema Python se incorporó al validador estático; ahora hay 5 schemas runtime versionados.

El grafo de conocimiento del repositorio no estuvo disponible (`Transport closed`) durante el descubrimiento. Conforme a `AGENTS.md`, se usó la búsqueda local sólo después de ese fallo y se documentó el límite.

## Abstención y negativos

| Caso | Reachability | Path distance | ETA |
|---|---|---|---|
| NAV ausente | `null/unavailable` | `null/unavailable` | `null/unavailable` |
| Origen o destino fuera de 96 world units | `null/unavailable` | `null/unavailable` | `null/unavailable` |
| Grafo válido sin conexión dirigida | `false/derived` | `null/unavailable` | `null/unavailable` |
| Peso cero, negativo, `NaN` o infinito | contrato rechazado | contrato rechazado | contrato rechazado |
| Disponibilidad posterior a `T0` | contrato rechazado | contrato rechazado | contrato rechazado |

La distancia directa puede seguir disponible cuando falta el NAV, pero nunca se convierte en reachability, path distance ni ETA. No se publican cero o infinito como sustitutos de una ausencia.

## Pruebas añadidas

Se añadieron 12 pruebas Go de nivel superior y 8 subcasos sintéticos. Cubren:

- grafo conectado;
- grafo desconectado;
- obstáculo que hace el camino mayor que la distancia directa;
- origen, destino y ambos extremos fuera de tolerancia;
- NAV ausente;
- disponibilidad de extremo o grafo posterior a `T0`;
- invariancia ante mutaciones futuras y oracle;
- IDs sólo para joins;
- determinismo ante distinto orden de nodos y aristas;
- rechazo de pesos cero, negativos y no finitos;
- regresión: raycast limpio o línea directa no prueban reachability ni ETA;
- instantánea NAV defensiva, ordenada y basada en conexiones reales.

Todas usan grafos en memoria. No abren las 44 partidas ni ningún archivo de demo.

## Verificación

| Comando | Resultado | Duración |
|---|---|---:|
| `go test -count=1 -v ./pkg/navigation ./pkg/maps ./parser` | 126 pruebas de nivel superior pasadas; 1 gate de demo real omitido | 4,165 s |
| `npm run test:go` | todos los paquetes Go pasaron | 5,590 s |
| `go vet ./pkg/navigation ./pkg/maps ./parser` | sin errores | 0,958 s |
| `python -m ai_coach.contracts.validate_contract` mediante el Python del venv | 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 5 schemas y 18 reason codes válidos | 0,104 s |
| `python -m pytest ai_coach/tests -q` mediante el Python del venv | 23 pruebas pasadas | 1,703 s; pytest 1,04 s |
| `npm run test:python` | 43 pruebas pasadas | 4,021 s; pytest 2,18 s |
| `npm run lint:python` | sin errores | 1,280 s |
| `python -m json.tool ai_coach/contracts/observed_navigation_path.schema.json` mediante el Python del venv | JSON válido | 0,107 s |
| `git diff --check` | sin errores de whitespace; sólo avisos informativos LF→CRLF | 0,061 s |

Hashes SHA-256 de las piezas centrales:

| Archivo | SHA-256 |
|---|---|
| `backend/go-service/pkg/navigation/graph.go` | `2f16f056d78f178eba63994c3215213bb10e51872aff58a9f35b62521f341465` |
| `backend/go-service/parser/observed_navigation_path.go` | `0f17ac0641c6d97a99a22b0dc4b61e7c4b4bfdc5d2740efef0e8a7d8fba7235c` |
| `ai_coach/contracts/observed_navigation_path.schema.json` | `e252e6d40be9d6c0905f6e40648bdc8cc94de8528c1a1e990abec568042f093a` |

## Estado preservado

La comprobación de sólo lectura antes de implementar registró:

| Área | Archivos | Bytes |
|---|---:|---:|
| `Faceit-Demos/demos` | 100 | 13.589.310.951 |
| `Faceit-Demos/analyzed_demos` | 3.013 | 27.264.064.316 |
| `Faceit-Demos/failed` | 727 | 5.245.908.800 |
| `Faceit-Demos/staging` | 0 | 0 |
| `Faceit-Demos/quarantine` | ausente | ausente |

La comprobación final conserva esos valores y `Faceit-Demos` continúa sin cambios de Git. No se descargó ni reprocesó una demo, no se entrenó ningún modelo, no se creó una release y no se modificaron `CURRENT`, staging, Docker, manifests, bundles, demos, exclusiones ni cuarentena. La release activa documentada sigue siendo `stratai-2c6db440231d463a`.

El catálogo v1.2 marca `FLD-PATH-DISTANCE`, `FLD-REACHABILITY` y `FLD-TRAVEL-TIME` como `DERIVABLE`, con `artifact=null` porque sólo existe el contrato candidato. El inventario de campos queda en 26 verificados, 9 derivables, 6 parciales, 5 por extraer, 5 por clasificar y 4 no observables.

Gate 1B permanece pendiente, `training_allowed=false` y la elegibilidad sobre los 44 bundles no cambia: 0 capacidades `eligible`, 3 `ineligible` y 14 `not_materialized`.

## Siguiente blocker

El siguiente hecho medido es `FLD-WEAPON-AMMO`: cargador, reserva y reload siguen parciales y no están materializados causalmente a 16 Hz. La siguiente fase queda limitada a ese contrato. No incluye velocidad inicial de utilidad, clasificación espacial, modelos, corpus, release ni promoción.
