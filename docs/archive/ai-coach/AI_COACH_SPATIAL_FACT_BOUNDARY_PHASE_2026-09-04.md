# Cierre 1B.11 — frontera entre hechos espaciales y juicio táctico

Fecha: 4 de septiembre de 2026  
Estado: **cerrada en código y smoke de una demo; Gate 1B sigue pendiente**

## Resultado

La disponibilidad de clearance angular, oclusión del volumen y ruta NAV ya no depende de una opinión humana. Los tres sidecars factuales pasan a contrato `2.0.0` y publican IDs inequívocos:

| Hecho automático | Campo | Schema de fila |
|---|---|---|
| Sectores de clearance geométrico | `FLD-GEOMETRIC-CLEARANCE-SECTORS` | `stratai.observed_exposure_angle_diagnostic@2` |
| Oclusión geométrica por postura/dirección | `FLD-GEOMETRIC-OCCLUSION` | `stratai.observed_cover_diagnostic@2` |
| Transitabilidad dirigida y coste NAV | `FLD-NAV-EXIT-PATH` | `stratai.observed_exit_route_diagnostic@2` |

Cada fila conserva un bloque `tactical_assessment` separado y cerrado:

```json
{
  "status": "unavailable",
  "reason_codes": ["tactical_decision_context_required"]
}
```

El bloque permanece igual tanto si el hecho objetivo está disponible como si falta físicamente. Los paquetes espaciales `@1` se conservan sin modificar para geometría, coordenadas y procedencia; sus placeholders o estados humanos históricos no se propagan al sidecar v2.

## Compatibilidad y consumidores

- El bundle canónico `3.8.0`, sus 44 bundles, lineage, release y `CURRENT` no cambian ni se reinterpretan.
- Los sidecars `@2` siguen siendo candidatos externos al manifest canónico. `DecisionExample` y `TaskEligibility` los declaran `missing` hasta que un contrato sucesor materialice el join; nunca usan formularios humanos como fallback.
- Los IDs mixtos `FLD-EXPOSURE-ANGLES`, `FLD-COVER` y `FLD-EXIT-ROUTE` se conservan por compatibilidad documental, pero se retiran de los prerrequisitos de capacidades.
- El catálogo `1.5.0` tiene 58 campos: los tres nuevos son `DERIVABLE`; eso no vuelve elegible ninguna capacidad en los bundles `3.8.0` actuales.

## Límite del ground truth

Los ocho schemas de `stratai.spatial_human_review_*@1` y `stratai.spatial_context_review_*@2` quedan marcados `deprecated_visual_qa_only`, con autorización de ground truth, evaluación y entrenamiento a `false`. La entrada común `ai_coach.spatial.review.ground_truth` mantiene una allowlist vacía y rechaza de forma cerrada los ocho IDs, los IDs desconocidos y fuentes vacías mediante `TACTICAL_GROUND_TRUTH_SOURCE_FORBIDDEN`.

Esto no elimina el material histórico ni sus hashes. Sólo impide confundir QA visual de rayos, polígonos o rutas con una etiqueta sobre qué decisión táctica era mejor. El contrato humano sucesor se diseñará después de materializar decisiones completas y generar alternativas físicamente viables.

## Validación

- Los paquetes Go de exposure, cover, exit-route, parser y exportador pasan sus pruebas focalizadas.
- `ai_coach/tests` pasa con tests positivos, negativos, `unavailable`, causalidad, recomputación, manipulación de hashes/assets/mapa, separación objetivo/táctica y rechazo de schemas humanos v1/v2.
- `npm run check:all` pasa: 88 pruebas frontend, 14 Node pasadas/3 omitidas, todos los paquetes Go, 43 pruebas backend Python y build Vite.
- El catálogo y los contratos runtime validan 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 58 campos, 28 schemas y 19 reason codes.

## Smoke obligatorio

Se procesó exactamente la demo aprobada `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, en tres pasadas especializadas sobre el mismo match y un intento por pasada. Cada destino era nuevo, vacío y estaba bajo `%TEMP%`.

- Exposure: 12.802 filas; 883 hechos geométricos; 11.919 ausencias físicas; 12.802 juicios tácticos no disponibles.
- Cover: 12.802 filas; 883 hechos geométricos; 21.192 direcciones; 11.919 ausencias físicas; 12.802 juicios tácticos no disponibles.
- Exit-route: 12.802 filas; 883 candidatos; 878 caminos disponibles; 5 ausencias NAV; 11.919 posiciones fuera de cobertura; 883 juicios tácticos no disponibles.
- El validador semántico del bundle canónico terminó con cero errores.
- Demo, GLTF/bin, NAV, tres paquetes y tres manifests conservaron exactamente sus bytes y SHA-256.
- Después de registrar la evidencia se eliminaron las tres salidas temporales, aproximadamente 2,2 GB en total.

Comandos, duraciones, conteos y hashes completos: [evidencia máquina](../../evidence/AI_COACH_SPATIAL_FACT_BOUNDARY_SINGLE_DEMO_SMOKE_2026-09-04.json).

## No autorizado por este cierre

No se descargaron demos, no se reprocesó 44/44, no se creó release, no se promovió soporte, no se recogieron respuestas humanas y no se entrenó ningún modelo. Gate 1B y `training_allowed=false` permanecen sin cambios.

## Siguiente frontera

La siguiente fase debe definir un contrato canónico sucesor candidato y materializar estos tres sidecars en `DecisionExample`/`TaskEligibility` sobre una única demo temporal. Debe conservar `3.8.0` como referencia inmutable y no autoriza migración 44/44, publicación, descarga ni entrenamiento.
