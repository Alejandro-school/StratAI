# Corrección del plan de ground truth táctico — 3 de septiembre de 2026

## Decisión

StratAI separa desde este cierre la validación automática de hechos y el juicio humano de decisiones:

- raycasting, colisiones y geometría validan clearance, oclusión y exposición física;
- NAV/pathfinding valida asociación de polígonos, conectividad dirigida, recorrido, distancia y coste físico;
- los humanos comparan la acción observada con alternativas físicamente viables dentro del contexto causal disponible en `T0`.

Una opinión humana no rellena un hecho ausente ni certifica números, rayos, polígonos o caminos recomputables. Del mismo modo, un hecho geométrico correcto no acredita por sí solo seguridad, calidad, intención, preferencia ni recomendación táctica.

## Instrumentos v1 y v2

El lote package-only v1 ya estaba invalidado porque ocultaba la situación observada y representaba las rutas como flechas rectas. La reparación v2 añadió radar, contexto T0 y polilíneas NAV, pero mantuvo como unidad de revisión un sector, perfil de oclusión o conexión dirigida aislados. Por tanto:

- v1 y v2 se conservan como antecedentes reproducibles y QA visual;
- mantienen cero respuestas y adjudicaciones;
- no se ejecutan con revisores;
- sus respuestas/schemas no son una fuente autorizada de entrenamiento, calibración, evaluación humana o soporte de producto;
- sus evidencias históricas no se reescriben.

## Ground truth compuesto

El ground truth futuro tiene tres componentes físicamente separados:

1. hechos observed y geometría/NAV reproducibles, validados automáticamente;
2. outcomes posteriores y oracle, sólo como targets o evaluación y nunca como features de `T0`;
3. preferencia táctica humana sobre decisiones completas, con razones, confianza y abstención.

La unidad humana contiene contexto factual legal en `T0`, acción observada, entre una y cuatro alternativas parametrizadas que hayan superado filtros físicos, radar/replay y Evidence Pack. La respuesta compara alternativas o declara equivalencia, ninguna razonable o contexto insuficiente. Ganar una ronda, conseguir una kill o sobrevivir no convierte automáticamente una decisión en correcta.

## Escalado humano

No se pretende revisar todas las situaciones posibles. Antes del primer recomendador se crea una muestra inicial estratificada y un test bloqueado. Después, el aprendizaje activo selecciona por incertidumbre, desacuerdo, rareza, huecos de cobertura y OOD, siempre con una cuota aleatoria para auditar el sesgo del selector.

Las etiquetas rutinarias pueden tener un revisor experto y control ciego muestral. El test, la calibración, los casos ambiguos y los desacuerdos reciben doble revisión; un tercero distinto adjudica conflictos. Se registran acuerdo, abstención, consistencia pairwise y deriva por versión.

Las respuestas crean datasets versionados. Nunca actualizan un modelo online: control de calidad, reentrenamiento offline, evaluación contra el test bloqueado, promoción y rollback son operaciones separadas.

## Orden de ejecución corregido

1. cerrar la extracción factual y sus validadores automáticos por capacidad;
2. materializar una versión nueva del golden y medir elegibilidad/cobertura;
3. construir reglas, acción observada, generadores de alternativas y filtros físicos;
4. congelar el protocolo táctico, crear el ground truth inicial y reservar el test bloqueado;
5. medir déficits y adquirir demos FACEIT sólo para cubrirlos;
6. reconstruir el dataset ampliado;
7. entrenar outcomes, valor y ranking, y comparar con baselines;
8. incorporar nuevos lotes humanos mediante aprendizaje activo versionado y offline;
9. promover únicamente tras los gates factual, táctico, de calibración, abstención y producto.

## Estado y siguiente fase

Gate 1B y `training_allowed=false` no cambian. La fase 1B.11 corregirá los contratos y consumidores espaciales para que la disponibilidad objetiva no dependa de `human_review_pending`, mantendrá separada toda valoración táctica e impedirá ingerir v1/v2 como etiquetas. No autoriza descargar demos, entrenar, reprocesar 44/44, construir una release ni promover soporte.
