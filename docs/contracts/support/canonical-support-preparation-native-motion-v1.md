# Movimiento nativo propio y aliado en C0 v1

Extensión optativa `1.0.0-candidate.1` del [censo C0](canonical-support-preparation-v1.md), basada en [movimiento nativo 3E.8](canonical-support-native-motion-v1.md). El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina el estado y la siguiente acción. No modifica selección, IDs, T0, exclusiones, ejecución ni los seis contextos C0 previos.

## Contrato y hechos

`stratai.support_preparation_native_motion_cut@1` normaliza una fila por corte, con actores propios/aliados seleccionados y todos sus pares ordenados. Cada actor conserva los seis hechos de 3E.8: `native_velocity`, `native_horizontal_speed`, `native_speed`, `ground_contact`, `ground_reference_state` y `movement_mode_code`. Las magnitudes requieren un vector mundial nativo actual, finito, tipado y concordante, con parent/attachment acreditados. La ausencia no equivale a cero. Velocidad base, históricos y diferencias de posiciones no completan la velocidad instantánea.

Contacto positivo exige bit OnGround y referencia actual de suelo resuelta; contacto negativo exige bit falso y sentinel inválido. Handle cero puede identificar el mundo. Referencia `world`, `entity` o `none` no acredita ruta física. MoveType_t conserva su código nativo, sin etiquetas semánticas inventadas. La demo de desarrollo acredita suelo/modo; los positivos de velocidad mundial permanecen limitados a fixtures hasta disponer de fuente real positiva.

## Corte y continuidad

El acceso inicial exige candidato/actor/aliado/secuencia/tick/FrameDone exactos. Recorta support, cortes/estados/nacimientos/entregas de posición y cortes/estados/entregas de movimiento antes de validar. Las fronteras declaradas excluyen incluso futuro malformado y entregas posteriores del mismo tick/FrameDone. Posición aporta el corte nativo común; su geometría no deriva movimiento.

El consumidor 3E.8 contrasta predecesores, creación/reemisión/destrucción, snapshots contra entregas, pawn actual, vida nativa, handle completo/serie, instancia y época. Además, desde la primera muestra del estado retenido hasta C0, exige continuidad de actor/lado/vida de pawn/handle/instancia/época. Una discontinuidad produce `native_motion_c0_actor_continuity_unaccredited`, sin eliminar candidatos. Un pawn no disponible conserva `current_motion_pawn_unavailable`. No se exige entrega nueva en cada sample.

`initial_features(support, position, motion, identifier, actor, ally, sequence, tick, frame)` devuelve copias de los hechos iniciales C0 y `native_motion` por rol. Identidades, referencias nativas, matriz y ejecución quedan fuera. `motion_reference` separa estado, vida nativa, vida de pawn, secuencia y frame; `provenance` separa corte/orden y fronteras de estados/entregas.

## Materialización y lectura

`support_preparation_native_motion_cuts.jsonl` y `support_preparation_native_motion_manifest.json` usan dos schemas estrictos. El manifiesto referencia por tamaño/SHA-256 las siete fuentes y el protocolo C0, conserva conteos y matriz actor/aliado/lado/instancia/época. El lector valida la cadena completa de fuentes, recompone filas, referencias, estados/abstenciones, matriz y hash; rechaza cambios, omisiones, filas extra y reordenaciones.

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.support.preparation_native_motion `
  --source $motionSource --output $newEmptyContextTemp --require-observed
```

Añadir `--validate` para releer. `--require-observed` exige candidatos y positivos de suelo/referencia/modo, conservando la ausencia real de velocidad. La captura usa `--ai-coach-support-native-motion-candidate`, sin cambios Go. Productor y materializador rechazan destinos no temporales, no vacíos y sobrescrituras antes de procesar la fuente. El cierre exige suites, una nueva pasada de la misma única demo aprobada, auditoría independiente, conservación de C0/contextos/canónico y salidas/inventarios retenidos.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` siguen en `false`; entrada física y compromiso quedan `not_accredited`. No acredita intención, cooperación, contacto visual, NAV físico, readiness completa ni apoyo viable S2–S3. Todas sus dependencias y abstenciones permanecen abiertas.
