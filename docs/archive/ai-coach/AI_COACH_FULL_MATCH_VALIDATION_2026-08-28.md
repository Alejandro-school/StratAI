# Validación del AI Coach sobre una partida completa

Fecha: 28 de agosto de 2026.

## Resultado

La validación completa ha pasado sobre la misma demo aprobada de Mirage usada en los smokes anteriores, con alias opaco `demo-a11eb0d600ee702c9edb`. Se analizaron sus 24 rondas y todos los frames disponibles. No se descargó ninguna demo, no se entrenó ningún modelo, no se reprocesaron los 44 bundles y no se publicó ningún resultado.

El nuevo modo `--ai-coach-full-demo-validation` parsea la demo una sola vez por intento y produce conjuntamente:

- el bundle canónico temporal;
- los dos smokes acotados ya existentes;
- un barrido NAV temporal de toda la partida;
- un barrido de contacto visual de todas las parejas enemigas vivas en todas las muestras disponibles.

Los detalles se escriben como JSONL gzip determinista para no retener más de una ronda visual en memoria. Los informes guardan tamaño y SHA-256 tanto de los bytes lógicos sin comprimir como del fichero físico comprimido. Un validador Python independiente vuelve a recorrer los dos streams y recalcula filas, orden, estados, contadores, distribuciones, hashes, procedencia, privacidad y assets.

Gate 1B sigue pendiente y `training_allowed=false`. Esto demuestra que el código actual funciona sobre una partida completa; no materializa todavía estos candidatos dentro del corpus canónico.

## Qué significa «completo»

La cobertura no es idéntica en los dos diagnósticos:

- Visión sí evalúa todas las parejas direccionales entre enemigos vivos en cada muestra disponible de las 24 rondas.
- Navegación evalúa exactamente una pareja direccional determinista por cada frame elegible: el origen elegible que se mueve más rápido y su compañero vivo más lejano. Cubre toda la línea temporal, pero no todas las parejas posibles. Se declara expresamente `pair_exhaustive=false`.

Esta selección NAV sirve para someter el cálculo a casos exigentes y variados, pero sesga las distribuciones hacia jugadores rápidos y compañeros lejanos. No debe interpretarse como el comportamiento medio de todos los jugadores.

## Resultados reales de navegación

| Medida | Resultado |
|---|---:|
| Rondas | 24 |
| Frames del replay | 29.520 |
| Frames elegibles / no elegibles | 28.718 / 802 |
| Consultas NAV | 28.718 |
| Rutas alcanzables | 28.277 (98,464 %) |
| Rutas desconectadas | 209 |
| Endpoint fuera de tolerancia | 232 |
| Proyecciones con arma activa | 26.512 |
| Proyecciones con ritmo observado | 27.761 |
| Contextos en movimiento / quietos | 28.201 / 517 |
| Walking / moving non-walking | 3.179 / 25.074 |
| Knife / ducking / scoped | 14.329 / 880 / 161 |
| Violaciones de tiempo exacto para el usuario | 0 |
| Valores tratados como predicción de llegada | 0 |

No existe una velocidad fija. Se observaron contextos con Knife, rifles, pistolas, subfusiles, francotiradores, granadas y C4. Los baselines conocidos usaron siete velocidades de arma entre 200 y 250 unidades por segundo; cuando el objeto activo no tiene una velocidad de movimiento autorizada en la tabla —por ejemplo, una granada o C4— el escenario de arma queda no disponible.

Las medianas y percentiles confirman que la ruta NAV importa:

| Distribución | P50 | P95 | Máximo |
|---|---:|---:|---:|
| Distancia directa | 1.819,10 | 3.254,53 | 4.186,36 |
| Distancia por NAV | 3.330,28 | 6.017,61 | 8.140,04 |
| Relación ruta/directa | 1,666 | 2,942 | 33,012 |
| Velocidad horizontal observada | 238,63 | 267,94 | 570,57 |
| Referencia optimista interna | 13,32 s | 24,07 s | 32,56 s |
| Referencia interna por arma | 13,99 s | 25,57 s | 35,71 s |
| Proyección interna del ritmo observado | 15,29 s | 42,37 s | 3.207,89 s |

El máximo de 570,57 UPS puede contener movimiento excepcional o una discontinuidad entre muestras. El máximo de 3.207,89 segundos aparece al dividir una ruta por un ritmo positivo casi nulo. Ambos son una prueba útil de por qué estas cifras sólo pueden ser señales internas condicionales: no son una ETA fiable para decirle al jugador «tardarás exactamente X segundos».

## Resultados reales de contacto visual

| Medida | Resultado |
|---|---:|
| Frames fuente / canónicos | 29.520 / 29.520 |
| Frames con alguna pareja evaluable | 27.200 |
| Evaluaciones direccionales enemigo-enemigo | 777.284 |
| Contacto visual verdadero por muestra | 24.893 (3,203 %) |
| Línea de visión geométrica libre | 52.004 |
| Dentro del FOV | 392.808 |
| Efectos de humo válidos distintos | 108 |
| Muestras cuya línea atravesó humo | 99.690 (12,825 %) |
| Muestras donde el humo fue el bloqueo final | 4.717 (0,607 %) |
| Episodios direccionales continuos | 2.992 |
| Episodios cerrados / censurados al final de ronda | 2.703 / 289 |
| Reapariciones del contacto dentro de la misma ronda | 2.130 |
| Jugadores evaluados / con algún contacto | 10 / 10 |
| Parejas enemigas únicas sin dirección / con dirección | 25 / 50 |
| Identidades Steam retenidas | 0 |

