# Contexto económico nativo por mensaje y frontera

Contrato `1.0.0-candidate.1`, 6 de septiembre de 2026. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y siguiente acción. Esta extensión optativa conserva los candidatos económicos v1/v2, las [fuentes nativas](canonical-economy-native-sources-v1.md), las [referencias históricas](canonical-economy-rule-references-v1.md) y el export base `3.8.0`.

## Fuentes y significado

El selector económico **4** ejecuta el mismo parser una vez, produce v2 y fuentes nativas @1, y añade este registro separado. La [sonda versionada](../../../backend/go-service/models/economy_rule_context_probe.json) fija 56 nombres de variables y cuatro propiedades de reglas. Sólo contiene nombres y tipos; no contiene valores por defecto. Go incorpora sus bytes en la build y el lector Python exige su SHA-256 exacto.

- `CNETMsg_SetConVar.convars.cvars`: conserva las asignaciones permitidas en orden, incluidas repeticiones dentro de un mensaje. Registra todos los mensajes recibidos, incluso cuando ninguno de sus nombres está permitido. Lee los punteros nativos: valor ausente, cadena vacía, cero y `false` son distintos. No conserva nombres ajenos a la sonda, datos de conexión ni chat.
- `RoundStart` y `RoundFreezetimeEnd`: capturan las últimas asignaciones vistas en el orden de callbacks. Cada variable conserva valor, secuencia del mensaje y tick de origen; cada frontera conserva su propia secuencia y tick. Se comprueba la correspondencia exacta con la instancia v2. Ninguna asignación posterior se aplica retroactivamente, aunque comparta tick.
- `GameRules` en esas fronteras: consulta `m_iNumConsecutiveCTLoses`, `m_iNumConsecutiveTerroristLoses`, `m_bCTCantBuy` y `m_bTCantBuy`, bajo `m_pGameRules`. Conserva presencia, tipo, origen y disponibilidad. Son sondas de propiedades nativas; un contador observado no acredita por sí solo la fórmula ni el importe del bonus.
- `CSVCMsg_ServerInfo`: registra presencia del objeto `game_session_config` y presencia, longitud y SHA-256 de `game_session_manifest` y `game_session_config.data`. No guarda el contenido opaco ni interpreta su hash como vínculo con assets. Ausente y presente con cero bytes se distinguen.

La fuente técnica es el SDK instalado `demoinfocs v5.2.0`, especialmente [SetConVar](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/net_messages.go), [mensajes base](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/proto/networkbasetypes.proto) y [ServerInfo](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/proto/netmessages.proto). Su evento `ConVarsUpdated` usa un mapa y getters; por eso esta extensión consulta el mensaje directamente. El orden de callbacks no acredita orden de subtick, ni prueba cuándo el jugador conoció una variable del servidor.

## Lector y comprobación

El lector verifica la cadena de hashes contexto → fuentes nativas → candidato v2 → bundle base, identidad opaca y build, esquema, sonda, secuencias completas y fronteras. Reconstruye cada contexto desde el prefijo de mensajes y exige que todas las fronteras v2 tengan su captura correspondiente. Un dato de fuente con tick posterior al de la frontera queda inutilizable, incluso si su callback precede a la captura.

El informe distingue:

- `observed_native_parameter`: valor numérico o booleano presente y disponible en esa frontera.
- `not_seen_in_probed_message_prefix`: el nombre fue consultado, pero no aparece antes de esa frontera en los mensajes capturados. **No demuestra ausencia en toda la demo ni en otros canales.**
- `native_value_absent`: el nombre aparece con puntero de valor ausente.
- `native_value_unsupported`: el contenido existe pero no es un número finito o booleano interpretable; se conserva como dato nativo sin usarlo como parámetro.
- `source_tick_after_boundary`: el tick de origen impide utilizar el valor en esa frontera.

Los defaults de los paquetes históricos se muestran aparte, sin rellenar huecos ni acreditar configuración efectiva. Se contrastan valores equivalentes numéricamente y se conservan conflictos entre referencias. El validador del informe vuelve a buscar las asignaciones en los mensajes, sin llamar al constructor del informe.

