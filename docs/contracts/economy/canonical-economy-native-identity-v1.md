# Identidad y relojes económicos nativos v1

Contrato candidato `stratai.economy_native_identity@1`, versión `1.0.0-candidate.1`. El selector optativo **8** conserva las salidas de los selectores 1–7 y añade `economy_native_identity.json`, enlazado por ruta, tamaño y SHA-256 a la traza de 3D.3. Export base `3.8.0`, parser `v16`, catálogo `1.7.0` y elegibilidad se conservan.

## Procedencia y relojes

La dependencia instalada es `demoinfocs-golang/v5@v5.2.0`, comprobada con `go mod verify`; la evidencia de fase fija los hashes de los archivos examinados. En Source2, `parseFrame` toma el tick de la cabecera del comando de demo y lo envía a `handleIngameTickNumber`. Ese es el valor que devuelve `GameState.IngameTick()`: su comentario genérico no demuestra que sea el mismo reloj que el `server_tick` de un evento. [Parser](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/parsing.go), [estado del SDK](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/game_state.go).

`handleDemoPacket` lee los mensajes y los ordena establemente por prioridad: `CNETMsg_Tick` y contexto tienen prioridad −10; mensajes ordinarios, 0; `PacketEntities`, 5. El orden de entrega no es el orden original de los bytes ni el orden subtick del servidor. La extensión captura **cada `CNETMsg_Tick` entregado**, con presencia, valor, tick del comando, fotograma local, secuencia y segmento de reloj. [Prioridades del SDK](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/s2_commands.go).

Los segmentos —`epoch`— sólo separan muestras observadas. Cambian ante retroceso del tick de demo, retroceso de red o un valor de red ausente, con razón explícita; no certifican un reinicio del servidor. Un valor ausente rompe continuidad y no se convierte en cero. Las filas enlazan únicamente la última muestra de red entregada antes de ellas. Las coincidencias entre tick de evento y red se cuentan por igualdad exacta **en el mismo fotograma y tick de demo**. Si falta el reloj, procede de otro fotograma o difiere, se conserva esa categoría. No se calcula ni aplica un desplazamiento global ni se interpolan instantes de ejecución.

Las fronteras `RoundStart`, `RoundFreezetimeEnd` y `RoundEndOfficial` conservan la instancia del observador económico y se recontrastan con las fronteras v2 verificadas. No se enlazan por número de ronda repetido. Toda la extensión es evidencia retrospectiva; no entra en T0 ni acredita conocimiento del jugador.

### Orden entre consumidores del proyecto

