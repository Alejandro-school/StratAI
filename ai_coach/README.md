# Motor AI Coach

Paquete Python para contratos, validación, preparación de datos e interpretación del coach. Es una biblioteca y un conjunto de herramientas; no es un quinto servidor HTTP. Su presencia en el repositorio no acredita que todas las capacidades del producto estén implementadas o validadas.

Consulta la [definición del producto](../docs/AI_COACH_PRODUCT_DEFINITION.md) y el [plan canónico](../docs/AI_COACH_IMPLEMENTATION_PLAN.md) para conocer el alcance y la siguiente acción. El [índice de diseño](../docs/ai-coach/README.md) y el [índice de contratos](../docs/contracts/README.md) permiten profundizar por tema.

## Módulos

| Carpeta | Responsabilidad | Cuándo entrar |
|---|---|---|
| [contracts/](contracts/) | Catálogo máquina, schemas y validadores | IDs, estructura, restricciones y validación de artefactos |
| [datasets/](datasets/) | Acceso a bundles y preparación de ejemplos y candidatos | Construir entradas a partir de datos existentes |
| [economy/](economy/) | Observación y reconstrucción económica | Compras, objetos, tenencia y conciliación |
| [support/](support/) | Anclajes y contextos factuales de apoyo y preparación | Cortes causales, recursos propios/aliados y dependencias |
| [spatial/](spatial/) | Datos y revisión espacial | Candidatos espaciales y sus límites |
| [interpretation/](interpretation/) | Contexto, descripción e interpretación temporal | Separar hechos, hipótesis, incertidumbre y revisión |
| [audits/](audits/) | Auditoría de bundles | Comprobar disponibilidad y coherencia de fuentes |
| [tests/](tests/) | Pruebas del paquete | Verificar reglas, causalidad, validadores y regresiones |

Go produce las observaciones a partir de demos. Este paquete consume y valida datos, y construye las representaciones de cada dominio. FastAPI y React tienen sus propios adaptadores; no debe trasladarse el parsing de demos a la interfaz.

## Desarrollo y pruebas

Utiliza el entorno Python descrito en el [README principal](../README.md) y ejecuta desde la raíz para resolver los imports `ai_coach.*`:

```powershell
backend\venv\Scripts\python.exe -m pytest ai_coach/tests
```

`npm run test:python` cubre `backend/tests/`. El comando `npm run check:all` actual no añade automáticamente `ai_coach/tests/`; ejecuta también la suite de este paquete cuando lo modifiques.

No ejecutes módulos de materialización o validación sobre el corpus por exploración. Sus entradas, salidas y gates se describen en el contrato correspondiente y en la [guía de validación aislada](../docs/operations/ai-coach-validation.md).

## Límites entre módulos

Conserva los IDs del [catálogo](contracts/capability_catalog.json), la disponibilidad causal y la procedencia de los hechos. Una capa posterior no puede transformar una ausencia en observación ni una hipótesis en hecho. Los lectores y validadores deben conservar la distinción entre datos publicados, candidatos y resultados de revisión.

Añade lógica en el dominio propietario, con pruebas en `tests/`. Si una herramienta fechada en `outputs/` resulta reutilizable, extrae su responsabilidad al módulo adecuado y conserva el artefacto histórico original.
