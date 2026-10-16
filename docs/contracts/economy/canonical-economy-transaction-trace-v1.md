# Traza de canales económicos v1

Contrato candidato `stratai.economy_transaction_trace@1`, versión `1.0.0-candidate.1`, selector optativo **7**. Go conserva los canales en `economy_transaction_trace.json`; su referencia por ruta, bytes y SHA-256 enlaza el manifest económico v2. Los selectores 1–6 mantienen sus rutas y contratos. Export base `3.8.0`, parser `v16` y catálogo `1.7.0` conservados.

## Fuentes y significado

La dependencia instalada es `demoinfocs-golang/v5@v5.2.0`, verificada mediante `go mod verify`. La evidencia de fase conserva los hashes de los archivos consultados. La implementación efectiva prevalece sobre comentarios genéricos del SDK.

| Canal | Fuente y extracción | Qué acredita y qué falta |
|---|---|---|
| `native_event` | `CMsgSource1LegacyGameEvent`, descriptores recibidos de `CMsgSource1LegacyGameEventList`; cinco nombres sondeados: `item_purchase`, `item_pickup`, `item_pickup_slerp`, `item_remove`, `item_refund` | Evento entregado y campos tipados por ordinal. Sin descriptor nativo se cuenta como no mapeado; no se incorpora silenciosamente el descriptor fallback del SDK. `server_tick` conserva presencia; no es orden subtick. |
| `sdk_generic` | `GenericGameEvent`, sólo nombre permitido | Control del canal que mapea el SDK; no conserva campos ni confirma su procedencia nativa frente al fallback. |
| `sdk_callback` | `ItemPickup`, `ItemDrop`, `ItemRefund` | Aviso del SDK, con actor y entidad cuando están disponibles. No acredita una transacción itemizada. |
| `property_baseline` | Lectura explícita al crear `CCSPlayerController` | Estado ya decodificado; las actualizaciones de creación ocurrieron antes de registrar el observador. No es una transacción ni un historial completo desde el nacimiento. |
| `property_update` | `OnUpdate` de `m_pInGameMoneyServices.m_iAccount` y `m_pInGameMoneyServices.m_iCashSpentThisRound` | Cada entrega posterior al registro, incluso si repite valor/tick/fotograma. Un cambio acredita saldo o contador replicado, no su causa. |
| `probe` | Clase/propiedad consultada inexistente | Ausencia en esta sonda, no ausencia global en la demo. |

`game_events.go` transforma `item_pickup` en `ItemPickup` con entrega aplazada; `item_remove` en `ItemDrop`. Resuelve el arma por actor y tipo mediante `itemEvent`/`getPlayerWeapon`, no mediante un identificador nativo exclusivo de esa transacción. El propio comentario de `item_remove` deja su interpretación como drop en duda. Se conserva por ello el aviso sin acreditar un drop voluntario. [Implementación del SDK](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/game_events.go).

`ItemRefund` es una **heurística del SDK**: subida de dinero, destrucción de una entidad de arma en el mismo tick y dueño en zona de compra. La coincidencia no prueba que esa subida proceda de esa devolución ni su importe. [Implementación en `bindWeaponS2`](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/datatables.go).

Source2 registra `OnUpdate` sin entregar inmediatamente el valor inicial, aunque el comentario de la interfaz diga lo contrario. El observador lee la base explícitamente. `readFields` entrega cada cambio después de decodificarlo; los callbacks de creación se registran después de los campos iniciales. [Implementación Source2](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/sendtables/sendtablescs2/entity.go).

**Precisión de procedencia añadida en 3D.5:** al final de `OnPacketEntities`, el SDK también reenvía las propiedades registradas de entidades recién creadas, en orden alfabético. Por tanto, `property_update` en este contrato conserva entregas de `OnUpdate`; no todas prueban una modificación nueva llegada por red. Se mantienen artefactos y conteos históricos. La [extensión de identidad](canonical-economy-native-identity-v1.md) distingue lecturas base, callbacks del fotograma de creación potencialmente reenviados y callbacks de fotogramas posteriores, sin rellenar el pasado.