El informe también agrupa entradas nativas de armadura/casco por ID, coste registrado, armadura previa cruda y casco previo, y vuelve a contar cada grupo. Son ocurrencias de entradas en capturas, no compras únicas. El valor nativo `-1` permanece sin interpretación física. Estos grupos no certifican una fórmula de precio o devolución.

Todos los artefactos son evidencia de desarrollo. `comparison_allowed`, `training_allowed` y `product_supported` son siempre `false`; no se conectan al acceso a features iniciales ni a consejos. El contexto del servidor no se convierte en información legal del jugador. La identidad exacta demo–assets, las transacciones completas, la cooperación y las reglas condicionales requieren acreditación separada.

## Artefactos y ejecución

| Artefacto | Schema |
|---|---|
| `economy_rule_context.json` | [economy_rule_context@1](../../../ai_coach/contracts/economy_rule_context.schema.json) |
| `economy_rule_context_manifest.json` | [economy_rule_context_manifest@1](../../../ai_coach/contracts/economy_rule_context_manifest.schema.json) |
| `economy_rule_context_assessment.json` | [economy_rule_context_assessment@1](../../../ai_coach/contracts/economy_rule_context_assessment.schema.json) |

Registro de la entrega 3D.2B: 49 schemas y 19 motivos; 3D.2C añade el número 50 y la [extensión de manifiesto 3D.2D](canonical-economy-resource-manifest-v1.md) eleva el total vigente a 51. Catálogo `1.7.0` e IDs conservados. Desde `backend/go-service`, ejecutar el exportador con `--ai-coach-economy-candidate --economy-candidate-version 4`, además de demo aprobada, SHA-256, alias opaco, build, mapas y destino TEMP nuevo y vacío. Se aplican los prechecks de la [guía de validación](../../operations/ai-coach-validation.md) antes de analizar la demo.

Una vez construido el paquete histórico con sus seis fuentes verificadas, desde la raíz:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.rule_context --source $ruleContextManifest --package "$rulePackageOutput/economy_rule_package.json" --sources $ruleSources --output $ruleContextOutput
```

El destino del informe también debe existir, estar vacío y bajo TEMP; se comprueba antes de abrir entradas. El materializador exige mensajes con entradas económicas y capturas no vacías. No escribe ni promueve al corpus, release o `CURRENT`. Cada cierre funcional aplica el smoke de una única demo aprobado en `AGENTS.md`.

## Extensión de sesión 3D.2C

El selector **5** añade `economy_session_payload.json`, contrato independiente `stratai.economy_session_payload@1`, versión `1.0.0-candidate.1`. Conserva los artefactos de los selectores anteriores. Su [schema](../../../ai_coach/contracts/economy_session_payload.schema.json) eleva el registro runtime a **50**; catálogo `1.7.0`, IDs, export `3.8.0` y elegibilidad permanecen iguales.

El artefacto enlaza por ruta fija, tamaño y SHA-256 el manifest de contexto @1 y, a través de él, fuentes nativas, candidato v2 y bundle base. La [sonda incorporada a la build](../../../backend/go-service/models/economy_session_payload_probe.json) especifica nueve rutas JSON Pointer y tipos, sin defaults. La raíz nativa tiene nombre vacío: por eso los campos empiezan por `//`. Cada aparición conserva `path`, `occurrence` desde cero, `node_index` desde uno en recorrido previo de nodos, presencia y tipo nativo. Su bloque padre conserva ruta protobuf, presencia, tamaño y hash; la sesión conserva tick y secuencia del mismo callback que el contexto @1.

Se admite únicamente configuración con firma `VBKV`, CRC32 IEEE válido sobre todos los bytes posteriores a la cabecera de ocho bytes, claves y cadenas UTF-8 terminadas en NUL, tipos **0 objeto, 1 cadena y 2 entero de 32 bits little-endian**, y terminadores 11. Se exige consumo total. Los otros tipos/dialectos quedan sin interpretar: el SDK y los lectores públicos difieren en algunos códigos compilados, y esta demo sólo requiere 0/1/2. No hay descompresión en este formato. Límites locales de seguridad del lector, no límites universales del motor: 65.536 bytes por bloque, 4.096 nodos, 4.096 bytes por cadena/clave y profundidad 16. Un fallo no publica campos parciales.

