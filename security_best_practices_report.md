# Auditoría de seguridad de StratAI

> **Correcciones del 15 de octubre de 2026:** se han aplicado los cambios de código descritos en el [cierre y configuración de seguridad](docs/operations/security-hardening-2026-10-15.md). Este informe conserva los hallazgos y reproducciones anteriores a la corrección. Por decisión expresa del usuario, las estadísticas de otros jugadores siguen disponibles para cuentas autenticadas aunque no compartan partidas.

Fecha: **15 de octubre de 2026**. Revisión del árbol de trabajo local, incluidos sus cambios sin confirmar. Referencia Git: `e1825795307d27a6d1a62e95db84a7d8c9012b21`, rama `agent/steam-pipeline-replay-v2`. Esta referencia no representa por sí sola todos los archivos auditados; el inventario de evidencias incluye hashes de las fuentes citadas.

## Resultado ejecutivo

**Hay fallos que permiten divulgar datos y debilitan el cierre de sesión. Recomiendo resolver los tres hallazgos de severidad alta antes de exponer la aplicación a usuarios externos.** He reproducido acceso anónimo a los tres endpoints de replay, acceso a un bundle situado fuera del directorio de exportaciones en Windows y escritura de una clave de Steam sintética en los logs HTTP.

También he reproducido la reutilización de una cookie después de cerrar sesión, la evasión del límite de tamaño del cuerpo HTTP, la aparición del chat de otra cuenta en la misma pestaña y la creación de anotaciones para partidas inexistentes. Son problemas distintos que requieren correcciones propias.

El inventario contiene **12 hallazgos: 3 altos, 8 medios y 1 bajo**, además de riesgos condicionados y decisiones de privacidad pendientes. La severidad refleja el contexto del proyecto, no simplemente el CVSS de una dependencia. No he confirmado una vulnerabilidad crítica de ejecución remota ni una filtración externa ya ocurrida. Una auditoría de código no certifica la ausencia de vulnerabilidades.

No he modificado código funcional, dependencias, credenciales ni datos reales. Las pruebas nuevas y sus resultados se guardan en [outputs/security-audit-2026-10-15](outputs/security-audit-2026-10-15/). No se ha procesado ninguna demo ni alterado el corpus.

| ID | Severidad | Hallazgo | Evidencia |
|---|---|---|---|
| SEC-01 | Alta | Replays accesibles sin autenticación ni autorización de partida | Reproducido contra la aplicación ASGI, datos sintéticos |
| SEC-02 | Alta | Escape del directorio de exportaciones mediante `match_id` en Windows | Reproducido por HTTP, bundle sintético externo |
| SEC-03 | Alta | La clave de Steam aparece en logs de HTTPX | Reproducido con clave ficticia y transporte simulado |
| SEC-04 | Media | Logout no revoca una cookie de sesión previamente copiada | Reproducido |
| SEC-05 | Media | El límite de cuerpo HTTP se evade sin `Content-Length` | Reproducido: más de 2 MB aceptados frente al límite de 1 MB |
| SEC-06 | Media | Chat persistido sin separación entre cuentas | Dos pruebas con el hook y componente reales |
| SEC-07 | Media | Anotaciones sin cuota total ni validación de acceso a la partida | Creación reproducida; agotamiento de recursos inferido |
| SEC-08 | Media | SSE ejecuta lecturas bloqueantes en la conexión Redis compartida | Análisis de código y semántica documentada de Redis |
| SEC-09 | Media | Configuración permite saltar la firma interna en producción | Reproducido con una bandera insegura explícita |
| SEC-10 | Media | Descargas legacy degradadas de HTTPS a HTTP | Transformación reproducida; interceptación no ejecutada |
| SEC-11 | Baja | CSP del frontend permisiva y cabeceras incompletas en el HTML | Análisis estático; sin XSS demostrado |
| SEC-12 | Media | Dependencias con avisos y ausencia de control automatizado de seguridad | npm audit, OSV y govulncheck |

## Alcance y método

Se revisaron las fronteras entre React/Vite, FastAPI, el servicio Node de adquisición y colas, el servicio Go de parsing, Redis, almacenamiento en disco y Steam. Se siguieron rutas, dependencias de autenticación, llamadas internas, lectura/escritura de archivos, persistencia del navegador, generación de logs y configuración de ejecución. Se inspeccionaron además los flujos locales relevantes de AI Coach y la automatización de calidad.

La exploración estructural usó codebase-memory-mcp. Su generación registrada era del 9 de octubre y la cobertura indicaba metadatos cambiados: los hallazgos se fundamentan en la fuente actual y en pruebas, no en asumir que el grafo estuviera actualizado. Los huecos detectados se revisaron directamente. La cobertura del índice es orientativa, no una prueba de exhaustividad. El árbol de trabajo no estaba congelado; aparecieron dos archivos ajenos a la auditoría durante la revisión. No se modificaron y el estado final queda registrado; los hashes delimitan las fuentes citadas al cierre.

