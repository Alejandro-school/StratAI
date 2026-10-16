# Reparación AUD-F01 — compatibilidad Node → Go → validador

Fecha: 20 de septiembre de 2026. Alcance: reparación técnica del contrato v16 ya acordado; no cambia criterios de juego ni implementa el intérprete.

## Resultado

**AUD-F01 reparado y verificado en el recorrido previo a publicación.** Los trabajos nuevos de Node, los ya encolados con v11 y la recuperación desde v11 emiten v16. Cada intento registra la versión del worker antes de enviar la petición, conservando la identidad y los demás metadatos del trabajo. Go conserva el valor por defecto v16 para una versión omitida y devuelve HTTP 422 para una versión distinta antes de rutas, slots, lectura/hash de demo, locks, parsing o salida. Mantener 422 conserva la compatibilidad con la recuperación existente. No se rebajó el validador ni se admitieron versiones arbitrarias.

La expectativa v3 de **AUD-T01** se corrigió a v16 y se añadió regresión ejecutada sin Redis. Esto cierra el desfase contractual y su cobertura local; los tres casos que requieren Redis real siguen omitidos y no se afirman aprobados. La auditoría del 19/09 y sus evidencias quedan intactas como resultados históricos.

## Cambios y pruebas que los sostienen

- `backend/node-service/services/pipelineV2.js`: la constante del parser pasa de v11 a v16. Los caminos de alta, petición y recuperación usan esa constante. `runAnalysis` actualiza primero los metadatos de intentos antiguos o sin versión y envía siempre la versión del worker actual; una versión guardada en la cola ya no puede sustituirla. Se añaden `runAnalysis` y `runDownload` al objeto `testHooks` ya existente para comprobar sus cuerpos reales.
- `backend/go-service/api/handlers.go`: rechazo explícito de versión incompatible inmediatamente después de aplicar el valor por defecto. Se mantienen los demás gates y el orden de publicación.
- `backend/node-service/test/pipeline.integration.test.js`: expectativa de recuperación alineada a v16. La prueba condicionada no se usa como sustituto de la nueva regresión.
- `pipeline.contract.test.js` y helpers: ejecutan el módulo real de Node sustituyendo exclusivamente transporte, Redis, colas, descarga y métricas. Verifican alta/default/recuperación, actualización de un trabajo ya encolado antes del envío HTTP, preservación de identidad, ausencia de reintento para versión actual o fallo ajeno y ausencia de finalización ante rechazo.
- `api/request_contract_test.go`: consume las tres peticiones realmente serializadas por Node; comprueba que atraviesan el gate de versión y llegan al 404 del archivo de fixture ausente. El caso omitido sigue el mismo camino. Versiones v11/v3/v999, espacio inicial y mayúsculas se rechazan con 422 incluso con el semáforo lleno y el contexto cancelado, sin mutaciones de salida.
- `scripts/test_node_parser_contract.py`: ejecuta el emisor real de peticiones Node y confronta cada versión con `validate_root_catalog`, incluida la negativa v11. Este fixture sólo certifica el contrato raíz; el bundle completo se comprueba en el smoke real.
- `.github/workflows/quality.yml`: el job Go declara Node 20.19 para ejecutar la nueva prueba entre servicios; el helper usa sólo módulos nativos y no necesita instalar dependencias Node en ese job. CI remoto no se ejecutó desde esta sesión.

Las nuevas regresiones Node y Go fallaron antes de cambiar el contrato y pasaron después. Durante la revisión final, una cuarta regresión reprodujo que un trabajo ya encolado con v11 seguía sustituyendo la constante actual; pasó después de corregir el intento. Se repitieron las suites y el smoke con esa última corrección. No se tomaron lecturas de constantes o exit code como prueba suficiente del recorrido real.

## Verificaciones ejecutadas

