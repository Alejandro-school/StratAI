# Prompt vigente — Fase 1B.1: DecisionExample y TaskEligibility

> **Estado:** VIGENTE — siguiente fase después del cierre de Gate 1A.
>
> **Modelo recomendado:** `gpt-5.6-sol` con razonamiento `high`; usar `xhigh` únicamente si se prioriza profundidad sobre tiempo y coste.
>
> Este prompt ejecuta la fase posterior al cierre documentado de Gate 1A. Ejecuta una sola fase y no autoriza entrenamiento ni cambios en Go.

Quiero continuar el desarrollo de StratAI desde el cierre verificado de Gate 1A. Ejecuta esta fase completa; no te limites a analizar o recomendar. Explícame el resultado final en español y con palabras sencillas.

## Repositorios

- StratAI: `E:\Carpeta compartida\Proyecto IA\StratAI`
- Corpus Mirage: `E:\Carpeta compartida\Proyecto IA\Faceit-Demos`

## Estado que debes preservar

- Gate 1A está cerrado.
- Hay 50 demos fuente de Mirage, 44 bundles canónicos 3.8.0 aprobados y 6 exclusiones documentadas.
- `training_allowed=false`.
- La release activa es `stratai-2c6db440231d463a`.
- `staging` debe permanecer vacío.
- Las fuentes antiguas de `golden-demos-v2` se eliminaron intencionadamente. No las busques ni intentes restaurarlas.
- Puede haber cambios del usuario en ambos repositorios. No los reviertas ni sobrescribas.
- No hagas commit, push ni PR.

Confirma este estado con comprobaciones breves de solo lectura. No repitas las auditorías completas de Gate 1A salvo que encuentres una contradicción real.

## Lectura obligatoria reducida

Antes de modificar archivos:

1. Lee completos los `AGENTS.md` aplicables.
2. Lee en `README.md` la jerarquía documental del AI Coach.
3. Lee en `docs/AI_COACH_IMPLEMENTATION_PLAN.md` únicamente:
   - orden de ejecución;
   - estado actual;
   - Fase 1 y Gate 1B;
   - verificación;
   - backlog ejecutable y próxima acción.
4. Lee en `docs/AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md` las secciones sobre:
   - invariantes causales;
   - decisiones, acciones y outcomes;
   - disponibilidad y procedencia;
   - tests y gates.
5. Valida `ai_coach/contracts/capability_catalog.json` con la herramienta existente y consulta sólo las entradas necesarias para implementar decisiones y elegibilidad. No vuelques sus 4.000 líneas al contexto si no hace falta.
6. Lee en `docs/AI_COACH_MIRAGE_44_DATA_AUDIT_2026-08-27.md` las decisiones materializadas, la cobertura por capacidad y la conclusión.
7. Consulta `docs/canonical-export-schema-v3.8.md` sólo para manifest, estados, decisiones y separación de particiones.

La arquitectura, el snapshot técnico, las evidencias detalladas y los documentos históricos son referencias. Consúltalos únicamente para resolver una duda concreta; no los releas completos por defecto.

Para descubrir código usa primero el grafo del repositorio. Usa búsqueda de texto para documentación, configuración, mensajes y huecos no cubiertos por el grafo.

## Objetivo único

Convertir una parte del contrato de Gate 1A en código Python ejecutable y de solo lectura:

- `DecisionExample`;
- `TaskEligibility`.

El objetivo es demostrar que las decisiones actuales pueden unirse con el estado legal en T0, que las particiones permanecen separadas y que cada capacidad explica por qué es o no utilizable.

Esta fase no crea un dataset de entrenamiento, no entrena modelos y no decide si una jugada fue correcta o incorrecta.

## Alcance funcional

Construye ejemplos estructurales para las dos familias ya materializadas:

- `spacing_or_trade_connection`;
- `peek_hold_or_reposition`.

Implementa `TaskEligibility` para las 17 capacidades del catálogo. Las capacidades todavía no materializadas deben indicarlo expresamente; no deben fingir que tienen ejemplos disponibles.

Estados mínimos:

- `eligible`;
- `ineligible`;
- `not_materialized`.

Cada resultado debe incluir:

- capacidad;
- estado;
- reason codes estables;
- campos requeridos ausentes;
- evidencia utilizada;
- versión del contrato.

## Reglas causales obligatorias

Mantén físicamente separados:

- `observed`;
- `oracle`;
- `outcomes`;
- perfil histórico.

Para cada `DecisionExample`:

- T0 es el último instante legal anterior a la acción evaluada;
- cada feature de entrada cumple `availability_tick <= t0_tick`;
- oracle sólo puede usarse para auditoría, etiquetas posteriores o evaluación;
- outcomes describen lo sucedido después, pero nunca se convierten en observed;
- los identificadores sirven para joins, nunca como variables predictivas;
- un dato ausente no se rellena con cero, media ni una suposición;
- no se etiqueta una alternativa no observada como mejor;
- no se generan recomendaciones ni labels de «correcto» o «incorrecto».

