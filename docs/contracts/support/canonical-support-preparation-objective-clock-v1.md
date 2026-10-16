# Relojes causales del objetivo en C0 v1

Extensión optativa `1.0.0-candidate.1`, fase 3E.16. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado y continuación. Une los [relojes nativos 3E.6](canonical-support-objective-clock-v1.md) al corte exacto del [censo C0](canonical-support-preparation-v1.md), conservando sus contextos [físico](canonical-support-preparation-physical-v1.md), de [objetivo](canonical-support-preparation-objective-v1.md) y de [C4](canonical-support-preparation-c4-v1.md).

## Fuente y frontera

Reutiliza el selector `--ai-coach-support-objective-clock-candidate`, sin cambiar Go ni añadir propiedades. Sus fuentes support, objetivo, C4 y reloj se enlazan por alias, checksum, build, bytes y SHA-256, incluyendo el manifest canónico. La unión exige secuencia support, tick, FrameDone, instancia/época, orden del objetivo, estado C4 y entrega del reloj exactos. Actor/pawn, vida, handle completo, propietario y objeto conservan las comprobaciones de 3E.5–3E.6. No se traslada un anclaje próximo ni otro FrameDone con igual tick.

Los diez hechos por actor conservan la semántica de 3E.6: pausa y correspondencia temporal; tiempo restante y estado del reloj de ronda y bomba; estado, tiempo restante y progreso nativo del intento; margen firmado de desactivación. Cada hecho mantiene estado, valor y motivo. Cero, vencimiento, ausencia, conflicto y no aplicabilidad siguen siendo distintos.

Toda resta exige tipos nativos, dominio GameTime_t cuando corresponde, intervalo observado, pausas y origen causal acreditados. La prueba de origen de ronda sólo se propaga dentro de la misma instancia/época, vida de rules, origen y sesión de intervalo, sin retrocesos. La vía de simulación propia exige igualdad exacta y escala nativa compatible con el decoder fijado. Un origen descubierto después no completa T0. Tiempo transcurrido público y defaults no sustituyen estas pruebas.

Una interrupción requiere intento propio/aliado en la entrega support inmediatamente anterior, ausencia actual acreditada, misma fase pública, instancia/época, lado, vida y handle del pawn. No conserva un timer cancelado ni salta huecos. Un plazo vencido no acredita éxito.

## Artefactos y acceso

Dos schemas estrictos elevan el registro a **94 schemas/19 razones**:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_objective_clock_cuts.jsonl` | `stratai.support_preparation_objective_clock_cut@1` | Todos los cortes, hechos por actor seleccionado y referencias de todos los candidatos |
| `support_preparation_objective_clock_manifest.json` | `stratai.support_preparation_objective_clock_manifest@1` | Fuentes/protocolo por hash, bytes/filas, conteos y matriz por par/lado/instancia/época |

Cada actor/corte se almacena una vez aunque participe en varios pares dirigidos. Cortes excluidos o sin pares conservan listas vacías. La falta de reloj nunca elimina candidatos, IDs, T0, denominadores, primeras muestras, exclusiones ni ejecución. La procedencia guarda los órdenes exactos, estado C4 y testigo causal del reloj; propiedades, entidades y mensajes permanecen en las fuentes enlazadas. Las capas C0 previas conservan sus artefactos y APIs.

`preparation_objective_clock.initial_features(support, objective, ownership, data, candidate_id, actor, ally, sequence, tick, frame)` comprueba pertenencia exacta al censo. Recorta cada stream antes de validar: support por ordinal, objetivo por corte y frontera pública, C4 por corte/estado y relojes por entrega. Se detiene en el corte exacto o primer localizador posterior, sin recorrer el resto del futuro. Una muestra exacta ausente mantiene ausencia explícita.

Devuelve copias `self` y `ally` con posición/estado vivo C0 y diez hechos bajo `objective_clock`. Identidad, procedencia, testigos, geometría enemiga, evaluación, conteos y futuro quedan fuera de features. Los estados aliados son proxy de información del equipo; no acreditan atención, comunicación o intención.

El lector completo valida todas las fuentes, schemas, bytes, hashes, filas y matriz por recomputación. La matriz conserva estados/motivos y disponibilidad conjunta de cada hecho; `both_available` admite observado o no aplicable y no mide viabilidad táctica. El smoke añade un auditor independiente que recalcula valores y abstenciones desde las fuentes, y recompone población C0 y contextos anteriores. Pares y ventanas solapadas siguen correlacionados.

## Ejecución y límites

Tras las suites públicas y AI Coach, procesar exactamente la misma única demo aprobada a TEMP nuevo/vacío y ejecutar desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective_clock --source <support_objective_clock_observations.json> --output <TEMP-nuevo-vacío> --require-observed
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective_clock --source <support_objective_clock_observations.json> --output <mismo-TEMP-materializado> --validate --require-observed
```

El preflight rechaza destinos no temporales, enlaces/reparse points, escapes, directorios no vacíos y sobrescrituras antes de cargar fuentes. `--require-observed` exige candidatos y registros observados de ronda, bomba, intento, progreso y margen. Se comprueban también 72 artefactos canónicos. Salidas e inventarios se retienen; fuente, 44 bundles y temporales anteriores son de sólo lectura.

Esta extensión no une el origen/progreso de plantado de 3E.6B ni acredita urgencia táctica, posición física de C4 transportada/caída, entrada física, compromiso, NAV/contacto legal, readiness completa o alternativas superiores. Cada capa conserva sus dependencias positivas. Catálogo `1.7.0`, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparaciones, recomendaciones, entrenamiento, producto e interfaz temporal mantienen su alcance. S2–S3 permanecen abiertos; sin adquisición, limpieza, migración 44/44, promoción, publicación ni cambios de `CURRENT`.
