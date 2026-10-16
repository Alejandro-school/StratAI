# Fase 1B.10 — protocolo humano espacial y lote piloto Mirage

Fecha de cierre: 2026-09-01  
Estado: cerrada a nivel de protocolo y preparación; la revisión humana real sigue pendiente.

## Resultado

Se fijó `stratai.spatial_human_review_protocol@1` en la versión `1.0.0` y se construyó un lote piloto determinista, exclusivamente de revisión, a partir de los paquetes Mirage congelados de `exposure_angle`, `cover` y `exit_route`. El lote contiene 12 unidades: cuatro por familia. No se abrieron demos, no se capturaron identidades, no se ejecutó revisión humana y no se inventaron respuestas, clasificaciones ni adjudicaciones.

El lote no concede soporte científico ni habilita entrenamiento. Todos los elementos permanecen en `pending_review`; los campos de autorización de soporte y entrenamiento son falsos.

## Contrato de revisión

El protocolo separa de forma explícita:

- geometría objetiva e inmutable procedente del paquete congelado;
- contexto observado, que en este piloto de preparación se declara no disponible;
- respuestas independientes de `reviewer_a` y `reviewer_b`;
- adjudicación por una tercera persona distinta cuando exista conflicto.

Las opciones neutrales son `supported`, `not_supported`, `missing_context`, `not_evaluable` y `abstain`. Los estados permitidos son `pending_review`, `reviewed`, `needs_adjudication`, `adjudicated`, `rejected` y `unavailable`. Una respuesta o adjudicación nunca reescribe geometría ni concede soporte automáticamente. Cualquier cambio de geometría invalida las revisiones anteriores y exige una nueva versión.

Los cuatro contratos JSON Schema Draft 2020-12 cubren protocolo, lote, respuesta y adjudicación. El manual operativo versionado define independencia, cegamiento, privacidad, evidencia, abstención y resolución de conflictos.

## Construcción y validación

Comando de construcción, ejecutado dos veces en destinos temporales nuevos y vacíos:

```text
python -m ai_coach.spatial.review.build_pilot --output-root <directorio-temporal-nuevo>
```

Comando de validación independiente para cada ejecución:

```text
python -m ai_coach.contracts.validate_spatial_review_pilot <directorio-temporal>/spatial_review_pilot_batch.json
```

Ambas ejecuciones produjeron exactamente 36.533 bytes y el mismo SHA-256:

```text
19648364aee7716221eba7ebded9c0c44a71ad6c513e9a6b9a5f7f9a21bf1b06
```

Las duraciones fueron 0,957 s y 0,917 s para la construcción, y 0,919 s y 0,907 s para la validación. El protocolo embebido se ancla al SHA-256 `1d7089845f7618224bf7856c2d3c1c2cc8bc06986c9cb982b1bb7e96a59b89c3`; el manual, a `b0a10ac2fa62b5f2e11f5bf4259743dcfea85f9a14d34fd294a40b7b45938818`.

El constructor y el validador fallan de forma cerrada ante destinos no temporales, no vacíos, enlaces o puntos de reanálisis, sobrescrituras, hashes incoherentes, paquetes alterados, selecciones no deterministas, identidades expuestas o datos humanos prellenados. La selección se reconstruye de forma independiente desde los inventarios fuente y la semilla versionada.

## Verificación

- `python -m pytest ai_coach/tests`: 195 pruebas superadas.
- `python -m ruff check ai_coach`: sin incidencias.
- `python -m ai_coach.contracts.validate_contract`: contrato válido, 21 esquemas runtime registrados.
- `npm run check:all`: suite raíz completa superada, incluido build de frontend.
- Pruebas específicas: 14 casos de protocolo, determinismo, privacidad, rechazo y manipulación superados.
- Los seis hashes de paquete y manifiesto protegidos permanecieron idénticos antes y después de ambas construcciones.

La puerta de smoke de una demo no aplica: este cierre sólo consume paquetes congelados y no cambia parsing, exportadores, hechos derivados ni contratos runtime de demos. Por tanto, se procesaron exactamente cero demos.

## Límites y siguiente acción

La fase demuestra que el instrumento, las vistas ciegas, la trazabilidad y la validación están preparados; no aporta todavía evidencia humana. El Gate 1B permanece pendiente y el entrenamiento sigue deshabilitado.

La siguiente fase es 1B.11: ejecutar el piloto con personas reales autorizadas, dos revisores independientes y un adjudicador separado, conservando intactos los paquetes y sin simular respuestas si no hay disponibilidad humana.

La evidencia compacta y reproducible está en `docs/evidence/AI_COACH_SPATIAL_HUMAN_REVIEW_PROTOCOL_MIRAGE_PILOT_2026-09-01.json`.
