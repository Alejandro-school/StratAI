# Temporización nativa propia y aliada en C0 v1

Extensión optativa `1.0.0-candidate.1` del [censo C0](canonical-support-preparation-v1.md), basada en [temporización nativa 3E.9](canonical-support-weapon-timing-v1.md). El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina estado y siguiente acción. Conserva candidatos, IDs, T0, exclusiones, ejecución y siete contextos C0 previos.

## Hechos y abstenciones

`stratai.support_preparation_weapon_timing_cut@1` contiene una fila por corte y los actores propios/aliados de sus pares ordenados. Conserva ocho hechos: `primary_remaining_seconds`, `primary_deadline_state`, `secondary_remaining_seconds`, `secondary_deadline_state`, `postpone_remaining_seconds`, `postpone_deadline_state`, `pawn_remaining_seconds` y `pawn_deadline_state`.

El plazo GameTime_t del servicio del pawn exige tipo, valor finito, reloj causal acreditado y controles de pausa. Su restante es `max(0, deadline - now)`; el estado es `pending`, `elapsed` o `paused` según el contrato fuente. Un cero de semántica no acreditada conserva abstención. Un plazo vencido no acredita `can_fire` ni readiness. Los seis hechos del arma conservan las abstenciones GameTick_t/fracciones y la no aplicabilidad de Knife/C4/utilidad: no se introduce conversión de ticks sin prueba positiva.

## Corte e identidad

El acceso inicial exige candidato, actor, aliado, secuencia, tick y FrameDone exactos. Recorta siete entradas antes de validar: support, movimiento, físico, objetivo, tenencia C4, reloj y temporización; incluye fronteras de estados/entregas y eventos públicos. Una actualización posterior del mismo tick/FrameDone tampoco completa C0. La fuente física se une por `source_sequence`, sin inventar campos de frame o tick.

La validación fuente comprueba instancia/época, orden causal, snapshots contra entregas, creación/reemisión/destrucción, pawn actual, vida nativa, handle completo/serie, arma física actual y propiedad bilateral. Desde la primera muestra del estado de pawn retenido hasta C0 se exige continuidad de actor/lado/vida/handle/instancia/época; una discontinuidad produce `native_timing_c0_actor_continuity_unaccredited`. Ausencia de pawn, arma o prueba de reloj conserva candidatos y denominadores.

`initial_features(support, motion, physical, objective, ownership, clock, data, identifier, actor, ally, sequence, tick, frame)` devuelve copias de los hechos iniciales C0 y `weapon_timing` por rol. Identidades, referencias y testigos quedan fuera de las features. `timing_reference` conserva estados/vidas nativas de pawn y arma, corte y secuencia del testigo; `provenance` separa orden, fronteras de estados/entregas y `clock_witness`.

## Materialización y lectura

Dos schemas estrictos regulan `support_preparation_weapon_timing_cuts.jsonl` y `support_preparation_weapon_timing_manifest.json`. El manifiesto referencia por tamaño/SHA-256 nueve fuentes y el protocolo C0. Conserva conteos y matriz actor/aliado/lado/instancia/época para los ocho hechos en ambos roles. El lector valida la cadena fuente completa y recompone filas, referencias, testigos, hechos, abstenciones, matriz y hash; rechaza cambios, omisiones, filas extra y reordenaciones.

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.support.preparation_weapon_timing `
  --source $timingSource --output $newEmptyContextTemp --require-observed
```

Añadir `--validate` para releer. `--require-observed` exige candidatos y positivos de ambos hechos del pawn; las abstenciones del arma no se reinterpretan. La captura usa `--ai-coach-support-weapon-timing-candidate`, sin cambios Go. Productor y materializador rechazan destinos no temporales, no vacíos y sobrescrituras antes de procesar la fuente. Cada cierre exige suites, una pasada nueva de la única demo aprobada, auditoría independiente y preservación de siete contextos previos/canónico, con nuevas salidas e inventarios retenidos.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` siguen en `false`; entrada física y compromiso permanecen `not_accredited`. S2–S3 siguen abiertos, junto con contacto visual, NAV físico, intención/cooperación, velocidad instantánea, geometría transportada/caída y readiness completa.
