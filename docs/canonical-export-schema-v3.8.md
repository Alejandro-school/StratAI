# Contrato de export canónico 3.8

Estado: candidato de release verificado en working tree el 24 de agosto de 2026; Gate 1 global pendiente.

Este documento define el bundle canonical-only producido por el parser `v16`.
La raíz publicada es `backend/data/exports/match_<match-id>/`; su
`manifest.json` operativo es el marcador de commit y `canonical/manifest.json`
es el catálogo verificable del contenido.

## Versiones cerradas

| Contrato | Versión |
| --- | --- |
| Parser | `v16` |
| Export | `3.8.0` |
| Quality report | `12` |
| Manifest canónico | `stratai.canonical_manifest@3` |
| Validador | `stratai.canonical_validator@2` |
| Corpus de demos golden | `stratai.golden_demo_corpus@2` (`golden-demos-v2`, evidencia semántica parcial) |

Una discrepancia de versión es un fallo hard de publicación.

## Árbol del bundle

```text
match_<match-id>/
├── manifest.json
└── canonical/
    ├── manifest.json
    ├── core/
    │   ├── match.json
    │   ├── match_metadata.json
    │   ├── participants.json
    │   └── rounds.json
    ├── states/
    │   ├── player_states/round_NNN.jsonl
    │   └── tactical/
    │       ├── sampling.json
    │       ├── observed.jsonl.gz
    │       ├── oracle.jsonl
    │       └── gaps.jsonl
    ├── events/
    │   ├── combat_events.jsonl
    │   ├── objective_events.jsonl
    │   └── utility_events.jsonl
    ├── derived/
    │   ├── clutch_events.json
    │   ├── economy_players.json
    │   ├── economy_rounds.json
    │   ├── engagements.json
    │   ├── player_stats.json
    │   └── trades.json
    ├── causal/
    │   ├── decisions.jsonl
    │   ├── decision_features.jsonl
    │   ├── oracle_context.jsonl
    │   ├── outcomes.jsonl
    │   └── quality_masks.jsonl
    ├── presentation/replay/
    │   ├── index.json
    │   └── round_NNN.json.gz
    └── diagnostics/quality_report.json
```

El catálogo declara ruta relativa, tipo, schema, encoding, compresión cuando
corresponde, tamaño, número de registros y SHA-256 de cada artefacto. Los
consumidores no deben descubrir archivos legacy ni inferir artefactos fuera
del catálogo.

## Particiones causales

Las cinco particiones son físicamente distintas y tienen una fila por
`decision_id`. Deben tener el mismo conjunto y orden de claves
`(round_number, t0_tick, decision_id)`.

### `decisions.jsonl`

Schema `stratai.decision@1`. Enlaza la decisión con actor, referencia de estado
observado, acción realizada, disponibilidad, rol causal, scope de visibilidad y
provenance. Los IDs son exclusivamente claves de join y no forman parte del
vector entrenable. Las familias P0 son `spacing_or_trade_connection` y
`peek_hold_or_reposition`.

### `decision_features.jsonl`

Schema `stratai.decision_features@1`. Sólo contiene información disponible en
o antes de `t0_tick`. `availability_tick_max` debe ser menor o igual a `t0_tick`.
Incluye contexto causal como participantes, estados observados, reloj de ronda,
contexto de bomba/economía y sus estados de disponibilidad. La distancia exacta
actor-target permanece `null/unavailable` hasta disponer de evidencia visual
causal; la geometría retrospectiva no se promueve a observed.

No puede contener ganador, perdedor, outcome, score futuro, rating, identidad
personal ni nombres. `match_id` y `decision_id` son claves de linkage, no
features de modelo.

### `oracle_context.jsonl`

Schema `stratai.oracle_context@1`. Reserva información oculta o sólo disponible
para evaluación. Si el parser no dispone de ella, escribe `available=false`,
`status=unavailable`, una lista de campos vacía y la abstención explícita. No
se inventa un oracle.

### `outcomes.jsonl`

Schema `stratai.decision_outcome@1`. Contiene únicamente etiquetas posteriores
o simultáneas a `t0`: outcome, `outcome_tick`, duración, kills terminales,
trades, supervivencia y cierre del engagement. `outcome_tick` nunca puede ser
anterior a `t0_tick`. Cada fila declara exactamente horizontes de 2, 5 y 10
segundos.