Se inventariaron **33 rutas explícitas de FastAPI** mediante AST. Se hicieron pruebas acotadas con cuentas, claves, partidas y archivos ficticios; las conexiones externas y almacenes reales se sustituyeron donde correspondía. Las pruebas del frontend usan Vitest/JSDOM, no un navegador desplegado. Se ejecutó la selección de tests de seguridad de Python y la suite de Node. Se consultaron bases públicas de vulnerabilidades enviando nombres y versiones de paquetes, sin código ni credenciales.

No se realizó un pentest de un dominio público, una prueba destructiva de carga, una auditoría de permisos del sistema operativo, firewall, TLS del proxy, Redis desplegado, copias de seguridad ni logs de producción. El modo temporal documentado inicia frontend/FastAPI; los riesgos específicos de Node/Go afectan cuando se utiliza el flujo completo. No se interpreta ese modo temporal como un bypass de autenticación.

## Hallazgos de severidad alta

### SEC-01 — Lectura anónima de replays

**Localización:** [dashboard.py](backend/app/routes/dashboard.py), líneas 2197–2212, 2215–2247 y 2250–2279; [main.py](backend/app/main.py), líneas 72–79.

Las rutas `GET /match/{match_id}/replay`, `/replay/metadata` y `/replay/round/{round_num}` no reciben usuario autenticado ni verifican acceso a la partida. El router tampoco tiene una dependencia global que lo imponga. En cambio, el detalle de partida sí autentica y comprueba pertenencia, en `dashboard.py:523–561`: esa protección no se hereda en las rutas de replay.

**Prueba:** con la aplicación en configuración de producción y sus middlewares, una petición sin cookie recibió **200 en las tres rutas** y obtuvo nombres/identificadores y posiciones sintéticos; el endpoint de detalle devolvió 401. El atacante necesita conocer u obtener un identificador válido. No se ha demostrado que pueda enumerar todos los identificadores, pero la dificultad de adivinarlos no constituye autorización.

**Impacto:** divulgación de información de partidas y participantes a cualquier cliente que pueda alcanzar FastAPI. El volumen de los replays también ofrece una vía de consumo de recursos sin autenticación.

**Solución:** establecer una dependencia compartida que autentique, cargue la partida y compruebe participación o un permiso explícito de compartición antes de leer sus artefactos. Aplicarla a las tres rutas y a cualquier futura variante. La protección de rutas en React y CORS no sustituyen el control del servidor. Es el principio de autorización por recurso recomendado por [OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html).

**Criterio de cierre:** anónimo → 401; usuario sin permiso → 403 o 404 consistente; participante o destinatario autorizado → 200. Cubrir replay completo, metadatos, rondas y respuestas cacheadas.

### SEC-02 — Traversal de `match_id` fuera de exportaciones en Windows

**Localización:** [canonical_repository.py](backend/app/matches/canonical_repository.py), líneas 70–71 y 250–257; entrada desde las rutas de SEC-01.

`find_canonical_match` elimina el prefijo `match_` y concatena el resto a una ruta. No exige un identificador de formato cerrado ni comprueba que el directorio resuelto siga dentro de `exports_directory`. `CanonicalMatch` resuelve la ruta, pero no impone esa frontera.

**Prueba:** `/match/x%5C..%5C..%5Cprivate_bundle/replay/metadata` devolvió 200 y un marcador situado en un bundle sintético hermano de `exports`, fuera del directorio permitido. La prueba incluyó una carpeta `match_x` existente como primer segmento; puede utilizarse el nombre de una partida existente para ese recorrido. En Windows las barras inversas codificadas pueden permanecer dentro del segmento HTTP y luego funcionar como separadores del sistema de archivos.

**Impacto:** lectura de bundles compatibles accesibles al proceso fuera del árbol autorizado. La reproducción **no demuestra lectura arbitraria de cualquier archivo**, como `.env`: el código espera manifests y artefactos con formato conocido. Las comprobaciones internas de rutas de artefactos no arreglan que la raíz de la partida ya haya escapado. La variante reproducida depende de Windows; no debe extrapolarse tal cual a Linux.

**Solución:** rechazar identificadores que no cumplan el formato real del contrato, con `fullmatch` y longitud máxima; no convertir silenciosamente entradas maliciosas en identificadores válidos. Resolver raíz y destino, comprobar pertenencia con una operación de rutas como `relative_to`, y rechazar escape, enlaces/junctions externos y la raíz misma. Aplicar autorización de partida aparte de esta validación.

**Criterio de cierre:** pruebas de `%5C`, separadores, `..`, codificaciones, carpetas hermanas y enlaces; todas deben fallar antes de leer fuera de la raíz. Mantener casos válidos del contrato en Windows y Linux.

### SEC-03 — Clave de Steam registrada en texto claro

**Localización:** [logging_config.py](backend/app/logging_config.py), líneas 6–23; [steam_auth.py](backend/app/routes/steam_auth.py), líneas 113–120. Patrón adicional en [steam_service.py](backend/app/routes/steam_service.py), líneas 23–38, y [dashboard.py](backend/app/routes/dashboard.py), líneas 428–446.

