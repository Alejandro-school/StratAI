# Reparación del instrumento de revisión espacial situada

Fecha: 2026-09-02  
Estado: instrumento v2 construido y validado; respuestas humanas pendientes  
Gate 1B: pendiente

## Resultado

El piloto package-only de 1B.10 no era un instrumento humano válido para juzgar
la geometría en una situación de partida. Sus 12 unidades declaraban el contexto
observado como no disponible, dibujaban sectores fuera del mapa y reducían las
rutas a una flecha recta. El hash histórico
`19648364aee7716221eba7ebded9c0c44a71ad6c513e9a6b9a5f7f9a21bf1b06` se
conserva como evidencia de aquella construcción, pero queda **invalidado para
revisión táctica humana y supersedido por v2**. No deben recogerse respuestas con
los formularios v1.

La reparación crea `stratai.spatial_context_review_protocol@2` y los schemas v2
de lote, respuesta y adjudicación. Cada unidad entregada contiene ahora:

- radar real de Mirage registrado con la transformación usada por Replay 2D;
- posición de un participante mediante alias opaco, ronda, tick T0 y
  `availability_tick <= T0`;
- orientación cuando la familia la necesita, y postura/arma/velocidad para las
  rutas cuando están disponibles;
- sector y rayos situados para exposure;
- dirección de entrada y matriz real de nueve probes para cover;
- polilínea completa de centros NAV para exit-route, nunca una flecha recta
  sustitutiva.

La pregunta humana se limita a correspondencia física visible. No pregunta si
la acción era segura, óptima, intencional, recomendable o ganadora.

## Smoke real de una demo

Se procesó exactamente un match distinto, la demo Mirage aprobada
`demo-a11eb0d600ee702c9edb`, SHA-256
`a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`.
Tres pasadas especializadas sobre esa misma fuente produjeron sidecars de
exposure, cover y exit-route; no se abrió una segunda demo.

Los tres validadores independientes pasaron:

| Familia | Filas | Contextos geométricos | Resultado relevante |
| --- | ---: | ---: | --- |
| Exposure | 12.802 | 883 observed | 0 futuro, identidades crudas u outcomes |
| Cover | 12.802 | 883 observed | 21.192 direcciones; probes recomputados |
| Exit-route | 12.802 | 883 observed | 878 rutas físicas; 5 abstenciones por tolerancia NAV |

Las salidas canónicas temporales de unos 736 MB por pasada se eliminaron sólo
después de validar y conservar sus sidecars pequeños. No se modificaron la demo,
los 44 bundles aprobados, packages, support manifests, exports, staging,
`CURRENT` ni releases.

## Lote y formularios entregables

El lote v2 contiene 12 casos observados —4 por familia—, cero respuestas y cero
adjudicaciones. Sus cuatro rutas suman 248 puntos de polilínea NAV. El validador
independiente recomputó protocolo, handbook, packages, radar, IDs, orden,
causalidad, privacidad, hashes visibles y geometría por familia.

- Batch ID: `spatial-context-pilot-f4f947d350253af23ecbca1b`
- Batch SHA-256:
  `bf614a09347433f5118e1b78fc2b5de4607eb8510eb66aebf03587089ae95e3c`
- Render: `mirage_radar_context_overlay@2`
- Respuestas presentes: `0/24`
- Entrenamiento autorizado: `false`

Se generaron dos HTML autocontenidos y ciegos, uno por reviewer. El radar viaja
embebido como `data:image/webp`; no hay `fetch`, almacenamiento del navegador,
red, `audit_context`, respuestas ajenas ni identidad pública. Playwright validó
las tres familias a 1440×1100 y 390×844, con cero errores o warnings de consola
y sin seleccionar ninguna respuesta.

## Verificación

- `go test ./pkg/navigation ./cmd/spatial-review-paths ./cmd/canonical-export`:
  correcto;
- validadores independientes de los tres sidecars: correctos;
- validador independiente del lote v2: 12 unidades, 12 contextos observed,
  4 polilíneas/248 puntos, cero respuestas;
- tests de revisión v1+v2: 20 pasados;
- suite `ai_coach/tests`: 201 pasados;
- `npm run check:all`: correcto —frontend 88, Node 14 pasados/3 omitidos,
  Go completo, backend Python 43 y build Vite—;
- navegador real desktop y móvil: correcto, sin respuestas generadas.

La evidencia detallada queda en
[`docs/evidence/AI_COACH_SPATIAL_CONTEXTUAL_REVIEW_INSTRUMENT_REPAIR_2026-09-02.json`](../../evidence/AI_COACH_SPATIAL_CONTEXTUAL_REVIEW_INSTRUMENT_REPAIR_2026-09-02.json).

## Estado y siguiente acción

La reparación del instrumento está cerrada, pero 1B.11 no lo está. La siguiente
acción es que dos personas distintas abran únicamente sus HTML v2, respondan los
12 casos de forma independiente y entreguen los JSON. Un tercer humano sólo
interviene en conflictos. Ningún agente o script puede simular respuestas. Esta
revisión tampoco autoaprueba Mirage, cambia soporte, habilita entrenamiento ni
autoriza migración o reproceso 44/44.
