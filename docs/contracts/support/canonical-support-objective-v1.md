# Contexto causal del objetivo — candidato v1

Contrato `1.0.0-candidate.1`, fase 3E.4. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y continuación. Añade hechos del objetivo a los mismos [anclajes v1](canonical-support-anchors-v1.md), preservando [contexto físico](canonical-support-physical-v1.md), [diagnósticos NAV](canonical-support-navigation-v1.md), IDs, selección y T0. No acredita apoyo viable ni una decisión previa a la entrada.

## Captura y unidad

El selector exclusivo `--ai-coach-support-objective-candidate` produce el bundle base `3.8.0`, los dos sidecars anteriores y `support_objective_observations.json`. Captura propiedades frescas dentro del callback support/replay y una traza de entregas públicas del SDK fijado `demoinfocs v5.2.0`. No usa la proyección de final de ronda ni el reloj histórico con defaults de 115/40 segundos.

Cada muestra conserva `source_sequence`, tick del parser, contador FrameDone, instancia, época y ordinal de entrega. Cada evento tiene secuencia, ordinal y esos relojes; el corte registra la última secuencia incluida. Un evento posterior al corte queda excluido aunque comparta tick. El ordinal acredita orden de entrega, no orden físico subtick. Los eventos Source 2 pueden ser sintetizados por el SDK desde callbacks de entidades: su tick es de disponibilidad, no un tick original de red acreditado.

La traza contiene sólo `round_start`, `freeze_end`, `bomb_planted`, `bomb_defused`, `bomb_exploded` y `round_end`. Omite actor, ganador, sitio y coordenadas. Las señales terminales sólo tienen efecto después de su entrega; nunca completan un T0 anterior. El prefijo se reinicia por instancia y época, incluso si se repite el número de ronda. Las instancias preliminares siguen sin acreditación competitiva.

## Hechos y disponibilidad

| Campo | Fuente y significado | Límite |
|---|---|---|
| `objective_phase` | Prefijo público: inicio de ronda o primer fin de freeze → `preplant`; plant → `planted`; defuse/explode/fin de ronda → `resolved`. | Sin origen en la misma instancia/época queda ausente. Una contradicción positiva con `m_bBombPlanted`, o fase terminal heredada mientras el corte replay vuelve a `playing`, invalida la fase. Otro fin de freeze sin nuevo `RoundStart` no borra silenciosamente la contradicción. |
| `bomb_planted`, `freeze_period`, `warmup` | Bool nativo presente de `m_pGameRules.m_bBombPlanted`, `m_bFreezePeriod`, `m_bWarmupPeriod`. | Ausente, tipo inesperado y `false` son distintos. No se convierte un entero cero en bool. |
| `round_duration_seconds` | Entero nativo no negativo de `m_pGameRules.m_iRoundTime`. | Es duración nominal observada, no tiempo restante ni configuración imputada. Cero presente se conserva. |
| `round_elapsed_seconds` | Diferencia de ticks desde la entrega `freeze_end` de la misma instancia/época, dividida por el tick rate del corte. | Tiempo transcurrido en el reloj del parser; no reconstruye pausas ni el contador efectivo del HUD. |
| `bomb_elapsed_seconds` | Diferencia de ticks desde la entrega pública `bomb_planted`, mientras la fase acreditada sigue plantada. | Cero es válido. Fuera de esa fase es `not_applicable`; sin fase/origen válido queda ausente. No usa duración nominal de C4 ni calcula una cuenta atrás. |
| `in_bomb_zone` propio/aliado | Bandera nativa `m_bInBombZone` del pawn exacto. | No identifica A/B ni demuestra intención de plantar. No es una comprobación geométrica de pertenencia a un site o de control territorial. |
| `defusing` propio/aliado | Bandera nativa `m_bIsDefusing` del pawn exacto. | No acredita progreso físico completo, duración restante, kit utilizado o atención. |
| `has_defuse_kit` propio/aliado | Bool nativo `m_pItemServices.m_bHasDefuser` del pawn exacto. | No acredita viabilidad ni conveniencia de desactivar. |

