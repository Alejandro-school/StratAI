# Contratos de datos

Este índice agrupa los contratos por responsabilidad. Una versión candidata o un suplemento no sustituye el contrato de publicación ni demuestra por sí solo una capacidad del coach.

## Fuentes estables

| Necesidad | Fuente |
|---|---|
| Interpretar los bundles publicados | [Export canónico 3.8](../canonical-export-schema-v3.8.md) |
| Significado, disponibilidad y causalidad de capacidades | [Contrato humano](../AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) |
| IDs y referencias consumidas por software | [Catálogo máquina](../../ai_coach/contracts/capability_catalog.json) |
| Schemas ejecutables y validadores | [ai_coach/contracts/](../../ai_coach/contracts/) |
| Estado, dependencias y siguiente acción | [Plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) |

## Contratos por dominio

| Área | Contenido |
|---|---|
| [Contexto e interpretación](context/README.md) | Contexto factual, replay, rondas, hipótesis y fichas de revisión |
| [Economía](economy/README.md) | Estados, oportunidades, identidad, propiedad, fuentes y conciliación |
| [Apoyo y preparación](support/README.md) | Anclajes, cortes C0, recursos, objetivo, movimiento, armas y límites visuales |

Las referencias que utiliza directamente el software permanecen en su ruta anterior y se enlazan desde estos índices. Cada contrato tiene una única copia mantenida. El [mapa de ubicaciones](../references/document-locations.json) permite consultar rutas citadas en evidencia histórica.

Para modificar un contrato, identifica productor y consumidores, actualiza sus validadores y aplica los gates de [AGENTS.md](../../AGENTS.md). Para interpretar una prueba real, consulta la [guía de validación](../operations/ai-coach-validation.md).
