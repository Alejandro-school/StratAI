# Preparación anterior a la ejecución — cierre documental 3E.11

12 de septiembre de 2026. **Diseño C0 congelado y ensayado; entrada física y compromiso táctico todavía no acreditados.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente acción. [Protocolo](../../ai-coach/AI_COACH_SUPPORT_PRE_ENTRY_PROTOCOL.md), [casos del diseño](../../references/support-preparation-selection-cases-v1.json) y [evidencia](../../evidence/AI_COACH_SUPPORT_PRE_ENTRY_PROTOCOL_DESIGN_2026-09-12.json).

## Decisión

Se define `support_preparation_census/0.1.0-design.1`: un censo prospectivo de pares propios/aliados utilizables en cada corte support actual. Incluye permanencia, movimiento ya iniciado y cambios posteriores; no selecciona por aumento de distancia, muerte, trayectoria futura o posición de un enemigo. T0 es el corte actual, anterior a la ventana de ejecución que se estudiará. No se afirma que sea anterior al comienzo físico de un movimiento que podría estar en curso.

El protocolo 3E.1 se conserva: su T0 ya contiene un aumento observado de separación. Elegir `T0-1 muestra` manteniendo ese disparador introduce dependencia del futuro. El ensayo reproduce dos continuaciones con el mismo prefijo —aliado que avanza y aliado que no avanza—: el retroceso selecciona distinto, mientras C0 conserva el mismo corte y estado inicial.

C0 es una población de preparación candidata, con unidad, claves, exclusiones y denominadores definidos. No es un detector de entradas ni un reconocimiento de decisiones mentales. Se especifican por separado cambio de posición muestreada, inicio físico del movimiento, entrada a una región y compromiso táctico. Para una entrada específica faltan una frontera orientada definida antes de seleccionar y fuentes espaciales/temporales que acrediten el instante y la continuidad. La revisión humana tampoco sustituye esas pruebas físicas.

## Ensayo sobre evidencia retenida

