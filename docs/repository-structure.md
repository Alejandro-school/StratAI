# Estructura y convenciones del repositorio

Esta guía ayuda a localizar el código correcto antes de empezar un cambio.

Si te incorporas al proyecto, comienza por la [guía de incorporación](getting-started.md). La instalación y los comandos de arranque están en el [README principal](../README.md).

## Mapa del directorio de trabajo

```text
StratAI/
  frontend/              Aplicación React
  backend/
    app/                 API FastAPI y dominios de agregación
    node-service/        Steam, descargas y colas
    go-service/          Captura, análisis y exportación de demos
    tests/               Pruebas de FastAPI
    testdata/            Fixtures mantenidos
    scripts/             Migraciones y tareas puntuales de backend
    data/                Datos locales y runtime; conservar
  ai_coach/              Motor Python, contratos y pruebas del coach
  docs/
    ai-coach/            Diseño, criterios y protocolos
    contracts/           Contratos por contexto, economía y apoyo
    operations/          Validación y procedimientos operativos
    references/          Referencias técnicas y mapa de traslados
    archive/             Informes y propuestas anteriores
    evidence/            Recibos de ejecuciones
    prompts/             Siguiente prompt ejecutable
  outputs/               Entregables y evidencia de trabajo retenida
  output/                Ubicación heredada de capturas locales
  scripts/               Comprobaciones de mantenimiento del repositorio
  .agents/               Catálogo de skills del repositorio
  .github/               Automatización de CI
```

La raíz de `docs/` conserva el producto, plan y contratos canónicos, las guías de entrada, cinco referencias consumidas directamente por el software y el protocolo CS3 en edición activa. Los [índices temáticos](README.md) permiten recorrerlas sin duplicar documentos. El [mapa de traslados](references/document-locations.json) resuelve rutas de documentos citadas en registros anteriores.

## Responsabilidad de cada aplicación

| Aplicación | Es propietaria de | No debería contener |
| --- | --- | --- |
| React (`frontend/`) | Navegación, interacción, visualización y estado de interfaz. | Descargas de Steam o análisis de demos. |
| FastAPI (`backend/app/`) | API pública, sesiones, autorización y agregación de resultados. | Automatización del cliente de Steam o parsing de demos. |
| Node.js (`backend/node-service/`) | Steam, descargas, colas y coordinación del pipeline. | Presentación o cálculos estadísticos propios de la API. |
| Go (`backend/go-service/`) | Parsing, eventos y análisis de demos. | Autenticación de usuarios o lógica de interfaz. |
| AI Coach (`ai_coach/`) | Contratos, validación, contexto factual e interpretación por dominio. | Adaptadores HTTP, componentes React o adquisición de demos. |
| Redis | Estado compartido, trabajos y resultados procesados. | Datos que solo existen durante el renderizado del frontend. |

## Dónde realizar un cambio

| Necesidad | Ubicación inicial |
| --- | --- |
| Añadir una pantalla o interacción | `frontend/src/pages/` y su feature en `frontend/src/features/`. |
| Consumir un endpoint desde React | `api/` dentro de la feature propietaria. |
| Cambiar autenticación o sesiones | `backend/app/auth/`, `backend/app/security/` o `backend/app/middleware/`. |
| Añadir un endpoint público | `backend/app/routes/`. |
| Cambiar una descarga o integración de Steam | `backend/node-service/services/`. |
| Cambiar la coordinación de trabajos | `backend/node-service/services/pipelineV2.js`. |
| Extraer una métrica de una demo | `backend/go-service/analyzers/` o `handlers/`. |
| Cambiar el modelo de salida de Go | `backend/go-service/models/` y `parser/`. |
| Cambiar contexto o interpretación del coach | `ai_coach/interpretation/`, con contratos y pruebas del paquete. |
| Cambiar hechos de economía, apoyo o espacio | `ai_coach/economy/`, `support/` o `spatial/`, y su productor Go si corresponde. |
| Cambiar el catálogo o un schema del coach | `ai_coach/contracts/`, revisando productores y consumidores. |
| Ejecutar una migración puntual | `backend/scripts/`. |
| Añadir una prueba | Junto al código en React, Node y Go; en `backend/tests/` para FastAPI. |
| Añadir una prueba del motor del coach | `ai_coach/tests/`; su ejecución está descrita en [su README](../ai_coach/README.md). |

Antes de modificar un contrato entre servicios, busca a todos sus consumidores. Los cambios en claves de Redis, payloads HTTP o modelos exportados suelen requerir ajustes coordinados en más de una aplicación.

## Convenciones de nombres

### Nombres orientados al dominio

El nombre debe explicar la responsabilidad:

- Usa `MatchHistoryPage`, `ReplayController` o `PerformanceAggregator`.
- Evita nombres nuevos como `utils`, `helpers`, `misc`, `common` o `manager` sin un dominio explícito.
- Una carpeta técnica solo es apropiada cuando todos sus elementos comparten realmente esa función.

### Convenciones por lenguaje

| Elemento | Convención | Ejemplo |
| --- | --- | --- |
| Componente o página React | `PascalCase` | `MatchHistoryPage.jsx` |
| Hook React | Prefijo `use` | `useMatchProgress.js` |
| Feature o sección frontend | `kebab-case` | `match-history/` |
| Módulo Python | `snake_case` | `replay_annotations.py` |
| Módulo Node.js | `camelCase` | `demoDownload.js` |
| Paquete Go | minúscula y singular | `middleware/` |
| Prueba | Convención del lenguaje | `middleware_test.go`, `security.test.js` |

No añadas números al nombre de una carpeta para indicar el orden visual. El orden pertenece al componente que compone la pantalla.

## Organización del frontend

El repositorio está migrando gradualmente desde carpetas por tipo (`components`, `hooks`, `styles`, `utils`) hacia módulos por feature. Para código nuevo, utiliza:

```text
features/
  nombre-feature/
    api/
    components/
    domain/
    hooks/
    index.js
```

No es necesario mover una zona antigua para realizar un arreglo aislado. Cuando una funcionalidad reciba cambios sustanciales, migra juntos sus componentes, hooks, lógica y estilos, manteniendo un `index.js` como API pública.

## Organización de los dominios Python y Go

FastAPI ya dispone de `matches/` y `tactical/` además de sus adaptadores `routes/`. Coloca las reglas y transformaciones del dominio en esos módulos o en otro dominio explícito, dejando al endpoint la validación HTTP, autorización y composición de la respuesta.

El [motor del coach](../ai_coach/README.md) separa contratos, datasets, economía, apoyo, espacio e interpretación. Mantén esa separación al extraer lógica reutilizable de una herramienta de revisión. No conviertas los scripts fechados de `outputs/` en dependencias implícitas del producto.

Go combina capas técnicas (`handlers/`, `models/`, `parser/`) con paquetes por dominio en `pkg/`. Mover un archivo Go a otra carpeta cambia su paquete: una reorganización funcional debe revisar imports, visibilidad y consumidores, y pasar sus pruebas y smoke cuando corresponda. Agrupar documentación no acredita esa refactorización.

## Datos, artefactos y estado local

| Ubicación | Tratamiento |
|---|---|
| `backend/data/`, `dump.rdb` | Datos locales o runtime. No mover ni borrar como parte de una limpieza de fuentes. |
| `outputs/` | Entregables, evidencia y herramientas fechadas. Consultar su [guía](../outputs/README.md). |
| `output/playwright/` | Capturas locales heredadas, excluidas de Git. |
| `.cache/` | Trabajo auxiliar regenerable e inventarios locales de mantenimiento. |
| `node_modules/`, `backend/venv/`, cachés de Python/Go | Dependencias y estado de herramientas, separados de las fuentes. |
| `.browser-profile/`, `.codebase-memory/`, `.tokensave/` | Estado local de navegador y herramientas. |
| `stratai-go-release/` | Artefactos locales de release; conservar su trazabilidad. |

Los outputs de smoke deben seguir usando un TEMP nuevo, aislado y retenido según [AGENTS.md](../AGENTS.md). No unifiques carpetas de resultados sin comprobar sus referencias e inventarios.

## Contratos entre servicios

El flujo principal es:

```text
FastAPI → Node.js → Go
    └──────── Redis ────────┘
```

- El navegador solo se comunica con FastAPI.
- Node.js y Go son internos.
- Los secretos internos deben coincidir entre los servicios.
- Las claves y estados del pipeline se aíslan mediante `PIPELINE_NAMESPACE`.
- Cualquier cambio de contrato debe incluir pruebas en productor y consumidor.

## Documentación al cambiar estructura

Actualiza en el mismo cambio:

1. Imports y exportaciones.
2. Pruebas afectadas.
3. El README del servicio si cambia su responsabilidad o arranque.
4. Esta guía si cambia la ubicación recomendada.
5. Los ejemplos de entorno si aparece una variable nueva.

Ejecuta `npm run check:docs` para comprobar destinos locales de enlaces, documentos trasladados sin duplicados y el único prompt activo. El chequeo cubre `docs/` y las guías de entrada mantenidas; no verifica URLs externas, anclas de sección ni copias históricas retenidas en `outputs/`.

Última revisión: 5 de octubre de 2026.
