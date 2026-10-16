# Cierre 3E.8: movimiento y suelo nativos propios y aliados

10 de septiembre de 2026. Extensión factual implementada y validada con una demo real. El [contrato](../../contracts/support/canonical-support-native-motion-v1.md) fija los campos; la [evidencia](../../evidence/AI_COACH_SUPPORT_NATIVE_MOTION_SINGLE_DEMO_SMOKE_2026-09-10.json) conserva comandos, resultados e inventarios. El [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene el orden de trabajo.

## Resultado

La capa acredita contacto con suelo, tipo de referencia y código nativo de movimiento en **32340 anclajes por rol**. La demo no declara `m_vecVelocity` ni `m_vecAbsVelocity` en el pawn observado y ambos permanecen ausentes en todos los estados capturados. **No hay velocidad instantánea real acreditada ni magnitudes derivadas de ella.** El consumidor conserva esa abstención; `m_vecBaseVelocity`, velocidades históricas de salto/aterrizaje y media entre posiciones no la sustituyen.

Se añaden productor Go, dos schemas `1.0.0-candidate.1`, consumidor Python, acceso inicial causal, materializador y lector con recálculo completo. Selector `--ai-coach-support-native-motion-candidate`. Runtime **78 schemas/19 razones**; parser **v16**, export **3.8.0**, quality **12** y catálogo **1.7.0** conservados. Cero decisiones elegibles, recomendaciones, comparaciones, entrenamiento o soporte de producto.

## Cobertura real

| Unidad | Resultado |
|---|---|
| Entregas / estados / cortes | 91998 / 19888 / 30716 |
| Vidas nativas | 12 creadas, 11 capturadas con pawn utilizable; dos destruidas dentro del prefijo auditado |
| Instancias | 24 cambios de dominio en los cortes; estados retenidos sin cruzar instancia/época |
| Anclajes / matriz | 32340 / 998; 30678 competitivos y 1662 preliminares conservados |
| Contacto por rol | 30357 positivos y 1983 negativos corroborados |
| Referencia de suelo por rol | 30206 mundo, 151 otra entidad y 1983 ninguna |
| Código nativo por rol | 32246 valores `2` y 94 valores `9`; sin etiquetas importadas de otro motor |
| Velocidad instantánea y dos magnitudes por rol | 32340 abstenciones por fuente actual ausente en cada hecho |

Los conteos por rol son registros correlacionados de una sola partida. Los estados son snapshots compartidos: **3409** con contacto y **16479** sin contacto. Las referencias resueltas corresponden a **3386** estados `CWorld` y **23** estados `CCSPlayerPawn`; el contacto con otra entidad no acredita una superficie estática o una ruta. Los códigos de movimiento aparecen en **19551** estados con `2` y **337** con `9`.

La velocidad base está presente en los **19888** estados, con **3** vectores no nulos, conservados como arrays nativos `[3]float32`. Hay valores históricos no nulos de aterrizaje y salto; ninguno se utiliza como velocidad actual. `m_flSimulationTime` está declarado pero ausente en el pawn; `m_nTickBase` permanece ausente. Se contrastan **427794** campos duplicados con la fuente de relojes, incluyendo su tipo declarado y la misma identidad/corte/orden.

Dos lecturas acotadas de declaraciones de **la misma demo** precedieron a la implementación: una selección y el inventario completo de **268 grupos de serializers**. Ambas se cancelaron deliberadamente al recibir `CDemoSendTables`, antes de procesar la partida completa. Sirven para investigar fuentes, no son pasadas funcionales ni acreditan valores en T0. Los helpers, comandos, salidas y hashes se retienen. El contrato condicional `VectorWS`, sus magnitudes y ceros se verifican con fixtures; no se presentan como cobertura positiva de esta demo.

## Verificación

- `npm run check:all`: pasa en **81.57 s**. **1049 pruebas AI Coach**, incluidas **48 nuevas**, pasan; cinco funciones Go nuevas, Ruff, `go vet ./...` y `go mod verify` correctos. Suites finales completadas antes del smoke.
- Una pasada completa de `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, a TEMP nuevo. Parser **95.381 s**, materializador **2403.695 s**, lector completo **1684.585 s**, canónico **252.778 s**, cero errores. Son tiempos transcurridos de la validación aislada, no una medición de latencia de producto.
- Auditoría sin importar derivaciones de producción: **7058601 campos** reconstruidos desde el prefijo, **194009 enlaces a estados retenidos**, **388080 hechos**, todos los cortes y matriz completa. Se comprueban flags, handles completos, identidades y disponibilidad actual.
- **7 controles reales** cubren contacto positivo/negativo, mundo/entidad/ninguna y ambos códigos de movimiento en ambos roles. Coinciden con el informe; modificar movimiento ajeno e introducir futuro malformado no cambia las features iniciales, que excluyen identidades.
- Fixtures cubren arrays fijos/slices, finitud, cero frente a ausencia, fuentes contradictorias, parent/attachment, referencia de suelo ausente o inconsistente, baseline/reemisión, destrucción/reutilización de handles y vidas, cambios de dominio/vida de pawn, fronteras y recálculo del lector.
- Los **72 artefactos canónicos** conservan SHA-256 y conteos. **Siete fuentes y nueve contextos anteriores** conservan hechos, selección, IDs/T0 y matrices, excluyendo únicamente build/referencias de procedencia actualizados. El informe final coincide con el previo salvo los hashes de referencias; la relectura no modifica los archivos sellados.

El grafo estuvo disponible. Se contrastaron símbolos y snippets con el código y SDK fijados; su cobertura con metadatos cambiados se trató como señal orientativa. Los inventarios finales acreditan el código efectivamente compilado y validado.

## Conservación y dependencia restante

Salida final **91 archivos / 2274326444 bytes**, retenida en `C:/Users/kerch/AppData/Local/Temp/stratai-3e8-smoke-xyu2ffmh`, junto con el trabajo y sus inventarios. Se conserva íntegra la salida, trabajo y evidencia de 3E.7, además de los 44 bundles, 50 fuentes, releases, exports, catálogo, interfaz temporal y cambios ajenos. Sin limpieza, adquisición, migración, promoción ni publicación.

Para resolver velocidad instantánea hace falta una **fuente positiva de velocidad actual**, con declaración, valor, sistema de referencia, vida del pawn y disponibilidad causal acreditados en la build correspondiente. Una propiedad histórica, un vector base o una estimación posicional no satisfacen esa dependencia. Esta evidencia acota las declaraciones y rutas consultadas; no demuestra ausencia de toda fuente posible.

La siguiente pieza del plan es **3E.9: temporización nativa del arma propia/aliada**. Las declaraciones retenidas contienen candidatos de ataque primario/secundario y `m_flNextAttack` del servicio de armas, cuya presencia y semántica temporal todavía deben validarse. Un plazo vencido no acredita por sí solo poder disparar. Entrada/compromiso, LOS/contacto, transformación del objetivo transportado, observación legal de C4 caída, NAV físico, readiness completa, cooperación/intención y economía siguen pendientes según su dependencia específica.
