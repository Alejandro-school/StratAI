# Dependencia de identidad exacta y contenido económico

Especificación documental de aceptación, 6 de septiembre de 2026. Resultado de 3D.2E: **material insuficiente para acreditar la cadena completa**. El [plan](../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene el bloqueo y la siguiente acción independiente. Este documento no es un schema runtime ni amplía permisos de adquisición. [Informe](../archive/ai-coach/AI_COACH_ECONOMY_EXACT_ASSET_BINDING_PHASE_2026-09-06.md) y [evidencia](../evidence/AI_COACH_ECONOMY_EXACT_ASSET_BINDING_AUDIT_2026-09-06.json).

## Cadena que hace falta demostrar

| Eslabón | Evidencia disponible | Dato pendiente |
|---|---|---|
| Observación en la demo | Alias `demo-a11eb0d600ee702c9edb`, SHA-256 aprobado, `CDemoFileHeader.build_num=10896`, `patch_version=14177`, protocolo de ServerInfo `14177` | Identificar el componente que emitió cada versión y su relación con los binarios de servidor/cliente relevantes. La declaración protobuf del campo de cabecera no acredita esa relación. |
| Build observada → distribución | Dos referencias históricas de patch `1.41.7.7`; metadatos del tracker y marcadores de revisión | Registro verificable que conecte la build de cabecera con la distribución efectivamente pertinente a esa partida. No basta el patch o una fecha. |
| Distribución → archivos originales | IDs de depots seleccionados por el tracker; inventario local de otra versión | Manifests históricos de los depots exactos, archivos originales y hashes; correspondencia con binarios, configuración y posibles addons relevantes. |
| Archivo original → representación consumida | Copias textuales fijadas de `weapons.vdata`, `steam.inf` y CFG; proceso de extracción público | Hash del contenedor y del recurso compilado cuando proceda, herramienta/revisión, argumentos y salida reproducible. El hash del texto transformado no sustituye al del original. |
| Assets exactos → reglas efectivas | Parámetros nativos parciales de 3D.2B; configuración y referencias de sesión decodificadas | Overrides, herencia y reglas condicionadas efectivas. Este es un requisito económico separado, incluso si se consigue la distribución exacta. |

La observación de build procede de [un campo opcional de `CDemoFileHeader`](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/msg/proto/demo.proto), cuyo archivo local conserva el SHA-256 fijado anteriormente. No se afirma que 10896 sea una versión de distribución de Steam ni que la cabecera, por sí sola, identifique el binario que aplicó todas las reglas del servidor.

## Qué aportan realmente las referencias históricas

| Referencia del tracker | Cliente/servidor | Patch | `SourceRevision` | `built_from_cl.txt` | ID de actualización citado por el tracker |
|---|---|---|---|---|---|
| `fd04856b8668f11cb31ed2b22574607dd9c365e0` | 2000888 | 1.41.7.7 | 10936007 | 10935971 | 24916958 |
| `acfe24d588d2df0a26da0f964e44d780bd3070eb` | 2000891 | 1.41.7.7 | 10937988 | 10937911 | 24934554 |

Las asociaciones de la última columna están en los mensajes de los commits [fd04856b](https://github.com/SteamTracking/GameTracking-CS2/commit/fd04856b8668f11cb31ed2b22574607dd9c365e0) y [acfe24d5](https://github.com/SteamTracking/GameTracking-CS2/commit/acfe24d588d2df0a26da0f964e44d780bd3070eb). Son procedencia declarada por el tracker; no son una asociación acreditada con la demo ni IDs de manifests de depot. Las fechas de commit —24 y 25 de agosto de 2026— tampoco completan el vínculo. No se equiparan los números de las columnas.

El [proceso fijado de actualización](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/update.sh) invoca extracción/descompilación VPK mediante `--vpk_decompile`, incluyendo `vdata_c`, y construye el commit a partir de `ClientVersion` y `steam_buildid.txt`. Por ello las copias textuales registradas deben tratarse como representaciones publicadas por ese proceso, sin atribuirles los bytes originales del recurso compilado. La revisión exacta de la herramienta usada para cada extracción no está acreditada por esos archivos.

Su [configuración de selección](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/files.json) enumera depots 2347770, 2347771, 2347773 y 2347779; son IDs y filtros de selección, no una lista de manifests históricos. Su [`.gitignore`](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/.gitignore) excluye binarios, VPK, `*.vpk.manifest.txt` y `steam_buildid.txt`. Los dos árboles Git obtenidos sin truncado no contienen esos archivos originales en las categorías comprobadas. El filtro de nombres y las sondas de identidad no prueban ausencia universal de información en otras fuentes.

Steamworks distingue el manifest de cada versión de depot del BuildID global de una build de aplicación. La documentación describe cómo se generan ambos, pero no proporciona la correspondencia de esta demo. [Documentación oficial de SteamPipe](https://partner.steamgames.com/doc/sdk/uploading).

La instalación local conserva cliente/servidor 2000899, patch 1.41.7.8 y Steam app build 25000182. Se recontrastaron cinco hashes anteriores: `steam.inf`, appmanifest y tres binarios conocidos. Su marcador `built_from_cl.txt` es 10948893; no se interpreta como build nativa. No se selecciona ese paquete para la demo.

## Material concreto que permitiría reabrir la acreditación

Se acepta material de quien custodie la instalación histórica, del distribuidor con acceso legítimo a esos manifests, o de un archivo público que conserve la misma cadena verificable. Los mantenedores del tracker pueden ser una fuente del registro original de captura/extracción; no se presupone que aún lo conserven. Esta fase no ha contactado con nadie ni autorizado descargas privadas.

1. **Registro de origen:** alias y SHA-256 de la demo; componente emisor de `build_num`, protocolo y versiones; plataforma, arquitectura y hashes de los binarios relevantes. Debe explicar de forma comprobable cómo se obtiene la versión observada de ese binario/distribución. Un log sin vincular al binario y al paquete sólo es una referencia candidata.
2. **Metadatos de distribución:** app ID y BuildID, lista completa de depots/manifests necesarios, rama/plataforma, `steam.inf`, y origen verificable de la captura. Preservar versiones cliente/servidor, `SourceRevision` y marcadores como datos distintos. No enviar credenciales, tokens ni identificadores personales.
3. **Archivos originales mínimos:** binarios o registros verificables suficientes para el vínculo de versión, `pak01_dir.vpk` y segmentos que contengan los recursos económicos, configuración relevante, y manifiesto de rutas/tamaños/hashes. Identificar el recurso compilado de armas —por ejemplo `scripts/weapons.vdata_c` cuando corresponda— y las dependencias que realmente utilice. No imponer ese ejemplo si el paquete acredita otra estructura.
4. **Transformación reproducible:** versión/revisión y SHA-256 de la herramienta, argumentos exactos, hash de entrada original, hash de salida textual y cualquier dependencia. Conservar por separado integridad del contenedor, recurso y representación. Un SHA-256 demuestra igualdad de bytes al contrastarlos; no autentica por sí solo quién los distribuyó.
5. **Límites de uso:** declarar qué archivos y componentes cubre el paquete y qué sigue ausente. Un paquete exacto no prueba loadout, cooperación, transacciones intrafotograma ni todos los overrides de la partida.

## Pruebas de aceptación al recibir material

Son criterios para la futura implementación, **no pruebas ejecutadas con un paquete que todavía falta**. Antes de cambiar estados de compatibilidad deberá existir una cadena positiva completa y pasar estos rechazos:

| Caso | Resultado exigido |
|---|---|
| Sólo coinciden patch, fecha, precio o nombre de recurso | Mantener candidato; no acreditar identidad. |
| Se mezcla build de cabecera, versión cliente/servidor, SourceRevision, changelist o BuildID | Rechazar la equivalencia sin una prueba de enlace explícita. |
| Se mezclan manifests, plataforma o archivos de dos distribuciones | Registrar conflicto; no escoger la coincidencia más favorable. |
| Falta un binario, manifest, segmento VPK o dependencia necesaria | Dato ausente; no rellenar con la instalación actual. |
| Cambia un byte del contenedor, recurso o representación vinculados | Rechazar integridad; mantener separados sus hashes. |
| Herramienta de extracción no identificada o salida no reproducible | Transformación sin acreditar, aunque el texto sea plausible. |
| Sólo hay hashes de nombres o de una lista de referencias | No convertirlos en hashes de contenido ni en prueba de carga. |
| Paquete íntegro pero no consta relación con la demo y su componente emisor | Referencia verificada del paquete; vínculo demo–assets pendiente. |
| Cadena exacta acreditada pero faltan reglas de sesión | Permitir únicamente el ámbito factual acreditado; no abrir comparación, entrenamiento o producto. |

La recepción de ese material es la condición para reabrir el comprobador exacto de 3D.2. Mientras tanto puede avanzarse la extracción factual de transacciones desde la demo sin usar precios/defaults externos. El [plan](../AI_COACH_IMPLEMENTATION_PLAN.md) selecciona esa tarea independiente como 3D.3; la investigación del mismo bloqueo no se repite automáticamente.