El logger raíz se configura a nivel INFO y escribe `record.getMessage()` sin redactar secretos. Las solicitudes a Steam incluyen `key=STEAM_API_KEY` en la query. HTTPX registra a nivel INFO la URL completa, incluida esa query.

**Prueba:** se llamó a `_load_profile` usando una clave ficticia y `httpx.MockTransport`, sin tráfico a Steam. La clave ficticia apareció en la salida del logger configurado por la aplicación. El resultado conservado registra esa comprobación, sin claves reales.

**Impacto:** exposición de la clave a quien pueda consultar stdout, ficheros de logs, agregadores, informes de error o copias de esos registros. No implica por sí solo que alguien externo haya accedido a ellos. Esta fuga puede producirse aunque `.env` esté bien excluido de Git.

**Solución:** limitar `httpx`/`httpcore` a WARNING o superior y añadir redacción centralizada para URLs, cabeceras y campos sensibles. Cubrir `key`, códigos de autenticación, cookies, tokens y firmas; registrar campos permitidos en vez de cuerpos completos. No basta con ocultar el parámetro en un único mensaje propio. Revisar los logs históricos con acceso restringido; si contienen la clave, rotarla y tratar sus copias según la política de retención. [OWASP Logging](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) recomienda excluir credenciales y tokens de los registros.

**Criterio de cierre:** pruebas de éxito, timeout, error de Steam y excepción con valores centinela; ninguno debe aparecer en ningún handler de logging.

## Hallazgos de severidad media

### SEC-04 — El cierre de sesión no revoca cookies existentes

**Localización:** [main.py](backend/app/main.py), líneas 63–70; [steam_auth.py](backend/app/routes/steam_auth.py), líneas 185–188; [dependencies.py](backend/app/auth/dependencies.py), líneas 13–21; [security.py](backend/app/middleware/security.py), líneas 84–97; [config.py](backend/app/config.py), línea 115.

La sesión es una cookie firmada de Starlette. Logout elimina la cookie del navegador actual, pero no revoca en el servidor el material ya emitido. La autenticación solo comprueba el contenido firmado. La caducidad predeterminada es de siete días y se renueva con actividad, sin una duración absoluta de sesión observada.

