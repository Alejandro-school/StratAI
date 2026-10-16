# Cierre de fase — `DecisionExample` y `TaskEligibility`

## Resultado sencillo

El paso 13 está completo. StratAI ya puede leer los bundles aprobados sin modificarlos, unir cada decisión con el último estado legal anterior, mantener observed/oracle/outcomes/perfil separados y explicar por qué cada capacidad se puede usar o no.

Esto no es un dataset de entrenamiento y no juzga si una jugada fue correcta. `training_allowed` permanece en `false` y Gate 1B sigue pendiente.

## Qué se construyó

- Contratos máquina versionados para `DecisionExample`, `TaskEligibility` y 18 reason codes estables.
- Validación fail-closed de tipos, campos obligatorios, IDs duplicados, reason codes desconocidos, oracle dentro de observed y disponibilidad posterior a T0.
- Lector read-only que descubre rutas sólo desde manifests, procesa un bundle cada vez y lee JSONL/gzip por streaming sin descomprimir en disco.
- Join de las cinco particiones causales, el `observed_state_ref` de 2 Hz y el último snapshot táctico self de 16 Hz disponible en T0.
- Evaluador determinista para las 17 capacidades del catálogo.
- CLI que conserva sólo agregados y cuatro muestras anónimas; no persiste ninguno de los 36.181 ejemplos individuales.

Un `DecisionExample` es una ficha estructural de una decisión real: qué acción se observó, qué datos legales había en T0 y qué sucedió después en una partición separada. `TaskEligibility` es el semáforo por capacidad: indica si existe la decisión, qué campos faltan y qué blocker impide usarla.

## Resultado de Mirage 44

| Medida | Resultado |
|---|---:|
| Bundles esperados / encontrados / procesados | 44 / 44 / 44 |
| Ejemplos estructurales | 36.181 |
| `spacing_or_trade_connection` | 26.648 |
| `peek_hold_or_reposition` | 9.533 |
| Joins correctos / fallidos | 36.181 / 0 |
| Violaciones `availability_tick > t0_tick` | 0 |
| Outcomes anteriores a T0 | 0 |
| Campos oracle dentro de observed | 0 |
| Filas gzip recorridas | 59.909.699 |
| Ejemplos individuales persistidos | 0 |

Las acciones también coinciden con Gate 1A: trade contiene 1.453 `connected` y 25.195 `disconnected`; peek contiene 5.153 `hold`, 4.363 `peek` y 17 `engage`.

## Elegibilidad de las 17 capacidades

| Estado | Cantidad | Capacidades |
|---|---:|---|
| `eligible` | 0 | Ninguna todavía |
| `ineligible` | 3 | `CAP-TEAM-001`, `CAP-PEEK-001`, `CAP-TIME-001` |
| `not_materialized` | 14 | Las demás capacidades del catálogo |

Trade es la familia más cercana, pero su contrato completo sigue bloqueado por la ausencia de `FLD-PATH-DISTANCE` y porque `FLD-LINE-OF-SIGHT` es sólo un proxy parcial. Peek tiene estado self en 16 Hz, pero no dispone de contacto visual causal completo, FOV, humo, exposición, cobertura ni salida. La capacidad temporal comparte decisiones peek, pero carece de `FLD-TRAVEL-TIME`.

Las 14 capacidades restantes no fingen cobertura: devuelven `not_materialized` porque sus tipos de decisión todavía no existen en estos bundles, aunque algunos hechos sueltos sí estén presentes.

## Causalidad, límites y fallos

No se detectaron fugas de futuro u oracle. Los IDs permanecen sólo en `join_keys`; no se convierten en variables predictivas. Los nulos siguen siendo nulos: por ejemplo, el reloj de ronda está presente en 8.408 decisiones y nulo en 27.773, mientras que el reloj de bomba está presente en 1.125 y nulo en 35.056.

No hubo corrupción, referencias rotas ni contradicciones canónicas. La CLI no recalcula los hashes ya aprobados por Gate 1A; confía en la validación comprometida del manifest y vuelve a validar estructura, referencias, tipos, orden, tiempo y conteos de los streams que consume.

## Pruebas y evidencia

| Comando | Resultado | Duración |
|---|---|---:|
| `python -m ai_coach.contracts.validate_contract` mediante el Python del venv | Contrato del catálogo y 3 contratos runtime válidos | 0,108 s |
| `python -m pytest ai_coach/tests -q` mediante el Python del venv | 23 pasadas | 0,941 s de pared; 0,50 s de pytest |
| `npm run test:python` | 43 pasadas | 2,275 s de pared; 1,02 s de pytest |
| Constructor read-only 44/44 | Correcto | 302,919 s |
| Repetición de determinismo 44/44 | Mismo SHA-256 | 301,491 s |
| `python -m json.tool` sobre la evidencia mediante el Python del venv | JSON válido | 0,111 s |

La segunda ejecución produjo exactamente el mismo SHA-256: `ad4849fefcbd6e71cfc571d2e99447a842ea1965d19ae6952501ace95d6d34af`. La evidencia completa está en [AI_COACH_DECISION_EXAMPLE_COVERAGE_MIRAGE_44_2026-08-27.json](../../evidence/AI_COACH_DECISION_EXAMPLE_COVERAGE_MIRAGE_44_2026-08-27.json).

Las pruebas sintéticas cubren join válido, referencia ausente, referencia duplicada, ID con tipo incorrecto, disponibilidad posterior a T0, mutación del futuro, oracle inyectado, gzip corrupto, campo obligatorio ausente, reason code desconocido, `not_materialized`, determinismo y ausencia de lectura anticipada del siguiente bundle.

## Siguiente acción única

Implementar y validar el ledger causal de contacto visual observed que combine raycast, FOV y oclusión por humo, sin oracle ni futuro. El prompt histórico de contacto visual, ya retirado del catálogo activo, limitaba la siguiente fase a ese blocker y no se ejecutó durante este cierre.
