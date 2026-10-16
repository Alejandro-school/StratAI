# Cierre de fase 1B.6 — última posición conocida y edad de información causales

Fecha: 29 de agosto de 2026

Estado: cerrada en código y validada sobre exactamente una demo aprobada. Gate 1B sigue pendiente y `training_allowed=false`.

## Resultado

Se añadió el contrato candidato separado `stratai.observed_last_known_position@1`. Por cada evaluación dirigida observer→subject del ledger `stratai.observed_visual_contact@1`, publica la última posición del pawn/origin del subject que ese observer pudo conocer, el tick y la disponibilidad del contacto que la acreditó, y la edad de esa información en el T0 actual:

`information_age_ms = (t0_tick - last_observed_tick) / tick_rate_hz * 1000`

Sólo `visual_contact=true` con estado `observed` actualiza la posición. Un contacto negativo o indisponible conserva el último hecho legal y aumenta su edad; sin contacto confirmado previo, posición y edad quedan `unavailable`. El estado se reinicia por ronda, identidad o secuencia de vida. No se interpola, no se clasifica la frescura y no se consulta posición actual oculta, radar/spotted, sonido, LOS aislado, oracle, lifecycle futuro ni outcomes.

El productor full-demo genera, fuera del bundle canónico:

- `observed_last_known_position_full_demo.json`;
- `observed_last_known_position_rows.jsonl.gz`.

La integración reutiliza el mismo `ParseDemoResult` de `--ai-coach-full-demo-validation`: una ejecución abre la demo una sola vez y alimenta el bundle temporal y todos los candidatos. La publicación temporal continúa siendo validada y atómica.

## Smoke real de una demo

Se usó exclusivamente `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, ya aprobado como parser `v16`, export `3.8.0`. La fuente, los 44 bundles, corpus, staging, exports, `CURRENT` y releases fueron de sólo lectura.

El primer intento falló cerrado antes de publicar archivos: la ordenación interna comparaba IDs de texto (`steam:10` antes de `steam:2`) mientras el contrato público exige orden por alias. Se corrigió la ordenación por `participant-NNN` y se añadió una regresión de IDs de dos dígitos. La repetición utilizó el mismo match y un directorio temporal nuevo y vacío; terminó en 149,360 s.

El validador Python independiente detectó después que el resumen comparaba de forma distinta números JSON equivalentes (`0`/`8500` frente a `0.0`/`8500.0`). Se corrigió la igualdad numérica sin equiparar booleanos; una pasada diagnóstica sobre las 777.284 filas confirmó igualdad completa y la validación estricta final, incluida Draft 2020-12 por fila, terminó en 664,097 s.

Resultados reales:

| Métrica | Resultado |
|---|---:|
| Rondas | 24 |
| Filas dirigidas | 777.284 |
| Participantes / parejas dirigidas | 10 / 50 |
| Actualizaciones por contacto | 24.893 |
| Refreshes de una pareja ya conocida | 24.031 |
| Carries desde contacto negativo | 320.271 |
| Filas `unavailable` | 432.120 |
| Hechos con edad observada | 345.164 |
| Edad exactamente cero | 24.893 |
| Edad positiva | 320.271 |
| Edad mínima / p50 / p95 / máxima | 0 / 8.500 / 55.437,5 / 139.312,5 ms |

Las 432.120 ausencias carecían de contacto visual confirmado previo en el scope legal. En esta demo no hubo contactos visuales `unavailable` ni resets de vida observables en el stream; las pruebas sintéticas sí cubren indisponibilidad, muerte/respawn, cambio de identidad y reset de ronda.

El stream comprimido ocupa 16.425.270 bytes y 1.059.773.230 bytes lógicos. Sus hashes son `8bb7519e5cad02b123fba9e18dc60480a9eee2c37b4fe6c8f90b0bf1b615f3be` y `011d46178b7cc76053d6d627ca3f9b56405ad888cde5ce2c4a5f68da44de5dc0`, respectivamente. El validador recomputó cero ticks futuros, cero posiciones actuales ocultas, cero mismatches de edad/carry y cero identidades crudas.

## Verificación

- `go test -count=1 ./parser ./cmd/canonical-export`: correcto después de la regresión de orden.
- `python -m pytest ai_coach/tests -q`: 143 pruebas pasadas.
- `python -m ruff check ai_coach`: correcto.
- `python -m ai_coach.contracts.validate_contract`: 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 8 schemas runtime y 18 reason codes.
- `npm run check:all`: frontend 88/88; Node 14 pasadas y 3 omitidas; todos los paquetes Go; backend Python 43/43; build frontend correcto.
- Los validadores de NAV/contacto acotados, NAV/contacto full-demo, arma y velocidad inicial de utilidad pasaron sobre la misma salida.
- `golden_demo_semantic_runner.validate_direct_bundle`: cero errores; bundle temporal `3.8.0`/`v16`, 73 ficheros y 72 artefactos declarados.
- Dos auditorías directas de `DecisionExample` dieron el mismo digest `02a69f5e897500d8f8f3d11801637f30796b09de25c0ad091eb3e2e7e3a6fb21`, 918 ejemplos/joins y cero fugas temporales u oracle.

La [evidencia máquina](../../evidence/AI_COACH_LAST_KNOWN_POSITION_INFORMATION_AGE_SINGLE_DEMO_SMOKE_2026-08-29.json) conserva intentos, comando, contratos, conteos, abstenciones, hashes y pre/postflight.

## Límites y siguiente paso

El candidato no forma parte de `3.8.0`, no se materializó en los 44 bundles y no cambia el catálogo histórico: `FLD-LAST-KNOWN-POSITION` y `FLD-INFORMATION-AGE` continúan `FALTA_EXTRAER` para esos bundles. No se entrenó, descargó, publicó, promovió ni creó una release.

El siguiente blocker no autorizado para ejecución en esta fase es el contexto espacial de `peek_hold_or_reposition`, empezando por un contrato y asset versionado de ángulos de exposición Mirage con revisión humana. Debe mantenerse separado de cobertura y salida, no convertir mesh/NAV en semántica táctica automática y no autoriza migración, reproceso 44/44 ni release.
