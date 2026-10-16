# Cierre de fase 1B.7 — contrato multimapa de ángulos de exposición y referencia Mirage

**Fecha:** 29 de agosto de 2026  
**Estado:** cerrada en código y validada sobre una demo real  
**Gate 1B:** pendiente  
**Entrenamiento:** no autorizado (`training_allowed=false`)

La fase congela `stratai.exposure_angle_map_package@1`, su manifiesto de soporte y el sidecar diagnóstico `stratai.observed_exposure_angle_diagnostic@1`. El núcleo selecciona paquetes por `map_id` y versiones exactas, valida schema, hashes, procedencia, bounds, cobertura, ray grid, sectores, solapamientos y revisión antes de exponer geometría. No contiene IDs, aliases ni excepciones de Mirage. Un mapa desconocido, un paquete ausente o alterado, una versión incompatible, un `map_id` distinto o un asset fuente incoherente devuelve `unavailable`; nunca recibe Mirage como fallback.

## Qué aporta al AI Coach

El Coach dispone ahora de una base espacial reproducible para preguntar, en T0, qué sectores de clearance geométrico rodean una posición muestreada y cómo quedan respecto al yaw actual. La convención común fija coordenadas Source 2, unidades, cero angular, giro positivo, normalización `[0,360)`, intervalos semiabiertos, altura, resolución, procedencia y límites. Esto permite añadir en el futuro un paquete de otro mapa sin cambiar la semántica central.

La mejora es deliberadamente prudente: clearance de raycast no equivale a exposición táctica, un callout no es una etiqueta y NAV no acredita cover. La geometría objetiva y las anotaciones tácticas son tipos separados. Como no existe revisión humana atribuible, las 25 anotaciones Mirage permanecen `pending_review`, sin clasificación, autor, revisor ni evidencia fabricados; toda salida táctica se abstiene.

## Referencia candidata de Mirage

El paquete `de_mirage` `1.0.0-candidate.1` se regeneró de forma determinista desde los GLTF/bin de física protegidos y ocho coordenadas usadas sólo como semillas. Sus dos regeneraciones y el fichero almacenado dieron el mismo SHA-256 `9f76c6556ac2e3e51adabf2f618524470815076fc167965687212fc5d7a90b94`.

- 8 muestras con IDs estables `sample-001`…`sample-008`.
- 25 sectores geométricos derivados de rays cada 15°.
- alcance máximo 2.048 unidades, clearance mínimo 512 y ojo de muestreo a 64 unidades.
- cobertura declarada `partial_reference_samples`; nunca cobertura total.
- revisión geométrica automatizada y reproducible; revisión táctica humana pendiente.
- `product_supported=false` y `training_use_authorized=false`.

El NAV de Mirage se auditó por hash, pero no se usó como verdad táctica. Dust II, Inferno, Ancient, Nuke, Anubis, Train y cualquier otro mapa continúan sin paquete implementado ni soporte declarado.

## Productor candidato y causalidad

El modo aislado `--ai-coach-exposure-angle-validation` usa el replay ya parseado, muestrea a 1 Hz por ronda y publica sólo alias locales opacos. Cada fila conserva `map_id`, versión del contrato y asset, `T0`, `availability_tick`, posición, yaw y hashes de procedencia. Los sectores absolutos proceden del paquete; los relativos se recalculan desde el yaw causal de esa misma muestra. Posiciones fuera de la cobertura parcial producen ausencia explícita. No se leen enemigos ocultos, futuro, outcomes, oracle, nombre ni Steam ID.

El sidecar y su reporte están separados del bundle canónico `3.8.0`: no cambian el manifest histórico, la elegibilidad ni el catálogo. Tampoco generan decisión, recomendación, intención, riesgo, etiqueta good/bad, cover, salida u outcome.

## Smoke real obligatorio

Se procesó exactamente una demo aprobada, `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, en un directorio nuevo y vacío bajo `%TEMP%`. El único intento del productor terminó en 102,464 s.

Resultados reales:

- 24 rondas, 29.520 frames de entrada, 1.857 muestras temporales, 10 aliases y 12.802 filas.
- 883 filas geométricas observadas; las ocho muestras tuvieron registros no vacíos.
- 11.919 ausencias `outside_declared_sample_coverage`, coherentes con la cobertura parcial.
- 0 solapamientos ambiguos y 0 observaciones tácticas.
- 12.802 abstenciones tácticas: 883 por `human_review_pending` y 11.919 por ausencia geométrica.
- 0 ticks futuros, campos oracle/outcome, identidades crudas, mismatches angulares o mismatches de bounds/selección.
- stream gzip de 422.357 bytes, 15.274.659 bytes lógicos, SHA-256 físico `20b86c93ad7c014db06a7257b66a0d7f552be8b9576dbcb71d2dce46d0b3aa4a` y lógico `75c47f5b94576cb1204c657b94ef1e36771218fc4c8bb09ddfbf6acc00d6dce5`.
- bundle temporal `3.8.0`/`v16` de 73 ficheros y 72 artefactos declarados; `validate_direct_bundle` obtuvo cero errores.

El validador Python independiente comprobó Draft 2020-12, hashes físico/lógico, identidad, mapa y versiones, orden determinista, causalidad, privacidad, selección de muestra, distancias, sectores absolutos/relativos, wrap-around, intervalos semiabiertos, pertenencia angular, conteos y abstención por revisión pendiente. El inventario pre/post reprodujo 100 demos, 44 bundles/3.013 ficheros, 727 ficheros failed, staging vacío, 1.242 ficheros en exports, los 44 manifests, Git de Faceit-Demos y `CURRENT=stratai-2c6db440231d463a` sin cambios.

La [evidencia máquina](../../evidence/AI_COACH_MULTIMAP_EXPOSURE_ANGLES_MIRAGE_SINGLE_DEMO_SMOKE_2026-08-29.json) conserva el comando exacto, contratos, hashes, conteos, procedencia, soporte por mapa, checks y limpieza.

## Verificación

- `go test -count=1 ./pkg/exposure ./parser ./cmd/canonical-export ./cmd/exposure-package-generator`: pasó.
- `python -m pytest -q ai_coach/tests`: 158 pruebas pasaron.
- Ruff sobre los paths afectados: pasó.
- `npm run check:all`: frontend 88; Node 14 pasadas/3 omitidas; todos los paquetes Go; backend Python 43; build frontend correcto.
- regeneración doble del paquete Mirage: byte a byte determinista.
- validador independiente del sidecar real: pasó.
- validador semántico del bundle temporal: 0 errores.

## Qué continúa pendiente

`FLD-EXPOSURE-ANGLES` sigue `REQUIERE_CLASIFICACION` en los 44 bundles históricos porque no existe revisión humana táctica ni materialización canónica. Gate 1B continúa pendiente y ninguna capacidad gana elegibilidad. No hay mapa productivamente soportado. Quedan por definir y revisar `FLD-COVER` y `FLD-EXIT-ROUTE`, y cualquier migración 44/44, release, promoción o entrenamiento requerirá autorización posterior explícita.

## Siguiente paso

La única fase sucesora es 1B.8: congelar el contrato agnóstico al mapa de `FLD-COVER` e implementar únicamente una referencia parcial de Mirage, manteniendo separada la geometría reproducible de la clasificación humana. No incluye rutas de salida, migración canónica, otros mapas ni entrenamiento.
