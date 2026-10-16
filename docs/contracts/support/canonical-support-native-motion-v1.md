# Movimiento nativo propio y aliado v1

Extensión optativa `1.0.0-candidate.1`: `stratai.support_native_motion_observations@1` y `stratai.support_native_motion_context@1`. Conserva los [anclajes](canonical-support-anchors-v1.md), el [contexto físico](canonical-support-physical-v1.md) y todas las capas hasta la [posición del objetivo](canonical-support-objective-position-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene el estado de validación y la siguiente acción.

## Fuente y corte

El selector `--ai-coach-support-native-motion-candidate` observa `CCSPlayerPawn` mediante sus campos declarados. Se consultan velocidad, velocidad absoluta/base, variantes de componentes, flags, referencia de suelo, código de movimiento, parent y algunos campos históricos del servicio de movimiento. Cada valor conserva presencia, tipo declarado, tipo nativo y estado de decodificación. Las rutas se resuelven antes de consultar el SDK; no se desciende por un vector o handle escalar como si fuera una estructura.

Se conservan entregas `created`, `creation_replay`, `update` y `destroyed`. Una vida local del observador identifica el objeto nativo desde su creación; no sustituye la vida de pawn del contrato support. El corte enlaza ambas mediante el mismo objeto actual, handle completo, serie, actor y condición de disponibilidad de las capas anteriores. Se vuelve a comprobar la entidad actual, sin usar un pawn cacheado que ya no existe.

Los estados se comparten por pawn sólo cuando valores, referencias y últimas entregas coinciden dentro de la misma instancia y época. Cada corte conserva frame, tick y orden común, frontera de entregas, frontera de estados y correspondencia de jugadores. Un valor retenido es el estado replicado vigente, con su procedencia; no implica una entrega nueva en cada sample ni una medición física continua.

Las referencias de suelo y parent se resuelven contra las entidades actuales y la serie completa. Sus vidas de referencia se asignan al primer encuentro; no afirman conocer el nacimiento nativo de esa superficie. El handle cero puede identificar una entidad mundial válida. La ausencia y el sentinel inválido se mantienen separados.

## Hechos iniciales

Cada rol `self` y `ally` aporta seis hechos independientes:

| Hecho | Condición |
|---|---|
| `native_velocity` | Vector nativo finito de tres componentes en `m_vecAbsVelocity` o `m_vecVelocity`, declarado `VectorWS`, con referencia mundial explícita y contexto de parent sin contradicción. |
| `native_horizontal_speed` | Magnitud horizontal del mismo vector acreditado; conserva sus unidades nativas. |
| `native_speed` | Magnitud de los tres componentes del mismo vector acreditado. |
| `ground_contact` | Bit `OnGround` nativo de `m_fFlags`, declarado `uint32`, corroborado por la referencia actual y tipada de suelo. |
| `ground_reference_state` | `world`, `entity` o `none`, sólo con la misma prueba positiva de contacto o ausencia. |
| `movement_mode_code` | Código entero nativo declarado `MoveType_t`; no se inventan etiquetas semánticas tomadas de otra versión del motor. |

El vector mundial se conserva si el SDK entrega un array fijo o un slice de tres flotantes. Las fuentes positivas simultáneas deben concordar. Un parent no acreditado, parent alternativo o attachment contradictorio impide utilizar esa vía. La magnitud cero de un vector presente es válida; un campo ausente no se convierte en un vector cero. Un vector declarado simplemente `Vector`, sin referencia espacial suficiente, conserva abstención.

La velocidad base, los límites de velocidad, la caída y las velocidades históricas de salto o aterrizaje permanecen como datos de procedencia. No sustituyen la velocidad instantánea del pawn, ni se reetiqueta como nativa la velocidad media por diferencias de posición de 3E.2. La declaración de un campo tampoco demuestra que exista un valor en un corte concreto. Las variantes que sólo aparezcan en fixtures se distinguirán de la cobertura de la demo en la evidencia de cierre.

Para contacto con suelo, el bit positivo exige referencia resuelta. El bit negativo exige el sentinel inválido explícito. Un desacuerdo entre ambas fuentes, una serie incorrecta o un campo ausente conserva abstención. El contacto con una entidad móvil no acredita suelo estático, una ruta física ni intención de saltar.

## Lectura y validación

`support_native_motion_observations.json` referencia mediante tamaño y SHA-256 las seis fuentes lógicas previas, incluida la posición del objetivo. `support_native_motion_context.json` conserva selección, IDs/T0, estado competitivo o preliminar y matriz por instancia/actor/aliado. Los denominadores de hechos cuentan anclajes; los conteos de campos nativos cuentan estados del pawn capturados.

El consumidor contrasta identidad de fuente, censo de jugadores, corte común, fronteras, creación/reemisión/destrucción, predecesores de cada entrega, campos de cada estado contra el prefijo y referencias actuales. Una vida nativa no puede heredarse por otra vida de pawn, ni un estado atravesar una instancia o época.

`initial_features` recibe support, la fuente de posición que fija el corte común y la fuente de movimiento. Recorta primero las tres a T0, valida el contrato afectado y devuelve sólo los hechos propios y aliados. Las identidades, los índices de procedencia y la geometría ajena quedan fuera de las features. `motion_reference` conserva esos vínculos por separado. El materializador y el lector completo validan también la cadena de fuentes y contextos anteriores.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-native-motion-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.native_motion `
  --source $motionSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

Añadir `--validate` al segundo comando para recomponer y contrastar todos los contextos. El productor rechaza destinos no temporales, no vacíos o sobrescribibles antes de procesar la demo; la materialización exige TEMP vacío. El cierre requiere suites, la misma única demo aprobada, registros afectados no vacíos, recálculo independiente, conservación de capas y canónico e inventarios retenidos.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` permanecen en `false`. Estos hechos no acreditan trayectoria física, entrada/compromiso, intención, contacto visual, readiness completa, cooperación ni apoyo viable S2–S3. Las dependencias de posición del objetivo, NAV físico y economía conservan su alcance propio.

La validación de una demo se registra en el [cierre 3E.8](../../archive/ai-coach/AI_COACH_SUPPORT_NATIVE_MOTION_PHASE_2026-09-10.md) y su [evidencia](../../evidence/AI_COACH_SUPPORT_NATIVE_MOTION_SINGLE_DEMO_SMOKE_2026-09-10.json). Acredita contacto con suelo y código de movimiento en los anclajes propios/aliados; los dos vectores actuales consultados permanecen ausentes en todos los estados capturados. La vía positiva de velocidad mundial se comprueba con fixtures y conserva su dependencia de fuente real.
