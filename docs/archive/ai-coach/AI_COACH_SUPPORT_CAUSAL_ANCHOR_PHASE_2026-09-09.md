# Anclaje factual de apoyo — cierre 3E.1

Fecha: 9 de septiembre de 2026. Implementación y validación funcional terminadas; TEMP retenido por instrucción del usuario. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente acción. [Contrato candidato](../../contracts/support/canonical-support-anchors-v1.md) y [evidencia con comandos, matriz y hashes](../../evidence/AI_COACH_SUPPORT_CAUSAL_ANCHOR_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Resultado y significado

Se pueden localizar **inicios observados de aumento de separación entre actor y aliado**, sin esperar a una muerte ni consultar el desenlace. El productor, protocolo, materializador, lector inicial y validador son optativos. Cada registro conserva tres muestras propias/aliadas, distancia al cuadrado, identidad local, tick/fotograma, disponibilidad y procedencia.

**T0 es posterior al aumento ya observado.** No demuestra que estemos antes de una entrada física ni que exista una oportunidad táctica de apoyar. Es una pieza de investigación para S2–S3, no un nuevo `DecisionExample`, una recomendación ni una revisión humana. `DEC-TRADE-001` conserva su comienzo en la muerte observada. Catálogo `1.7.0`, base `3.8.0`, parser `v16`, quality `12` y manifest `@3` permanecen iguales; tres schemas candidatos llevan el registro runtime a **62 schemas y 19 razones**.

## Fuentes y criterio de selección

El grafo permitió descubrir y trazar los handlers de replay y parser; se contrastaron las fuentes locales del SDK `demoinfocs v5.2.0`. La cobertura final no registra problemas de parseo en las rutas consultadas, pero marca metadatos cambiados: se conservan lectura directa y hashes de **362 archivos funcionales** como referencia de la build. La cobertura del grafo es una señal parcial.

La captura ocurre en el mismo callback que añade el replay, sin avanzar el parser. Relee conexión y referencia de pawn, comprueba índice y serie completos y asigna vida al objeto SDK, con invalidación por destrucción. Cuenta todos los `FrameDone` y abre época ante retroceso de tick. No necesita el orden entre consumidores económicos. Una vida de entidad no equivale automáticamente a una vida física del jugador; se exigen además instancia, lado y continuidad viva muestreada, sin afirmar qué ocurrió entre muestras.

Se verifica presencia de los seis componentes nativos de posición y de salud/estado de vida. El vector cero que el SDK devuelve cuando faltan campos no se convierte en posición observada. Cero numérico y `false` presentes conservan su valor; ausencia, identidad no resuelta y valores no finitos mantienen estado explícito.

El protocolo selecciona `d²[1] <= d²[0]` y `d²[2] > d²[1]`, con tres muestras contiguas de la misma instancia/época/vidas/lado, ambos conectados y vivos, en `playing` y a cadencia exacta. X/Y están redondeadas como en replay. No hay distancia considerada tácticamente correcta: la cuantización o un cambio pequeño pueden producir un anclaje. Un aumento continuo no se repite hasta otro paso no creciente. Ambos sentidos del par se conservan como roles distintos y correlacionados.

El acceso inicial exige actor, ID, tick y fotograma exactos; valida sólo el prefijo y devuelve copias de hechos propios y del aliado. Enemigos, identidades como features y outcomes quedan fuera. La posición aliada es un proxy de información del equipo en la demo, no prueba de percepción o conocimiento exacto del jugador.

## Cobertura real de una sola partida

Demo `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Una pasada, sin reintentos, build `working-tree-support-3e1-20260909-r1`.

| Medida | Conteo |
|---|---:|
| Muestras fuente / en `playing` | 30.716 / 27.948 |
| Estados de jugador con identidad/posición/estado vivo presentes | 306.740 |
| Instancias observadas / vidas de objeto pawn | 25 / 11 |
| Filas de matriz actor/instancia | 250 |
| Pares dirigidos examinados | 660.686 |
| Anclajes de inicio de aumento | 32.340 |
| Estados contrastados con replay canónico | 294.825 |
| Muestras fuente unidas al replay / sin unión | 29.520 / 1.196 |

Las 1.196 muestras sin unión y sus **1.662 anclajes** son de la instancia preliminar 1. Conservan procedencia de instancia observada; no se les atribuye ronda competitiva ni se usa una etiqueta oficial futura para seleccionarlos. Las otras 30.678 filas de anclaje tienen tick de replay, lo que tampoco acredita una decisión táctica.

| Motivo | Conteo | Unidad |
|---|---:|---|
| Actor no disponible | 74.113 | Muestra de actor, principalmente estado no vivo |
| Ningún aliado vivo disponible | 5.870 | Muestra de actor |
| Historia insuficiente | 64 | Par examinado |
| Ventana de muestras incompatible | 2.000 | Par examinado |
| Historia del par no disponible | 156 | Par examinado |
| Cambio de vida o lado | 80 | Par examinado |
| Sin comienzo de aumento | 626.046 | Par examinado |
| Comienzo observado | 32.340 | Par examinado |

Los dos primeros motivos tienen un denominador distinto de los otros seis. La evidencia conserva los ceros y la matriz completa, no sólo el agregado. No se presentan 32.340 filas de una partida como 32.340 observaciones independientes.

## Ventana posterior y casos sin muerte

La ventana separada termina exactamente dos segundos después de T0: **32.024** completas, **264** excluidas por discontinuidad/frontera y **52** por cambio/ausencia del par. Cero exclusiones de `missing_exact_horizon_sample` en esta partida. Hay **30.606** ventanas completas sin una muestra del par en estado no vivo; eso no se transforma en una kill o su ausencia.

Después de reconstruir todos los anclajes, un lector independiente consultó eventos canónicos dentro de ventanas competitivas completas:

- **29.162** anclajes sin muerte registrada de actor o aliado; de ellos, **26.346** sin ninguna kill en la ventana.
- **1.294** con muerte registrada del par.
- **1.568** con ventana canónica no acreditada y **316** con horizonte de muestras incompleto; se conservan excluidos.

El resultado posterior no selecciona ni desplaza T0. La ausencia de una muerte registrada en dos segundos no demuestra seguridad, buena decisión ni cooperación.

## Pruebas, controles y retención

Pasan `npm run check:all`, **706 pruebas del AI Coach —47 nuevas—**, cinco funciones nuevas de prueba Go, Ruff y `go mod verify`. La suite pública incluye 43 pruebas Python del backend, 88 del frontend y 14 Node; tres pruebas Node de integración permanecen omitidas y se identifican en la evidencia. No se afirma que esas tres hayan pasado.

| Operación | Segundos | Resultado |
|---|---:|---|
| Productor Go, una demo | 80,226 | Correcto |
| Materializador candidato | 144,484 | Schema, hashes, replay y registros no vacíos correctos |
| Relectura y reconstrucción del informe | 116,211 | Coincidencia completa |
| Recálculo independiente, sin importar implementación | 18,991 | Todos los pares, historias, distancias, matriz y horizontes coinciden |
| Validación semántica canónica completa | 436,607 | Cero errores |

Los controles de prefijo sobre 1.000 y 15.358 muestras conservan exactamente **1.406** y **16.490** anclajes. Alterar enemigos y datos posteriores del primer anclaje real no cambia sus hechos iniciales. Son mutaciones de control, no nuevos eventos observados. Las pruebas cubren desconexión, ausencia nativa, serie completa, reutilización/destrucción, cambio de lado/vida/época, tick repetido, ventana exacta ausente, límites, deduplicación, hashes y manipulación del informe. En la demo se observan 24 cambios de instancia y 92.843 estados no vivos; hay **cero** desconexiones, duplicados de actor/pawn, saltos de época y muestras consecutivas del mismo tick. Todas sus propiedades de identidad/posición/estado vivo están presentes; las ausencias correspondientes son fixtures.

Los **72 artefactos canónicos base** conservan exactamente sus hashes y conteos frente al control 3D.8; los sidecars económicos no se vuelven a producir en este modo. Los 44 bundles, 50 fuentes, release y exports protegidos mantienen el inventario de **4.356 archivos y 181 hashes de control**. Las 30 evidencias previas y el catálogo siguen intactos.

Se conservan **75 archivos / 881.206.999 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3e1-smoke-_exvo0fm`, con inventario y SHA-256. Scripts, logs y resultados auxiliares permanecen en el temporal de trabajo enlazado en la evidencia. **Cero intentos de borrado y cero solicitudes de revisión de limpieza.** No se modificaron temporales anteriores, corpus, release ni `CURRENT`; no hubo adquisición, promoción, entrenamiento o migración 44/44.

## Qué falta

Cada uno de los 32.340 anclajes declara ausentes entrada física, compromiso táctico, ruta, LOS, contacto legal acreditado, objetivo, cooperación, intención, orientación y preparación del arma. Los tres gates permanecen en `false`; cero entradas físicas o decisiones elegibles quedan acreditadas.

La continuación seleccionada en el plan, **3E.2**, comprobará movimiento, orientación y arma propios/aliados exactamente en los mismos anclajes, reutilizando señales acreditadas y capturando sólo lo imprescindible. Aun con esos datos, S2–S3 necesitarán acreditar el momento previo y el contexto de la decisión, alternativas viables y revisión humana. Economía sigue parcial por sus dependencias de transacción y assets; perfil, misiones y el ciclo completo de coaching continúan pendientes.
