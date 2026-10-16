# Contexto físico de apoyo — cierre 3E.2

Fecha: 9 de septiembre de 2026. Implementación y validación funcional terminadas; TEMP retenido por instrucción del usuario. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y continuación. [Contrato](../../contracts/support/canonical-support-physical-v1.md) y [evidencia con matriz, comandos y hashes](../../evidence/AI_COACH_SUPPORT_PHYSICAL_CONTEXT_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Qué hemos conseguido

Los **32.340 anclajes factuales** de 3E.1 conservan selección, historia, IDs, T0 y ventana posterior. Un contexto separado añade orientación, walking, ducking, arma activa, munición y recarga propias/aliadas en ese mismo corte, más velocidad media calculada desde el prefijo de posiciones. No se ha seleccionado por muerte, resultado ni información enemiga.

**31.862 anclajes** tienen los siete componentes observados para ambos participantes o explícitamente no aplicables; **478** conservan alguna ausencia. Dentro de intervalos competitivos canónicos, **30.264 de 30.678** tienen esos componentes. La instancia preliminar mantiene sus **1.662** anclajes sin acreditación competitiva. Los conteos describen cobertura factual, no readiness completa, oportunidades tácticas independientes o apoyo viable.

Se implementaron productor optativo, dos schemas, materializador, acceso inicial, matriz y validador. Modo `--ai-coach-support-physical-candidate`, contratos `1.0.0-candidate.1`, **64 schemas runtime/19 razones**. Catálogo `1.7.0`, parser `v16`, export `3.8.0`, quality `12`, manifest `@3`, `DecisionExample` y significado de `DEC-TRADE-001` se conservan.

## Captura y límites de las fuentes

El grafo permitió localizar y trazar replay, movimiento, arma y tracker de recarga. Se contrastaron cinco fuentes locales del SDK fijado `demoinfocs v5.2.0`; `go mod verify` pasó. La cobertura del grafo marca metadatos cambiados, sin problemas registrados en las rutas consultadas: se conservan lectura directa y hashes de la build como referencia, sin afirmar exhaustividad del índice.

La captura física lee inmediatamente dentro del callback support/replay y referencia su `source_sequence`. No avanza el parser ni recupera otra muestra. La orientación exige vector nativo completo, finito y de tipo correcto; walking exige bool y ducking flags uint64. Se comprueba handle de arma completo, serie y vida del objeto SDK; un objeto destruido no se reutiliza y la repetición del handle entre objetos invalida la recarga heredada.

La munición se contrasta con propiedades presentes: se retienen cargador codificado y reserva, con su fuente. El SDK fijado calcula cargador como uint32 codificado menos uno. **336 valores codificados cero** se excluyen: el wrap unsigned del accessor SDK no acredita miles de millones de balas. Codificado uno sí conserva cero balas. Los lectores históricos no se reinterpretan ni quedan revalidados por este nuevo filtro. Knife/C4/utilidad tienen `not_applicable`, no munición cero; una clase desconocida mantiene ausencia.

La recarga necesita el tracker causal inicializado y una observación actual para la identidad exacta. Se conserva el tick original de señal, separado del tick refrescado. La antigüedad del seed real va de **0 a 5.867 ticks**, con **1.025** seeds en el mismo tick de la muestra; no significa que la acción de recargar dure esa cantidad. Es el origen del tracker que se revalida, no una traza nueva de cada transición física ni orden subtick demostrado.

El movimiento se calcula como `(posición_actual − posición_anterior) / duración`, con XYZ, extremos temporales y norma horizontal. X/Y mantienen la cuantización del replay. Se recalcularon **64.680** registros de movimiento y **12.144** intervalos horizontales estacionarios. Es velocidad media del intervalo que termina en T0, no velocidad instantánea, tecla, aceleración, recorrido por curva ni viabilidad futura.

## Cobertura y abstenciones reales

La fuente tiene **30.716** muestras, **213.897** contextos físicos presentes y **92.843** contextos ausentes para actores no vivos. Se observan **935 vidas de objeto arma**. La matriz tiene **998** filas actor/aliado/instancia con anclajes; conserva los denominadores y motivos de cada campo.

| Componente disponible para ambos | Anclajes |
|---|---:|
| Orientación | 32.340 |
| Walking | 32.340 |
| Ducking | 32.340 |
| Arma activa | 32.280 |
| Cargador observado o no aplicable | 32.216 |
| Reserva observada o no aplicable | 32.280 |
| Recarga observada o no aplicable | 31.926 |
| Todos los componentes | 31.862 |

Por rol, de 32.340 filas: 30 carecen de identidad de arma; el cargador tiene otros 32 casos excluidos por codificado cero; la recarga tiene otros 178 sin señal válida. Munición/recarga no aplican en **12.218 filas por rol**. Ambos sentidos del par son correlacionados; no se suman como decisiones independientes.

En toda la fuente, las ausencias de arma son **541**, las de cargador codificado cero **336**, y las de señal de recarga **1.608**. Hay 77.757 estados de arma sin munición de cargador. Los bools reales incluyen walking 47.007 `true` / 166.890 `false`, ducking 18.339 / 195.558 y recarga 1.007 / 132.984; cero/falso se conservan cuando están acreditados.

No aparecen ausencias o tipos inválidos de orientación/postura en esta demo; tampoco señales futuras, handles reutilizados con recarga heredada o contextos seleccionados sin muestra física exacta. Esos casos están cubiertos por fixtures y no aumentan la cobertura real. Los cambios de vida/lado/época, discontinuidades y deduplicación de anclajes conservan los controles v1.

## Validación

Pasan `npm run check:all`, **743 pruebas AI Coach —37 nuevas—**, cinco funciones nuevas de prueba Go, Ruff y `go mod verify`. La suite pública incluye 43 pruebas Python del backend, 88 del frontend y 14 Node; tres integraciones Node permanecen omitidas, identificadas en la evidencia.

Una demo aprobada, una pasada, sin reintento: `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`; build `working-tree-support-3e2-20260909-r1`.

| Operación | Segundos | Resultado |
|---|---:|---|
| Parser/exportador | 72,382 | Correcto |
| Materializador físico y v1 | 394,378 | Schemas, fuentes y registros no vacíos correctos |
| Relectura y reconstrucción completa | 359,02 | Coincidencia completa |
| Recálculo independiente, sin importar implementación | 16,128 | Todos los hechos, movimientos, codificados, matriz y resúmenes coinciden |
| Canónico completo | 343,267 | Cero errores |

El cálculo independiente contrasta **203.030** estados de orientación, walking, ducking y arma activa con replay, cada componente por separado. Los ángulos se comparan a precisión float32: las diferencias de representación decimal no se convierten en un desacuerdo factual. Las observaciones v1 y sus anclajes/matriz/resumen son idénticos al control 3E.1. Los **72 artefactos canónicos** conservan exactamente sus hashes y conteos.

El acceso inicial del primer anclaje real permanece idéntico al modificar enemigos, muerte, daño y resultado posteriores. No devuelve alias, handles, vidas de arma, seed de recarga, etiqueta competitiva ni outcomes. Son mutaciones de control, no variantes observadas. La información aliada es un proxy de equipo según la demo, no percepción, intención ni cooperación demostradas.

## Conservación y continuación

Los 44 bundles, 50 fuentes y demás datos protegidos conservan **4.356 archivos / 181 hashes de control**. Las **31 evidencias previas** y el catálogo mantienen sus hashes. **371 archivos funcionales** coinciden con la build validada. El cierre documental verifica enlaces locales, un único prompt, alcance sobre el worktree previo y `git diff --check`; cifras finales en la evidencia.

Se conservan **77 archivos / 1.136.710.346 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3e2-smoke-upd4ly43`, con inventario y hashes. Scripts/logs/resultados auxiliares permanecen en el temporal de trabajo. Cero intentos de borrado de TEMP y cero solicitudes de revisión de limpieza. Fuentes, corpus, release, `CURRENT` y temporales anteriores permanecen de sólo lectura; no hubo descargas, promoción, entrenamiento ni migración 44/44.

Todos los anclajes siguen sin ruta NAV, LOS/contacto, objetivo, entrada/compromiso previo, cooperación, intención, velocidad instantánea ni readiness completa acreditados. Los tres gates permanecen en `false`; cero decisiones elegibles o recomendaciones.

El déficit seleccionado para **3E.3** es la conexión NAV actor → aliado en los mismos anclajes, usando fuentes locales con procedencia y cobertura verificables. La distancia recta y el movimiento medio no prueban un camino transitable. Esa continuación tampoco cierra el momento previo de decisión S2–S3 ni sustituye alternativas viables y revisión humana. Economía mantiene sus dependencias positivas de transacción y assets; perfil, misiones y el ciclo completo de coaching siguen pendientes.