Decisiones ya fijadas en Gate 1A:

- `enemy_los` es un proxy geométrico parcial y no demuestra percepción real;
- FOV y humo no están completamente integrados en esa señal;
- ammo/reload táctico a 16 Hz sigue bloqueado;
- nav y raycast no proporcionan por sí solos zonas tácticas, cobertura o rutas;
- voz, intención, sonido subjetivo y futuro contrafactual son `NO_OBSERVABLE`;
- trade dispone de mayor base factual;
- peek debe resultar inelegible cuando falte el contexto espacial requerido.

No reabras estas decisiones sin evidencia nueva y reproducible.

## Implementación obligatoria

### 1. Contratos versionados

Crea o completa contratos de máquina bajo `ai_coach/contracts/` para:

- `DecisionExample`;
- `TaskEligibility`;
- reason codes;
- referencias mínimas a acción observada, outcome y evidencia.

No desarrolles todavía un contrato prescriptivo completo de candidatos o feedback. Incluye sólo las referencias mínimas necesarias para que `DecisionExample` pueda ampliarse después.

Los contratos deben:

- tener versión explícita;
- validar tipos;
- rechazar reason codes desconocidos;
- impedir campos oracle dentro de observed;
- distinguir campos obligatorios y opcionales;
- evitar IDs duplicados.

### 2. Constructor read-only

Implementa bajo `ai_coach/datasets/` un lector que:

- descubra artefactos únicamente mediante manifests;
- procese un bundle cada vez;
- lea JSONL y gzip por streaming;
- no descomprima archivos de forma persistente;
- mantenga memoria acotada;
- resuelva joins entre decisión, estado observed, contexto, outcome y calidad;
- detecte referencias ausentes, duplicadas o con tipo incorrecto;
- compruebe `availability_tick <= t0_tick`;
- no escriba dentro de `Faceit-Demos`;
- no persista todos los ejemplos individuales.

Para trade y peek construye `DecisionExample` estructurales suficientes para verificar joins, tiempo y disponibilidad. No valores la decisión.

### 3. TaskEligibility

Evalúa las 17 capacidades definidas en el catálogo.

Cada evaluación debe ser determinista e indicar:

- estado;
- reason codes;
- campos ausentes;
- blockers;
- versión del catálogo y del contrato.

No confundas un bundle válido con una capacidad materializada o elegible. Una partida puede ser válida y no servir para una tarea concreta.

### 4. CLI de auditoría

Añade una CLI read-only que recorra los 44 bundles uno por uno. Puede escribir únicamente evidencias nuevas bajo `StratAI/docs/evidence/`.

Debe producir un resumen determinista con:

- bundles esperados, encontrados y procesados;
- joins correctos y fallidos;
- violaciones temporales;
- decisiones estructurales por familia;
- elegibilidad por capacidad y reason code;
- campos requeridos presentes, ausentes y nulos;
- exclusiones;
- errores con ruta y línea cuando sea posible;
- método de streaming y limitaciones;
- una muestra pequeña, acotada y sin datos personales innecesarios.

Si detectas corrupción o una contradicción canónica, no repares el bundle. Detén o marca la auditoría según la gravedad y documenta la ruta afectada.

## Pruebas obligatorias

Añade pruebas sintéticas para:

- join válido;
- referencia ausente;
- referencia duplicada;
- tipo de identificador incorrecto;
- `availability_tick > t0_tick`;
- mutación del futuro que no altera observed;
- oracle inyectado en observed;
- gzip corrupto con error comprensible;
- campo obligatorio ausente;
- reason code desconocido;
- `not_materialized`;
- salida determinista;
- streaming sin cargar todos los bundles simultáneamente.

Los casos negativos deben demostrar que el sistema rechaza estructuras peligrosas. No uses los bundles reales como única prueba.

## Entregables permitidos

Como mínimo:

- contratos versionados y validados;
- constructor read-only de `DecisionExample`;
- evaluación de `TaskEligibility`;
- CLI reproducible;
- pruebas positivas y negativas;
- `docs/evidence/AI_COACH_DECISION_EXAMPLE_COVERAGE_MIRAGE_44_2026-08-27.json`;
- `docs/AI_COACH_DECISION_EXAMPLE_PHASE_2026-08-27.md`.

No crees otro plan, backlog, arquitectura, definición de producto ni contrato humano. Actualiza las fuentes existentes cuando corresponda.

El informe Markdown debe explicar de forma sencilla:

- qué se construyó;
- qué significan `DecisionExample` y `TaskEligibility`;
- cuántos bundles se recorrieron;
- cuántos ejemplos estructurales se construyeron;
- qué capacidades son elegibles y cuáles no;
- si hubo fugas de futuro u oracle;
- blockers, limitaciones y pruebas.

## Verificación mínima

Adapta los nombres exactos sólo si el código implementado lo exige:

```powershell
python -m ai_coach.contracts.validate_contract
python -m pytest ai_coach/tests -q
npm run test:python
python -m ai_coach.datasets.build_examples "E:\Carpeta compartida\Proyecto IA\Faceit-Demos\analyzed_demos" --expected-bundles 44 --summary-only --output "docs\evidence\AI_COACH_DECISION_EXAMPLE_COVERAGE_MIRAGE_44_2026-08-27.json"
python -m json.tool "docs\evidence\AI_COACH_DECISION_EXAMPLE_COVERAGE_MIRAGE_44_2026-08-27.json"
```

Valida también contratos, referencias, IDs y enlaces Markdown. No ejecutes pruebas Go o Node si esos servicios no cambian. Usa `npm run check:all` sólo si modificas código compartido entre varios servicios.

Registra comando, resultado, número de pruebas, duración y si el recorrido fue completo o una muestra. No afirmes que algo pasó si no lo ejecutaste.

## Cierre documental obligatorio

Antes de terminar:

1. Actualiza `docs/AI_COACH_IMPLEMENTATION_PLAN.md` en la misma revisión:
   - incrementa versión y fecha;
   - marca el paso 13 únicamente si todos sus criterios pasan;
   - añade evidencia, comandos y resultados;
   - actualiza el progreso de Gate 1B;
   - no declares Gate 1B completo mientras quede cualquiera de sus criterios pendiente;
   - fija una sola siguiente acción.
2. Actualiza `README.md` para mostrar el estado breve, la restricción `training_allowed=false`, el informe nuevo y el único prompt vigente.
3. Traslada el resultado de esta ejecución al informe y al plan. Cuando el sucesor esté creado y correctamente enlazado, elimina este prompt ejecutado; el historial permanece en el informe, las evidencias y Git.
4. Crea exactamente un prompt sucesor bajo `docs/prompts/`, limitado al siguiente blocker demostrado. Debe ser el único archivo que permanezca en esa carpeta. Enlázalo desde el plan y el README, pero no lo ejecutes.
5. No crees un backlog o plan paralelo. El plan de implementación sigue siendo la única fuente de progreso.

Si la implementación prevista queda incompleta, Gate 1B continúa pendiente y la única siguiente acción debe resolver el blocker demostrado. No abras una fase posterior.

El plan, README, informe y único prompt sucesor deben nombrar la misma siguiente acción.

## Prohibiciones

- No entrenes ni ajustes modelos.
- No descargues demos ni llames a FACEIT.
- No amplíes el corpus.
- No reproceses demos.
- No cambies Go, Node, releases, `CURRENT`, staging, Docker ni servicios.
- No modifiques, muevas, renombres o borres demos, bundles, manifests o cuarentena.
- No descomprimas `.gz` de forma persistente.
- No escribas dentro de `Faceit-Demos`.
- No cambies el formato 3.8.0 ni `golden-demos-v2`.
- No debilites controles de calidad ni fabriques datos ausentes.
- No permitas fugas de oracle o futuro.
- No generes labels prescriptivos.
- No hagas commit, push ni PR.
- No cambies configuración global de Git.
- No reviertas cambios del usuario.

## Criterios de finalización de esta fase intermedia

El paso 13 sólo se marca como completado cuando:

- los contratos parsean y validan;
- las pruebas positivas y negativas pasan;
- los 44 bundles se recorren de uno en uno;
- el corpus permanece intacto;
- no existen violaciones temporales sin explicar;
- las 17 capacidades devuelven un estado válido;
- la evidencia es determinista y válida;
- el informe coincide con la evidencia;
- plan, README e informe están actualizados y el prompt ejecutado se ha retirado;
- existe exactamente un prompt sucesor;
- `docs/prompts/` contiene exactamente ese único archivo vigente;
- todos los documentos coinciden sobre el siguiente paso;
- `training_allowed=false` permanece sin cambios.

## Informe final

Termina con un informe breve y sencillo que indique:

- resultado del paso 13;
- archivos creados o modificados;
- 44/44 recorridos o punto exacto del bloqueo;
- ejemplos estructurales obtenidos;
- elegibilidad por capacidad;
- fallos detectados;
- pruebas ejecutadas;
- confirmación de que el corpus quedó intacto;
- confirmación de `training_allowed=false`;
- única siguiente acción;
- ruta del único prompt sucesor.

No continúes automáticamente con la fase sucesora.