| Comprobación | Resultado | Duración | Evidencia |
|---|---|---:|---|
| Regresión Node antes del arreglo | Fallo esperado: reproduce el defecto | 0.441 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/node-contract-before.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/node-contract-before.log) |
| Regresión Go antes del arreglo | Fallo esperado: reproduce el defecto | 15.795 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/go-contract-before.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/go-contract-before.log) |
| Regresión Node corregida | Código 0 | 0.316 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/node-contract-after.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/node-contract-after.log) |
| Regresión Go corregida | Código 0 | 2.966 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/go-contract-after.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/go-contract-after.log) |
| Trabajo antiguo en cola antes del arreglo | Fallo esperado: reproduce el defecto | 0.340 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/node-stale-before.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/node-stale-before.log) |
| Trabajo antiguo en cola corregido | Código 0 | 0.344 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/node-stale-after.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/node-stale-after.log) |
| Suite pública completa final | Código 0 | 22.036 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/check-all-final.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/check-all-final.log) |
| Scripts Python de Go finales | Código 0 | 26.774 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/scripts-tests-final.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/scripts-tests-final.log) |
| Suite Go tras robustecer el caso cancelado | Código 0 | 8.544 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/go-final.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/go-final.log) |
| Contrato Node–Python final | Código 0 | 0.932 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/python-node-contract-final.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/python-node-contract-final.log) |
| Lint de la prueba Python nueva | Código 0 | 0.346 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/new-python-lint.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/new-python-lint.log) |
| Contrato AI Coach | Código 0 | 5.540 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/contract.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/contract.log) |
| Primer smoke, misma partida | Código 0 | 426.693 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/real-api-smoke.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/real-api-smoke.log) |
| Smoke final tras la corrección de trabajos en cola | Código 0 | 433.506 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/real-api-smoke-r2.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/real-api-smoke-r2.log) |
| Comprobación independiente | Código 0 | 1.612 s | [recibo](../../../outputs/stratai-aud-f01-repair-20260920/independent-verification.json); [log](../../../outputs/stratai-aud-f01-repair-20260920/independent-verification.log) |

`check:all` final: 88 tests frontend, 18 Node aprobados y tres omitidos, 22 paquetes Go (API ejecutada; 21 desde caché), 43 FastAPI, lint selectivo y build frontend. La suite complementaria final ejecutó **187 tests** Python de herramientas Go, incluido el contrato Node–Python. El contrato conserva 104 schemas, 19 razones, 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes y 58 campos. Se repitió Go tras reforzar el test de cancelación y el test Python después de ordenar sus imports; después de corregir el trabajo en cola se repitieron `check:all` y los 187 tests, antes del smoke final. El código productor no cambió después de esas suites. Los 1.840 tests AI Coach ejecutados en la auditoría anterior no se atribuyen a esta sesión: sus módulos no se modificaron en esta reparación.

Entorno: Node 25.9.0, npm 11.12.1, Go 1.26.2 windows/amd64, Python 3.14.4. Dependencias existentes, resolución offline y ninguna instalación/descarga. [Versiones](../../../outputs/stratai-aud-f01-repair-20260920/versions.json). El aviso Python sobre `<prefix>` no impidió las comprobaciones. La integración Redis/Steam y la publicación operacional no se ejecutaron.

## Smoke de una única demo aprobada

Alias `demo-a11eb0d600ee702c9edb`; SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. **Una sola partida distinta, dos ejecuciones del parser**, cada una en un TEMP nuevo y después de sus suites. La primera pasó; al detectar el caso de cola v11 se conservó íntegramente y se repitió la misma partida después de la última corrección y sus tests. El cierre utiliza la segunda ejecución. Se comprobó la aprobación entre los 44 manifests y el SHA de la fuente; ambas copias se crearon exclusivamente dentro de TEMP. Fuente, corpus y bundles originales se leen solamente.

Se ejecutó **el cuerpo JSON emitido por Node → `HandleProcessDemo` real → parser real → exportador real → manifiesto operativo → `publication_validator.py` real**, con mapas locales y build `working-tree-aud-f01-20260920`. El manifiesto raíz tiene v16, export 3.8.0, validador `stratai.canonical_validator@2` y **73 artefactos**; el manifiesto canónico `stratai.canonical_manifest@3` cataloga 72.

Instrumentación limitada y explícita:

1. Redis se sustituye por un hook local que sólo admite adquisición/liberación del lock (`set`, `eval`) y prohíbe red y cualquier operación de datos. No se arranca el bot, Redis ni un servidor HTTP real; el handler recibe una petición local de `httptest`.
2. El overlay de compilación cambia **únicamente** `defer os.RemoveAll(stagingRoot)` por un comentario para retener la salida aislada. No cambia el handler productivo del repositorio, sus controles, parsing, exportación ni validación. Su diferencia y hashes se conservan en [preflight final](../../../outputs/stratai-aud-f01-repair-20260920/smoke-preflight-r2.json) y el inventario.
3. Después de que el validador real devuelve `passed` sin errores, el callback de prueba devuelve un error deliberado para detenerse antes de snapshot/commit/publicación. El HTTP 422 final es esa barrera intencionada, **no un rechazo de versión o un fallo del bundle**. Se comprueba que sólo hubo `set/eval`, que no existe la carpeta final publicada y que la salida temporal sigue presente. No se anuncia una respuesta operacional HTTP 200 ni una publicación completa probada.

