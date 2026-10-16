# Candidato de observación económica v1

La [versión 2](canonical-economy-candidate-v2.md) añade sondas de inicio y correspondencia competitiva. Este contrato y su lector conservan su significado para artefactos v1.

Contrato `1.0.0-candidate.1`. Describe una salida aislada de desarrollo; el [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado y siguiente trabajo. No modifica los bundles `3.8.0`, `DecisionExample`, la elegibilidad del catálogo ni los consumidores de producto.

## Qué registra

El observador toma dinero, gasto nativo acumulado, inventario, armadura, casco, kit, zona de compra y valor nativo del equipo cuando las propiedades de la demo están presentes. El inventario contiene armas y utilidad con identidades opacas. Un inventario vacío observado y un inventario desconocido son estados diferentes.

Cada hecho contiene `value`, `status`, `source`, `observation_tick` y `availability_tick`. Un dato no observado tiene valor y ticks `null`; cero y `false` se conservan cuando la propiedad existe. No se convierte una propiedad ausente, un handle inválido o un tipo nativo desconocido en un cero observado. Los handles de armas se contrastan con la entidad y su número de serie; no se reutiliza una identidad por compartir índice.

El contexto recoge marcador por lado y seis variables del servidor: compra, freeze, dinero máximo, dinero inicial, rondas máximas y prórroga. Su presencia no acredita por sí sola una combinación completa de reglas. Las tablas heredadas de precios y economía se referencian mediante versión y SHA-256 con `reference_only_not_applied`; su compatibilidad permanece sin verificar. Este contrato no calcula precios, bonus de derrota, recompensas, ingresos futuros ni superioridad de una compra.

## Tres compartimentos

| Compartimento | Observación | Uso permitido |
|---|---|---|
| `initial_state` | Primer `FrameDone` disponible tras `RoundStart` | Estado inicial factual con disponibilidad explícita |
| `execution` | Callbacks estrictamente después del tick inicial y antes del tick de fin de freeze; snapshot del primer frame tras ese cierre | Describir ejecución observada; nunca completar el estado previo |
| `reconciliation` | Snapshot del primer frame tras `RoundEnd`, callbacks de los límites y posteriores, diferencias con agregados antiguos | Diagnóstico posterior; nunca features previas |

`prepurchase_status=available` requiere jugador conectado, captura en el mismo tick de inicio, dinero/inventario/gasto observados, gasto nativo cero y ninguna actividad de objetos observada para ese actor antes de la captura. Una primera muestra tardía conserva su tick real y queda no disponible como precompra. Si el freeze ya está cerrado, tampoco acredita precompra. La etiqueta sólo acredita estas condiciones factuales; no significa que todas las reglas, alternativas o intenciones del equipo sean conocidas.

`callback_sequence` conserva el orden del parser. No demuestra orden del motor dentro del tick ni precisión subtick. Los callbacks en los propios ticks de inicio o fin de freeze quedan fuera de la ejecución estricta. El snapshot de freeze conserva su tiempo real: puede incluir cambios del tick límite y no debe usarse para adjudicar esas acciones a una ventana más estrecha.

Las instancias de ronda se numeran por orden de aparición. Reinicios con el mismo número visible conservan instancias distintas y no se unen a agregados heredados ambiguos. El lado pertenece a cada observación y no identifica un roster permanente. Un participante que desaparece después de ser observado en la ronda conserva hechos ausentes.

## Recogida, compra y transferencias

- `pickup` significa callback de recogida. Su atribución inicial es `purchase_or_pickup_unresolved`; no demuestra gasto ni compra.
- `drop` y `refund` conservan el tipo observado, sin inventar precio o devolución monetaria.
- Una recogida puede referenciar el último drop anterior de otro actor del mismo lado y de la misma entidad: `prior_allied_drop_same_entity`. La referencia se consume al recoger, no cruza rondas y no prueba intención, cooperación o idoneidad táctica.
- El gasto nativo no sustituye una lista completa de compras. No se reconstruyen armaduras, devoluciones o recogidas desconocidas por diferencia de dinero.

## Identidad, acceso y verificación

Los actores usan alias locales `p0001`; los objetos, `w00001`. El puente de reconciliación es SHA-256 de `source_sha256 + "\n" + canonical_player_id`. No se incluyen nombres ni Steam IDs originales en el candidato. Son claves de unión locales a la partida, excluidas de futuras señales predictivas.

El archivo completo contiene observaciones de ambos lados para auditoría. **No se entrega entero como contexto del coach.** `initial_team_state` es un acceso limitado sobre un candidato previamente validado: exige T0 exacto y cinco jugadores del mismo lado con precompra disponible; devuelve una copia del contexto y del equipo propio. Excluye rival, ejecución y reconciliación. Todavía no existe un consumidor de decisiones económicas ni prueba de atención o comunicación del jugador.

El validador independiente comprueba los dos schemas, hashes, puente con la economía canónica, identidad de partida, conteos, procedencia, tiempos, cronología, particiones, motivos de abstención y referencias de drops. Recalcula las diferencias monetarias contra el artefacto canónico cuyo hash figura en su manifest. Estas diferencias comparan fronteras de observación diferentes: un delta distinto de cero no es automáticamente un error económico.

## Archivos y ejecución

- [Schema de datos](../../../ai_coach/contracts/observed_economy_candidate.schema.json): `stratai.observed_economy_candidate@1`.
- [Schema de manifest](../../../ai_coach/contracts/economy_candidate_manifest.schema.json): `stratai.economy_candidate_manifest@1`.
- [Validador y acceso inicial](../../../ai_coach/contracts/validate_economy_candidate.py).

El modo `--ai-coach-economy-candidate` procesa una demo una sola vez y exporta el bundle normal junto a `observed_economy_candidate.json` y `economy_candidate_manifest.json`, bajo `smoke_run_<alias>`. No activa productores espaciales. Requiere una carpeta ya creada, vacía, dedicada y bajo `%TEMP%`; rechaza enlaces, junctions, escapes y sobrescrituras antes de parsear. Los modos candidatos no se combinan en esta ejecución.

```powershell
# $approvedDemo, $sourceAlias y $sourceSHA256 identifican una misma demo aprobada.
$economyRunRoot = Join-Path ([IO.Path]::GetTempPath()) "stratai-3a-smoke-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $economyRunRoot -ErrorAction Stop | Out-Null
Push-Location backend/go-service
try {
  go run ./cmd/canonical-export --demo $approvedDemo --output $economyRunRoot `
    --alias $sourceAlias --checksum $sourceSHA256 --maps-root ../data/maps `
    --build-id <identificador-de-build> --ai-coach-economy-candidate
} finally { Pop-Location }
backend/venv/Scripts/python.exe -B -m ai_coach.contracts.validate_economy_candidate `
  "$economyRunRoot/smoke_run_$sourceAlias/economy_candidate_manifest.json" `
  --expected-source-sha256 $sourceSHA256 --require-observed
```

Además se valida semánticamente el bundle según la [guía de validación aislada](../../operations/ai-coach-validation.md). La evidencia pequeña se guarda antes de retirar la salida grande. Entrenamiento y soporte de producto permanecen en `false`; la integración de ventanas, acciones y oportunidades económicas necesita su propio contrato.
