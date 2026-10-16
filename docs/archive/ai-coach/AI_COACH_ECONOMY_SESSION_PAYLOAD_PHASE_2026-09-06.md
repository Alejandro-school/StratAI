# Datos nativos de sesión — 3D.2C

Fecha: 6 de septiembre de 2026. Se ha implementado la extracción verificable de configuración. **El formato del manifiesto y la compatibilidad económica completa siguen pendientes.** El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija el siguiente trabajo; la [evidencia](../../evidence/AI_COACH_ECONOMY_SESSION_PAYLOAD_SINGLE_DEMO_SMOKE_2026-09-06.json) conserva comandos, fuentes fijadas, hashes y comprobaciones.

## Qué se pudo verificar

Se inspeccionó sólo la cabecera de la demo aprobada para recuperar ambos bloques en TEMP privado; no se retuvieron sus contenidos opacos en el repositorio. Después de los tests se ejecutó una pasada completa del productor cambiado sobre **la misma partida**, `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. El destino nuevo y vacío fue `C:/Users/kerch/AppData/Local/Temp/stratai-3d2c-smoke-cvhglvjh`.

| Bloque | Resultado comprobado | Límite de la conclusión |
|---|---|---|
| `game_session_config.data` | 570 bytes; Binary KeyValues con firma VBKV y CRC32 `2cc6f339`; 30 nodos y 27 valores escalares; consumo y roundtrip completos | Contiene configuración de sesión y nombres de modelos, sin vínculo exacto acreditado con una compilación de assets |
| `game_session_manifest` | 7.160 bytes; fingerprint idéntico al de 3D.2B | Formato y compresión no acreditados; no se afirma que carezca de versión, assets o reglas |

La configuración contiene cinco enteros seleccionados: `c_game_type=0`, `c_game_mode=1`, `default_game_type=0`, `default_game_mode=0` y `numSlots=10`. Se conservan los nombres nativos: no se trata un default como configuración efectiva ni se interpreta la diferencia entre ambos modos como un error. También hay doce referencias de modelos: dos de brazos, cinco entradas CT y cinco T. Se retienen hashes de sus nombres, no nombres de personas ni hashes del contenido de assets.

Las listas CT/T utilizan cinco claves vacías repetidas por lista. Un diccionario convencional conservaba sólo una; la auditoría independiente lo detectó antes del smoke. La implementación conserva las doce referencias y sus ordinales, sin escoger cuál estuvo equipado. El recorrido completo se contrastó con `ValvePython/vdf` fijado por revisión y SHA-256, usando `VDFDict` y `merge_duplicate_keys=False`; su serialización reproduce los **570 bytes originales**. Los **17 campos seleccionados** coinciden con Go, sin diferencias.

## Investigación y límites de formato

Las definiciones de `demoinfocs v5.2.0` identifican las rutas protobuf y la presencia, pero declaran ambos contenidos como `bytes`: no especifican su estructura interior. Para VBKV se contrastaron los lectores públicos originales de ValvePython y ValveKeyValue, además de las declaraciones de tipos del SDK CS2; las URLs inmutables y hashes están en la evidencia y en el [contrato](../../contracts/economy/canonical-economy-rule-context-v1.md). No se consideran documentación oficial de la semántica económica de estos campos.

Se admite el subconjunto observado de tipos 0/1/2, CRC32 correcto, terminadores 11 y consumo completo. Los límites son locales y explícitos. Tipos fuera del subconjunto, truncamientos, cadenas inválidas, exceso de tamaño/profundidad/nodos y basura final fallan sin retener campos parciales. No se asignaron significados a offsets desconocidos del manifiesto. La prueba de Snappy sobre el bloque completo devolvió `corrupt input`; esto descarta esa interpretación concreta, **no** toda compresión ni posibles envolturas. Sus bytes no empiezan por la firma VBKV de la configuración. La búsqueda acotada en los protobufs instalados, el árbol público del SDK CS2 y las interfaces públicas consultadas no aportó un codec verificable del manifiesto de red. No se confundió con los archivos de recursos `.vrman_c`, que son otro formato.

El recorrido de **todas las claves** de la configuración no encontró ninguno de los 56 nombres de la sonda económica. Las claves `requires_attr_reward` y `reward_drop_list` no se transforman en recompensas monetarias: falta una definición que acredite esa semántica. No se obtuvo una nueva fórmula de bonus, precio condicionado, devolución, límite monetario o regla de compra. Los 22 parámetros observados en el canal anterior y las cuatro propiedades de `GameRules` conservan su procedencia independiente. El contador de pérdidas no equivale al importe del bonus; las 101 entradas de casco a 350 siguen siendo ocurrencias de capturas, no compras únicas ni una regla universal.

## Identidad y entrega

- **Identidad exacta:** no acreditada. El build nativo 10896 y patch/protocolo 14177 se conservan; los nombres o hashes de referencias de modelos no identifican los archivos usados por esa build.
- **Referencias candidatas:** siguen siendo las dos compilaciones históricas 2000888 y 2000891 registradas en 3D.2. No se crea un candidato nuevo por coincidencia de nombres, fechas o precios.
- **Conflicto:** las dos referencias históricas tienen números de compilación distintos; no se resuelve eligiendo una. Los campos nativos de modo y modo por defecto tienen funciones distintas y no se fuerzan a coincidir.
- **Dato ausente o no acreditado:** no hay vínculo demo–assets demostrado en lo decodificado; el manifiesto sigue sin interpretar. Esto no equivale a ausencia en toda la demo.

El selector optativo **5** produce los artefactos anteriores y `stratai.economy_session_payload@1`. La sonda versionada fija nueve rutas y admite sus apariciones repetidas. El artefacto registra tipos, presencia, posición de nodo, ordinal, tick 0, secuencia 1 y fingerprints de origen; enlaza el manifest de contexto por tamaño y SHA-256. El lector verifica toda la cadena, las sesiones y su causalidad. No se conecta a features iniciales del jugador. Catálogo `1.7.0`, export `3.8.0`, parser `v16`, quality `12` y elegibilidad permanecen iguales; el nuevo schema lleva el registro runtime a 50.

## Verificación y cierre

Pasaron `npm run check:all` y **483 pruebas del AI Coach**, 30 más que al inicio. Ruff pasó en los archivos Python afectados y el contrato global registró 50 schemas/19 motivos. La consulta adicional de Ruff a todo `ai_coach` detectó dos avisos anteriores en los módulos espaciales, ajenos al cambio; se conservaron. Un test que fijaba el recuento anterior de 49 schemas se actualizó y la suite completa pasó después. La prueba real se inició sólo tras ese resultado.

Parser real: **106,439 s**, una pasada completa; reconstrucción nativa: **7,460 s**. La validación de sesión, procedencia, recálculo independiente, candidato v2, fuentes nativas y oportunidades tardó **32,232 s**. Se verificaron ambos hashes de bloque, CRC32, roundtrip, 30 nodos, 27 escalares y 17 apariciones seleccionadas. La comprobación temporal rechaza la sesión antes del tick o antes de su callback, incluso compartiendo tick.

La validación semántica canónica completa pasó con **cero errores en 379,582 s**. Se conservaron los 77 actores iniciales acreditados, 173 abstenciones, tres equipos completos y 50 fichas de oportunidad, con 60 diferencias y 15 transiciones. El inventario protegido conserva sus 4.356 archivos y 181 hashes; las 21 evidencias anteriores y los 301 archivos de fuentes de la build permanecen iguales. No se modificó el corpus ni se promovieron salidas. Las puertas de comparación prescriptiva, entrenamiento y producto permanecen cerradas.

La evidencia retuvo **83 archivos / 743.837.378 bytes** de la salida nueva antes del intento de limpieza. La revisión automática rechazó `Remove-Item -LiteralPath <destino TEMP nuevo> -Recurse -Force` con el motivo `blocked by policy`, antes de crear el proceso. Una comprobación posterior de sólo lectura confirmó que el destino sigue presente y sus 83 hashes coinciden. La limpieza 3D.2C queda pendiente, sin reintento ni alternativa para eludir el rechazo. No se tocaron los temporales anteriores.

El cierre documental comprobó **301 enlaces locales de 50 documentos**, sin destinos ausentes, y un único prompt vigente. De los 966 archivos previos inventariados, 951 permanecen iguales y 15 cambios corresponden al alcance de la fase, incluida la retirada del prompt completado. `git diff --check` pasó y las fuentes funcionales coinciden con la build validada.

## Dependencia seleccionada

La continuación **3D.2D** debe resolver una dependencia concreta: una descripción o implementación verificable del serializador/lector de **`CSVCMsg_ServerInfo.game_session_manifest` compatible con la build nativa 10896**, con fixture y procedencia; si hace falta un paquete de instalación, debe acreditar también su vínculo con build y manifests de distribución. El SDK consultado sólo declara un bloque opaco y no basta para escribir ese codec. No se autoriza deducir offsets ni sustituirlo por otra comparación de defaults.

El siguiente prompt exige primero inventariar las fuentes aprobadas disponibles y fijar pruebas de aceptación de esa dependencia. Si no existe, debe dejar una solicitud concreta del material faltante, sin repetir una demo para confirmar el mismo bloqueo ni abrir una búsqueda indefinida. Una implementación nueva sólo podrá cerrarse con tests y la misma única demo en otro TEMP nuevo. La información económica y de transacciones que siga faltando conservará abstención; el coaching personalizado aún necesita esos datos antes de comparar alternativas de compra.
