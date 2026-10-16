# Documentación de StratAI

Empieza por la [guía de incorporación](getting-started.md) si eres nuevo, o por el [README principal](../README.md) para instalar y arrancar. Este índice separa fuentes canónicas, diseño, contratos, operación y evidencia.

## Fuentes canónicas

| Necesidad | Documento |
|---|---|
| Promesa, alcance y límites del producto | [Definición de producto](AI_COACH_PRODUCT_DEFINITION.md) |
| Estado, orden, dependencias y siguiente acción | [Plan de implementación](AI_COACH_IMPLEMENTATION_PLAN.md) |
| Capacidades, datos y causalidad | [Contrato humano](AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) |
| IDs, campos y referencias para software | [Catálogo máquina](../ai_coach/contracts/capability_catalog.json) |
| Interpretar los bundles publicados | [Contrato canónico 3.8](canonical-export-schema-v3.8.md) |
| Preparar el siguiente trabajo indicado por el plan | [Prompts](prompts/) |

El plan es la única fuente de progreso y orden de ejecución. Los documentos de diseño, contratos candidatos e informes fechados no autorizan fases, entrenamiento, adquisición ni publicación. Consulta la [forma de trabajo por fase](AI_COACH_IMPLEMENTATION_PLAN.md#forma-de-trabajo-por-fase).

## Explorar por tarea

| Sección | Qué encontrarás |
|---|---|
| [Incorporación](getting-started.md) | Primera lectura, entorno y primer cambio |
| [Estructura del repositorio](repository-structure.md) | Responsabilidades y ubicación del código |
| [Diseño del AI Coach](ai-coach/README.md) | Arquitectura, criterios, propuestas, revisión y evaluación |
| [Contratos](contracts/README.md) | Índices de contexto, economía y apoyo |
| [Operación](operations/README.md) | Validación, publicación, rollback y corpus golden |
| [Referencias](references/README.md) | Material técnico y mapa de documentos trasladados |
| [Entregables](../outputs/README.md) | Cómo consultar workbooks, revisiones y resultados fechados |

## Historia y evidencia

- [Archivo del AI Coach](archive/ai-coach/README.md): informes fechados y propuestas anteriores.
- [Evolución del parser](archive/parser-evolution/): contratos y auditorías anteriores.
- [Evidencias máquina](evidence/): comandos, resultados, conteos y hashes de cada cierre.
- [Limpieza del 04/10/2026](evidence/PROJECT_CLEANUP_2026-10-04.json): alcance y registro de la limpieza previa.
- [Revisión de estructura del 05/10/2026](references/repository-review-2026-10-05.md): hallazgos, reorganización y límites de esta revisión.
- [Revisión de skills del 06/10/2026](archive/skills-review-2026-10-06.md): selección aplicada, fuentes y recuperación del catálogo anterior.

Archivar o mover un documento no cierra una fase. Los registros históricos conservan su alcance original. Para resolver una ruta anterior, usa el [mapa de ubicaciones](references/document-locations.json).

## Dónde guardar documentación nueva

Mantén en la raíz las fuentes canónicas y las rutas estables consumidas por software. Guarda diseño y protocolos en `ai-coach/`, contratos del dominio en `contracts/`, guías operativas en `operations/`, referencias técnicas en `references/`, informes fechados en `archive/` y recibos máquina en `evidence/`. Cada directorio temático mantiene su índice.

`prompts/` contiene únicamente el siguiente prompt ejecutable. El estado se actualiza en el plan, sin crear un segundo roadmap en un README o informe.