En cada ejecución, la petición incompatible v11 obtiene 422 antes de producir salida. La omitida pasa la normalización y se detiene por cancelación con 408, sin parsing adicional. La petición v16 realiza el único parsing de ese intento. En la ejecución final el validador real terminó en 332.172 s; el comando completo duró 433.506 s. Las advertencias de cobertura se conservan: causal_availability, combat_consistency, corpus_quality, economy_consistency, engagement_consistency, lineage_completeness, utility_consistency.

## Contraste independiente y límites

El helper independiente usa sólo la biblioteca estándar. Verificó hashes/tamaños y cobertura de los **73 archivos del catálogo raíz**, identidad/versión/procedencia y la igualdad byte a byte de los **72 artefactos de datos canónicos** con los hashes retenidos en la auditoría del 19/09. Cambian la identificación del build en el manifiesto canónico y la envoltura operacional nueva; no se reetiquetan silenciosamente hechos de la partida.

Hay **24 rondas, 10 participantes y 918 registros por cada una de las cinco particiones causales**. Se comprobó la alineación completa de esas particiones por ronda/T0/decisión, 33650 estados y 918 referencias observadas, límites de disponibilidad ≤ T0, outcomes ≥ T0, horizontes 2/5/10 y disjunción de máscaras. Los 918 oracles siguen absteniéndose. Conteos completos de ausencias, estados y hashes: [verificación independiente](../../../outputs/stratai-aud-f01-repair-20260920/independent-smoke.json).

La igualdad de los datos y la validación causal no acreditan tránsito físico del NAV candidato, contacto legal, prescripción, S2–S3 o evaluación independiente. No se recalcularon algoritmos de juego que esta reparación no modifica. El resultado acredita la compatibilidad y validación previa a publicación; siguen fuera del alcance el transporte/autenticación desplegados, Redis real y commit/rollback operacional.

## Preservación y siguiente acción

Se registraron Git/HEAD/diffs y se copiaron las fuentes existentes antes de modificarlas. El inventario protegido final coincide con el inicial: 44 bundles aprobados, 50 demos, 4.356 archivos, 42.372.398.910 bytes; SHA de 181 archivos y tamaño/mtime de todos. No es un rehash completo de los 42 GB. [Inventario inicial](../../../outputs/stratai-aud-f01-repair-20260920/protected-before.json); [final](../../../outputs/stratai-aud-f01-repair-20260920/protected-after.json).

Se retienen **164 archivos / 2192565177 bytes** de ambas ejecuciones, incluidas copias de entrada, overlays, ejecutables, peticiones, recibos del validador y salidas, con SHA de cada uno. [Inventario y rutas TEMP](../../../outputs/stratai-aud-f01-repair-20260920/smoke-inventory.json). No se borraron salidas de smoke ni datos protegidos; tampoco hubo promoción, publicación, adquisición, lote 44/44, entrenamiento, commit ni cambio de CURRENT. La interfaz temporal y los libros permanecen intactos. La comprobación final del worktree y de las evidencias históricas está en [preservación](../../../outputs/stratai-aud-f01-repair-20260920/source-preservation.json).

**Única siguiente acción, no ejecutada:** corregir y verificar **AUD-C01**, delimitando el NAV candidato como diagnóstico de grafo y evitando su uso como transitabilidad/coste físico acreditado. Conservar valores, IDs y procedencia; no cerrar conexiones físicas ni reprocesar los 44 bundles. AUD-D01/D02/D03, E01/E02 y O01 siguen pendientes en sus propios alcances. El intérprete y K3 no se inician; tampoco cambian S2–S3, las 18 hipótesis ni la elegibilidad.

[Evidencia de cierre](../../evidence/AI_COACH_PIPELINE_VERSION_REPAIR_2026-09-20.json). [Diff de esta reparación](../../../outputs/stratai-aud-f01-repair-20260920/repair.patch); [comprobaciones documentales](../../../outputs/stratai-aud-f01-repair-20260920/closure-checks.json). Los comandos conservados no autorizan repetir directamente scripts que apuntan a estas salidas: cualquier nueva ejecución debe usar destinos nuevos y conservar los anteriores.
