# Servicio Steam

Servicio Node.js responsable de la comunicación con Steam, la descarga de demos y la coordinación del pipeline de procesamiento.

## Responsabilidades

- Mantener las sesiones de los bots de Steam.
- Descubrir partidas y resolver URLs de demos.
- Descargar y validar archivos.
- Encolar y supervisar trabajos mediante Redis y BullMQ.
- Solicitar el análisis al servicio Go.
- Exponer a FastAPI el estado operativo del pipeline.

No es una API pública para el navegador.

## Cuenta técnica y amistad

La cuenta de Steam es una dependencia interna para resolver un sharecode mediante
el Game Coordinator. El usuario no necesita agregarla como amigo: el historial se
consulta con su código de autenticación y los sharecodes contienen los identificadores
y el token de la partida. Se han eliminado las invitaciones, las comprobaciones de
amistad y su caché en Redis. Las claves antiguas pueden caducar con su TTL existente.

`GET /steam/status` conserva el diagnóstico protegido por autenticación interna
(`logged_in`, `bot_steam_id`, `gc_session`, `redis_ready`); ya no publica
`friends_ready`. Las rutas internas `/steam/check-friend` y
`/steam/send-friend-request` ya no existen.

`test/steamGateway.test.js` ejecuta el resolver con un GC simulado que no permite
consultar relaciones sociales. Comprueba resolución, recuperación tras indisponibilidad
del GC y reutilización de URLs en caché. No realiza un inicio de sesión real en Steam.

## Estructura

```text
controllers/    Adaptadores HTTP
middleware/     Seguridad de las peticiones
services/       Steam, Redis, descargas y pipeline
test/           Pruebas unitarias y de integración
index.js        Composición y arranque del servicio
```

`pipelineV2.js` conserva el sufijo mientras convive con el despliegue anterior. Antes de renombrarlo, completa el rollout descrito en [`docs/operations/pipeline-v2-rollout.md`](../../docs/operations/pipeline-v2-rollout.md).

## Desarrollo

Desde la raíz:

```powershell
npm --prefix backend/node-service ci
Copy-Item backend/node-service/.env.example backend/node-service/.env
npm run node-service
```

Requiere Redis y credenciales válidas de Steam. El puerto local predeterminado es `4000`.

`npm run start:all` inicia los cuatro servicios después de comprobar Redis y la
coherencia de sus secretos y namespace. `/ready` sólo devuelve éxito con Redis,
workers y una sesión real de GC disponibles. El onboarding importa la partida
aportada una vez que Steam valida el historial y mantiene el cursor de las siguientes.
El evento `discovery_completed` cierra el progreso de búsqueda; cada demo conserva
su identidad y progreso aunque haya varias en curso. El resolver libera solicitudes
y temporizadores tanto al terminar como al fallar o agotarse el tiempo de espera.
Si el CDN rechaza un enlace caducado, la resolución elimina su caché y consulta al
GC de nuevo. La cadena permite dos renovaciones antes de declarar la descarga no
disponible; no genera un ciclo infinito de trabajos. Un rechazo de los códigos
de historial publica un fallo para cerrar el progreso y pedir su renovación.

## Pruebas

```powershell
npm run test:node
```

Las pruebas de integración con Redis se activan en CI mediante `REDIS_INTEGRATION=true`.

Última revisión: 15 de octubre de 2026.