El inventario incluye, fuera de la sonda implementada, `CUserMessageItemPickup` y `CCSUsrMsg_ItemPickup` (texto de objeto), y `CCSUsrMsg_ItemDrop` (`itemid`, `death`). Esos cuerpos no contienen por sí solos un actor acreditado ni el vínculo con una transacción. No se declara su ausencia real: no se capturan en este contrato. Las listas de inventario y sellback siguen en las capturas y el reconstruidor 3D.1; una diferencia en ellas tampoco demuestra la causa. [Tipos de mensajes CS](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/cstrike15_usermessages.pb.go), [tipos de mensajes base](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/usermessages.pb.go).

## Identidad, orden y ausencias

Cada fila retiene secuencia consecutiva del observador, tick, fotograma local, canal, fuente, instancia, actor, lado, entidad y valor. El fotograma local empieza en cero; no equivale a un tick del servidor. No se deduplica por tick, objeto, nombre o valor. Entidades SDK distintas reciben alias distintos, incluso si reutilizan índice/serial. El alias de entidad del callback identifica la entidad que resolvió el SDK; no mejora su correspondencia con el objeto nativo del evento.

Los actores usan exclusivamente aliases existentes en el observador v2 en ese momento. Un alias aún ausente permanece `null`; no se completa al terminar la partida. No se exportan SteamID, nombres de participantes, userid ni handles crudos. `userid` se redacta y se intenta resolver por el índice conectado siguiendo la reducción de userid del SDK. Se aceptan tipos coincidentes 4 y 9 con `ValShort` explícito: `itemEvent` consulta ese campo también para el tipo 9 observado en CS2. Se conserva el tipo nativo, sin convertirlo al tipo 4. Es un mapeo del SDK, no una identidad nativa exportada. Las duplicidades o falta de campo impiden resolverlo. La resolución secundaria desde `rawPlayers` no está disponible en esta sonda: se conserva la ausencia.

Los campos de evento permitidos son `item`, `defindex`, `silent`, `inventory`, `canbuy`, `team`. Se conservan ordinal, nombre, tipo de descriptor, tipo de mensaje, estado y valor. Los campos repetidos no se convierten en un mapa. Los valores cero, falso y cadena vacía se distinguen de un puntero ausente. Otros campos tienen estado `outside_probe`; los identificadores se redactan. No se conservan cargas arbitrarias.

## Lectura y límites

`python -B -m ai_coach.economy.transaction_trace --source $traceFile --output $newEmptyTemp --require-observed` valida schema, identidad, procedencia, orden, presencia y enlaces. El destino debe estar vacío bajo TEMP, sin enlaces/reparse points y sin sobrescrituras; se comprueba antes de leer la entrada. La entrada debe ser el archivo fijo acompañado de su cadena v2 validada.

El enlace retrospectivo requiere **instancia**, actor y lado coincidentes con el roster inicial y tick de entrega estrictamente posterior al inicio y anterior al fin de freeze. Los ticks frontera se excluyen, igual que identidades ausentes o desacuerdos. Un número de ronda repetido nunca sustituye la instancia. Se informa de cobertura por ID de oportunidad existente; ese enlace no acredita estado inicial completo ni compra legal. Clasifica cuándo se entregó la evidencia al observador: no acredita por sí solo el instante de ejecución del suceso en el servidor. `server_tick` y tick del parser se conservan por separado; su relación no se supone ni se corrige con un offset adivinado.

La traza pertenece íntegramente a evidencia de ejecución. No es información que el jugador necesariamente conocía y no completa T0. Las series de cuenta permiten contar variaciones y compensaciones intrafotograma cuando se observan; ni su suma ni su diferencia se convierten en precio, recompensa o compra. Los canales permanecen separados ante coincidencias o desacuerdos. La clasificación de transacciones itemizadas se abstiene; comparación, entrenamiento y producto continúan deshabilitados.