### `quality_masks.jsonl`

Schema `stratai.quality_mask@1`. Clasifica cada campo como `available`,
`unavailable` o `inferred`; las tres listas son disjuntas, están ordenadas y no
ocultan gaps. `warning_flags` explica por qué una fila requiere cautela.

## Estado táctico a 16 Hz

`sampling.json` (`stratai.tactical_sampling@1`) declara la cadencia objetivo,
conteos e identidad join-only. `observed.jsonl.gz` (JSONL comprimido con gzip)
(`stratai.tactical_physical_observation@1`) sólo expone al observador su propio
estado, el subconjunto permitido de compañeros y enemigos demostrados mediante
geometría cargada más LOS. `oracle.jsonl` (`stratai.tactical_oracle_state@1`)
conserva el estado físico real en una partición separada. Cada tick objetivo sin
observación se materializa en `gaps.jsonl`
(`stratai.tactical_sampling_gap@1`); no existen huecos silenciosos.

Cada campo físico tiene disponibilidad explícita. Los datos sin productor
callback-backed, incluidos ammo y reload en el replay actual, son `null` y
`unavailable`, nunca cero inventado. Los nombres no se publican y los IDs sólo
se usan dentro de join keys.

## Quality report

`canonical/diagnostics/quality_report.json` conserva las métricas detalladas
de Blocks 1–6 y añade exactamente 20 dominios globales:

1. `bundle_manifest_contract`
2. `artifact_catalog_integrity`
3. `artifact_hash_integrity`
4. `artifact_record_count`
5. `cross_artifact_references`
6. `roster_consistency`
7. `round_consistency`
8. `objective_consistency`
9. `utility_consistency`
10. `combat_consistency`
11. `engagement_consistency`
12. `economy_consistency`
13. `player_state_consistency`
14. `replay_projection_consistency`
15. `causal_availability`
16. `future_leakage`
17. `schema_version_compatibility`
18. `determinism`
19. `lineage_completeness`
20. `corpus_quality`

Cada dominio contiene los campos:

```text
status, severity, expected, actual, coverage, unavailable_count,
inferred_count, warning_details, hard_failure_details,
source_artifacts, schema_versions
```

La cobertura se calcula como
`observed / (observed + unavailable + inferred)`. Un gap observado no se
convierte en cero ni en un hecho calculado. Los dominios hard deben estar en
`pass`; los dominios de cobertura pueden publicar como `warning` con el gap
cuantificado. Cualquier `fail`, status desconocido o métrica hard distinta de
cero bloquea el commit.

`usable_for_training=true` expresa únicamente que el bundle supera este
contrato técnico. No autoriza por sí solo entrenamiento global ni el uso del
bundle para cualquier familia del AI Coach. Esa autorización requiere
`task_eligibility`, separación causal, soporte, splits y el gate de
entrenamiento definidos en el plan vigente.

## Lineage

`canonical/manifest.json.lineage` fija o marca explícitamente como no
disponibles:

- checksum SHA-256 de la demo;
- parser, demoinfocs, export, validator y build;
- mapa y reglas de tick rate;
- tabla de precios y su SHA-256;
- versiones y hashes de algoritmos/configuración;
- schemas de todos los artefactos;
- fuente y checksum de metadatos;
- hashes exactos de physics map, nav mesh y callouts, junto con si se usaron o
  sólo se inspeccionaron;
- quality flags, warnings y abstenciones;
- versión e ID del corpus golden.

`processing_timestamp` se declara `operational_only` y queda fuera del árbol
determinista.

## Validación

El validador recalcula catálogo, tamaños, record counts, SHA-256, referencias,
cronología, schemas, las cinco particiones causales, continuidad táctica,
quality gates, lineage y
corpus. Su salida es una receipt JSON; sólo `status=passed` permite publicar.

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\publication_validator.py `
  backend\data\exports\match_<match-id> `
  --match-id <match-id> `
  --checksum <sha256-demo>
```

Los consumidores de producción deben usar únicamente el manifest y los paths
canónicos. El formato 3.7 no autoriza a reintroducir
`metadata.json`, `quality.json`, `combat.json`, `economy.json`,
`grenades.json`, `tracking.json` o `replay.json` legacy.
