# Contexto físico propio/aliado en C0 v1

Contrato optativo `1.0.0-candidate.1`, fase 3E.13. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado y siguiente acción. Extiende el acceso factual del [censo C0](canonical-support-preparation-v1.md) mediante la [captura física 3E.2](canonical-support-physical-v1.md). No cambia selección, IDs, T0, exclusiones, denominadores ni descriptores de ejecución de C0.

## Fuentes y unión exacta

Reutiliza `support_physical_observations.json`, captura `same_support_FrameDone_native_physical@1`, y `support_observations.json`. No añade propiedades ni modifica Go. El lector comprueba schemas, identidad de demo/build, hashes de ambas fuentes y manifest canónico. `source_sequence` enlaza una entrega support exacta: tick, FrameDone, época, instancia, actor, handle completo y vida del pawn proceden de esa misma entrega. No se busca un anclaje anterior ni una muestra cercana.

La identidad de arma, vida del objeto, valores nativos de munición y señal/tick originales de recarga permanecen en procedencia. Orientación, walking/ducking, arma, munición y recarga conservan el origen, presencia y límites de 3E.2. `observed=false`, cero observado, `unavailable` y `not_applicable` son distintos. Knife, C4 y utilidad sin cargador no adquieren munición cero; una clase desconocida conserva ausencia.

## Artefactos normalizados

Dos schemas estrictos, ambos con entrenamiento desautorizado:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_physical_cuts.jsonl` | `stratai.support_preparation_physical_cut@1`, por línea | Una fila por cada corte support, candidatos C0 con sus IDs y referencias actor/aliado, contexto y procedencia por actor/corte seleccionado |
| `support_preparation_physical_manifest.json` | `stratai.support_preparation_physical_manifest@1` | Hashes de fuentes y protocolo C0, bytes/filas del JSONL, conteos y matriz por par/lado/instancia/época |

Cada actor/corte se almacena una vez aunque participe en varios pares. Todos los pares dirigidos seleccionados conservan una referencia explícita; no se submuestrean. Los cortes excluidos o sin pares también tienen fila, con listas vacías. Las exclusiones, posiciones y ejecución originales siguen en C0 y en su fuente enlazada, sin duplicar ni modificar esos artefactos. Esta normalización no reduce la población ni convierte pares correlacionados en decisiones independientes.

La matriz cuenta candidatos, disponibilidad conjunta de ocho componentes y estados/motivos por rol. `both_available` admite `observed` o `not_applicable`, no acredita readiness, apoyo viable ni superioridad táctica. Un contexto físico ausente mantiene al candidato y sus denominadores.

## Intervalo anterior y disponibilidad

La velocidad media XYZ usa sólo la muestra entregada inmediatamente anterior y la actual. Exige fase `playing`, época/instancia/reloj/cadencia compatibles, ticks separados exactamente por `sample_step`, y actor utilizable con el mismo lado, vida y handle. Nunca salta un hueco para encontrar una posición más antigua.

`segundos = (tick_actual − tick_anterior) / tick_rate`; el vector divide las diferencias XYZ por ese intervalo y la rapidez horizontal es la norma XY. Se conserva el método `quantized_position_interval_average@1`, extremos y duración. No es velocidad instantánea ni inicio de movimiento, distancia recorrida sobre una curva, aceleración o prueba de tránsito físico.

La primera muestra queda seleccionada con `no_previous_sample`. Los demás límites son `previous_cut_incompatible`, `previous_actor_unavailable` y `pawn_life_handle_or_side_changed`. Cada rol evalúa su propia continuidad; la ausencia del historial de un rol no inventa ni borra el del otro. Dos FrameDone con igual tick son cortes distintos y no definen un intervalo de duración positiva.

## Acceso inicial y lector completo

`preparation_physical.initial_features(support, physical, candidate_id, actor, ally, sequence, tick, frame)` recorta support por ordinal antes de validar y exige el candidato/par/corte exactos de C0. Recorre el stream físico ordenado hasta el corte exacto o el primer localizador posterior, sin validar el resto del futuro. Una fila posterior malformada después del corte exacto queda fuera. Un stream sin corte exacto mantiene ausencia física, sin reutilizar otra fila; la integridad completa se comprueba aparte.

Devuelve copias `self` y `ally` de las posiciones/estado vivo iniciales C0, hechos físicos y velocidad media disponible. Excluye aliases, vidas, handles, hashes, contextos nativos, geometría enemiga, conteos, clasificación posterior y ejecución. La información aliada sigue siendo un proxy de equipo de la demo, no atención o comunicación observadas.

El materializador escribe exclusivamente en un TEMP dedicado, nuevo y vacío; rechaza rutas externas, enlaces y sobrescrituras antes de cargar fuentes. `validate_artifact` verifica el manifest y vuelve a derivar todas las filas y la matriz; rechaza cambios de candidato, fuente, corte, identidad, hecho, orden, conteo, bytes o hash, así como filas ausentes o adicionales. El smoke añade recálculo independiente de hechos y matriz y contraste completo con la población C0 anterior.

## Comandos

Después de las suites, la misma única demo aprobada se procesa con el selector Go existente `--ai-coach-support-physical-candidate` hacia TEMP nuevo. Desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_physical --source $physicalSource --output $newEmptyTemp --require-observed
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_physical --source $physicalSource --output $newEmptyTemp --validate --require-observed
```

Se conservan fuente, 44 bundles y temporales anteriores como sólo lectura. Cada intento nuevo queda retenido e inventariado. Catálogo, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparaciones, recomendaciones, entrenamiento e interfaz temporal mantienen su alcance. Entrada física, compromiso, contacto legal, NAV físico, velocidad instantánea, C4 transportada/caída, temporización completa, cooperación e intención requieren sus propias pruebas positivas. La unión física básica no cierra S2–S3.