No se parseó ninguna demo ni se cambió el runtime. Se leyó el `support_observations.json` retenido por 3E.10, SHA-256 `c3526c6647483605f4d0f29b4048eca3f8bed90338e460804bda6e345ce59ac3`, con build y referencias originales. Pertenece a `demo-a11eb0d600ee702c9edb`, SHA-256 de demo `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. La validación funcional de esa fuente permanece en la evidencia 3E.10; esta fase comprueba un diseño documental sobre sus bytes.

Dos implementaciones de selección separadas, sin importar derivaciones de la aplicación, coinciden en todos los cortes. La versión final del ensayo completo tardó **22,487 s**.

| Denominador | Resultado |
|---|---:|
| Cortes retenidos | 30.716 |
| Cortes admisibles por fase/reloj de captura | 27.948 |
| Cortes `postround` excluidos / reloj inválido | 2.768 / 0 |
| Filas de roster observadas | 306.740 |
| Aliases actor/corte en cortes admisibles | 279.437 |
| Actor/cortes utilizables / no vivos | 205.324 / 74.113 |
| Pares del mismo lado examinados, incluidos no utilizables | 1.117.576 |
| Pares excluidos por actor / por aliado | 296.452 / 160.438 |
| Cortes/par seleccionados por C0 | **660.686** |
| Actor/cortes con al menos un aliado utilizable | 199.454 |
| Filas de la matriz actor/aliado/instancia/época | 1.040 |
| Anclajes 3E.1 localizados dentro de C0 | **32.340 de 32.340** |

Los 660.686 candidatos coinciden con los pares utilizables examinados por el selector antiguo antes de exigir su patrón de tres muestras. El nuevo denominador bruto incluye además exclusiones de disponibilidad; no se compara con el antiguo `pairs_examined` como si fueran la misma población. Se conservan motivos con cero y ecuaciones de conciliación.

No se exige roster completo ni se promueven instancias preliminares a rondas competitivas. La pertenencia canónica posterior no elige T0; este ensayo no produce una clasificación competitiva nueva para C0. Pares inversos, múltiples aliados y ventanas superpuestas están correlacionados. Estos conteos no equivalen a decisiones independientes ni a tamaño efectivo de una muestra de aprendizaje.

## Ejecución posterior y ausencia de avance

La ventana de investigación es `(T0, T0+2 segundos]` en ticks, con horizonte exacto y cadencia/identidad/lado/fase continuos. Por rol, sobre los mismos 660.686 candidatos:

| Descripción posterior | Ventanas |
|---|---:|
| Algún cambio entre posiciones muestreadas | 597.847 |
| Sin cambio entre posiciones muestreadas | **40.390** |
| Censura por muestra no viva | 14.774 |
| Censura por hueco/frontera | 6.619 |
| Identidad/lado no disponibles o cambiados | 1.056 |
| Otros motivos de censura presentes sólo en fixtures | 0 |

En **4.726 pares/ventanas** ninguno de los dos participantes muestra cambio entre posiciones exportadas. Eso no demuestra inmovilidad continua, ausencia de entrada entre muestras, espera, detención deliberada ni que mantener fuera correcto. La cuantización X/Y y la precisión Z se conservan sin introducir un umbral táctico para ocultar cambios pequeños.

El conjunto de anclajes anteriores, incluido íntegramente en C0, conserva el control ya acreditado por 3E.1 de **29.162 ventanas sin muerte registrada del par** y 26.346 sin ninguna kill, bajo sus condiciones originales de cobertura canónica. Es una referencia transferida por conservación de las claves anteriores; **no es un nuevo conteo de kills para toda la población C0**. No se consultan muertes para seleccionar el censo.

## Casos y causalidad

Pasan **31 casos sintéticos**: primera muestra sin historia, cero nativo, permanencia, avance posterior, movimiento ya iniciado, separación grande, ausencias/no finitos/tipos inválidos, muerte o desconexión actual, identidad compartida/duplicada, ausencia de aliado, fase/reloj inválidos, vida/época nuevas, mismo tick con otro FrameDone y censuras posteriores por vida/lado, desconexión, muerte muestreada, frontera, hueco y horizonte ausente. Los contextos sintéticos de freeze o función de mantener no cambian la selección ni se convierten en intención observada.

Cinco prefijos reales —1, 477, 1.000, 15.358 y 30.716 muestras— conservan exactamente las claves seleccionadas, incluido un prefijo que termina en `postround`. Cambiar escalares geométricos de enemigos, reordenar el roster e insertar futuro malformado fuera del prefijo no altera la selección legal comprobada. Los controles iniciales propios/aliados permanecen iguales y sus copias no modifican la fuente.

Los descriptores posteriores tienen pruebas de casos y controles reales; la igualdad de los dos selectores acredita selección, no constituye una segunda implementación independiente de toda la clasificación posterior. Esa recomputación runtime queda para la implementación. Ningún ensayo acredita una entrada física, un compromiso, una comparación, una recomendación o aprendizaje.

El auditor tuvo tres revisiones documentales, todas retenidas: se hicieron explícitos los ceros de cobertura, se amplió el control de prefijos a fronteras y se precisó que identidad ausente no equivale a identidad cambiada. No hubo cambio de productor, consumidor de aplicación ni reproceso de demo.

## Conservación y continuación

Se conservan el código funcional, los 82 schemas/19 razones, catálogo `1.7.0`, protocolo 3E.1, `DecisionExample`, `DEC-TRADE-001`, producto e interfaz temporal. El censo no se ha registrado como contrato runtime. Los 170 archivos de salida/trabajo 3E.10 y las evidencias anteriores conservan sus hashes; el inventario protegido de 44 bundles/50 demos mantiene sus 4.356 archivos y 181 hashes. El alcance y método de estas comprobaciones, comandos e inventario del nuevo trabajo temporal se retienen en la evidencia.

La siguiente intervención es **3E.12: implementar el censo prospectivo de preparación C0** con contratos, materializador/lector, acceso por prefijo y ejecución posterior separados. Se aprovechará la captura existente cuando sea suficiente. El protocolo de entrada física específico sigue pendiente de su material positivo, al igual que contacto legal, NAV físico, cinemática instantánea, C4 transportada/caída, temporización completa, cooperación/intención y economía. S2–S3 no quedan elegibles por este diseño.