Se retienen los enteros nativos `c_game_type`, `c_game_mode`, `default_game_type`, `default_game_mode` y `numSlots`. Una cadena numérica no se convierte en entero. Las referencias `ct_arms`, `t_arms`, `ct_models/` y `t_models/` se retienen como **SHA-256 del nombre UTF-8**, nunca como hash del archivo del asset. Las claves repetidas se conservan con su ordinal; no se convierten en un mapa ni se escoge una aparición como valor efectivo. Ninguno de estos campos acredita por sí solo el modo efectivo, el loadout, una versión exacta o una regla monetaria.

El lector recorre todas las claves decodificadas y compara sus nombres exactos con los 56 de la sonda económica anterior. Publica sólo los nombres permitidos encontrados, sin interpretar nombres parecidos ni cadenas opacas. Cero coincidencias significa exclusivamente que esos nombres no están en este árbol decodificado. Los campos con `reward` en su nombre no se convierten en importes de dinero sin una definición semántica verificable.

Dentro del artefacto de sesión @1, `game_session_manifest` conserva `format=unverified`, `compression=unverified` y `status=unsupported_format` cuando está presente dentro del límite. El nuevo selector 6 publica su interpretación acreditada en un [artefacto separado](canonical-economy-resource-manifest-v1.md), sin cambiar el significado de @1. No se ensayan offsets internos ni se trasladan cadenas a campos. Ausencia, formato no soportado, exceso del límite, contenido malformado y decodificación completa son estados distintos. Las rutas seleccionadas ausentes en un árbol válido conservan `present=false`, tipo/valor/índice nulos y `not_seen_in_decoded_tree`; un bloque no decodificado no permite afirmar esa ausencia.

El [lector Python](../../../ai_coach/economy/session_payload.py) verifica schema, sondas, cadena de hashes, cobertura exacta de sesiones, fingerprints originales, posiciones y ordinales. `available_sessions` filtra simultáneamente tick y secuencia; no alimenta features de jugador. Este lector puede comprobar coherencia y procedencia, pero no volver a demostrar el valor de un campo a partir de un hash sin el bloque fuente. El smoke compara además todos los campos contra un lector independiente de los bytes privados y exige roundtrip binario idéntico.

Fuentes de formato: [lector VBKV y CRC32 de ValvePython/vdf, revisión fijada](https://github.com/ValvePython/vdf/blob/d76292623e326fb165fe3bdb684832cdf30959d4/vdf/__init__.py), [lector Binary KV1 de ValveKeyValue, revisión fijada](https://github.com/SteamDatabase/ValveKeyValue/blob/27f501fc028cda19872c5d9f803450441f54c194/ValveKeyValue/ValveKeyValue/Deserialization/KeyValues1/KV1BinaryReader.cs), y [declaraciones del SDK CS2](https://github.com/alliedmodders/hl2sdk/blob/bd17582be4bc18970e4fe4c518359872fda8276d/public/tier1/KeyValues.h). Son fuentes primarias de esas implementaciones; no constituyen una especificación oficial del contenido económico de la sesión. Las definiciones protobuf sólo declaran los bloques como `bytes`. Revisiones y hashes completos quedan en la evidencia de fase.

Uso: ejecutar el exportador con los mismos prechecks de TEMP y `--ai-coach-economy-candidate --economy-candidate-version 5`. Después, desde Python, `session_payload.load_candidate(Path(session_artifact), require_observed=True)`, tras importar `Path` y `ai_coach.economy.session_payload` y asignar a `session_artifact` la ruta del JSON. El destino de exportación se comprueba antes de abrir la demo; el lector de sesión es sólo lectura. `exact_build_asset_binding=not_accredited`, `comparison_allowed=false`, `training_allowed=false` y `product_supported=false` son obligatorios. Identidad exacta, candidato histórico, conflicto entre referencias y dato no acreditado siguen siendo categorías separadas; los hashes de nombres no abren ninguna puerta.
