# Posición legal del objetivo en C0 v1

Extensión optativa `1.0.0-candidate.1`, fase 3E.18. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija su estado y continuación. Une los tres hechos de [posición 3E.7](canonical-support-objective-position-v1.md) al corte exacto del [censo C0](canonical-support-preparation-v1.md). Conserva los contextos [físico](canonical-support-preparation-physical-v1.md), [objetivo](canonical-support-preparation-objective-v1.md), [C4](canonical-support-preparation-c4-v1.md), [reloj](canonical-support-preparation-objective-clock-v1.md) y [origen de plantado](canonical-support-preparation-attempt-origin-v1.md).

## Fuente y frontera causal

Reutiliza `--ai-coach-support-objective-position-candidate` sin modificar Go ni ampliar las propiedades capturadas. Support, objetivo, C4, reloj, intento y posición se enlazan por alias, checksum, build, bytes y SHA-256, incluyendo el manifest canónico. La unión exige secuencia support, tick, FrameDone, instancia/época, orden del reloj y fronteras exactas de estados, nacimientos y entregas.

Cada corte conserva el estado replicado vigente con su procedencia. No afirma que se haya recibido una nueva coordenada en cada muestra. Un estado futuro o un corte distinto con el mismo tick no completa T0. Se validan predecesores, creación/replay/destrucción, instantáneas contra entregas, censo de objetos, vidas y handles completos, propietario y desactivador contra C4/reloj.

La geometría mantiene la aritmética y los tipos nativos de 3E.7: `celda × 512 − 16384 + desplazamiento`. Una coordenada de celda sólo es mundial con raíz y parent/attachment acreditados, sin contradicciones de coordenadas absolutas. La posición del portador, los defaults del SDK y la geometría de otro objeto no sustituyen esa prueba. La ausencia y cero permanecen distintos.

## Tres hechos y disponibilidad legal

Cada actor seleccionado tiene `physical_objective_position`, `objective_object_state` y `position_availability`, con estado, valor y motivo separados. Las vías positivas conservan las exigencias de 3E.7:

| Vía | Evidencia exigida |
|---|---|
| `current_owned_c4` | Fase preplant, pawn actual, propietario exacto, inventario bilateral y geometría mundial de la C4 propia. |
| `current_defuse_contact` | Fase planted, desactivación nativa actual y referencia exacta del objeto al pawn del rol. |
| `witnessed_stationary_plant` | Intento propio nativo acreditado hasta el frame anterior, destrucción CC4 y nacimiento CPlantedC4, evento público, reloj/deadline y raíz mundial positivos; posición sin cambios desde nacimiento. |

El conocimiento de un plantado exige continuidad desde el **inicio nativo del intento** hasta T0: mismo alias C0, lado, vida y handle del pawn, instancia y época en todas las entregas support desde la última anterior o igual al inicio. La cadena de 3E.6B exige también continuidad de cada frame activo, C4/propietario/deadline/dominio y ausencia acreditada de pausa. Si falla la continuidad C0, los tres hechos quedan indisponibles con `native_position_c0_actor_continuity_unaccredited`.

Cualquier cambio corporal, movimiento, parent/attachment, propiedad o destrucción relevante revoca la prueba estacionaria; volver a la misma posición no la repara. Una actualización corporal no decodificable tampoco conserva el conocimiento. Un contacto nativo actual de desactivación puede acreditar una nueva disponibilidad. No se hereda conocimiento a través de cambios de identidad o lado.

Evento global, spotted, tenencia pasada, muerte del portador y progreso uno no acreditan posición legal. Una C4 caída puede tener coordenada mundial válida y seguir sin observación atribuible. El rol aliado conserva el proxy del equipo; no demuestra atención, voz o comunicación.

## Artefactos y acceso

Dos schemas estrictos elevan el registro a **98 schemas/19 razones**:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_objective_position_cuts.jsonl` | `stratai.support_preparation_objective_position_cut@1` | Todos los cortes/candidatos, tres hechos por actor y referencia legal separada |
| `support_preparation_objective_position_manifest.json` | `stratai.support_preparation_objective_position_manifest@1` | Fuentes/protocolo por hash, bytes/filas y matriz por par/lado/instancia/época |

Cada actor/corte aparece una vez aunque participe en varios pares. Se mantienen IDs/T0, primeras muestras, exclusiones, denominadores y ejecución. Cortes excluidos o sin pares conservan listas vacías. La ausencia de posición no elimina candidatos. `position_reference` registra vía, vida de objeto, corte, frame de disponibilidad y nacimiento fuera de las features. La procedencia del corte mantiene las cinco fronteras exactas; el diagnóstico geométrico global permanece en la fuente/capa 3E.7.

`preparation_objective_position.initial_features(support, objective, ownership, clocks, attempt, data, identifier, actor, ally, sequence, tick, frame)` recibe el ID del candidato en `identifier` y comprueba pertenencia exacta al censo. Recorta las seis fuentes **antes** de validar: corta estados, nacimientos, frames activos, entregas y eventos por sus fronteras declaradas, sin recorrer futuro malformado, incluso dentro del mismo tick o FrameDone.

Devuelve copias `self` y `ally` con posición/estado vivo C0 y tres hechos bajo `objective_position`. Identidad, procedencia, geometría ajena, testigos, evaluación, conteos y futuro no entran en las features. Las APIs de las capas anteriores permanecen separadas.

El lector completo valida fuentes, schemas, hashes, bytes, filas y matriz por recomputación. `both_available` admite observado o no aplicable; no mide viabilidad táctica. El smoke añade recálculo independiente de coordenadas, snapshots, nacimiento/plantado, disponibilidad, abstenciones, referencias y matriz, además de conservación C0 y sus cinco contextos. La cobertura real se separa de las variantes sintéticas; la demo de desarrollo y los pares/ventanas correlacionados no acreditan generalización.

## Ejecución y límites

Después de suites públicas, AI Coach, Ruff y contrato, procesar la misma única demo aprobada a TEMP nuevo/vacío, separado por fase. Desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective_position --source <support_objective_position_observations.json> --output <TEMP-nuevo-vacío> --require-observed
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective_position --source <support_objective_position_observations.json> --output <mismo-TEMP-materializado> --validate --require-observed
```

El preflight rechaza destinos no temporales, enlaces/reparse points, escapes, directorios no vacíos y sobrescrituras antes de cargar fuentes. `--require-observed` exige candidatos y observaciones positivas de los tres hechos. Se comprueban 72 artefactos canónicos; se retienen salidas e inventarios, con fuente, corpus y temporales anteriores de sólo lectura.

La coordenada legal no acredita attachment transportado, observación de C4 caída, entrada física, compromiso, NAV/contacto visual, velocidad instantánea, readiness completa, intención, cooperación o alternativa superior. Catálogo `1.7.0`, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparación, recomendaciones, entrenamiento, producto e interfaz temporal conservan su alcance. S2–S3 y sus dependencias positivas siguen abiertos. Sin limpieza, adquisición, migración 44/44, promoción, publicación ni cambios de `CURRENT`.