`godispatch@v1.4.1` inicializa su caché recorriendo mapas de Go. Su comentario sobre orden de registro no constituye una garantía de implementación. En el primer smoke de esta fase, el consumidor de contexto se ejecutó antes del productor de frontera y faltaron las 25 sondas de inicio: el validador rechazó la salida. [Implementación del despachador](https://github.com/markus-wa/godispatch/blob/v1.4.1/dispatch.go).

El selector 8 registra un solo callback del proyecto por tipo concreto y entrega sus consumidores en orden explícito de registro. Así la frontera precede al contexto y la traza precede a su enlace de identidad. Los selectores 1–7 mantienen sus rutas. La corrección **no garantiza el orden relativo entre lógica interna del SDK y consumidores del proyecto**: un evento nativo y su callback pueden entregarse en orden distinto entre ejecuciones. Sus secuencias son evidencia de esa ejecución, no una identidad estable para unir builds. Los controles históricos deben contrastar contenido y multiplicidad por canal, además de registrar cualquier cambio de orden; nunca ordenar la evidencia retenida para simular causalidad.

## Vida de entidades y referencias

La sonda observa `CCSPlayerPawn` y las clases que el SDK reconoce como equipamiento por `m_iItemDefinitionIndex` y `m_iClip1`. Cada objeto de entidad recibe un alias `native-entity-*` nuevo. Se conservan índice y número de serie locales de la demo, clase, creación observada y callback de destrucción. La creación se observa después de decodificar los valores iniciales: no implica compra ni acredita el historial anterior de ese objeto.

Las referencias `m_hOwnerEntity` y los 64 slots `m_pWeaponServices.m_hMyWeapons.0000`–`.0063` conservan el handle efímero original, presencia, estado de resolución y alias de destino. El índice ocupa los 14 bits bajos; el resto se compara íntegramente con el número de serie. El parser Source2 lee **17 bits de serie**. No se trunca a los diez bits de una constante heredada. Se comprueba además que la entidad esté disponible en ese momento y pertenezca a la clase prevista. [Lectura y búsqueda Source2](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/sendtables/sendtablescs2/entity.go).

Esto evita el acceso sólo por índice de `GameState.EntityByHandle`/`Participants.FindByPawnHandle`. Ausencia, tipo no admitido, sentinel inválido, entidad no disponible, serie distinta y clase distinta son estados separados. Una referencia anterior a la creación no se completa más tarde. El handle es local a esa demo; no es SteamID, ID de artículo de cuenta ni una identidad estable entre partidas. Los actores conservan exclusivamente aliases ya disponibles; su controlador debe apuntar mediante `m_hPlayerPawn` al pawn con índice y serie coincidentes. No se exportan identificadores personales ni nombres de jugadores.

Al crear entidades, el SDK vuelve a entregar a los handlers las propiedades registradas, ordenadas alfabéticamente, al final de `OnPacketEntities`. Por eso las lecturas iniciales son `*_baseline` y los callbacks del mismo fotograma se marcan **`creation_frame_or_replay`**. Esa marca admite que puedan mezclarse entregas replicadas y reenvíos de creación: no declara que cada callback sea una nueva modificación nativa. Los posteriores tienen `later_frame_callback`. Las actualizaciones de longitud del inventario se conservan separadas de los slots; un slot fuera de la longitud activa no prueba pertenencia. [Reenvío de propiedades de creación](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/sendtables/sendtablescs2/entity.go).

La definición de objeto identifica su tipo; no identifica una compra. El callback de destrucción no identifica su causa. Un dueño o slot observado tampoco demuestra drop voluntario, regalo, cooperación o devolución. El enlace con un callback de la traza 3D.3 conserva la entidad concreta que devolvió el SDK; su asociación con el evento nativo sigue dependiendo del mecanismo del SDK, que puede resolver por tipo.

## Mensajes de recogida y drop

| Fuente observada | Contenido conservado | Límite |
|---|---|---|
| `CUserMessageItemPickup` | Presencia y texto acotado `itemname` | El cuerpo no contiene actor ni entidad. |
| `CCSUsrMsg_ItemPickup` | Presencia y texto acotado `item` | No se transforma en compra ni se une por nombre. |
| `CCSUsrMsg_ItemDrop` | Alias local del `itemid` presente y booleano `death` con presencia | El ID de artículo se mantiene en su propio espacio; no se equipara a un handle. No se inventa actor. |
| `CSVCMsg_UserMessage`, tipos económicos permitidos | Tipo, bytes y hash del payload | Payload sin interpretar y sin destinatario acreditado; no se supone que corresponda a otra entrega directa. |

Se conservan contadores explícitos para las cuatro fuentes, incluidos ceros. Un cero sólo describe esta sonda sobre esta demo. Los wrappers no se desempaquetan mediante posiciones o convenciones adivinadas. [Cuerpos CS](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/cstrike15_usermessages.pb.go), [mensajes base](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/usermessages.pb.go), [wrapper](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/netmessages.pb.go).

## Validación

Desde la raíz: `backend/venv/Scripts/python.exe -B -m ai_coach.economy.native_identity --source $identityFile --output $newEmptyTemp --require-observed`. Se rechazan destinos no temporales, no vacíos y enlaces/reparse points antes de leer datos. La entrada fija y su cadena de hashes se validan hasta el candidato v2 y los artefactos canónicos.

El lector recalcula la secuencia conjunta de relojes/filas, los segmentos, disponibilidad causal del reloj, vida/serie/clase de cada referencia, fases de creación, enlaces exactos a la traza y fronteras de instancia. Exige registros reales de reloj, creación, destrucción, referencias resueltas y traza. No sustituye cobertura ausente de mensajes con fixtures. Un recálculo independiente y la revisión canónica completa siguen formando parte del gate real de una única partida.

El contrato no habilita cestas legales, transacciones itemizadas, consejo, entrenamiento ni producto. La compatibilidad exacta de assets y las reglas efectivas completas mantienen sus dependencias separadas.
