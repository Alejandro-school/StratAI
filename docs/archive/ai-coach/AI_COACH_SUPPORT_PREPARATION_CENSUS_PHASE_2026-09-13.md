# Censo prospectivo de preparación C0 — cierre 3E.12

13 de septiembre de 2026. **Censo C0 materializado y validado; cero decisiones elegibles.** [Plan](../../AI_COACH_IMPLEMENTATION_PLAN.md), [contrato](../../contracts/support/canonical-support-preparation-v1.md), [diseño 3E.11](../../ai-coach/AI_COACH_SUPPORT_PRE_ENTRY_PROTOCOL.md) y [evidencia](../../evidence/AI_COACH_SUPPORT_PREPARATION_CENSUS_SINGLE_DEMO_SMOKE_2026-09-13.json).

## Entrega

`ai_coach.support.preparation` reutiliza la captura support existente, sin cambios Go. Sus cuatro schemas estrictos elevan el registro a **86 schemas/19 razones**. Protocolo, cortes y pares JSONL y manifest conservan claves, disponibilidad, exclusiones, matriz, hashes y conteos. El lector comprueba todos los registros y recompone el censo completo.

C0 selecciona todos los pares actuales propios/aliados utilizables. No exige historia, proximidad, movimiento futuro o muerte. El acceso inicial corta por ordinal antes de validar, exige el par/T0 exactos y devuelve copias sólo de posición, estado vivo y disponibilidad. Identidad, procedencia, enemigos y descriptores posteriores permanecen separados. Los contratos de captura siguen siendo estrictos: un fixture que el selector excluye por tipo inválido no autoriza materializar una fuente inválida.

## Pruebas y demo real

Pasan `npm run check:all`, **1202 pruebas AI Coach —69 nuevas—**, Ruff y la validación contractual. Los 31 casos del diseño están contrastados con sus expectativas documentales; las pruebas adicionales cubren claves, primer corte, prefijos, copias, enemigos, futuro malformado, denominadores, handles inválidos, primera censura, sobrescrituras y rechazo de filas alteradas, duplicadas, ausentes o reordenadas.

Después de las suites se procesó una sola vez `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, con build `working-tree-support-preparation-3e12-20260913-r1`, hacia un TEMP nuevo y vacío. Parser **111.966 s**, materializador C0 **849.209 s**, lector **862.538 s**, validación canónica **474.361 s**. Comandos exactos y binding privado de fuente quedan inventariados en la evidencia. No hubo otra demo, reintento funcional, descarga ni adquisición.

| Denominador | Conteo |
|---|---:|
| Cortes / admisibles / excluidos por fase | 30.716 / 27.948 / 2.768 |
| Cortes con reloj inválido | 0 |
| Filas de roster | 306.740 |
| Aliases actor/corte admisibles | 279.437 |
| Actor/cortes utilizables / no vivos | 205.324 / 74.113 |
| Pares examinados | 1.117.576 |
| Excluidos por actor / por aliado | 296.452 / 160.438 |
| Pares seleccionados | **660.686** |
| Actor/cortes con aliado utilizable | **199.454** |
| Filas de matriz | 1.040 |
| Anclajes anteriores incluidos | **32.340 de 32.340** |

Por rol: **597.847** ventanas con cambio muestreado, **40.390** sin cambio y **22.449** censuradas: 14.774 por estado no vivo, 6.619 por hueco/frontera y 1.056 por identidad/lado no disponibles o cambiados. Los demás motivos conservan cero en la demo y cobertura sintética cuando corresponde. Hay **4.726** pares/ventanas sin cambio muestreado en ninguno de los roles.

## Recálculo y conservación

Una vía independiente, sin importar la derivación de aplicación, recorre todos los cortes y pares. Usa el selector pareado congelado en 3E.11 y otra implementación de ventanas que reúne fallos y reduce por primer índice/precedencia. Coinciden **3.964.116 componentes XYZ**, **1.321.372 descriptores posteriores, incluidas sus censuras**, IDs, referencias, exclusiones, conteos y matriz. El auditor tarda **86.308 s**. Cinco prefijos conservan las huellas de selección 3E.11; los controles reales de features en secuencias 1, 1134 y 30614 pasan, con mutaciones enemigas/futuro en los dos primeros y aislamiento de copias.

Los **72 artefactos canónicos** conservan hashes y conteos. La captura support recién producida conserva las observaciones 3E.10; sólo cambian build y manifest de procedencia. Los anclajes regenerados conservan filas, IDs/T0, selección, matriz y resumen. Las capas 3E.2–3E.10 no se modificaron ni se volvieron a materializar: sus fuentes/código retenidos conservan sus hashes y sus pruebas forman parte de la suite completa.

Se retienen **79 archivos/1757279216 bytes** en la nueva salida, iguales antes y después del lector. Los 202 archivos temporales de 3E.10/3E.11, evidencias anteriores y corpus de 44 bundles/50 demos conservan sus comprobaciones. El inventario protegido compara tamaños/mtime de 4.356 archivos y 181 hashes seleccionados; no equivale a rehash completo de todo el corpus. Inventarios, fuentes de build y método exacto quedan en la evidencia, sin limpieza, promoción o migración.

## Límites y siguiente acción

T0 precede a una ventana futura de dos segundos, pero puede caer durante un movimiento ya iniciado. Sin cambio de coordenadas exportadas no acredita inmovilidad continua, espera deliberada, ausencia de entrada entre muestras ni mantener correctamente. Pares inversos, aliados múltiples y ventanas superpuestas están correlacionados y no son decisiones independientes. Una muestra no viva no identifica por sí sola un evento de muerte; no se ha producido un nuevo conteo de kills para toda la población C0.

`DecisionExample`, catálogo `1.7.0`, `DEC-TRADE-001`, producto e interfaz temporal permanecen intactos. Entrada física, compromiso, contacto legal, NAV físico, cinemática instantánea, C4 transportada/caída, temporización completa y cooperación conservan sus dependencias. No se habilitan comparaciones, recomendaciones o entrenamiento ni se cierra S2–S3.

Siguiente **3E.13: contexto físico básico propio/aliado en el T0 exacto de C0**. Reutilizar la captura 3E.2 sin cambiar la población ni arrastrar un anclaje cercano; la primera muestra sin historia debe conservarse con la ausencia correspondiente. El contexto físico básico tampoco acreditará por sí solo entrada previa o apoyo viable.
