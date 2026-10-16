# Origen nativo del intento de plantado v1

Extensión optativa `1.0.0-candidate.1`: `stratai.support_attempt_origin_observations@1` y `stratai.support_attempt_origin_context@1`. Conserva los [anclajes](canonical-support-anchors-v1.md), la [tenencia C4](canonical-support-c4-v1.md) y los [relojes nativos](canonical-support-objective-clock-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija el estado y la siguiente entrega.

## Origen y resolución

El selector `--ai-coach-support-attempt-origin-candidate` añade una captura de transiciones nativas de `CC4.m_bStartedArming`, asignaciones de `m_fArmedTime` y cambios de `m_hOwnerEntity`. No toma el primer sample de apoyo a 16 Hz como inicio del plantado. Tampoco convierte el último `BombPlantBegin` del SDK en una duración ni aplica una duración convencional.

En `demoinfocs-golang/v5@v5.2.0`, `sendtablescs2.Entity.readFields` aplica cada valor y después llama a los observadores de esa propiedad. Al crear una entidad, el SDK lee baseline y campos iniciales antes de registrar los observadores, ejecuta `OnCreateFinished` y posteriormente vuelve a emitir las propiedades registradas en orden alfabético. Por eso `OnCreateFinished` no permite distinguir por sí solo un inicio. La captura conserva `created`, `creation_replay`, `update` y `destroyed`; excluye como inicio todas las emisiones en el frame de creación, aunque el flag sea verdadero. Esa exclusión es conservadora también si se produjese un cambio adicional dentro de ese frame.

Cada entrega guarda secuencia, frame, tick de demo, vida C4, propiedad, valor anterior y nuevo, presencia, tipo decodificado y tipo declarado. Un cero nativo sigue siendo un valor. Los callbacks de creación/destrucción delimitan la vida; reutilizar el mismo handle completo no hereda un intento.

Un observador integrado en el callback existente de `FrameDone` sella las entregas antes de decidir si corresponde el sample support. Registra cada frame activo y los frames con cambios o fin de actividad. Los restantes frames inactivos no se duplican. Conserva todos los mensajes de tick de red, tick de entidades e intervalo recibidos en el frame; no presume orden entre callbacks del proyecto y el procesamiento interno del SDK. Más de un tick nativo distinto dentro del frame deja el origen temporal sin acreditar.

El tiempo de inicio representa el **tick nativo de la transición replicada**. No acredita el instante de pulsación, el subtick físico, el momento de llegada al cliente ni el redondeo de su barra de progreso.

## Prueba positiva y continuidad

Para derivar duración se exige:

1. Una única transición posterior a creación, de `false` nativo a `true` nativo, con predecesor registrado y orden coherente.
2. Una asignación positiva y no ambigua de `m_fArmedTime` en ese mismo frame. Puede preceder o seguir al flag dentro del frame; se lee el estado final al cerrarlo. Un deadline heredado de otro intento no basta.
3. Ambos ticks nativos presentes y concordantes en el frame, intervalo explícito y una prueba de origen 3E.6 ya disponible. Vida de rules, instancia, época, origen e intervalo/sesión deben coincidir. Una prueba posterior no completa un inicio anterior.
4. Controles nativos que acrediten ejecución sin pausa, fase `playing`, objeto C4 actual, clase implícita del callback `CC4` y definición nativa 49.
5. Propietario con handle completo y vida actual, valor nativo de propietario concordante e inventario con una referencia exacta a esa C4. Duplicados y reclamaciones contradictorias impiden acreditar el propietario.
6. Duración `deadline − tiempo_nativo_del_inicio` positiva y dentro del límite contractual de 120 s. Ese límite rechaza valores fuera del dominio de la sonda; no es un default de duración.

Desde el origen hasta T0, la continuidad exige todos los `FrameDone` activos, reloj sin retrocesos, el mismo dominio/ronda/época, deadline inalterado y el mismo propietario/pawn. Pausa, dato insuficiente, cambio de dueño, sustitución de vida, cambio de deadline o hueco invalidan la prueba. No se repara con el estado posterior. Una nueva transición corroborada puede abrir otro intento después de una cancelación.

La creación activa, el inicio ausente o tardío, la cancelación, los cambios de vida o propietario, la ambigüedad temporal y la duración insuficiente conservan motivos distintos en la fuente o derivación. Una pausa intermedia no se rellena suponiendo cuánto duró ni se conserva un porcentaje sin acreditar su continuidad.

## Acceso propio y aliado

La extensión genera cuatro hechos por cada rol, `self` y `ally`, en los mismos T0 y pares de las fases anteriores:

| Hecho | Significado |
|---|---|
| `planting_origin_state` | Origen de transición nativa acreditado o plazo expirado; insuficiencias y no aplicabilidad permanecen separadas. |
| `planting_duration_seconds` | Horizonte derivado entre la transición replicada y el deadline nativo asignado. |
| `planting_elapsed_seconds` | Duración menos tiempo restante, limitado al horizonte del intento. |
| `planting_progress` | Fracción `1 − restante / duración`, entre 0 y 1, con resolución de tick nativo. |

T0 exige además tenencia bilateral positiva 3E.5, intento de plantado y tiempo restante acreditados por 3E.6, frame exacto y concordancia de la identidad del intento. Un progreso de 1 debido al plazo expirado no acredita que la bomba se haya plantado. El progreso de desactivación y los demás relojes continúan en el contrato 3E.6 sin alterar sus artefactos.

`origin_reference` conserva procedencia por rol —entrega, frame, tick nativo, vida, inicio, deadline y duración— fuera de las features. `initial_features` recorta primero todas las fuentes hasta T0, valida ese prefijo y devuelve únicamente hechos propios/aliados. No devuelve identidades, handles, futuro ni timers de enemigos ocultos. Los hechos aliados conservan su significado de proxy de información del equipo, sin afirmar atención o comunicación.

## Artefactos y validación

`support_attempt_origin_observations.json` referencia con tamaño y SHA-256 las fuentes support, objetivo, C4 y relojes. Contiene entregas, frames sellados y un corte por cada muestra support; cada corte fija las fronteras de frames y entregas y el orden exacto de 3E.6. El schema es estricto. El validador comprueba predecesores, creación/destrucción, fuentes y fronteras, snapshots contra entregas y concordancia con los objetos del reloj en el corte común.

`support_attempt_origin_context.json` referencia también los contextos anteriores. Conserva IDs, selección, T0, denominadores, intervalos competitivos separados y anclajes preliminares. Su matriz cuenta los estados y motivos de los ocho hechos por actor/aliado/instancia. El lector recompone todas las capas anteriores y vuelve a calcular el informe nuevo.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-attempt-origin-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.attempt_origin `
  --source $attemptSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

Añadir `--validate` al segundo comando ejecuta la relectura. Se rechazan destinos no temporales, no vacíos y sobrescrituras antes del parseo. El cierre exige pruebas públicas y AI Coach, la única demo aprobada, recálculo independiente, controles de información legal y conservación de las capas anteriores y el canónico. Cada salida temporal se conserva con inventario y hashes.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` permanecen en `false`. La fracción temporal no acredita intención, compromiso táctico, ubicación física legal del objetivo, conexión NAV física, LOS/contacto, viabilidad de completar el intento ni superioridad de una alternativa. El contrato no cierra S2–S3 ni autoriza migración, promoción, entrenamiento o publicación.

El [informe de cierre](../../archive/ai-coach/AI_COACH_SUPPORT_ATTEMPT_ORIGIN_PHASE_2026-09-09.md) y la [evidencia](../../evidence/AI_COACH_SUPPORT_ATTEMPT_ORIGIN_SINGLE_DEMO_SMOKE_2026-09-09.json) registran conteos medidos, controles independientes y variantes reales frente a fixtures.