Los números cuadran entre sí:

- `24.893` muestras visibles son `2.992` inicios más `21.901` continuaciones.
- `752.391` muestras no visibles son `749.688` inactivas más `2.703` finales.
- Las `777.284` evaluaciones se dividen exactamente entre visibles y no visibles; no hubo valores indisponibles.
- Había `29.610` contactos potenciales con geometría y FOV favorables. El humo bloqueó `4.717`, un 15,93 % de esos contactos potenciales; los otros `24.893` quedaron visibles.
- Sólo un 4,732 % de las `99.690` líneas que atravesaron humo fueron casos donde el humo era el bloqueo decisivo. En el resto ya impedían el contacto una pared, el FOV u otra condición.
- Las 25 parejas posibles entre los cinco jugadores de cada equipo tuvieron algún contacto durante la partida; las 50 direcciones observador→objetivo también aparecieron.

Por tanto, `99.690` no significa que «el humo bloqueó la visión 99.690 veces» en lenguaje humano. Es un contador técnico por pareja y por muestra de 16 Hz. La cifra más cercana a «el humo fue realmente el causante» es `4.717`, que sigue siendo un contador por muestra, no un número de eventos humanos únicos.

Del mismo modo, `2.992 episodios` no significa 2.992 enemigos diferentes vistos. Un episodio es una racha continua de contacto de un observador hacia un objetivo. Si se pierde y recupera el contacto, empieza otro episodio; por eso hubo 2.130 recuperaciones dentro de ronda.

## Aclaración de la validación acotada anterior

El valor histórico de `1.796` procedía sólo de 256 muestras de un prefijo de ronda, no de toda la partida. Contaba intersecciones humo–línea por pareja y muestra. Los `10 episodios` eran diez rachas direccionales continuas en esa misma ventana; no eran diez enemigos únicos ni diez contactos de toda la partida.

Ese smoke se mantiene porque comprueba rápidamente una ventana pequeña y diversa. El nuevo barrido completo lo complementa, no lo sustituye.

## Defecto encontrado y corregido

El primer intento produjo los artefactos, pero el validador independiente falló cerrado en la fila NAV 38. Cuando una ruta era imposible, se conservaba la velocidad horizontal observada pero su procedencia perdía el sufijo `position_delta`.

Se corrigió el productor para conservar `causal_horizontal_speed_at_t0:position_delta` incluso si la distancia de ruta no está disponible. Se añadió una prueba específica, pasó toda la suite Go y se repitió exactamente la misma demo y SHA-256.

La corrección afectó sólo a la procedencia de las 441 filas sin ruta: los resúmenes NAV quedaron idénticos. El resumen y el stream visual fueron idénticos entre ambos intentos, con SHA-256 lógico `8ad062fdc9164fc4beeaec17c7e79c1337653629757e5c965bf18e040d191144`.

## Verificación final

- 59 pruebas de contratos y datasets del AI Coach: correctas.
- Ruff del AI Coach: correcto.
- `npm run check:all`: correcto; frontend 88 pruebas, Node 14 correctas y 3 omitidas, Go completo, backend Python 43 pruebas y build de frontend.
- Validador de ambos streams full-demo: correcto.
- Smokes NAV y visual acotados: correctos.
- Bundle canónico `3.8.0`/parser `v16`: 72 artefactos, cero errores semánticos.
- `DecisionExample`: 918 ejemplos, 918 joins correctos, cero fallidos, cero observaciones posteriores a T0 y cero oracle dentro de observed.
- Dos auditorías directas idénticas: SHA-256 `02a69f5e897500d8f8f3d11801637f30796b09de25c0ad091eb3e2e7e3a6fb21`.
- `TaskEligibility`: 0 elegibles, 3 inelegibles y 14 no materializadas.

La evidencia máquina completa está en [AI_COACH_FULL_MATCH_VALIDATION_2026-08-28.json](../../evidence/AI_COACH_FULL_MATCH_VALIDATION_2026-08-28.json).

## Seguridad y límites

Antes y después se mantuvieron exactamente 100 demos, 44 bundles aprobados, 3.013 archivos de bundles, 727 archivos en `failed` y cero en `staging`. Los hashes de la demo y de su manifest aprobado no cambiaron; `Faceit-Demos` siguió limpio.

No se entrenó, descargó, publicó, promovió ni reprocesó el corpus. Los informes completos son diagnósticos temporales, no nuevos datos canónicos.

Las dos carpetas de intento se verificaron como hijas directas de `%TEMP%`, sin reparse points, y se eliminaron después de conservar la evidencia pequeña. Se borraron 158 archivos y 1.547.228.478 bytes; la eliminación no es recuperable.

El contacto visual actual usa geometría estática, FOV y el modelo vigente de humo cilíndrico en dos dimensiones. Todavía no incorpora ceguera por flash. El barrido NAV no es exhaustivo por parejas y su selección está sesgada deliberadamente. Estas limitaciones deben acompañar cualquier interpretación de los valores.

## Siguiente fase

La siguiente acción canónica no cambia: materializar ammo de cargador, reserva y estado de recarga causales a 16 Hz, con sentinels y ausencias explícitos. Esa fase deberá volver a pasar el gate de exactamente una demo aprobada.
