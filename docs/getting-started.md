# Incorporarse a StratAI

Esta guía permite entender el repositorio y preparar un primer cambio. No necesitas leer todos los contratos ni los informes históricos para empezar.

## Primera lectura

1. Lee el [README principal](../README.md) y prepara el entorno local.
2. Consulta la [definición de producto](AI_COACH_PRODUCT_DEFINITION.md) para distinguir el objetivo del coach de las capacidades ya demostradas.
3. Lee el estado y la siguiente acción del [plan canónico](AI_COACH_IMPLEMENTATION_PLAN.md). Es la única fuente de progreso y prioridades.
4. Usa el [mapa del repositorio](repository-structure.md) y el README del módulo que vayas a modificar.
5. Abre el contrato del área desde el [índice de contratos](contracts/README.md) cuando tu cambio afecte a datos o a la comunicación entre módulos.

## Elegir un punto de entrada

| Trabajo | Empieza aquí | Verificación habitual desde la raíz |
|---|---|---|
| Pantallas, interacción y replay | [Frontend](../frontend/README.md) | `npm run test:frontend` y `npm run build:frontend` |
| Endpoints, sesiones y agregación | [Backend](../backend/README.md) | `npm run lint:python` y `npm run test:python` |
| Steam, descargas y colas | [Servicio Steam](../backend/node-service/README.md) | `npm run test:node` |
| Captura y análisis de demos | [Servicio Go](../backend/go-service/README.md) | `npm run test:go` |
| Contexto factual, interpretación y validación del coach | [Motor AI Coach](../ai_coach/README.md) | Suite específica de `ai_coach/tests/`, descrita en su README |
| Contratos o guías | [Índice documental](README.md) | `npm run check:docs` y revisar consumidores de rutas y fuente canónica |

El arranque habitual `npm run start:all` comprueba Redis y la configuración antes
de iniciar frontend, FastAPI, Node y Go. Consulta el [flujo completo de demos](../backend/README.md#acceso-al-historial-y-descarga-de-demos)
y la [preparación de Redis en Windows](../backend/README.md#redis-y-wsl-en-windows).

## Preparar el primer cambio

Comprueba `git status --short` antes de editar: este directorio de trabajo puede contener cambios previos y archivos todavía no versionados. Delimita tu cambio y conserva los demás.

Localiza el productor y los consumidores del dato o comportamiento. Usa el grafo de código cuando esté disponible; confirma en el archivo actual las partes que el índice no cubra o tenga desactualizadas. Coloca la lógica en el dominio propietario y deja rutas y componentes como adaptadores.

Ejecuta las comprobaciones del área y consulta [AGENTS.md](../AGENTS.md) para los gates adicionales. Un cambio de parser, exportador, hechos derivados, schema o contrato runtime también requiere el [smoke real de una demo](operations/ai-coach-validation.md). Las fases nuevas del coach requieren el encuadre experto definido en el plan.

## Código, datos y resultados

- `frontend/`, `backend/app/`, `backend/node-service/`, `backend/go-service/` y `ai_coach/` contienen el código mantenido.
- `backend/data/` contiene datos locales y runtime; no es una carpeta de limpieza.
- [outputs/](../outputs/README.md) conserva entregables y evidencias de trabajo. Sus scripts fechados no son comandos públicos de desarrollo.
- [output/](../output/README.md) contiene la ubicación heredada de capturas locales.
- Dependencias, perfiles de navegador, índices y cachés son estado local. No se incluyen en una aportación de código.

Una carpeta con fecha no significa que su contenido sea prescindible. Los informes, workbooks, artefactos aprobados y salidas retenidas conservan sus referencias y condiciones de retención.
