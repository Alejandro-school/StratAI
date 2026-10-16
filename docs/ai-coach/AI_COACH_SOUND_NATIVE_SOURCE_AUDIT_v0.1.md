# Auditoría de fuentes sonoras nativas v0.1

12/10/2026. Incremento de fuentes dentro del [encuadre recibido](AI_COACH_SOUND_INFORMATION_SCOPE_v0.2.md).
El estado y siguiente acción pertenecen al [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md).

El módulo [source_audit.go](../../backend/go-service/pkg/sound/source_audit.go)
y el comando [sound-source-audit](../../backend/go-service/cmd/sound-source-audit/main.go)
observan la demo con demoinfocs **v5.2.0**, sin alterar los productores históricos
ni conectar información nueva al coach. Contrato exclusivo del diagnóstico:
`stratai.native_sound_source_audit@1`. No es una versión nueva de replay,
bundle, proyección T/CT o entrada de Qwen.

| Fuente | Dato positivo que puede recoger | Límite |
|---|---|---|
| `events.Footstep` | Callback de paso, emisor y posición nativa cuando hay pawn | No publica radio |
| `events.PlayerSound` | Callback genérico, radio y duración nativos | No clasifica el sonido; semántica y unidades del alcance por verificar |
| `CSVCMsg_Sounds` | Registros decodificados, coordenadas y `sound_level` presentes | `sound_level` no se convierte en radio |
| `CCSUsrMsg_WeaponSound` | Mensaje, nombre y coordenadas nativas presentes | Sin radio explícito |
| `CMsgSosStartSoundEvent` | Mensaje de inicio, hash y parámetros empaquetados | Se retiene longitud/hash del paquete; no se adivina su decodificación |

Los mensajes de parada, actualización de parámetros y biblioteca se cuentan
separadamente. Las definiciones de `player_sound` y `player_footstep` se recogen
si llegan por la lista de eventos. Una definición no acredita una emisión.
Los contadores de eventos de juego crudos son independientes de los callbacks
tipados. Además se cuentan `GenericGameEvent` resueltos por el parser: la lista
interna de respaldo puede dar nombre al evento aunque falte el paquete de
definiciones. Un paquete sin nombre no se declara sonido ausente. La coincidencia
entre eventos resueltos y callbacks tipados se contrasta en la auditoría real;
los nombres crudos originales permanecen intactos. Los IDs numéricos de mensajes
no soportados se registran para delimitar la cobertura del decoder.

Disparos, recargas, lanzamientos y acontecimientos C4 se cuentan como **acciones**.
No fabrican señales ni recepción auditiva. `reception_status` permanece
`unavailable`; un radio positivo queda `native_radius_semantics_unverified`.
Radio cero/negativo queda inválido; radio ausente sigue ausente. No hay radios
predeterminados ni clasificación por coincidencia temporal.

El tick y su disponibilidad pertenecen al estado del parser en el callback;
no prueban la hora física de escucha. Las coordenadas de pawn son nativas a
ese callback, a la altura de sus pies. Pawn ausente no se convierte en `(0,0,0)`
observado. Estas posiciones pertenecen a auditoría y no se publican como
localización enemiga reconocida. Cada ID de participante observado se sustituye
por un alias local antes de retener el artefacto, sin mapa inverso ni nombres.

El comando exige un destino nuevo, vacío y físicamente dentro de TEMP; rechaza
rutas externas, enlaces/reparse points y sobrescrituras antes de abrir la demo.
Comprueba el SHA-256 antes de crear el archivo de salida. El wrapper del cierre
además verifica suites sobre los mismos hashes de fuente y la única partida
aprobada, junto con los 44 manifiestos antes/después. Todos los temporales se
retienen. Cero señales es un resultado de cobertura del decoder sobre esa
partida; no demuestra que CS2 fuese silencioso ni que ninguna otra demo o fuente
pueda contener información sonora.

Fuentes primarias del código fijado: [eventos](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/v5.2.0/pkg/demoinfocs/events/events.go),
[callbacks](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/v5.2.0/pkg/demoinfocs/game_events.go)
y [decodificación de mensajes](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/v5.2.0/pkg/demoinfocs/s2_commands.go).
Los hashes locales y límites de cada fuente se conservan en el
[registro](../../outputs/stratai-sound-20261012/source-references.json).
