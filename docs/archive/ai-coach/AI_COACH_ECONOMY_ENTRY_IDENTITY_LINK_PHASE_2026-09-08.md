# Entradas económicas y vida de objeto 3D.7 — 8 de septiembre de 2026

Entrega funcional validada. TEMP retenido por indicación del usuario; sin intento ni revisión de borrado. [Evidencia completa](../../evidence/AI_COACH_ECONOMY_ENTRY_IDENTITY_SINGLE_DEMO_SMOKE_2026-09-08.json), [contrato](../../contracts/economy/canonical-economy-entry-identity-v1.md) y [plan vigente](../../AI_COACH_IMPLEMENTATION_PLAN.md).

El selector optativo **10** lee las entradas nativas dentro del mismo callback de tenencia, antes de avanzar el fotograma. Conserva los cinco miembros, sus tipos y valores originales, índices, duplicados y presencia. No une las capturas antiguas sólo por tick. Una referencia exige serie completa y vida disponible; un vínculo exacto exige también definición coincidente, actor único, listas completas y tenencia bilateral del mismo pawn.

La única demo aprobada aporta **35.485 cortes, 1.114 estados distintos y 25.255 registros de entrada/estado**. Los conteos siguientes incluyen repeticiones del mismo objeto en distintos estados; no son compras ni ejemplos independientes.

| Resultado principal | Registros |
|---|---|
| `exact_link` | **4.010** |
| `reference_unresolved` | **0** |
| `identity_ambiguous` | **1.728** |
| `special_entry_without_entity` | **18.906** |
| `ownership_disagreement` | **473** |
| `ownership_unaccredited` | **138** |
| `partial_list` | **0** |
| `invalid_member_value` | **0** |
| `actor_unavailable` | **0** |
| `definition_unavailable` | **0** |
| `definition_disagreement` | **0** |

Los **4.010 vínculos exactos** corresponden a **165 vidas de objeto**, con evidencia en **41 de 50 oportunidades**. La matriz por instancia y oportunidad, incluidos ceros, está en la evidencia. Las instancias 1 y 25 mantienen su enlace competitivo no verificado. Un handle resuelto puede aparecer sin vínculo económico exacto; los estados de referencia y tenencia conservan la razón.

Se observan **4.621 referencias resueltas y 20.634 sentinels inválidos**. El sentinel no acredita objeto, tipo de armadura ni motivo económico. Los metadatos de armadura conservan sus valores y tipos nativos, incluido `-1`; no se convierten a armadura física. Los costes siguen vinculados sólo a la entrada observada, sin precio general, pago efectivo ni devolución.

Cobertura real: **0 miembros ausentes/no interpretados, 0 listas parciales, 0 handles cero, 0 costes cero y 1.728 registros duplicados**. Hay **12 cortes consecutivos con el mismo tick y 0 cambios de estado dentro del mismo tick**. Los casos que aquí tienen cero sólo se acreditan mediante fixtures, no por la partida. La evidencia conserva distribución de tipos, metadatos, costes, definiciones y ejemplos con aliases locales. Cualquier abstención conserva `cause=not_accredited`, `refund_amount=null` y elegibilidad deshabilitada; también los vínculos exactos mantienen esos límites.

Pasaron `npm run check:all`, **619 pruebas del AI Coach**, incluidas **42 nuevas**, cinco nuevas pruebas Go, Ruff y `go mod verify`. Los 14 hashes del SDK y dispatcher fijados se recontrastaron. El grafo guio el descubrimiento; su cobertura es de mejor esfuerzo y marcó metadatos cambiados, contrastados con la fuente actual y hashes de build. La semántica de entrega se apoya en el [SDK v5.2.0](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/parsing.go); no acredita subtick del servidor ni conocimiento del jugador.

El smoke ejecutó **una partida aprobada, una pasada**, después de las pruebas: alias `demo-a11eb0d600ee702c9edb`, build `working-tree-economy-3d7-20260908-r1`. Parser **100.204 s**, materializador de entradas **83.034 s**, validación del informe **83.983 s** y validación canónica completa **317.201 s, cero errores**. Los comandos, SHA-256 completo, binario, fuentes y hashes de todos los artefactos están registrados.

Un programa independiente volvió a calcular **los 25.255 registros nuevos**, las series, vidas, cortes, presencia, duplicados, definición, actor y matriz. Otro recalculó los **60.517 registros y 213 cambios de tenencia**. Las ocho salidas anteriores conservan exactamente sus observaciones salvo build/procedencia. Se mantienen 77/250 actores inicialmente acreditados, 173 abstenciones, tres equipos competitivos completos, 50 oportunidades, 60 diferencias numéricas, 15 transiciones de inventario, 1.467 capturas históricas, 7.918 filas de traza y 463 referencias del manifiesto. Los 4.356 archivos protegidos y 181 hashes de control permanecen iguales, al igual que las 28 evidencias anteriores.

El registro pasa a **57 schemas/19 razones**, plan `2.18` y contrato humano `1.25`. Catálogo `1.7.0`, export `3.8.0`, parser `v16`, quality `12` y manifest `@3` conservan su significado. Cero transacciones itemizadas verificadas, comparaciones económicas, entrenamiento o capacidades soportadas como producto.

Ahora sabemos cuándo una entrada y su coste registrado corresponden al objeto que tiene un actor en un corte concreto. Falta conciliar **los cambios entre cortes** con las entregas de saldo y gasto, separar movimientos simultáneos y residuos sin asignación y acreditar la causa transaccional. La siguiente acción **3D.8** implementa esa conciliación factual; una igualdad de importes será consistencia observada, no prueba de compra, refund efectivo o cooperación. Se reutilizará la traza por fotograma sólo tras demostrar que comparte el corte. La dependencia de assets exactos permanece separada y exige material externo positivo.

Se retienen **90 archivos / 824.621.174 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3d7-smoke-p0_p3rde`, inventariados por SHA-256. Se actualizó `AGENTS.md` y el plan para no intentar ni revisar borrados de TEMP salvo nueva petición explícita. Los rechazos históricos y sus salidas permanecen intactos; la retención actual no es un bloqueo de cierre.
