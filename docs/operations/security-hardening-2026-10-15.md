# Correcciones de seguridad — 15 de octubre de 2026

Este trabajo corresponde a la auditoría y a la petición de aplicar sus correcciones. No cambia la fase ni los criterios de juego del AI Coach. El informe original se conserva en [security_best_practices_report.md](../../security_best_practices_report.md); la evidencia de corrección está en `outputs/security-fixes-2026-10-15/`.

## Política de acceso acordada

Por indicación expresa del usuario, una cuenta autenticada puede buscar jugadores y consultar sus estadísticas aunque no haya participado en sus partidas. No se ha añadido un filtro de partidas compartidas al buscador ni a los agregados de rendimiento.

Los replays, sus fragmentos y las anotaciones requieren una sesión válida y participación en la partida. Las anotaciones pertenecen a su autor. Los intentos anónimos o de otra cuenta se rechazan antes de devolver los archivos.

## Cambios aplicados

| Hallazgo | Corrección |
|---|---|
| SEC-01 | Autenticación y comprobación de participante en las tres rutas de replay. |
| SEC-02 | Identificadores estrictos y comprobación de la ruta real del bundle, incluidos enlaces que salgan del directorio. |
| SEC-03 | Redacción central de credenciales, URLs y cabeceras; HTTPX/HTTPCore en WARNING; handlers de Uvicorn enviados al mismo redactor. |
| SEC-04 | Sesiones opacas almacenadas en Redis, revocación en logout y límites de inactividad y duración absoluta. Los streams de progreso vuelven a comprobar la sesión antes de reenviar datos. |
| SEC-05 | Límite real de 1 MiB en el cuerpo ASGI, también sin Content-Length o con una longitud declarada falsa. |
| SEC-06 | Chat separado por identidad, cancelación de respuestas pendientes al cambiar de cuenta y limpieza de chat, revisiones y caché de consultas al cerrar sesión correctamente. Un logout fallido se muestra como error. |
| SEC-07 | Autorización de partida, límites de 500 anotaciones por partida, 200 archivos y 10 MiB por cuenta; escritura atómica y bloqueo entre procesos. Los PATCH con null se rechazan antes de escribir. |
| SEC-08 | Redis dedicado para cada lector SSE, máximo de dos conexiones por jugador y 100 por proceso; cierre del lector al desconectar, fallar o acumular contrapresión. |
| SEC-09 | Peticiones internas sin firma desactivadas por defecto y prohibidas en producción; rechazo de APP_ENV y NODE_ENV contradictorios. |
| SEC-10 | Descargas y redirecciones de demos solo por HTTPS y hosts permitidos. Las URLs legacy conocidas se elevan a HTTPS; nunca se degrada HTTPS a HTTP. |
| SEC-11 | CSP de producción sin JavaScript inline ni eval; orígenes explícitos y cabeceras HTTP en Vite/preview. El build exporta `security-headers.json`. |
| SEC-12 | Lock Python completo con hashes, actualización de dependencias afectadas, Go 1.26.9 y x/sys 0.44.0, Node 24 LTS en CI, análisis npm/pip/Go y Gitleaks. Dependabot semanal. |

También se cierra el acceso de Go mediante enlaces/junctions a archivos externos, se aceptan únicamente archivos regulares y Node/Go escuchan en loopback por defecto. El limitador de peticiones falla con 503 si Redis no está disponible y limita el tamaño de sus conjuntos de contadores. Las respuestas de la API llevan `Cache-Control: no-store`.

## Configuración y puesta en marcha

- Redis es necesario para autenticar, revocar sesiones y limitar peticiones. No se acepta una cookie como identidad si falta su registro de sesión. No publicar Redis en Internet; configurar sus permisos y persistencia en el entorno de despliegue.
- `SESSION_MAX_AGE` limita la inactividad y `SESSION_ABSOLUTE_MAX_AGE` la duración total; ambos tienen por defecto 604800 segundos. Los eventos SSE no renuevan la inactividad. Las cookies antiguas con el perfil dentro obligan a iniciar sesión otra vez al actualizar.
- `APP_ENV=production` debe coincidir con `NODE_ENV=production` si se definen ambos. `ALLOW_UNSIGNED_LOCAL_INTERNAL=false` en producción. Los secretos de sesión, HMAC y cifrado deben ser independientes y los secretos compartidos deben coincidir entre servicios.
- `NODE_SERVICE_HOST=127.0.0.1` y `GO_SERVICE_ADDR=127.0.0.1:8080` son los valores seguros por defecto. En contenedores separados, configurar explícitamente la dirección de la red privada y mantener los servicios internos fuera del acceso público.
- Servir el frontend por HTTPS y aplicar **todas** las cabeceras de `frontend/build/security-headers.json` a las respuestas HTML, incluidas las rutas de la SPA. Vite preview ya las aplica; un alojamiento estático/CDN distinto necesita esa configuración. La CSP en meta no puede sustituir `frame-ancestors` ni `X-Frame-Options` HTTP.
- `VITE_API_URL` usa el mismo origen por defecto en producción. Si la API usa otro origen, proporcionar su URL HTTPS al compilar. Solo se admite HTTP para loopback local. Las cabeceras se generan con ese origen concreto.
- Las descargas legacy cuyo servidor no acepte HTTPS se rechazan. No se ha realizado una descarga de Steam para comprobar disponibilidad del proveedor: el flujo temporal de interfaz sigue vigente.
- Mantener los directorios de demos, exportaciones y anotaciones bajo permisos de la cuenta del servicio. La comprobación de rutas no sustituye el aislamiento del proceso ni protege frente a un actor local que ya pueda sustituir archivos entre su validación y lectura.
- Revisar los logs históricos según la política de retención. La corrección evita nuevas exposiciones; no borra registros anteriores ni rota credenciales externas. No se ha demostrado una filtración real de credenciales en esta revisión.

