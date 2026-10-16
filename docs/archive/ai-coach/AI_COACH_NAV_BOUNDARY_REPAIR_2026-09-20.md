# Reparación AUD-C01 — diagnóstico NAV y evidencia física

20 de septiembre de 2026. Corrección técnica del criterio ya establecido: conservar el diagnóstico sin atribuirle tránsito físico. No cambia criterios tácticos ni implementa el intérprete.

## Resultado

**AUD-C01 corregido y verificado.** El candidato espacial `3.9.0-candidate.2` usa manifest `@2`, loader/validador `@2` y `DecisionExample@3`. Los schemas anteriores permanecen intactos como referencias históricas. El lector actual rechaza el manifiesto anterior; no lo reinterpreta ni migra salidas automáticamente.

`observed.spatial_facts.FLD-NAV-EXIT-PATH` tiene siempre `unavailable` y valores vacíos. `spatial_evidence` conserva el diagnóstico `partial_nav_graph`, estado/razones originales, números e IDs locales, referencia exacta a la fila, paquete y procedencia. Sus propiedades se validan con estructuras estrictas, sin el juicio táctico del sidecar. La evidencia física es null cuando existe muestra y missing sin join. Ambos casos conservan una causa explícita; true y false del grafo no acreditan conexión o ausencia física.

El catálogo clasifica el campo como PARCIAL. `TaskEligibility` no recibe cobertura física del diagnóstico. El resumen separa sus conteos y razones de grafo de la cobertura utilizable. Los otros dos campos geométricos conservan su proyección; los sidecars, IDs estables, algoritmos NAV, parser v16 y base 3.8 no cambian.

## Trazabilidad y alcance

Se siguió `MapManager.NavigationGraphSnapshot` → evaluador NAV compartido → `BuildObservedExitRouteRecord` → sidecar `@2` → joins exactos → `SpatialCandidate.attach` → cobertura de `process_manifest` → elegibilidad. El grafo sirvió para descubrir funciones; sus avisos de metadatos cambiados se contrastaron con fuentes y pruebas.

Los enlaces dirigidos y pesos entre centros permanecen idénticos. Los nombres heredados `objective_viability` y `physical_cost` sólo se conservan dentro del diagnóstico. No acreditan vínculo demo–asset, hull, condiciones de paso, escaleras, transformaciones o cobertura completa. Los contratos humano y espacial explican esta frontera y califican los resultados históricos sin reescribir sus informes.

## Pruebas

- Las seis regresiones nuevas fallaron antes de la corrección. La primera pasada posterior tuvo 50 aprobadas y un error en la propia prueba al consultar una clave de elegibilidad inexistente; se corrigió contra el catálogo. Los logs se retienen.
- `check:all`: 88 frontend, 18 Node aprobadas/3 omitidas, 22 paquetes Go, 43 FastAPI, lint selectivo y build, código 0.
- AI Coach completo: 1.844 aprobadas y dos expectativas antiguas de 104 schemas. Los módulos de test ya estaban cargados cuando se actualizaron sus archivos a 106; el log conserva ese desfase de línea/valor. Tras actualizar esas expectativas y endurecer el schema diagnóstico, **70 pruebas afectadas pasan**. No se afirma una segunda ejecución completa con código 0 ni se ocultan los dos fallos iniciales. No quedan fallos conocidos de esa ejecución sin corregir/recomprobar.
- Contrato: 106 schemas, 19 razones; 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes y 58 campos. Los dos schemas añadidos identifican la nueva versión; los anteriores no se modifican.
- Casos: conexión, desconexión, fuente no disponible, join ausente, promoción manipulando estado, manifiesto antiguo, disponibilidad futura e inyección de campos ajenos. Se comprueba preservación de valores/procedencia y bloqueo de elegibilidad.

El Ruff suplementario conserva UP033 e I001 preexistentes (AUD-O01); no se declaran reparados por C01. Redis real y servicios operacionales no se ejecutan.

## Una demo real después de las pruebas

Alias `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Una sola partida distinta y una llamada al parser. Fuente y 44 bundles aprobados en lectura; copia local y destino TEMP nuevo/vacío, sin enlaces/reparse ni sobrescrituras. Se compiló el productor Go y se ejecutó `--ai-coach-spatial-candidate`, seguido del materializador Python cambiado y del lector/validador persistido. La base pasó `validate_direct_bundle` y los tres validadores espaciales recomputaron sus datos desde los assets locales.

El overlay de compilación únicamente desactiva `defer os.RemoveAll(stagingRoot)` para conservar incluso salidas rechazadas. El renombrado interno del CLI queda dentro de TEMP; no se publica ni promueve a corpus/release/exports/CURRENT. Código productivo Go intacto. Build `working-tree-aud-c01-20260920`; comandos, duraciones y hashes están en los recibos enlazados por la evidencia.

Comprobación independiente con biblioteca estándar: **918 ejemplos, 2.754 joins**, hashes de 72 artefactos canónicos y descriptores candidatos; correspondencia completa fila/ronda/T0/alias, disponibilidad temporal y proyección exacta de los otros dos campos. Cada diagnóstico unido conserva sample/candidates/posture, números e IDs y procedencia; cero hechos NAV físicos presentes.

Conteos NAV de joins: `{'missing': 908, 'null': 9, 'present': 1}`. Reachability en fuente: `{'True': 878, 'None': 5}`; en joins de decisión: `{'True': 1}`. Sólo esas variantes naturales constituyen cobertura real; las demás se prueban con fixtures, sin fabricar otra partida. Elegibilidad: `{'ineligible': 3, 'not_materialized': 14}`. Los límites físicos y tácticos permanecen abiertos.

Se retienen 87 archivos / 1099808028 bytes, con inventario SHA completo. El inventario protegido anterior/posterior contrasta 4.356 archivos, 44 bundles, 50 demos, 181 hashes y tamaño/mtime de todos; no es un rehash completo de 42 GB.

## Continuación

Siguiente acción: alinear los contratos documentales de AUD-D01/D02/D03 en su propio cierre. AUD-E01/E02 conservan los límites de evidencia original ausente; AUD-O01 queda para su reparación de estilo/requisitos. Intérprete, K3, entrenamiento y promoción pendientes. Los libros y criterios expertos permanecen intactos.

[Evidencia y recibos](../../evidence/AI_COACH_NAV_BOUNDARY_REPAIR_2026-09-20.json). [Contraste independiente](../../../outputs/stratai-audit-followup-20260920/c01-independent-result.json). [Inventario TEMP](../../../outputs/stratai-audit-followup-20260920/c01-temp-inventory.json). [Preservación](../../../outputs/stratai-audit-followup-20260920/c01-preservation.json).
