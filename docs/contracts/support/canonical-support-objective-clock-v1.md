# Relojes causales nativos del objetivo v1

Extensión candidata `1.0.0-candidate.1`; schemas `stratai.support_objective_clock_observations@1` y `stratai.support_objective_clock_context@1`. Conserva los [anclajes](canonical-support-anchors-v1.md), el [objetivo público](canonical-support-objective-v1.md) y la [tenencia C4](canonical-support-c4-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) es la fuente de estado y siguiente acción.

## Captura y dominios

`--ai-coach-support-objective-clock-candidate` procesa una sola demo y añade el sidecar de relojes después de los cortes support, objetivo y C4 dentro del mismo callback `FrameDone`. Es optativo y exclusivo de los demás selectores. Los contratos anteriores y el bundle canónico `3.8.0` no cambian.

Cada corte conserva secuencia, frame, tick de demo, instancia, época, orden del objetivo, estado C4 y orden propio. Los últimos mensajes nativos se guardan como registros inmutables con orden, frame y tick de entrega. Un `FrameDone` posterior dentro del mismo tick puede observar un cambio. El orden es de entrega al observador; no acredita subtick físico ni orden entre callbacks internos del SDK.

| Fuente | Significado y comprobación |
|---|---|
| `CNETMsg_Tick.tick` | Tick nativo de red; se conserva presencia del puntero protobuf. |
| `CSVCMsg_PacketEntities.server_tick` | Tick nativo del mensaje de entidades. No se sustituye por el tick de demo. |
| `CSVCMsg_ServerInfo.tick_interval` | Intervalo nativo en segundos; ningún fallback de `TickRate()` lo rellena. |
| `CSVCMsg_SetPause.paused` | Estado explícito si el mensaje y campo están presentes. Ausencia no significa `false`. |
| `CCSGameRulesProxy.m_pGameRules.*` | Inicio y duración de ronda, reinicio, freeze, warmup, pausa general, espera de reanudación y pausas técnicas/de ambos equipos. |
| `CPlantedC4` | Vida y handle actuales, ticking/defused, deadline de explosión, duración nativa, deadline/duración de defuse y referencia completa al defusor. |
| `CC4` | Vida del mismo tracker C4, estado de arming y deadline `m_fArmedTime`; se sondea `m_flArmingLength` sin asignarle un default. |
| Pawn utilizable propio/aliado | Sonda de `m_flSimulationTime` y `m_nTickBase`, con identidad y vida exactas. Su ausencia permanece explícita. |

Los campos conservan presencia, tipo decodificado, valor y tipo declarado. El tipo declarado procede del `CSVCMsg_FlattenedSerializer` contenido en `CDemoSendTables`: se resuelve la ruta completa clase/propiedad, incluido `m_pGameRules`. Un nombre de campo o un número plausible no se convierten en prueba de `GameTime_t`. NaN/infinito, tipo no soportado y ausencia son estados diferentes, con valor JSON nulo. Cero y `false` observados se conservan.

Las vidas de rules/bomba se asignan a objetos creados y terminan al destruirse; se comprueba el mapa actual de entidades. Los objetos `CC4` reutilizan la vida ya acreditada en 3E.5. El defusor exige índice y serie completos y vida del pawn, sin usar una referencia SDK resuelta sólo por índice. Handles, vidas, ticks absolutos y aliases son procedencia, no features.

## Prueba positiva de correspondencia temporal

`Parser.CurrentTime()` en `demoinfocs-golang/v5@v5.2.0` usa el tick de la demo. No es el reloj empleado aquí para restar deadlines nativos.

El camino de origen de ronda exige simultáneamente:

1. Dos ticks nativos presentes, concordantes y recibidos en el frame exacto del corte.
2. Intervalo nativo observado entre `1/256` y `1/16` segundos.
3. `m_fRoundStartTime` observado y declarado `GameTime_t`, exactamente igual a `server_tick × tick_interval`.
4. Evento público `freeze_end` ya disponible en ese mismo frame/tick de demo, fase pública preplant coherente y réplica jugando.
5. Controles nativos de pausa/freeze/warmup/reinicio observados y compatibles con ejecución.

El informe conserva esta prueba en `clock_witnesses`, fuera del acceso inicial. Puede acreditar cortes posteriores sólo dentro de la misma instancia, época, vida de rules, mensaje de intervalo y origen nativo. Un cambio de origen, sesión/intervalo, vida, ausencia o desacuerdo de ticks o retroceso del reloj invalida la prueba. Una nueva prueba nunca rellena cortes anteriores. No se ajusta un offset usando el desenlace ni muestras futuras.

Existe además un camino directo cuando el pawn propio/aliado aporta `m_flSimulationTime` exactamente igual al tick nativo actual multiplicado por el intervalo. Se exige intervalo nativo `1/64` porque el decoder de esa propiedad en el SDK fijado usa explícitamente esa escala. La prueba de origen de ronda utiliza el intervalo nativo y no depende de ese decoder. El informe de fase distingue qué camino se ha verificado en la demo.

