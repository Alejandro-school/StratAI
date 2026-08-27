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

## Estructura

```text
controllers/    Adaptadores HTTP
middleware/     Seguridad de las peticiones
services/       Steam, Redis, descargas y pipeline
test/           Pruebas unitarias y de integración
index.js        Composición y arranque del servicio
```

`pipelineV2.js` conserva el sufijo mientras convive con el despliegue anterior. Antes de renombrarlo, completa el rollout descrito en [`docs/pipeline-v2-rollout.md`](../../docs/pipeline-v2-rollout.md).

## Desarrollo

Desde la raíz:

```powershell
npm --prefix backend/node-service ci
Copy-Item backend/node-service/.env.example backend/node-service/.env
npm run node-service
```

Requiere Redis y credenciales válidas de Steam. El puerto local predeterminado es `4000`.

## Pruebas

```powershell
npm run test:node
```

Las pruebas de integración con Redis se activan en CI mediante `REDIS_INTEGRATION=true`.

Última revisión: 26 de julio de 2026.
