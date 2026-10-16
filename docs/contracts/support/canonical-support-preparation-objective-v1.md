# Contexto causal del objetivo en C0 v1

Extensión optativa `1.0.0-candidate.1`, fase 3E.14. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado y continuación. Une los hechos de [objetivo 3E.4](canonical-support-objective-v1.md) a los cortes exactos del [censo C0](canonical-support-preparation-v1.md), conservando su población y el [contexto físico 3E.13](canonical-support-preparation-physical-v1.md).

## Captura y frontera

Reutiliza `support_objective_observations.json` y `support_observations.json`, producidos por `--ai-coach-support-objective-candidate`. No modifica Go ni añade propiedades. La captura del pawn comprueba actor, conexión, vida, posición y estado vivo observados, además de ID y serie del handle completo. La vida exportada enlaza ese handle con la misma entrega support; el lector comprueba la identidad de fuente/build, hashes y manifest canónico.

`source_sequence`, tick, FrameDone, instancia y época identifican el corte exacto. `order` y `event_sequence` delimitan el prefijo público entregado. Un evento posterior no completa el corte anterior aunque tenga el mismo tick. Otra entrega FrameDone con igual tick tampoco aporta una muestra de objetivo ausente. Los eventos del SDK acreditan disponibilidad, no un instante físico subtick ni atención del jugador.

## Hechos y límites

Se conservan los siete hechos globales de 3E.4: fase pública, bandera nativa de bomba plantada, freeze y warmup, duración nominal de ronda y tiempos transcurridos desde las entregas públicas de fin de freeze y plantado. Los tres hechos por actor son zona de bomba, desactivación y kit. Los bool nativos, ausencia, tipos no soportados, conflictos, `false`, cero y no aplicabilidad mantienen su significado original. `playing` no acredita fin de freeze ni ronda competitiva.

Los tiempos transcurridos usan ticks del parser y el reloj del corte; no son cuentas atrás efectivas, no reconstruyen pausas ni plazos del HUD. Zona de bomba no identifica A/B ni acredita control territorial. Desactivación no prueba progreso, duración ni viabilidad. El contexto no asigna valor táctico a mantener una posición.

Ubicación/tenencia C4, origen/progreso de intentos, reloj efectivo, entrada física, compromiso, contacto legal, NAV físico, velocidad instantánea, readiness, cooperación/intención y economía mantienen sus contratos y dependencias positivas. Ninguna capa posterior se incorpora automáticamente.

## Artefactos

Dos schemas estrictos, con entrenamiento desautorizado, elevan el registro a **90 schemas/19 razones**:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_objective_cuts.jsonl` | `stratai.support_preparation_objective_cut@1` por línea | Una fila por corte C0, hechos globales una vez, hechos propios/aliados por actor y referencias de todos los candidatos |
| `support_preparation_objective_manifest.json` | `stratai.support_preparation_objective_manifest@1` | Fuentes/protocolo por hash, bytes/filas del JSONL, conteos y matriz por par/lado/instancia/época |

Cada actor/corte se almacena una vez aunque intervenga en múltiples pares dirigidos. Identidad, vida, handle, propiedades nativas y prefijo público pertenecen a procedencia. Las filas sin candidatos conservan el corte y listas vacías, sin fabricar features. Un candidato con objetivo ausente conserva ID, T0 y denominador.

La matriz cuenta los 13 componentes globales y por rol, sus estados/motivos, fase observada y disponibilidad conjunta de los tres hechos de jugador. `both_available` sólo significa presencia factual; no acredita apoyo viable. Pares inversos y ventanas solapadas están correlacionados.

## Acceso inicial y lector

`preparation_objective.initial_features(support, objective, candidate_id, actor, ally, sequence, tick, frame)` comprueba la pertenencia exacta al censo C0. Recorta support por ordinal antes de validar; recorre el stream de objetivo hasta la fila exacta o el primer localizador posterior, y recorta eventos por la frontera registrada antes de validar ese prefijo. No recorre el resto del futuro. Una muestra ausente produce ausencia explícita, sin usar el contexto de un anclaje cercano.

Devuelve copias de las posiciones/estado vivo iniciales C0 en `self` y `ally`, los hechos de objetivo por rol bajo `objective` y los hechos públicos en `global`. Excluye IDs, handles, vidas, fuente/build, propiedades nativas, geometría enemiga, evaluación, conteos globales y ejecución posterior. La extensión física conserva su acceso y sus artefactos propios; no se duplican aquí.

La API de prefijo no sustituye la comprobación de integridad completa: `validate_artifact` valida el stream original entero, incluida la frontera frente a todos los eventos, y reconstruye cada fila y toda la matriz. Rechaza filas ausentes, adicionales, reordenadas o alteradas, discrepancias de fuente, schema, hash, bytes y conteos. La evidencia real añade un recálculo independiente que no importa consumidores de aplicación.

El catálogo `1.7.0`, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparación, recomendaciones, entrenamiento y producto conservan su alcance. S2–S3 permanecen abiertos.

## Ejecución aislada

Después de las suites, el productor existente procesa exactamente la misma demo aprobada hacia TEMP nuevo y vacío. Para materializar y releer, desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective `
  --source <support_objective_observations.json> `
  --output <directorio-nuevo-vacio-bajo-TEMP> --require-observed

backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_objective `
  --source <support_objective_observations.json> `
  --output <mismo-directorio-materializado> --validate --require-observed
```

El preflight rechaza destinos fuera de TEMP, enlaces/reparse points, escapes, directorios no vacíos y sobrescrituras antes de cargar la fuente. La verificación exige hechos afectados no vacíos, causalidad, semántica, procedencia, población C0 y contexto físico conservados, rutas reutilizadas y 72 artefactos canónicos. Fuente, 44 bundles y temporales anteriores son de sólo lectura. Las salidas e inventarios se retienen; no autoriza limpieza, adquisición, migración 44/44, publicación, promoción ni cambios de `CURRENT`.