## Hechos iniciales y pausas

Cada rol `self`/`ally` recibe diez hechos, todos con estado, valor y motivo:

| Hecho | Regla |
|---|---|
| `pause_state` | `paused` con indicador positivo; `running` sólo con controles suficientes observados. Ausencia/ambigüedad conserva abstención. |
| `clock_state` | Correspondencia por origen nativo de ronda o por simulación propia exacta; no devuelve timestamps ni identidades. |
| `round_remaining_seconds` | En preplant: `max(0, inicio_nativo + duración_nativa − ahora_nativo)`. |
| `round_clock_state` | `active`, `paused` o `expired`; con bomba plantada deja de ser el reloj aplicable. |
| `bomb_remaining_seconds` | En fase pública plantada y con una entidad activa corroborada: `max(0, m_flC4Blow − ahora_nativo)`. |
| `bomb_clock_state` | `active`, `paused` o `expired`; un plazo agotado no se convierte en desenlace observado. |
| `attempt_state` | Plantando/desactivando, pausado, plazo expirado, interrupción acreditada entre cortes o ausencia/insuficiencia. |
| `attempt_remaining_seconds` | Deadline nativo del intento propio/aliado menos el reloj nativo acreditado, limitado inferiormente a cero. |
| `attempt_progress` | `1 − restante/duración` sólo con duración nativa positiva y coherente. Un deadline solo no acredita porcentaje. |
| `defuse_deadline_margin_seconds` | `m_flC4Blow − m_flDefuseCountDown`, con ambos deadlines `GameTime_t` de la misma bomba y defusor propio/aliado corroborado. Puede ser negativo o cero. |

Toda resta exige pausa observada y correspondencia temporal acreditada en el corte. Se expresa en segundos del reloj nativo, sin prometer redondeo del HUD ni tiempo de pared. Una pausa positiva conserva estado separado; no se suma tiempo de demo ni se proyecta cuándo se reanudará. Cambios de reloj/origen no acreditados invalidan el cálculo afectado.

Plantar requiere tenencia bilateral 3E.5 y el flag del mismo objeto `CC4`. Desactivar requiere flag nativo del jugador y referencia nativa de defusor con handle/vida coincidentes; una contradicción no se trata como ausencia. Un intento de enemigo oculto no es un hecho global. La cuenta atrás pública de bomba no depende del deadline ni identidad de ese enemigo.

Una interrupción exige intento propio/aliado observado en la muestra inmediatamente anterior, ausencia actual acreditada, mismo pawn, instancia/época y fase pública. No se hereda un timer cancelado ni se escanea el futuro para decidir si terminó. Vida nueva o información insuficiente no se convierten en interrupción.

## Artefactos y verificación

`support_objective_clock_observations.json` referencia por nombre, tamaño y SHA-256 los sidecars support, objetivo y C4. El contexto añade las referencias a todas las capas 3E.1–3E.5 y a la fuente nueva. Schemas estrictos y comprobaciones semánticas verifican identidad, orden, fronteras y presencia. El lector recompone las capas anteriores y el informe nuevo; los archivos se abren de forma exclusiva al escribir.

La matriz por actor/aliado/instancia conserva los denominadores y estados/motivos de los veinte hechos. Los intervalos competitivos y anclajes preliminares mantienen su estado anterior. El acceso inicial recorta todas las fuentes hasta T0 antes de validar y calcular.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-objective-clock-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.objective_clock `
  --source $clockSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

`--validate` en el segundo comando relee y recalcula. El productor rechaza destinos no temporales, no vacíos y sobrescrituras antes de parsear. Cada cierre usa la única demo aprobada después de las pruebas; una corrección repite esa misma partida en otro TEMP. Todas las salidas se retienen inventariadas.

## Límites

Deadlines, progreso nativo y margen firmado no prueban viabilidad de llegar, completar el intento, responsabilidad táctica o superioridad de una alternativa. No acreditan ubicación física del objetivo, conexión NAV física, LOS/contacto, intención o cooperación. `comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` siguen en `false`; no se modifica `DecisionExample`, `DEC-TRADE-001` ni se cierra S2–S3.

Cada pieza insuficiente conserva su dependencia concreta. La [evidencia](../../evidence/AI_COACH_SUPPORT_OBJECTIVE_CLOCK_SINGLE_DEMO_SMOKE_2026-09-09.json) y el [informe de la fase](../../archive/ai-coach/AI_COACH_SUPPORT_OBJECTIVE_CLOCK_PHASE_2026-09-09.md) distinguen cobertura real y variantes probadas sólo con fixtures; una propiedad no observada en estas sondas no demuestra ausencia de todas sus posibles fuentes.