**Prueba:** cookie válida → autenticado; logout → 200 y navegador sin sesión; restaurar la cookie anterior → autenticado otra vez. Requiere que alguien haya obtenido previamente una cookie válida: no es un bypass que permita inventar una sesión sin conocer la firma. Las cookies firmadas de [Starlette](https://starlette.dev/middleware/) no proporcionan por sí solas un registro de revocación.

**Solución:** usar un identificador opaco de sesión y un registro revocable en Redis, o añadir una versión/identificador de sesión comprobado en servidor. Revocar en logout, rotar en login y fijar expiración absoluta además del tiempo de inactividad. Mantener HttpOnly, Secure en producción y SameSite. No guardar secretos de Steam en la cookie: firma no equivale a cifrado.

**Criterio de cierre:** una copia anterior de la cookie deja de autorizar inmediatamente tras logout; expiraciones, rotación y política de cerrar todas las sesiones tienen pruebas independientes. El frontend debe comprobar `response.ok` antes de presentar el cierre como exitoso.

### SEC-05 — Límite de cuerpo basado únicamente en una cabecera

**Localización:** [security.py](backend/app/middleware/security.py), líneas 64–81.

El middleware acepta `Content-Length` ausente como cero y no cuenta bytes del cuerpo. Una transferencia sin esa cabecera puede superar el límite anunciado mientras FastAPI lee y decodifica el contenido.

**Prueba:** un POST de onboarding con cuerpo transmitido por un iterador, sin `Content-Length`, de **2.097.225 bytes** obtuvo 202, pese al límite de **1.048.576 bytes**. Se usó sesión sintética y se sustituyeron Redis y el servicio Node. La prueba no sometió al servicio real a carga ni demuestra agotamiento de memoria; demuestra que la barrera es eludible.

**Solución:** un middleware ASGI que contabilice bytes de cada mensaje `http.request` y corte antes de que el cuerpo completo quede materializado, más un límite coherente en el proxy. Validar longitudes inválidas/negativas y definir excepciones pequeñas y explícitas por ruta. `extra="forbid"` en modelos puede reducir contenido inesperado, pero no sustituye el límite de transporte.

**Criterio de cierre:** contenido de tamaño máximo permitido se acepta; máximo + 1 se rechaza con 413 tanto con longitud declarada como con transferencia fragmentada/sin longitud. Probar también abortos y límites de tiempo. No confundir con Node y Go, que sí aplican límites de lectura propios.

### SEC-06 — El historial del chat pasa de una cuenta a otra en la misma pestaña

**Localización:** [useCoachChat.js](frontend/src/hooks/useCoachChat.js), líneas 211–240; [NavigationFrame.jsx](frontend/src/components/Layout/NavigationFrame.jsx), líneas 47–57.

El historial se guarda bajo una clave fija de `sessionStorage` y se carga sin comprobar a qué cuenta pertenece. Logout navega a `/`, pero no borra ese historial. El almacenamiento de sesión sobrevive a la navegación y al cambio de usuario dentro de la misma pestaña.

**Prueba:** el hook real, montado para una segunda cuenta, cargó mensajes ficticios de la primera. Una prueba separada sobre el componente real confirmó que un logout exitoso conserva esa clave. No es acceso entre dispositivos: requiere compartir esa sesión de navegador/pestaña.

**Solución:** asociar cualquier persistencia a la identidad autenticada, invalidarla cuando cambia y borrar los datos privados al cerrar sesión. Restablecer también estado React y cachés de consultas. Validar estructura/tamaño del JSON recuperado. Revisar bajo el mismo criterio las notas locales de revisión de [coachReviewNotes.js](frontend/src/features/replay2d/domain/coachReviewNotes.js), líneas 12–14 y 62–64, según si son material de una cuenta o de un operador local.

**Criterio de cierre:** secuencia cuenta A → conversación → logout → cuenta B, en la misma pestaña, sin mensajes de A; comprobar también fallo de logout, cambio de identidad sin desmontaje y datos persistidos corruptos.

### SEC-07 — Anotaciones ilimitadas y sin comprobar la partida

**Localización:** [replay_annotations.py](backend/app/routes/replay_annotations.py), líneas 70–93; [utils/replay_annotations.py](backend/app/utils/replay_annotations.py), líneas 12–39.

La API autentica y limita el tamaño de cada anotación, pero no valida existencia ni acceso a la partida y no impone cuota de anotaciones, archivos o bytes por usuario. Cada operación lee o reescribe el array JSON completo. Un usuario puede inventar identificadores válidos y acumular archivos o incrementar indefinidamente el coste de un archivo.

**Prueba:** ocho creaciones para una partida sintética inexistente devolvieron 201. No se hizo una inundación de disco. El aislamiento por directorio de usuario existe: este hallazgo no demuestra que pueda editar las anotaciones de otra persona.

**Solución:** verificar acceso a una partida real antes de escribir; imponer cuotas por partida y por cuenta, límite de bytes y rate limiting compartido entre procesos. Usar actualizaciones transaccionales o una estrategia que no reescriba sin límite todo el histórico. El `RLock` local tampoco coordina varios procesos: revisar las escrituras concurrentes y el `.tmp` compartido como riesgo adicional de integridad.

**Criterio de cierre:** partidas inexistentes/no autorizadas se rechazan sin crear archivos; superar cuotas produce una respuesta controlada; escrituras simultáneas no pierden anotaciones ni eluden la cuota. Extender presupuestos de recursos a búsquedas de rendimiento, onboarding/discovery y conexiones SSE según su coste real.

### SEC-08 — SSE bloquea la conexión Redis usada por el resto del servicio

**Localización:** [demoController.js](backend/node-service/controllers/demoController.js), líneas 49–113, especialmente 85–90; [redisClient.js](backend/node-service/services/redisClient.js), líneas 8–15; [security.js](backend/node-service/middleware/security.js), líneas 139–142.

Cada stream SSE llama a `xRead(..., { BLOCK: 25000 })` sobre el cliente singleton. Esa misma conexión se usa para estado y operaciones de autenticación interna, incluidos nonces. Un comando bloqueante mantiene ocupada su conexión; otras conexiones Redis siguen funcionando. Es la semántica descrita por [Redis XREAD](https://redis.io/docs/latest/commands/xread/).

**Impacto:** un stream sin eventos puede retrasar otros comandos de esa conexión hasta 25 segundos; múltiples streams agravan las colas. No hace falta corromper Redis. La ruta Node requiere autenticación interna y se alcanza mediante el flujo autorizado de eventos; no se afirma que esté abierta directamente a anónimos. Es una conclusión estática; no se bloqueó el Redis real para medirla.

**Solución:** lector dedicado mediante conexión duplicada/pool, o un consumidor dedicado que distribuya eventos a los clientes. Limitar streams por cuenta/IP y desconectar/cancelar el lector al cerrar el cliente, además de considerar reconexiones y backpressure. Nunca compartir una conexión bloqueante con la validación de firmas o la cola de trabajos.

**Criterio de cierre:** abrir streams inactivos no incrementa significativamente la latencia de una operación de estado/autenticación; desconectarlos libera lectores y conexiones sin esperar ciclos innecesarios.

### SEC-09 — Bandera de desarrollo puede desactivar la firma interna en producción

**Localización:** [config.js](backend/node-service/services/config.js), líneas 13–21; [security.js](backend/node-service/middleware/security.js), líneas 103–106.

`ALLOW_UNSIGNED_LOCAL_INTERNAL=true` habilita la excepción de loopback incluso si el entorno es `production`. Además, `NODE_ENV` tiene precedencia sobre `APP_ENV`, por lo que valores divergentes pueden activar reglas de desarrollo inesperadas.

**Prueba:** cargando el middleware real con configuración ficticia de producción y la bandera explícita a `true`, una petición local sin firma llegó a `next()`. La configuración de ejemplo desactiva la bandera. No se ha demostrado un bypass remoto con los valores de producción seguros.

**Impacto condicionado:** un proceso local puede invocar rutas internas sin HMAC. Si un proxy local expone esas rutas, la conexión del proxy aparece como local y puede trasladar el bypass a clientes externos. No basta con comprobar la dirección del socket para definir confianza de usuario.

**Solución:** rechazar el arranque de producción si se solicita la excepción; que el permiso efectivo requiera entorno de desarrollo y una activación explícita. Unificar cómo se determina el entorno entre servicios. Mantener Node/Go en una red privada y no publicar `/internal/*` en el proxy.

**Criterio de cierre:** con cualquier combinación de variables que declare producción, peticiones sin firma reciben 403 o el proceso rechaza la configuración. Cubrir `NODE_ENV`/`APP_ENV` contradictorios y un proxy de loopback.

### SEC-10 — Descargas de demos sin integridad de transporte autenticada

**Localización:** [demoDownload.js](backend/node-service/services/demoDownload.js), líneas 26–47 y 169–185.

El código acepta HTTP para hosts legacy concretos de Valve y convierte incluso URLs HTTPS de esos hosts a HTTP. La transformación está reproducida. Se calcula un SHA-256 del archivo recibido, pero un hash calculado después de una transferencia no autenticada no demuestra que el contenido venga de Valve.

**Impacto condicionado:** un atacante situado en la red puede sustituir o alterar la demo que después se descomprime y parsea. Puede corromper resultados o intentar consumir recursos. No se ha probado una vulnerabilidad del parser ni ejecución remota. La allowlist de hosts, la revisión de redirecciones y los límites de descarga/descompresión existentes son defensas útiles, pero no autentican los bytes recibidos.

**Solución:** confirmar soporte HTTPS en el origen real y conservarlo cuando exista. Si el servicio legacy exige HTTP, documentar esa dependencia y tratar cada demo como entrada hostil: proceso de parsing con pocos permisos, acceso restringido a archivos y red, límites reales de CPU/memoria/tiempo y verificación contra un digest autenticado si el proveedor lo ofrece. No sustituir el transporte a ciegas sin validar compatibilidad.

**Criterio de cierre:** no degradar un origen HTTPS funcional; probar redirecciones y contenido manipulado. Si HTTP sigue siendo necesario, verificar de forma efectiva el aislamiento y los límites del parser.

### SEC-12 — Dependencias y controles de suministro de software

**Localización:** [requirements.txt](backend/requirements.txt), líneas 1–14; [go.mod](backend/go-service/go.mod), línea 3; [quality.yml](.github/workflows/quality.yml), especialmente líneas 64–71 y 79–87. Evidencia: `npm-*.json`, `python-go-advisories.json`, `advisory-details.json` y `govulncheck-summary.json`.

Los tres `npm audit` sobre raíz, frontend y servicio Node devolvieron **cero vulnerabilidades conocidas** en sus resoluciones actuales. Esto no evalúa la lógica de la aplicación ni garantiza que todos los paquetes sean seguros.

La consulta OSV de **98 combinaciones de paquete/versión** identificó avisos en ocho paquetes del entorno Python instalado, `golang.org/x/sys` y la biblioteca estándar de Go. Hay aliases duplicados entre fuentes: no se deben sumar como vulnerabilidades independientes. `govulncheck` identificó 35 avisos distintos y trazas de llamada para 14; una traza a un símbolo amplio tampoco demuestra que se cumplan las condiciones de explotación.

| Componente observado | Versión auditada | Acción propuesta y contexto |
|---|---|---|
| AnyIO | 4.13.0 | Evaluar ≥ 4.14.2. Incluye TLS/IDNA y bloqueo del pool de procesos. Los destinos Steam observados son ASCII fijos; no se confirmó un hostname internacional controlado por atacante. |
| cryptography | 47.0.0 | Evaluar ≥ 50.0.0 para cubrir el conjunto de avisos consultado. Hay avisos sobre OpenSSL incluido, verificación X.509 y PKCS#7; el uso aplicativo examinado es AES-GCM, sin exposición observada de las APIs PKCS#7/X.509 afectadas. |
| idna | 3.13 | Evaluar ≥ 3.15. Entrada hostil muy grande necesaria para el escenario de consumo de CPU; no se identificó esa entrada en los destinos actuales. |
| pip | 26.0.1 | Evaluar ≥ 26.2.0. Riesgo de instalación/resolución, no una ruta HTTP de la aplicación. |
| PyJWT | 2.12.1 | Eliminar si es residuo del entorno; si se usa, evaluar ≥ 2.15.0 y volver a escanear. Comprobar especialmente el advisory de mutación de `options`, cuyo registro usa `last_affected` en vez de una versión `fixed`. La sesión actual de StratAI no utiliza JWT. |
| python-multipart | 0.0.26 | Eliminar si no se necesita o evaluar ≥ 0.0.31. No se observaron rutas `Form`, `UploadFile` o `request.form()` en la API pública revisada. |
| Starlette | 1.0.0 | Evaluar ≥ 1.3.1 con una versión compatible de FastAPI. Los avisos incluyen formularios, `StaticFiles`/UNC, `HTTPEndpoint` y construcción de URL. No se observaron los tres primeros usos en el servicio; existe TrustedHostMiddleware. |
| urllib3 | 2.6.3 | Evaluar ≥ 2.8.0. Revisar consumidores reales y configuración de proxy/redirecciones; las llamadas principales de FastAPI revisadas usan HTTPX. |
| golang.org/x/sys | v0.35.0 | Evaluar ≥ v0.44.0; el símbolo Windows afectado no apareció como llamado en govulncheck. |
| Go / stdlib | go1.26.2 local | Recompilar con una rama soportada parcheada; los avisos consultados incluyen correcciones hasta 1.26.9 / 1.27.2. Validar también el toolchain que realmente usa CI y producción. |

Estas son versiones candidatas basadas en los registros consultados en la fecha del informe, no actualizaciones aplicadas ni compatibilidad ya comprobada. Los avisos y correcciones están conservados íntegramente en las evidencias. Fuentes de ejemplo: [AnyIO TLS/IDNA](https://github.com/agronholm/anyio/security/advisories/GHSA-82r6-8w77-94w6), [cryptography/OpenSSL](https://github.com/pyca/cryptography/security/advisories/GHSA-537c-gmf6-5ccf), [PyJWT/options](https://github.com/jpadilla/pyjwt/security/advisories/GHSA-gvp8-978c-rx2q) y [Go HTTP/2](https://pkg.go.dev/vuln/GO-2026-6603). Se contrastó el inventario mediante la [API oficial de OSV](https://google.github.io/osv.dev/api/).

En Go, varios avisos requieren HTTP/2, ECH o entradas que no se observaron en los listeners HTTP planos y destinos configurados del proyecto. Se recomienda parchear sin presentarlos como exploits confirmados. El workflow anuncia Go 1.23.4 mientras `go.mod` declara 1.24 y la máquina auditada usa 1.26.2: esa discrepancia dificulta saber qué se valida realmente, especialmente con descarga automática de toolchains.

**Solución:** reconstruir un entorno Python mínimo y reproducible desde un lock con hashes; eliminar dependencias residuales justificadamente; actualizar por grupos compatibles y ejecutar tests. Añadir escaneo de dependencias, secretos y revisión de cambios de lock a CI, con excepciones de alcance documentadas y fecha de revisión. Fijar versiones de toolchains coherentes y valorar SHA inmutables para Actions. No ejecutar `audit fix --force` ni actualizaciones masivas sin comprobación.

**Criterio de cierre:** entorno limpio reproducible, inventario de ejecución separado del de desarrollo, escaneo actualizado y ausencia de avisos relevantes sin tratamiento; pruebas funcionales y de seguridad pasan con las versiones elegidas.

## Hallazgo de severidad baja

### SEC-11 — CSP permisiva y protección incompleta del documento frontend

**Localización:** [index.html](frontend/index.html), líneas 9–12; [vite.config.js](frontend/vite.config.js), configuración de desarrollo y build.

La CSP incluye `script-src 'unsafe-inline' 'unsafe-eval'`, además de permitir conexiones a cualquier origen `ws:`/`wss:` y puertos localhost. La política de desarrollo se incluye en el HTML que sirve de entrada al build. Esto reduce la contención de una futura inyección de scripts. No se ha encontrado una vía directa de XSS en los componentes revisados; la debilidad de CSP no constituye por sí sola un XSS.

No se define `frame-ancestors` para el documento en la configuración de frontend del repositorio. El `<meta http-equiv="X-Content-Type-Options">` no sustituye la cabecera HTTP `X-Content-Type-Options`. Las cabeceras de FastAPI no protegen automáticamente un HTML servido por otro host/proxy. Es posible que el despliegue añada estas defensas; no se ha verificado.

**Solución:** servir una CSP de producción mediante cabecera HTTP, con scripts propios o nonces/hashes según necesidad, sin evaluación dinámica ni inline genéricos; conservar solo el permiso WASM que se demuestre necesario. Restringir destinos de conexión, añadir `object-src 'none'`, `base-uri 'self'` y `frame-ancestors 'none'` o la allowlist funcional. Separar desarrollo/HMR de producción y configurar `nosniff` en el servidor. [MDN CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy) documenta las restricciones y diferencias de estas directivas.

**Criterio de cierre:** verificar las cabeceras de la respuesta HTML desplegada y probar login Steam, recursos 3D/WASM, workers y conexiones permitidas sin habilitar comodines por comodidad. Revisar primero violaciones en modo Report-Only, si procede.

## Riesgos condicionados y decisiones pendientes

### R-01 — Visibilidad global de jugadores y estadísticas

[performance.py](backend/app/routes/performance.py), líneas 23–43, 94–107 y 110–128, construye un índice global de jugadores y devuelve estadísticas del `steam_id` solicitado a cualquier usuario autenticado. La identidad `_user` no se utiliza para filtrar el conjunto autorizado. La prueba sintética confirmó esa ausencia de control en la ruta, con agregación sustituida; no consultó datos personales reales.

La interfaz incluye comparación de jugadores, por lo que este acceso podría ser deliberado. **Si los datos deben ser privados por cuenta, el impacto pasa a ser alto y hay que tratarlo junto a SEC-01.** Antes de ampliar uso, definir qué es público, compartido, exclusivo del participante o propio del corpus de investigación. Aplicar esa política a búsqueda y agregación; que un identificador Steam sea público no hace automáticamente públicos todos los análisis almacenados sobre él. Si la comparación es pública, minimizar campos y limitar extracción automatizada.

### R-02 — Frontera de archivos y aislamiento del parser Go

[handlers.go](backend/go-service/api/handlers.go), líneas 104–114 y 431–437, comprueba rutas con `filepath.Abs` y prefijo, sin resolver symlinks/junctions. Un enlace creado por alguien con permisos de escritura puede cruzar esa frontera. La ruta exige autenticación interna y acceso al sistema de archivos: no se ha demostrado que un usuario web pueda crear ese enlace ni leer un archivo arbitrario.

En el mismo archivo, líneas 545–562, el contexto se comprueba antes y después de `parser.ParseDemo`, sin trasladar cancelación a esa llamada. Los timeouts del servidor y el límite de concurrencia no equivalen a un presupuesto de CPU/memoria del parser. Resolver la ruta real con una política de enlaces, proteger permisos del directorio y aislar parsing en un proceso con límites verificables; considerar carreras entre validación y apertura. Esta revisión cobra más importancia por SEC-10.

### R-03 — Despliegue, cachés, retención y exposición de servicios

Node escucha sin host explícito en [index.js](backend/node-service/index.js), línea 74, y Go utiliza `:8080` en [main.go](backend/go-service/main.go), línea 46. Esto no demuestra exposición a Internet: faltan los datos de red y firewall. Confirmar que solo el frontend y el proxy público previsto sean accesibles, que Redis no tenga acceso público y que los servicios internos usen autenticación/ACL y transporte apropiados a su red. El servidor de desarrollo de Vite accesible en todas las interfaces no debe ser el mecanismo de publicación de producción.

Definir `Cache-Control: no-store` para respuestas privadas donde corresponda y verificar que un proxy/CDN no mezcle identidades. Definir retención y acceso a logs, credenciales persistidas, chats y datos de cuenta; distinguirlo del corpus aprobado y la evidencia técnica, cuya conservación no debe modificarse sin autorización. No se ha comprobado la política de copias de seguridad o borrado de cuentas.

`main.py:37–40` desactiva Swagger/ReDoc en producción, pero deja el OpenAPI predeterminado. No es un fallo de autenticación ni una vulnerabilidad alta; decidir si se quiere publicar ese esquema. También conviene que cabeceras de seguridad alcancen respuestas tempranas de todos los middlewares, no solo las que llegan al middleware más interno.

## Controles útiles observados

La base tiene defensas relevantes que deben conservarse al corregir los fallos:

- OpenID valida proveedor, identidad, campos firmados, estado y nonce; utiliza caducidad y almacenamiento antirreplay. La selección de tests de OpenID/seguridad pasó.
- Las llamadas internas tienen HMAC sobre método, URL y hash del cuerpo, comparación constante, ventana temporal y nonces Redis. La excepción de SEC-09 debe cerrarse sin debilitar ese diseño.
- Las credenciales almacenadas usan AES-GCM; la configuración de producción exige secretos independientes. Las cookies usan HttpOnly y configuración Secure/SameSite apropiada a producción.
- CORS utiliza una lista explícita, existe validación Host y una defensa CSRF para cambios autenticados. Deben comprobarse también en el despliegue real.
- La descarga de demos restringe hosts, revisa redirecciones y limita tamaño descomprimido. Go limita cuerpo HTTP y concurrencia; métricas administrativas se configuran en loopback y pprof es opt-in.
- En el frontend revisado no se localizaron usos directos de `dangerouslySetInnerHTML`, `innerHTML`, `eval` o `new Function` con datos de usuario. No se identificó persistencia de tokens de autenticación en localStorage. Estas búsquedas acotadas no excluyen todos los posibles XSS o comportamientos de dependencias.
- Las consultas SQLite inspeccionadas parametrizan valores. Las invocaciones de procesos relevantes revisadas usan argumentos separados; no se confirmó una inyección de comandos.
- La integración experimental de interpretación local con Ollama restringe loopback, redirecciones, tamaño y formato, y no se observó ejecución de herramientas ordenada por el modelo. Esto no equivale a una evaluación completa de seguridad del modelo.

## Secretos: resultado y límites

El escaneo inicial de código/documentación examinó 1.436 archivos y produjo seis candidatos, correspondientes a pruebas o documentación; no se confirmó una credencial activa incrustada. El escaneo por coincidencia exacta con secretos actuales examinó **5.396 archivos de texto** sin encontrar coincidencias. Excluyó datos de ejecución, dependencias/cachés y archivos grandes; hubo 63 archivos por encima del límite y 22 inaccesibles. No se copiaron los secretos al informe.

El historial de refs locales se examinó sobre **3.903 blobs de texto**, sin coincidencias con los secretos actuales ni los patrones de proveedor/clave privada utilizados; 968 blobs se excluyeron por tamaño o tipo. No fue un análisis completo de entropía, ni de refs remotas no descargadas, objetos Git inalcanzables, logs de producción o copias externas. Por tanto, el resultado es «sin secretos reales confirmados en este escaneo», no «imposible que exista una fuga». SEC-03 demuestra una vía de fuga en ejecución independiente de estos resultados.

## Orden propuesto de corrección

1. **Cerrar exposición de datos:** SEC-01, SEC-02 y SEC-03. Resolver la política de R-01 y aplicar sus permisos. Verificar logs y rotar la clave si se confirma que fue registrada.
2. **Aislar cuentas y limitar abuso:** SEC-04, SEC-05, SEC-06 y SEC-07, con regresiones de autenticación/autorización y cuotas.
3. **Asegurar el flujo completo:** SEC-08, SEC-09, SEC-10 y aislamiento Go. Comprobar red interna y comportamiento de los proxies.
4. **Hacer reproducible la seguridad:** SEC-12 y SEC-11, actualizaciones compatibles, checks en CI y verificación de las cabeceras de producción.

Cada corrección debe cerrarse con el criterio indicado en su hallazgo. Una actualización de dependencias no corrige por sí sola los fallos de autorización, sesión o almacenamiento local. Este informe propone trabajo; no declara corregido ninguno de los hallazgos.

## Evidencias y verificación ejecutada

Todos los archivos siguientes están en [outputs/security-audit-2026-10-15](outputs/security-audit-2026-10-15/). Se conservan también intentos fallidos por restricciones del entorno, sin confundirlos con resultados exitosos.

| Evidencia | Contenido / resultado |
|---|---|
| `reproduce_findings.py`, `reproduction-results.json` | Pruebas sintéticas de SEC-01 a SEC-05, SEC-07 y política R-01; todas las aserciones completadas |
| `node_probes.cjs`, `node-probe-results.json` | Bandera insegura de producción y degradación de HTTPS, sin peticiones de red |
| `frontend-probes.test.jsx`, `frontend-probe-run-3.txt` | Dos pruebas aprobadas de aislamiento de chat/logout |
| `python-security-tests-2.txt` | 11 aprobadas, 32 no seleccionadas; seguridad/OpenID |
| `node-tests-2.txt` | 18 aprobadas y 3 omitidas; integración Redis opt-in no ejecutada |
| `npm-root.json`, `npm-frontend.json`, `npm-node.json` | Cero avisos en los tres lockfiles consultados |
| `python-go-advisories.json`, `advisory-details.json` | Consulta OSV y registros completos para analizar condiciones/versiones |
| `govulncheck.json`, `govulncheck-summary.json` | Análisis Go de paquetes y llamadas con el toolchain local |
| `fastapi-route-inventory.json` | Inventario AST de rutas y dependencias explícitas |
| `secret-and-package-inventory.json`, `exact-secret-scan.json`, `history-secret-scan.json` | Inventarios y resultados sin valores de secretos |
| `index-coverage.json`, `source-hashes.json`, `evidence-inventory.json` | Cobertura declarada, hashes de fuentes y de evidencias |

Comandos principales de verificación (desde la raíz, salvo indicación):

```text
backend/venv/Scripts/python.exe outputs/security-audit-2026-10-15/reproduce_findings.py
node outputs/security-audit-2026-10-15/node_probes.cjs
node frontend/node_modules/vitest/vitest.mjs run --config outputs/security-audit-2026-10-15/vitest.config.mjs
npm run test:python -- -q -k "openid or service_security or security" --basetemp outputs/security-audit-2026-10-15/pytest-security -p no:cacheprovider
npm run test:node
npm audit --json
npm --prefix frontend audit --json
npm --prefix backend/node-service audit --json
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -json ./...
```

El último comando se ejecuta en `backend/go-service`; la resolución de `@latest` durante esta auditoría fue v1.9.0. Los tests Python se ejecutaron con configuración de prueba y claves ficticias. Las pruebas nuevas de reproducción configuran sus propias claves ficticias y sustituyen conexiones externas. No se ejecutó la suite completa `check:all` ni un smoke de demo, porque no se modificó comportamiento productivo; los resultados anteriores no deben presentarse como esa validación completa.
