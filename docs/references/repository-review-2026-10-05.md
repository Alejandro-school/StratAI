# Revisión de estructura del repositorio

Revisión del 5 de octubre de 2026 para facilitar la incorporación de una persona nueva. Se reorganiza la documentación y se explicitan los módulos, sus límites y sus verificaciones. No se cambia el comportamiento del producto ni el estado de las fases del coach.

## Hallazgos y cambios aplicados

| Hallazgo | Resultado |
|---|---|
| 74 documentos Markdown mezclados en la raíz de `docs/` | 62 documentos trasladados a diseño, contratos y operación; quedan 13 en la raíz, incluida la nueva guía de incorporación |
| README principal con 213 líneas, decenas de contratos especializados y añadidos después de la fecha de revisión | README de 147 líneas centrado en orientación, entorno, arranque y verificaciones; los contratos se consultan por dominio |
| `ai_coach/` ausente del mapa principal y sin README propio | Guía del paquete con sus ocho áreas, relaciones, límites y comando de pruebas |
| Guía del backend sin `matches/` ni `tactical/` | Mapa actualizado para situar la lógica de dominio fuera de los endpoints |
| `output/` y `outputs/` sin explicación de su diferencia | Guías de uso y retención, conservando las rutas citadas en evidencia |
| Rutas documentales usadas por software y recibos históricos | Rutas consumidas por software conservadas; mapa explícito de 62 traslados para referencias históricas |
| Sin comando público de comprobación documental | `npm run check:docs` comprueba destinos locales, traslados sin duplicados y un único prompt activo |
| Directorios residuales `pytest-cache-files-*` aparecían en el recorrido de Git | Regla específica en `.gitignore`; no se eliminan archivos |

## Organización resultante

| Destino | Documentos trasladados | Índice |
|---|---:|---|
| Diseño, criterios y protocolos | 18 | [AI Coach](../ai-coach/README.md) |
| Contratos de contexto e interpretación | 8 | [Contexto](../contracts/context/README.md) |
| Contratos de economía | 12 | [Economía](../contracts/economy/README.md) |
| Contratos de apoyo y preparación | 20 | [Apoyo](../contracts/support/README.md) |
| Guías operativas | 4 | [Operación](../operations/README.md) |

La [guía de incorporación](../getting-started.md) conduce del README al producto, plan, mapa del código y contrato pertinente. El [índice documental](../README.md) mantiene la distinción entre fuentes canónicas, propuestas y resultados históricos.

Se mantienen las rutas canónicas fijadas por `AGENTS.md` y cinco documentos citados por el catálogo o por `context_comparison.py`. También se conserva el protocolo CS3 en la raíz: otra sesión lo estaba editando durante esta revisión. El plan y los cambios concurrentes de CS3 conservan su contenido funcional.

## Observaciones sobre modularidad del código

La separación por servicios ya existe. El motor Python añade módulos de economía, apoyo, espacio e interpretación, y FastAPI dispone de dominios de partidas y táctica. La guía anterior no reflejaba toda esa estructura; el [mapa actualizado](../repository-structure.md) sí la incluye.

Hay áreas cuyo tamaño merece una revisión funcional específica. El inventario local al inicio de esta tarea midió 8.522 líneas en `backend/go-service/scripts/canonical_export_validator.py`, 3.466 en `backend/go-service/parser/canonical_exporter.go` y 2.280 en `backend/app/routes/dashboard.py`. `ai_coach/contracts/` contenía 144 archivos directos, y `backend/go-service/handlers/` 90, incluyendo schemas o pruebas según la carpeta. Estos tamaños son señales de coste de lectura, no una demostración de defectos ni de acoplamiento incorrecto.

Una extracción funcional debe seguir responsabilidades y consumidores, conservar imports y contratos y superar las pruebas pertinentes. En Go, mover archivos entre directorios cambia el paquete. La reorganización actual no realiza esas extracciones ni declara que el código esté completamente desacoplado. Estas observaciones no crean un roadmap alternativo ni autorizan nuevas fases; el [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) sigue fijando el trabajo del coach.

También se documenta un límite de la interfaz de pruebas actual: `npm run check:all` no incluye `ai_coach/tests/`. El README del motor indica cómo ejecutar esa suite adicional. La revisión no altera los comandos de arranque ni la composición de las suites existentes.

## Preservación y límites

Los cambios de esta tarea se limitan a documentación, su comprobador, un comando npm adicional y una regla de exclusión de caché. No modifican código runtime, contratos máquina, schemas, fuentes de demos, corpus, datos locales, releases, workbooks ni evidencia JSON preexistente. Los enlaces navegables de los informes Markdown se ajustan a las ubicaciones actuales; los recibos históricos conservan sus bytes y se interpretan mediante el [mapa de traslados](document-locations.json).

`outputs/` contiene herramientas históricas que pueden mencionar rutas anteriores. Se conservan tal como se ejecutaron; el mapa no es un redirect runtime y no convierte esos scripts en comandos reutilizables. Ninguna instrucción de incorporación requiere ejecutarlos.

La revisión combinó el grafo de código con inventario directo y lectura de los puntos de entrada. El índice señaló metadatos cambiados y exclusiones de cachés; por tanto, no se afirma una auditoría exhaustiva de comportamiento o dependencias. Dos carpetas residuales de pytest y una carpeta de fixtures de una entrega histórica no eran accesibles en el inventario inicial; no se intentó alterar sus permisos ni su contenido.

## Verificación

La [evidencia de esta reorganización](../evidence/PROJECT_STRUCTURE_2026-10-05.json) registra los resultados, las comparaciones con el estado inicial y las modificaciones concurrentes observadas.

```powershell
npm run check:docs
node --check scripts/check-docs.mjs
git diff --check
```

El comprobador cubre `docs/` y las guías de entrada mantenidas. No valida URLs externas, anclas de encabezados ni todas las copias históricas de `outputs/`. Su prueba aislada comprueba también que rechaza destinos ausentes, copias duplicadas tras un traslado, documentos trasladados sin enlace de entrada y más de un prompt activo.

No se ejecutan suites funcionales ni un smoke de demo por esta reorganización documental: los productores, hechos derivados, schemas y contratos runtime quedan fuera del cambio. Los inventarios y respaldos de trabajo se conservan en `.cache/project-structure-20261005/`.
