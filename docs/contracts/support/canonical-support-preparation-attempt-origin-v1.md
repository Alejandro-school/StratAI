# Origen y progreso nativo del plantado en C0 v1

Extensión optativa `1.0.0-candidate.1`, fase 3E.17. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija su estado y continuación. Une los cuatro hechos de [origen nativo 3E.6B](canonical-support-attempt-origin-v1.md) al corte exacto del [censo C0](canonical-support-preparation-v1.md). Conserva los contextos [físico](canonical-support-preparation-physical-v1.md), de [objetivo](canonical-support-preparation-objective-v1.md), [C4](canonical-support-preparation-c4-v1.md) y [reloj](canonical-support-preparation-objective-clock-v1.md).

## Fuente y causalidad

Reutiliza `--ai-coach-support-attempt-origin-candidate` sin cambios Go. Support, objetivo, C4, reloj y origen se enlazan por alias, checksum, build, bytes y SHA-256, incluyendo el manifest canónico. El corte exige entrega support, tick, FrameDone, instancia/época, reloj y fronteras de frames activos/entregas exactos. Un frame anterior o posterior con el mismo tick no sustituye T0.

Un origen positivo requiere una transición nativa `m_bStartedArming: false → true` posterior a creación/replay y una nueva asignación concordante de `m_fArmedTime` en ese frame. Exige propietario e inventario bilaterales, handles completos, vidas de C4 y pawn, definición nativa C4, tipos/dominio GameTime_t, ticks nativos concordantes, intervalo observado y origen temporal ya disponible al empezar. Creación activa, primer sample, deadline heredado o testigo posterior mantienen abstención.

La duración se deriva del deadline menos el tiempo nativo del inicio, con rango positivo acreditado por 3E.6B. Cada frame activo posterior debe ser consecutivo, conservar C4/propietario/pawn/deadline/dominio, reloj sin retroceso y ausencia acreditada de pausa. También se comprueba continuidad del alias C0, lado, vida y handle del pawn en las entregas support desde la última anterior o igual al inicio hasta T0, dentro de la misma instancia/época. Una continuidad no acreditada devuelve `native_origin_c0_actor_continuity_unaccredited`, sin retirar el candidato.

En T0 se contrasta el tiempo restante con el reloj C0. Los cuatro hechos son:

| Hecho | Valor observado |
|---|---|
| `planting_origin_state` | `native_transition` o `expired` |
| `planting_duration_seconds` | Deadline nativo menos tiempo nativo del inicio |
| `planting_elapsed_seconds` | Duración menos tiempo restante, limitado al intervalo del intento |
| `planting_progress` | Uno menos tiempo restante dividido por duración |

Estado, valor y motivo permanecen separados. Cero, vencimiento, cancelación, conflicto, falta de prueba y no aplicabilidad conservan su significado. Progreso uno no acredita plantado exitoso, compromiso ni intención. Un intento reiniciado exige su propia transición positiva.

## Artefactos y acceso

Dos schemas estrictos elevan el registro a **96 schemas/19 razones**:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_attempt_origin_cuts.jsonl` | `stratai.support_preparation_attempt_origin_cut@1` | Todos los cortes y candidatos, cuatro hechos por actor y referencia causal separada |
| `support_preparation_attempt_origin_manifest.json` | `stratai.support_preparation_attempt_origin_manifest@1` | Fuentes/protocolo por hash, bytes/filas y matriz por par/lado/instancia/época |

Cada actor/corte aparece una vez; todos los pares conservan IDs, T0, denominadores, primeras muestras, exclusiones y ejecución. Cortes excluidos o sin pares mantienen listas vacías. `origin_reference` registra entrega, frame, reloj y vidas/handles del origen fuera de las features. La procedencia del corte contiene secuencia support, orden de reloj y fronteras de frames/entregas. La matriz cuenta estados/motivos por rol; `both_available` admite observado o no aplicable y no mide viabilidad táctica.

`preparation_attempt_origin.initial_features(support, objective, ownership, clocks, data, candidate_id, actor, ally, sequence, tick, frame)` comprueba pertenencia exacta al censo. Recorta los cinco streams **antes** de validar. El origen se corta por fronteras declaradas, sin inspeccionar frames o entregas posteriores, incluso dentro del mismo tick o FrameDone. Una entrega exacta ausente no se completa con otra futura.

Devuelve copias `self` y `ally` con posición/estado vivo C0 y cuatro hechos bajo `attempt_origin`. La API de relojes permanece separada. Identidad, procedencia, geometría enemiga, testigos, evaluación, conteos y futuro no entran en las features. El rol aliado mantiene el proxy de información del equipo, sin acreditar atención o comunicación.

El lector completo valida fuentes, schemas, hashes, bytes, filas y matriz por recomputación. El smoke añade una vía independiente de transiciones, relojes, continuidad, hechos y abstenciones; comprueba también los contextos C0 anteriores frente a la captura nueva. Los casos sintéticos se distinguen de las variantes observadas en la demo de desarrollo; pares y ventanas solapadas siguen correlacionados.

## Ejecución y límites

Después de suites públicas, AI Coach, Ruff y contrato, procesar la misma única demo aprobada a un TEMP nuevo/vacío separado por fase. Desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_attempt_origin --source <support_attempt_origin_observations.json> --output <TEMP-nuevo-vacío> --require-observed
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_attempt_origin --source <support_attempt_origin_observations.json> --output <mismo-TEMP-materializado> --validate --require-observed
```

El preflight rechaza destinos no temporales, enlaces/reparse points, escapes, directorios no vacíos y sobrescrituras antes de cargar fuentes. `--require-observed` exige candidatos y observaciones positivas de los cuatro hechos. Se comprueban 72 artefactos canónicos y se retienen salidas e inventarios; fuente, corpus y temporales anteriores son de sólo lectura.

No acredita posición física legal de C4, attachment transportado, observación de C4 caída, entrada física, compromiso, NAV/contacto legal, readiness completa, urgencia o alternativas superiores. Catálogo `1.7.0`, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparaciones, recomendaciones, entrenamiento, producto e interfaz temporal conservan su alcance. S2–S3 y las dependencias positivas pendientes permanecen abiertos. Sin limpieza, adquisición, migración 44/44, promoción, publicación ni cambios de `CURRENT`.
