# Estructura y convenciones del repositorio

Esta guía ayuda a localizar el código correcto antes de empezar un cambio.

## Responsabilidad de cada aplicación

| Aplicación | Es propietaria de | No debería contener |
| --- | --- | --- |
| React (`frontend/`) | Navegación, interacción, visualización y estado de interfaz. | Descargas de Steam o análisis de demos. |
| FastAPI (`backend/app/`) | API pública, sesiones, autorización y agregación de resultados. | Automatización del cliente de Steam o parsing de demos. |
| Node.js (`backend/node-service/`) | Steam, descargas, colas y coordinación del pipeline. | Presentación o cálculos estadísticos propios de la API. |
| Go (`backend/go-service/`) | Parsing, eventos y análisis de demos. | Autenticación de usuarios o lógica de interfaz. |
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
| Ejecutar una migración puntual | `backend/scripts/`. |
| Añadir una prueba | Junto al código en React, Node y Go; en `backend/tests/` para FastAPI. |

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

Última revisión: 26 de julio de 2026.