## Dependencias y comprobaciones

Instalar Node según `.nvmrc`, usar el Go indicado por `go.mod` e instalar Python con verificación de hashes:

```powershell
backend\venv\Scripts\python.exe -m pip install --upgrade pip==26.2.1
backend\venv\Scripts\python.exe -m pip install --require-hashes -r backend/requirements-dev.txt
npm run check:all
```

Para un entorno de producción limpio usar `backend/requirements.txt`. Las entradas humanas están en `requirements.in` y `requirements-dev.in`; no editar manualmente los hashes. Generación utilizada con uv 0.13.1:

```text
uv pip compile backend/requirements.in --generate-hashes --universal --python-version 3.12 --output-file backend/requirements.txt
uv pip compile backend/requirements-dev.in --generate-hashes --universal --python-version 3.12 --output-file backend/requirements-dev.txt
```

La actualización del entorno Python local conserva paquetes adicionales preexistentes. Se actualizaron también pip, PyJWT y python-multipart instalados, aunque los dos últimos no forman parte del lock de la API. No se ha cambiado la instalación global de Node: la validación local usa una copia aislada de Node 24.22.0 verificada contra el SHA-256 oficial.

Gitleaks encontró siete coincidencias históricas que se comprobaron como ejemplos documentales o valores ficticios de pruebas. `.gitleaksignore` contiene exclusivamente sus huellas de commit, archivo, regla y línea. La excepción de `.gitleaks.toml` exige simultáneamente uno de dos valores ficticios exactos y uno de los cuatro archivos concretos que los usan. No se desactivan reglas ni carpetas. Los resultados retenidos están redactados.

La comprobación npm actualizada encontró avisos adicionales que no figuraban en el resultado inicial. Se corrigieron dentro de las ramas compatibles: concurrently 9.2.5, Vitest 4.1.11, Axios 1.20.0, Express 4.22.3 y adm-zip 0.6.1, además de sus dependencias transitivas afectadas. Los tres locks terminan con cero avisos en el análisis final.

Fuentes técnicas: [uv: generación y sincronización de locks](https://docs.astral.sh/uv/pip/compile/), [pip-audit](https://github.com/pypa/pip-audit), [Go: versiones oficiales](https://go.dev/dl/), [Gitleaks](https://github.com/gitleaks/gitleaks), [Vite: cabeceras de preview](https://vite.dev/config/preview-options).

## Verificación y alcance

La evidencia conserva pruebas de regresión que fallaban antes de las correcciones, resultados automatizados, auditorías de dependencias, comprobaciones de navegador y hashes. La revisión independiente detectó los dos casos adicionales de Uvicorn y streams activos; ambos tienen regresiones específicas.

El despliegue remoto y sus cabeceras, TLS, permisos de infraestructura y política de logs no se han inspeccionado ni modificado. Los cambios de CI están preparados en el repositorio; no se ha ejecutado GitHub Actions ni cambiado la protección de ramas.

La ejecución ampliada de backend y AI Coach terminó con **2.683 pruebas superadas**. Sus fuentes Python y locks no cambiaron durante esa ejecución. Una ejecución previa de `npm run check:all` también superó lint, 362 pruebas de frontend, 154 de Python, 50 de Node, las pruebas Go y el build; tres integraciones Node que requieren Redis quedaron omitidas localmente. Estos resultados están conservados en `verified-test-ai-coach-all.log` y `verified-check-all.log`.

**La validación final está pendiente.** El control de hashes rechazó el cierre de esa ejecución porque otras fuentes cambiaron mientras se ejecutaban las pruebas, entre ellas cambios simultáneos de otra sesión en el pipeline y la landing. Además, las últimas actualizaciones npm requieren comprobar de nuevo el conjunto final. El usuario ha confirmado que avisará cuando termine esa sesión. No se considera que los resultados anteriores validen esas modificaciones posteriores.

El smoke de una sola demo aprobada está preparado y todavía no se ha ejecutado. Se ejecutará después de superar las suites sobre fuentes estables, en TEMP nuevo, con alias locales y salida retenida. No descargará demos, modificará el corpus, promoverá exportaciones ni cambiará CURRENT. Sus resultados y límites se registrarán en el cierre de evidencia.