Los escalares conservan presencia, tipo nativo y valor originales. Las propiedades del jugador exigen actor presente, conexión, vida y posición observadas, handle completo y serie coincidentes con el pawn actual. No se arrastran estados de otro pawn o muestra. Se capturan participantes opacos para permitir lectura por cada actor; el acceso inicial devuelve sólo el propio y su aliado seleccionado.

La fase global no expone ubicación del objetivo ni tenencia enemiga. Tenencia/ubicación propia o aliada, sitio legalmente conocido, progreso de intentos y cuenta atrás efectiva **no están implementados por este contrato**. Requieren captura positiva y comprobación de disponibilidad específica; no se declara que la demo carezca de esas fuentes. Estado en el servidor no implica información legal para ambos lados. La posición SDK de la bomba puede depender del portador o conservar estado previo y no se reutiliza como feature.

## Artefactos, lector y controles

Schemas estrictos: [observaciones](../../../ai_coach/contracts/support_objective_observations.schema.json) y [contexto](../../../ai_coach/contracts/support_objective_context.schema.json). Registro runtime: **68 schemas/19 razones**. Catálogo `1.7.0`, parser `v16`, quality `12`, base `3.8.0`, `DecisionExample` y `DEC-TRADE-001` conservados.

El materializador reutiliza el de 3E.3; conserva las serializaciones de anclajes, contexto físico y NAV. El nuevo informe enlaza por bytes/SHA-256 observaciones, anclajes y contextos anteriores; la fuente enlaza el manifiesto canónico completo. Una discrepancia de fuente, build, tick, frame, instancia, época, vida o frontera de eventos se rechaza. El lector reconstruye el informe y ejecuta también la validación anterior de NAV/contexto físico/base.

Una fila por actor → aliado / instancia / anclaje. La matriz conserva denominador, acreditación competitiva, fases observadas y estados/motivos de los 13 campos globales y propios/aliados; no cuenta direcciones correlacionadas como decisiones independientes. Disponibilidad parcial no elimina anclajes ni habilita recomendaciones.

`initial_features(support, objective, anchor_id, actor, tick, frame)` valida exclusivamente el prefijo hasta el corte solicitado y devuelve copias de los hechos. Excluye identidades, vidas, referencias, ganadores, posiciones enemigas, etiquetas competitivas y observaciones posteriores. Falta de muestra exacta devuelve ausencia en todos los campos; no usa el vecino temporal.

`decision_eligible`, `comparison_allowed`, `training_allowed` y `product_supported` siguen en `false`. NAV físico, entrada previa, contacto, cooperación e intención conservan sus límites. Este contexto no acredita urgencia táctica, llegada a tiempo ni que el jugador consultara un reloj.

## Ejecución y verificación

Compilar en TEMP `cmd/canonical-export` y `cmd/support-navigation`. Desde `backend/go-service`, después de las pruebas:

```powershell
& $supportBinary --demo $approvedDemo --output $supportRunRoot --alias demo-a11eb0d600ee702c9edb --checksum a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262 --maps-root ../data/maps --build-id $buildID --ai-coach-support-objective-candidate
```

Desde la raíz, con otro directorio TEMP nuevo/vacío:

```powershell
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.objective --source $objectiveSource --output $objectiveOutput --evaluator $navigationBinary --maps-root backend/data/maps
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.objective --source $objectiveSource --output $objectiveOutput --evaluator $navigationBinary --maps-root backend/data/maps --validate
```

Los destinos se comprueban antes de leer/procesar; se rechazan no temporales, no vacíos, enlaces y sobrescrituras. El cierre exige pruebas públicas y AI Coach, una única demo aprobada, recálculo independiente de prefijos/relojes/matriz, controles anteriores y validación canónica completa. Se separan variantes reales y fixtures. TEMP se conserva inventariado con hashes; no se borra ni se pide limpieza. No autoriza migración 44/44, adquisición, entrenamiento o promoción.
