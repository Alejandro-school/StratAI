# Reconciliación de tenencia 3D.6 — 8 de septiembre de 2026

Entrega funcional validada; limpieza pendiente por rechazo automático. [Evidencia completa](../../evidence/AI_COACH_ECONOMY_OWNERSHIP_SINGLE_DEMO_SMOKE_2026-09-08.json), [contrato](../../contracts/economy/canonical-economy-ownership-v1.md) y [estado vigente](../../AI_COACH_IMPLEMENTATION_PLAN.md).

El selector optativo **9** añade una lectura fresca de dueño y slots activos al final de cada fotograma seleccionado del freeze. El SDK termina los mensajes del comando y procesa sus eventos diferidos antes de emitir `FrameDone`; ése es el corte de entrega utilizado. No equivale a subtick de ejecución, a terminar todos los comandos del mismo tick ni a conocimiento del jugador. No se rellenan identidades o T0 desde el futuro.

Se añade `economy_ownership_observations@1`, enlazado a la identidad 3D.5 por hash, y `economy_ownership_reconciliation@1`, enlazado a esa fuente. La captura enumera las vidas actuales, lee de nuevo los handles y verifica serie completa y clase. Se conserva un marcador por fotograma; los payloads idénticos sólo comparten almacenamiento después de volver a leerlos. Slots fuera de longitud, vidas sustituidas e inventarios desconocidos no se transforman en tenencia.

| Hecho observado en la única partida | Resultado | Límite |
|---|---|---|
| Cortes frescos | **35.485**, con **885 estados distintos** | Cubren todos los fotogramas con reloj observado dentro del freeze estricto; no acreditan lo que ocurre entre comandos. |
| Registros equipo/estado | **60.517** | Son registros repetidos de estado, no objetos únicos, transacciones ni ejemplos independientes. |
| Dueño e inventario coincidentes | **37.842** | Coincidencia por vida y pawn, actor único e inventarios completos en el ámbito observado. |
| Sólo dueño | **6.825** | Falta la segunda fuente; no se declara tenencia ni su causa. |
| Sin asignación en el ámbito observado | **15.850** | Sentinel inválido de dueño y ninguna referencia activa, con inventarios completos; no prueba ausencia física global. |
| Cambios entre cortes consecutivos | **213: 85 apariciones y 128 desapariciones** | Ningún cambio directo entre dos tenedores observado; no se etiquetan compras, drops voluntarios, regalos o devoluciones. |
| Matriz de oportunidades | **46 de 50** con estados coincidentes | Las instancias 1 y 25 conservan enlace competitivo no verificado. Cero comparaciones económicas habilitadas. |

Los 213 cambios usan el mismo alias de vida, fotogramas adyacentes, misma instancia y muestras de reloj presentes del propio fotograma dentro del mismo segmento. Hay 24 separaciones entre instancias que impiden unir extremos. No faltan relojes en los cortes observados. No se han observado aquí conflictos dueño/slot, duplicados, inventarios incompletos o aliases ambiguos: su cobertura corresponde a fixtures, no a esta demo. Un estado intermedio incierto impide enlazar tenedores separados por ese hueco.

El lector verifica el prefijo completo de identidad, las entidades vivas, las referencias y series, la longitud activa, los cortes estrictos y la cadena de hashes. El validador del informe recalcula todos los registros, transiciones y contadores. Un programa independiente, sin importar el lector, ha vuelto a calcular **los 60.517 registros y las 213 transiciones**, incluidos cortes, relojes, referencias y matriz. Los casos retenidos usan únicamente aliases locales.

Pasaron `npm run check:all`, **577 pruebas del AI Coach** —28 nuevas—, cuatro nuevas pruebas Go, Ruff y `go mod verify`. El smoke procesó **una única partida aprobada en una sola pasada**, en TEMP nuevo y vacío. Alias `demo-a11eb0d600ee702c9edb`; SHA-256 completo y comandos en la evidencia. Build `working-tree-economy-3d6-20260908-r1`. Parser **97,187 s**, reconstrucción nativa **6,155 s**, materializador CLI **65,779 s**, validación del informe **67,991 s** y canónico completo **309,461 s, cero errores**.

Las siete salidas de control —v2, fuentes, contexto, payload, manifiesto, traza e identidad— conservan exactamente sus observaciones salvo build y referencias de procedencia. Se mantienen 77/250 estados iniciales acreditados, 173 ambiguos, tres equipos completos, 50 oportunidades, 60 diferencias numéricas, 15 transiciones de inventario anteriores, 1.467 capturas nativas y 463 referencias del manifiesto. La traza sigue en 7.918 filas; sus relojes conservan 1.642 coincidencias y 41 desacuerdos. Los 4.356 archivos protegidos y sus 181 hashes de control no cambian.

Se usó el grafo para descubrir productores y leer funciones. La comprobación de cobertura reportó metadatos de archivos cambiados, sin garantizar exhaustividad; se contrastaron la fuente actual, sus pruebas y hashes de build. El SDK instalado y la fuente primaria `parsing.go` acreditan el corte de `FrameDone`. No se repitieron búsquedas de codecs, defaults o assets.

El registro tiene **55 schemas y 19 razones**, plan `2.17` y contrato humano `1.24`. Catálogo `1.7.0`, export base `3.8.0`, parser `v16`, quality `12` y manifest `@3` mantienen su significado. `comparison_allowed`, `training_allowed` y `product_supported` siguen en `false`. No se modificaron corpus, release, runtime o CURRENT ni se adquirieron demos.

El hecho nuevo es poder contrastar tenencia en un corte explícito y describir cambios observados entre cortes compatibles. Sigue faltando vincular las entradas económicas nativas y su coste registrado a esas vidas de entidad y cortes, y acreditar causa transaccional, reglas exactas y cooperación. La siguiente acción es **3D.7: vinculación de entradas nativas con objetos**. No se unen snapshots históricos sólo porque compartan tick ni se aplica su coste a una cesta hipotética. La dependencia externa de assets continúa separada.

La revisión automática de aprobación rechazó `Remove-Item` sobre la única nueva salida temporal con `blocked by policy`, antes de iniciar el proceso. Se conservan **88 archivos / 798.029.328 bytes**, inventariados y recontrastados por hash tras el rechazo. No se repitió ni eludió el intento ni se tocaron temporales anteriores. El cierre documental y la conservación de las 27 evidencias previas quedan registrados en la evidencia de esta fase.
