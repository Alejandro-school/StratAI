# Tenencia causal propia/aliada de C4 en C0 v1

Extensión optativa `1.0.0-candidate.1`, fase 3E.15. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado y continuación. Une los hechos de [tenencia C4 3E.5](canonical-support-c4-v1.md) al corte exacto del [censo C0](canonical-support-preparation-v1.md), preservando el [contexto físico](canonical-support-preparation-physical-v1.md) y el [contexto del objetivo](canonical-support-preparation-objective-v1.md).

## Fuente y unión exacta

Reutiliza `support_c4_observations.json`, `support_objective_observations.json` y `support_observations.json`, producidos por `--ai-coach-support-c4-candidate`. No modifica Go ni añade propiedades. La captura C4 lee de nuevo inventario, referencias y propietario en cada callback support. Los estados se comparten sólo cuando las lecturas frescas son idénticas dentro de la misma instancia/época; no son un inventario arrastrado de una muestra próxima.

El lector exige alias/checksum/build y hashes de las tres fuentes, su vínculo con el manifest canónico, secuencia support, tick, FrameDone, instancia/época y `objective_order` exactos. Comprueba `state_sequence`, `first_cut`, vida/handle del pawn, identidad/serie de objetos y correspondencia de cada slot resuelto. Otro FrameDone con el mismo tick puede tener otro propietario; no sustituye el corte pedido.

## Hechos y límites

Cada actor seleccionado conserva los tres hechos de 3E.5:

| Hecho | Significado |
|---|---|
| `has_c4` | Positivo con inventario y propietario actuales corroborados; negativo sólo con inventario completo y sin contradicción; ausente ante falta de prueba o conflicto |
| `carrier_player_position` | Posición actual del jugador propio/aliado cuando porta C4; `not_applicable:not_a_carrier` con ausencia acreditada y `unavailable:ownership_unavailable` sin prueba suficiente |
| `physical_c4_position` | Conserva `unavailable:not_observed_in_legal_view`; la posición del portador no mide el objeto C4 |

Una prueba positiva puede sobrevivir a una ranura adicional parcial. Una prueba negativa exige cobertura completa. Duplicados, múltiples C4, discrepancia clase/definición o inventario/propietario conservan conflicto; una referencia sin resolver no se convierte en ausencia. Se conservan handle cero, inventario vacío, posición cero, `false` y ausencia con sus tipos originales. Una definición cero no acredita un objeto distinto de C4.

La posición del aliado mantiene el alcance de proxy de información del equipo. No acredita atención, comunicación, responsabilidad táctica ni valor de mantener una posición. No incorpora attachment transportado, ubicación legal de C4 caída, reloj efectivo, progreso de intentos, entrada física, compromiso ni apoyo viable. Sus dependencias positivas y contratos propios se conservan.

## Artefactos y cobertura

Dos schemas estrictos, con entrenamiento desautorizado, elevan el registro a **92 schemas/19 razones**:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_c4_cuts.jsonl` | `stratai.support_preparation_c4_cut@1` por línea | Una fila por corte, contexto por actor seleccionado y referencias de todos los candidatos |
| `support_preparation_c4_manifest.json` | `stratai.support_preparation_c4_manifest@1` | Fuentes/protocolo por hash, bytes/filas del JSONL, conteos y matriz por par/lado/instancia/época |

Se conserva cada corte, incluso excluido o sin pares. Esas filas mantienen listas vacías y no inventan features. Cada actor/corte se almacena una vez aunque participe en múltiples pares dirigidos. Ausencia o conflicto de C4 nunca elimina candidatos, IDs, T0, denominadores, primeras muestras o ejecución posterior.

La procedencia conserva el corte C4, orden del objetivo, estado y primera captura, más el inventario nativo del actor. Las entidades globales permanecen en la fuente enlazada por hash; no se duplican como features. La matriz cuenta estados/motivos de seis hechos por par, tenencia positiva/negativa/insuficiente por rol y disponibilidad conjunta. `both_available` admite observado o no aplicable; no acredita apoyo viable. Los pares y ventanas solapadas están correlacionados.

## Acceso inicial y lector completo

`preparation_c4.initial_features(support, objective, data, candidate_id, actor, ally, sequence, tick, frame)` exige pertenencia exacta al censo. Recorta support, cortes de objetivo y C4, eventos públicos y estados referenciados **antes de validar**. Se detiene en la fila exacta o el primer localizador posterior, sin recorrer el resto del futuro. Un corte C4 ausente devuelve ausencia explícita, sin usar el estado anterior.

Devuelve copias `self` y `ally` de la posición/estado vivo C0 y los tres hechos bajo `c4`. Excluye identidad, vidas, handles, fuente/build, inventarios nativos, entidades ajenas, geometría enemiga, conteos, evaluación y ejecución posterior. El contexto físico y el del objetivo mantienen sus APIs y artefactos propios.

La integridad completa se valida por separado: `validate_artifact` comprueba todas las fuentes y fronteras, reconstruye todas las filas y la matriz y rechaza cambios de origen, corte, estado, actor, hecho, orden, hash, bytes o conteo. El smoke añade una vía independiente sin importar consumidores de aplicación para recalcular tenencia, posiciones y matriz y contrastar C0 y sus contextos previos.

Catálogo `1.7.0`, `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparaciones, recomendaciones, entrenamiento y producto mantienen su alcance. `playing` tampoco acredita fin de freeze o ronda competitiva. S2–S3 permanecen abiertos.

## Ejecución aislada

Tras las suites, el productor existente procesa la misma única demo aprobada a TEMP nuevo/vacío. Desde la raíz:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_c4 `
  --source <support_c4_observations.json> `
  --output <directorio-nuevo-vacio-bajo-TEMP> --require-observed

backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation_c4 `
  --source <support_c4_observations.json> `
  --output <mismo-directorio-materializado> --validate --require-observed
```

El preflight rechaza destinos no temporales, enlaces/reparse points, escapes, directorios no vacíos y sobrescrituras antes de cargar la fuente. Se exige tenencia positiva no vacía, schema, causalidad, semántica, procedencia, población C0 y contextos físico/objetivo conservados, rutas reutilizadas y 72 artefactos canónicos. Fuente, 44 bundles y temporales anteriores son de sólo lectura. Se retienen salidas e inventarios; no autoriza limpieza, adquisiciones, migración 44/44, publicación, promoción ni cambios de `CURRENT`.
