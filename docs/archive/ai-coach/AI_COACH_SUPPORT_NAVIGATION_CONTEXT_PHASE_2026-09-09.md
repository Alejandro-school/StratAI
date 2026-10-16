# Contexto NAV de apoyo — cierre 3E.3

Fecha: 9 de septiembre de 2026. Implementación y validación funcional terminadas; conexión física real **no acreditada**. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y continuación. [Contrato](../../contracts/support/canonical-support-navigation-v1.md) y [evidencia con matriz, comandos e inventario](../../evidence/AI_COACH_SUPPORT_NAVIGATION_CONTEXT_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Resultado

Se conservan los **32.340 anclajes** de 3E.1 y todos los hechos de 3E.2. Un productor optativo Go y un materializador/lector Python añaden el diagnóstico de conexión actor → aliado sobre el NAV local, usando las dos posiciones exactas de cada T0. No cambian selección, historia, ID, fuente, vida, época, lado ni ventana posterior.

| Resultado en el grafo candidato | Todos | Intervalo canónico | Preliminares sin acreditación |
|---|---:|---:|---:|
| Camino dirigido | 31.949 | 30.391 | 1.558 |
| Sin camino en el subgrafo retenido | 239 | 201 | 38 |
| Fuera de cobertura de asociación | 152 | 86 | 66 |
| Total | 32.340 | 30.678 | 1.662 |

Hay **998 filas de matriz actor/aliado/instancia**. Los 32.340 estados físicos iniciales conservan `insufficient_information`, con reachability/distancia/ruta físicos `null`; cero conexiones físicas observadas, cero ausencias físicas acreditadas y cero decisiones elegibles. No se transforma una distancia recta, una rapidez o un camino candidato en apoyo viable.

El diagnóstico geométrico es nuevo y verificable. La dependencia física queda especificada con material de aceptación concreto; **3E.3 no resuelve la conexión física de la demo ni cierra S2–S3**. Comparación, entrenamiento y producto permanecen en `false`.

## Productor y semántica

`--ai-coach-support-physical-candidate` conserva la única pasada de demo. `cmd/support-navigation` lee sólo el NAV y consultas de extremos sin identidades/relojes; reutiliza `NavigationGraphSnapshot` y `EvaluatePath`. `ai_coach.support.navigation` conserva los informes anteriores y añade `support_navigation_observations@1` y `support_navigation_context@1`, ambos `1.0.0-candidate.1`. El registro tiene **66 schemas/19 razones**; catálogo `1.7.0`, parser `v16`, export `3.8.0`, quality `12`, manifest `@3`, `DecisionExample` y `DEC-TRADE-001` no cambian.

La asociación conserva el método candidato existente: contención XY con diferencia Z al centro hasta 128 unidades, seguida de fallback al centro 3D más cercano hasta 96. En cada rol hay **26.216** asociaciones por contención candidata, **6.048** por centro cercano y **76** fuera de cobertura. Hay **854** anclajes con ambos extremos en el mismo polígono candidato. Estos supuestos no prueban superficie física, altura interpolada, hull ni portal transitable.

Dijkstra opera sobre enlaces dirigidos y pesos euclídeos entre centros; suma los dos offsets de extremos. El recorrido por centros es un diagnóstico aproximado, no un camino físico mínimo o comprobado contra colisiones. En el mismo polígono conserva ambos segmentos al centro, sin sustituirlos por distancia directa. Las distancias candidatas observadas van de **34,258 a 8.036,953 unidades**. No se introduce velocidad nominal, ETA, ranking ni umbral de buen apoyo.

## Asset y límites comprobados

El mapa sale del manifiesto canónico verificado. El NAV local `de_mirage/de_mirage.nav` tiene **471.905 bytes**, SHA-256 `83b7f9abbedf193df580608dba1eb3b691098c5e8a74f23c3a0c548d54efcc0e`, versión **35**, subversión **1**. El hash coincide con el input real del parser y se comprueba antes/después de cargar. Esto acredita el archivo usado por nuestros productores, no su pertenencia al mapa exacto de la partida.

Una lectura independiente del binario verifica **3.613 vértices, 2.544 polígonos/áreas y 6.381 enlaces dirigidos**. Todos los polígonos y enlaces coinciden exactamente con el snapshot. Los centros se contrastan con tolerancia float64 por suma/división frente a suma/multiplicación por recíproco: la diferencia máxima es **4,55 × 10⁻¹³ unidades**, sin diferencia geométrica material.

La misma lectura conserva límites concretos: **7 referencias de escalera** omitidas por el constructor —3 superiores, 4 inferiores—; **92 áreas** con flags nativos `65536` y 2.452 con cero, sin asignar un significado no acreditado al bit; hull nativo cero en todas las áreas; ningún registro de mesh móvil en este asset. El lector consume un prefijo de **262.748 bytes** y deja **209.157 bytes** sin interpretar, con hash retenido. Los IDs de bordes destino también se descartan en el snapshot. No se inventan enlaces, offsets, alturas, geometría o reglas para cubrir estas ausencias.

Por tanto, `candidate_no_directed_path` sólo prueba que no hay camino en ese subgrafo, y `candidate_connected` sólo prueba la conexión y coste según su representación. No acreditan ausencia física, transitabilidad real, seguridad, visibilidad, contacto, llegada a tiempo, cooperación ni superioridad táctica.

Para reabrir la conexión física hacen falta el material original del mapa y su cadena verificable hasta esta demo —recurso/distribución, manifiesto de contenido, hashes y extracción del NAV—, la interpretación de enlaces/atributos usados y la asociación sobre superficies/portales/hull con cuantización y alturas justificadas. La ausencia física exige cobertura completa de conexiones relevantes. Otro inventario del mismo asset local no satisface esa dependencia.

## Verificación

Pasan `npm run check:all`, **775 pruebas AI Coach —32 nuevas—**, tres funciones nuevas de prueba Go, Ruff, `go vet` de los paquetes afectados y `go mod verify`. Se declara `numpy==2.4.4`, la versión ya instalada que se usó en todas las pruebas y recálculos; no se instala ni descarga otra versión. Las tres integraciones Node omitidas conservan su estado previo.

| Operación | Segundos | Resultado |
|---|---:|---|
| Parser/exportador de la única demo | 71.462 | Correcto |
| Materialización 3E.1–3E.3 y recálculo independiente NAV | 418.195 | Schemas, fuentes y todos los registros correctos |
| Relectura, reconstrucción Go y validación Python completa | 465.031 | Coincidencia completa |
| Canónico completo | 256.637 | Cero errores |
| Control independiente de extremos, vidas, matriz y fases anteriores | 13.220 | Todos los joins y conteos coinciden |
| Recálculo físico 3E.2 independiente | 10.397 | 64.680 movimientos y campos físicos conservados |

El validador Python recalcula pesos, asociaciones, distancias mínimas y dirección de cada segmento; la relectura vuelve a producir el grafo y rutas desde el archivo NAV y el ejecutable fijados por hash. Se recomputan las 998 filas de matriz y se contrastan todos los extremos contra su muestra exacta, con fuente/vidas/lado/época/tick/FrameDone. El acceso inicial real permanece idéntico al cambiar enemigos, muertes y resultados futuros, y excluye identidades, diagnóstico candidato y acreditación competitiva.

El smoke observa las tres variantes candidatas, ambos roles fuera de cobertura por separado y casos de mismo polígono. Ambos extremos fuera simultáneamente, falta de archivo, formato/hash/mapa incorrectos, muestra ausente y mutaciones de futuro/identidad/época/lado son fixtures o controles de mutación; no aumentan la cobertura observada.

## Conservación y continuación

Una única demo aprobada: `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Una pasada del parser, sin reintento de la partida; build `working-tree-support-nav-3e3-20260909-r1`. La lectura de NAV y la reconstrucción de los informes no reparsean la demo.

Las observaciones v1 y físicas son idénticas al control 3E.2; anclajes, hechos, matriz, resumen y 1.662 anclajes preliminares conservan su estado. Los **72 artefactos canónicos** mantienen exactamente sus hashes y conteos. El inventario protegido conserva **4.356 archivos/181 hashes**, incluyendo 44 bundles y 50 fuentes; catálogo, interfaz temporal y evidencias anteriores permanecen intactos. La evidencia final registra las comprobaciones documentales y de fuentes de la build.

Se retienen **79 archivos / 1.365.465.380 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3e3-smoke-fzl1m96f`, con inventario y hashes. Scripts, binarios, logs y controles permanecen en el TEMP de trabajo indicado en la evidencia. Cero intentos de borrado y cero solicitudes de revisión de limpieza. No se han descargado demos ni assets, reprocesado el corpus, promovido datos, modificado `CURRENT`, entrenado modelos o emitido recomendaciones.

La siguiente pieza independiente, **3E.4**, es el contexto causal del objetivo en los mismos anclajes, a partir de fuentes nativas legales y presentes. Objetivo, entrada/compromiso previo, LOS/contacto, cooperación, intención y readiness completa siguen sin acreditarse conjuntamente para S2–S3. NAV físico y economía conservan sus dependencias positivas separadas; el siguiente prompt no las reabre por repetir proxies.
