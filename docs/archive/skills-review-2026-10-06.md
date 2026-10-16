# Revisión de skills de StratAI

Fecha: 6 de octubre de 2026.

El catálogo del repositorio pasa de **9 a 5 skills**. Se actualizan las fuentes de diseño y React, se retiran instrucciones ajenas al stack y se concentra la orientación 3D en React Three Fiber. La selección prioriza utilidad para StratAI, compatibilidad, procedencia verificable y precisión de activación; no pretende ser un ranking universal.

## Decisiones aplicadas

| Skill anterior | Decisión | Motivo y cobertura resultante |
| --- | --- | --- |
| `audit-website` | Retirada | Orientada a rastreo SEO mediante squirrelscan; el CLI no se encontró en PATH. Incluía instalación remota ejecutada directamente, preferencia por sitios públicos y configuración adicional. La revisión habitual de la aplicación queda cubierta por `web-design-guidelines` y las herramientas de navegador disponibles. Esto no sustituye una auditoría SEO especializada si se solicita en el futuro. |
| `crafting-effective-readmes` | Retirada | Aportaba plantillas genéricas y preguntas obligatorias para tareas cuyo contexto ya está en el repositorio. Los README siguen las convenciones documentales existentes. |
| `frontend-design` | Actualizada y adaptada | Nueva fuente oficial de Anthropic. Se conserva la dirección visual deliberada y se adapta a pantallas de análisis CS2, estilos existentes, español, accesibilidad y estados sin datos. Se retiran prohibiciones arbitrarias de fuentes y el rediseño implícito de cada componente. |
| `vercel-react-best-practices` | Actualizada y filtrada | Selección de 10 referencias de Vercel compatibles con el trabajo en React 18. Se retira el manual compilado duplicado y las reglas de Next.js, SSR/RSC, React 19 y SWR. Se explica el uso de Vite, TanStack Query y Zustand, con medición antes y después. |
| `web-design-guidelines` | Adaptada | Conserva la fuente de Vercel, acota el alcance a UX y accesibilidad e incorpora el flujo temporal de StratAI. Usa herramientas web disponibles, sin depender de un nombre de herramienta específico, y declara cuándo no pudo consultar las reglas actuales. |
| `python-performance-optimization` | Reemplazada en la misma ruta | Sus 874 líneas de tutorial pasan a un procedimiento breve de profiling y comparación de resultados. Prioriza herramientas estándar, el intérprete del proyecto y la conservación de semántica, causalidad y evidencia. |
| `threejs-fundamentals` | Sustituida | Integrada en `stratai-react-three-fiber`, centrada en el Canvas y ciclo de vida de Fiber. |
| `threejs-animation` | Sustituida | La nueva skill cubre animación con delta, refs, mixers y propiedad de recursos. |
| `threejs-interaction` | Sustituida | La nueva skill cubre eventos de Fiber, controles, coordenadas y accesibilidad. Excluye expresamente el replay Canvas 2D. |
| `playwright` personal | Conservada | Aporta automatización y comprobaciones de navegador. Sus ejemplos Bash requieren adaptación a PowerShell cuando se ejecuten en este entorno. No se duplica dentro del repositorio. |
| `security-best-practices` personal | Conservada | Su ámbito de Python, JavaScript y Go coincide con los servicios del proyecto y su activación se limita a trabajo de seguridad solicitado. |

La presencia real de React Three Fiber se comprobó en `frontend/package.json` y en `PerformanceScene.jsx`, tras localizar el componente mediante el grafo. Las guías Three.js anteriores no eran inútiles, pero sus ejemplos imperativos y referencias a skills ausentes encajaban peor que una guía de la integración existente.

Las skills del sistema y las aportadas por plugins se evaluaron por su ámbito declarado y se conservaron: documentación OpenAI, creación e instalación de skills, documentos, PDF, hojas de cálculo, presentaciones, imágenes, navegador, Pages, Sites, visualización y mascotas. Son capacidades de la aplicación para tareas puntuales; esta revisión no desinstala plugins ni modifica su caché o la configuración global. Su código y recursos internos no se han auditado exhaustivamente.

## Fuentes y selección

La documentación de OpenAI recomienda descripciones concretas, instrucciones centradas en una tarea y referencias cargadas solo cuando hacen falta. También mantiene `.agents/skills` como ubicación válida para skills propias de un repositorio. Se aplican esos criterios al catálogo de StratAI. [Documentación de skills](https://learn.chatgpt.com/docs/build-skills), [orientación de OpenAI sobre tamaño y alcance](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra).

Las actualizaciones externas se descargaron con el instalador de skills a un directorio temporal y se revisaron antes de aplicarlas. Se fijaron commits para poder identificar las fuentes exactas:

| Fuente | Versión consultada | Uso |
| --- | --- | --- |
| [Anthropic skills](https://github.com/anthropics/skills/tree/683bc88e56f3e09ba94f7055977f3d3aa499f202/skills/frontend-design) | `683bc88e56f3e09ba94f7055977f3d3aa499f202` | Base de diseño; licencia conservada. |
| [Vercel agent-skills](https://github.com/vercel-labs/agent-skills/tree/063bee94c3f4df8453406c830b0a7df0f2860278/skills) | `063bee94c3f4df8453406c830b0a7df0f2860278` | React y revisión de interfaces; atribución y licencia declarada conservadas. |
| [React Three Fiber](https://r3f.docs.pmnd.rs/advanced/pitfalls) | Documentación consultada el 06/10/2026 | Referencia para la nueva skill propia; no es una skill oficial de Poimandres. |
| [Python 3.11](https://docs.python.org/3.11/library/profile.html) | Documentación consultada el 06/10/2026 | Referencia para la reescritura de profiling; no es una skill oficial de Python. |

El repositorio [openai/skills](https://github.com/openai/skills) está marcado como deprecado y remite a [openai/plugins](https://github.com/openai/plugins). No se añadieron copias de skills ya disponibles ni paquetes de Next.js, Expo, despliegue o integraciones sin una necesidad del proyecto. Las adaptaciones locales no deben sobrescribirse mediante una actualización ciega del upstream.

## Comprobación y recuperación

Las cinco skills pasan el validador oficial `quick_validate.py`, la comprobación de nombres únicos y la resolución de enlaces locales. Los hashes de los 16 archivos instalados coinciden con la selección preparada. `npm run check:docs` pasa con 219 documentos y 1.601 enlaces locales comprobados; `git diff --check` pasa en el alcance modificado. Se revisó que sus descripciones separen diseño visual, rendimiento React, auditoría UX, profiling Python y escenas 3D. No se ha realizado una evaluación empírica del comportamiento del modelo; la mejora esperada se basa en los conflictos concretos corregidos.

Los puntos de entrada `SKILL.md` pasan de 3.235 a 161 líneas; es una medida del tamaño de las instrucciones, no una medición de tokens consumidos ni de rendimiento. Las referencias React se cargan según el problema. Esta revisión modifica instrucciones y documentación, sin cambios de producto ni contratos de datos; no requiere procesar demos.

El [registro de evidencia](../evidence/SKILLS_REVIEW_2026-10-06.json) conserva decisiones, inventarios SHA-256 y la ubicación de recuperación. Las nueve carpetas anteriores y un ZIP completo permanecen fuera de las rutas de descubrimiento, en el directorio temporal de esta revisión. Para restaurar una skill, comparar primero su copia con el estado actual y recuperar únicamente la carpeta elegida, sin sobrescribir cambios posteriores.

Las skills nuevas estarán disponibles en el siguiente turno. Si el catálogo no se actualiza, reiniciar Codex permite recargarlo.
