# Posición nativa del objetivo y disponibilidad legal v1

Extensión optativa `1.0.0-candidate.1`: `stratai.support_objective_position_observations@1` y `stratai.support_objective_position_context@1`. Conserva los [anclajes](canonical-support-anchors-v1.md), la [tenencia C4](canonical-support-c4-v1.md), los [relojes](canonical-support-objective-clock-v1.md) y el [origen de plantado](canonical-support-attempt-origin-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija el estado de entrega.

## Fuente y sistema espacial

`--ai-coach-support-objective-position-candidate` añade observación directa de `CC4` y `CPlantedC4` en el mismo corte support. Cada objeto conserva clase, handle completo, vida, frame de creación, campos con presencia y tipos nativo/declarado, última entrega por campo y referencias actuales de propietario, desactivador y parent. La ausencia no se convierte en cero. Se capturan las seis propiedades de celda/origen, parent, rotación, escala y sondas de transformación/attachment, además de los miembros directos declarados bajo `CBodyComponent.`. Una propiedad no decodificable conserva su estado; no se inventa una transformación. Las rutas se resuelven en las declaraciones entregadas por la demo antes de consultar el SDK: éste lanza una excepción si se intenta descender por un handle escalar como si fuese una estructura. Las colecciones nativas conservan su presencia y tipo no soportado; no se recorren mediante rutas inventadas sin índice. La enumeración usa nombres de propiedad, no nombres de serializers; éstos pueden diferir en `PropertyEntries()`. Sin declaración completa, la sonda queda ausente con tipo declarado nulo.

En el SDK fijado `demoinfocs-golang/v5@v5.2.0`, `sendtables/sendtablescs2/entity.go` reconstruye cada componente como `celda × 512 − 16384 + desplazamiento`. El consumidor la recalcula a partir de `m_cellX/Y/Z` declarados `uint16` y desplazamientos finitos de tipo nativo flotante, declarados `CNetworkedQuantizedFloat` o `float32`. El paso de celda 512 no es un límite del desplazamiento: la demo puede entregar valores mayores y el SDK los suma sin recortarlos. No se impone un rango no acreditado por la fuente. Conserva por separado esta coordenada de celda y la coordenada mundial acreditada. No utiliza `Bomb.Position()`, `LastOnGroundPosition` ni la posición del portador como coordenada del objeto.

La coordenada de celda sólo se acredita como mundial si `CBodyComponent.m_hParent` es un handle nativo entero de 32 bits, declarado `CGameSceneNodeHandle`, con índice inválido explícito en sus 14 bits inferiores y referencia concordante. El decodificador de ese tipo es `unsignedDecoder`; el SDK usa índice de entidad de 14 bits. Cero es un handle no nulo y no acredita ausencia de parent. Un parent presente exige una transformación mundial actual acreditada; esta versión conserva abstención cuando no dispone de ella. Un `m_hMoveParent` nativo no nulo, un índice o token de attachment distinto de cero o una coordenada absoluta observada que contradiga la reconstrucción impiden usarla. Rotación y escala de una raíz no desplazan su origen.

El diagnóstico de estado distingue `carried`, `unowned`, `planted` y `resolved`. En ese diagnóstico global, `carried` significa definición nativa C4 y propietario resuelto; la disponibilidad inicial exige adicionalmente la prueba bilateral de inventario. `unowned` no acredita por sí solo caída observada, contacto con el suelo o accesibilidad. El estado plantado exige ticking verdadero y defused falso. Ninguno de estos diagnósticos acredita información del jugador por sí solo.

## Vida, orden y retención

Se registran entregas `created`, `creation_replay`, `update` y `destroyed`, con valor anterior y nuevo. La lectura inicial ocurre después de baseline/campos iniciales del SDK. Todas las reemisiones dentro del frame de creación permanecen clasificadas como creación, sin inferir un movimiento o una acción previa. Un sello de nacimiento al cerrar ese frame conserva la primera instantánea completa; la sustitución o destrucción de una vida impide heredar su estado. Se utilizan las vidas existentes de C4 y relojes, sin alterar sus IDs ni avanzar sus órdenes.

Los vectores nativos se conservan tanto si el SDK entrega slices como arrays fijos de tres flotantes; las rotaciones `QAngle` de esta demo usan `[3]float32`. Cada corte fija fronteras de entregas, nacimientos y estados. El consumidor verifica secuencias, predecesores, creación/destrucción, presencia/tipos, instantáneas contra el prefijo de entregas, identidad y censo contra C4/relojes, y frame/tick/instancia/época/orden contra el corte común. Un estado retenido representa el valor replicado vigente con su procedencia; no afirma que se haya entregado una nueva coordenada en cada sample. No se comparten estados a través de cambios de instancia o época.

## Disponibilidad propia y aliada

Por cada rol `self` y `ally` se generan tres hechos: `physical_objective_position`, `objective_object_state` y `position_availability`. Una coordenada requiere una única identidad legal y geometría mundial acreditada. Las vías positivas son:

| Base | Prueba exigida |
|---|---|
| `current_owned_c4` | Fase preplant, pawn actual, propietario exacto y tenencia bilateral positiva de 3E.5. |
| `current_defuse_contact` | Fase planted, desactivación nativa actual del rol y referencia exacta del desactivador a su pawn actual. |
| `witnessed_stationary_plant` | El mismo pawn ha completado un intento propio acreditado; el nuevo objeto plantado tiene origen mundial acreditado y no ha cambiado desde ese nacimiento. |

El plantado corroborado exige un único intento 3E.6B válido en el frame inmediatamente anterior, destrucción de su CC4 en el frame de creación de una única `CPlantedC4` del mismo propietario, evento público de plantado en ese mismo frame/instancia/época y reloj nativo actual concordante. El reloj actual debe haber alcanzado el deadline, pero no se inventa un plazo máximo de creación: la continuidad nativa del intento hasta el frame anterior y la transición de objetos acreditan la terminación observada. Un plazo expirado por sí solo no acredita plantado; tampoco se supone que creación y deadline sean simultáneos. El objeto nuevo debe estar activo y tener posición mundial positiva en su nacimiento. Ni el evento global ni el propietario cacheado del SDK sustituyen estas pruebas.

El conocimiento retenido de ese plantado se revoca ante cualquier cambio posterior relevante del cuerpo, parent, attachment, origen o propietario, o destrucción, incluso si el objeto vuelve después a la misma coordenada. Una actualización corporal no decodificable también invalida la continuidad, aunque ambos valores exportados sean `null`. También exige el mismo pawn, instancia y época. El contacto de desactivación actual puede acreditar una nueva disponibilidad. Esta versión no mantiene una posición recordada como si fuese la posición física actual de un objeto que se ha movido.

Tenencia pasada, muerte del portador, plantado global, spotted global y geometría de objetos ajenos no revelan automáticamente coordenadas exactas. La geometría ajena tampoco decide el motivo inicial de abstención. La vía aliada conserva el significado de proxy del equipo de las fases anteriores; no demuestra atención, voz o comunicación. Una C4 caída puede tener coordenada mundial nativa válida y seguir sin una observación legal atribuible al jugador.

`position_reference` conserva vida, corte, frame de disponibilidad y nacimiento fuera de las features. `initial_features` recorta primero las seis fuentes a T0, valida ese prefijo y devuelve únicamente los seis hechos. Una actualización o identidad futura no completa una prueba anterior ni revoca retroactivamente un hecho ya disponible.

## Artefactos y ejecución

`support_objective_position_observations.json` referencia con tamaño y SHA-256 las cinco fuentes anteriores. `support_objective_position_context.json` conserva selección, IDs, T0, separación competitiva y preliminar, denominadores y matriz por instancia/actor/aliado. La tabla `native_geometry` es diagnóstico global de la fuente; `native_geometry_is_initial_information` permanece en `false`. La matriz de hechos iniciales cuenta estados y motivos por rol; sus denominadores son anclajes, mientras el diagnóstico geométrico cuenta objetos por estado capturado.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-objective-position-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.objective_position `
  --source $positionSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

`--validate` en el segundo comando recompone las capas anteriores y el nuevo informe. El productor rechaza destino no temporal, no vacío o sobrescritura antes de parsear. La materialización exige TEMP vacío. El cierre requiere pruebas públicas y AI Coach, una única demo aprobada, recálculo independiente, contraste de artefactos anteriores y canónicos e inventarios retenidos de cada salida.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` permanecen en `false`. La coordenada del objetivo no acredita vínculo físico NAV, LOS/contacto visual, posibilidad de completar un intento, intención, cooperación, compromiso ni apoyo viable S2–S3. No habilita `DecisionExample`, `DEC-TRADE-001`, recomendaciones, entrenamiento o producto.

El [cierre 3E.7](../../archive/ai-coach/AI_COACH_SUPPORT_OBJECTIVE_POSITION_PHASE_2026-09-10.md) y su [evidencia](../../evidence/AI_COACH_SUPPORT_OBJECTIVE_POSITION_SINGLE_DEMO_SMOKE_2026-09-10.json) distinguen cobertura real, fixtures, correcciones, controles y salidas retenidas.
