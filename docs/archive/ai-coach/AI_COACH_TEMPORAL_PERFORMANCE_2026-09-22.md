# Optimización del registro temporal — 22 de septiembre de 2026

Cerrado el ajuste técnico dentro del alcance ICT-I1.A. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y orden. [Evidencia](../../evidence/AI_COACH_TEMPORAL_PERFORMANCE_SMOKE_2026-09-22.json).

La generación sobre la misma captura de referencia pasa de 1237,647 s a 65.509 s. SHA-256 idéntico del JSONL completo: `c8b261d52fc9301621843cfac46f4b5c49bba73d455b3feb23da8c02e62993ff`. El benchmark comparte equipo con otros checks; no es una garantía de latencia de producción. El parser no se ha optimizado.

Se compilan únicamente los cinco schemas medidos con `jsonschema-rs==0.57.1`, Draft 2020-12. El contenido del schema determina la caché; nunca se cachean instancias. Se conserva la validación semántica y causal. Valores Python no JSON y errores siguen el motor de referencia. No cambia ningún schema, catálogo, contrato o criterio táctico.

Pasan `npm run check:all` y 1.901 pruebas del AI Coach. Tras añadir la compatibilidad con sustitutos Unicode aislados escapados, se repiten todas las pruebas afectadas de los cuatro consumidores, del registro, de equivalencia y de contratos: ver recibo `final-targeted`. Hay 14 pruebas nuevas de equivalencia/compatibilidad. Se conserva el contrato de 107 schemas/19 motivos. La comparación diferencial recorre todos los campos de fixtures de los cinco contratos, con supresión de propiedades, valores extra, tipos erróneos, enums, límites, booleanos, números y valores Python especiales.

Tras las pruebas se procesa exactamente una demo aprobada en TEMP nuevo. Parser: 75.348 s; generación: 64.729 s; lector/recomputación: 66.472 s. Se conservan 61.432 registros y todos los conteos previos. Auditor independiente y prueba de futuro/rival/etiquetas pasan. Los 72 artefactos canónicos son idénticos a los ya validados semánticamente y contrastados con su inventario sellado; el manifiesto sólo cambia la procedencia de build. No se repite una validación semántica de datos idénticos.

La fuente aprobada y todas las salidas anteriores permanecen de sólo lectura. Benchmark, primer smoke y repetición final de la misma partida quedan inventariados y retenidos. El JSONL final es idéntico byte a byte al del primer smoke. Sin promoción, descarga, entrenamiento ni recomendaciones. Este cierre habilita ejecutar el lote técnico acotado solicitado, no un modelo táctico.
