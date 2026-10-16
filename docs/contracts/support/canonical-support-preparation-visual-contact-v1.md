# Entradas visuales propias y aliadas en C0 v1

Extensión optativa `1.0.0-candidate.1` del [censo C0](canonical-support-preparation-v1.md), basada en [entradas visuales 3E.10](../../canonical-support-visual-contact-v1.md). El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina estado y siguiente acción. Conserva candidatos, IDs, T0, exclusiones, ejecución y ocho contextos C0 anteriores.

## Cinco hechos

`stratai.support_preparation_visual_contact_cut@1` contiene una fila por corte, actores seleccionados y pares ordenados. Los hechos por rol son `native_eye_offset`, `reference_fov_degrees`, `flash_clear`, `smoke_clear` y `visual_contact`.

El offset conserva las componentes finitas y tipadas consultadas por el accessor nativo del SDK; cero no es ausencia. No afirma posición mundial ni transformación de la cámara renderizada. El FOV de referencia exige origen/destino uint32 iguales, positivos y menores de 180, con scoped explícitamente falso. Ausencia, default/cero, rango, transición y óptica scoped conservan abstenciones separadas. No se imputa FOV 90 ni interpolación de otra build.

Flash, humo y contacto permanecen `unavailable/null`, respectivamente por `rendered_flash_opacity_unaccredited`, `smoke_volume_and_coverage_unaccredited` y `legal_visual_contact_inputs_unaccredited`. Ningún rayo, FOV o cero de duración/alpha de flash los convierte en contacto positivo o negativo.

## Prefijo, identidad y procedencia

`initial_features(support, timing, data, identifier, actor, ally, sequence, tick, frame)` exige pertenencia exacta a C0 y recorta las tres fuentes antes de validar, incluidas fronteras nativas de estados y entregas. Excluye futuro malformado y actualizaciones posteriores del mismo tick/FrameDone. Devuelve copias de los hechos iniciales y `visual_inputs` por rol; enemigo, rayos, índices, geometría, identidades, ejecución y conteos internos quedan fuera de features.

La fuente verifica corte común con temporización, instancia/época, orden, creación/reemisión/destrucción, snapshots contra entregas, pawn actual vivo/conectado, vida nativa, handle completo/serie y correspondencia de coordenadas actuales. El consumidor C0 exige además continuidad actor/lado/pawn/handle/instancia/época desde la primera muestra del estado retenido. Una discontinuidad abstiene offset/FOV con `native_visual_c0_actor_continuity_unaccredited`; los otros tres motivos permanecen fijos. Ausencia de pawn o estado conserva candidatos. `visual_reference` separa secuencia, frame, estado y vida; las fronteras nativas quedan en `provenance`.

## Artefactos y verificación

Dos schemas estrictos regulan `support_preparation_visual_contact_cuts.jsonl` y `support_preparation_visual_contact_manifest.json`. El manifiesto referencia diez fuentes por tamaño/SHA-256 y el protocolo C0; contiene conteos y matriz por actor/aliado/lado/instancia/época. El lector recompone filas, referencias, hechos/abstenciones, matriz, hashes y diagnóstico físico.

Materializador y lector ejecutan `cmd/support-visual-check` con los assets identificados por hash. El resultado queda separado en `evaluator_diagnostics.static_ray_recomputation`, con ámbito interno fijo. El verificador reconstruye endpoints con otra expresión, pero comparte el motor de intersección; no valida independientemente ese motor ni la suficiencia visual de la malla estática seleccionada, sus materiales, transformaciones u obstáculos dinámicos.

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.support.preparation_visual_contact `
  --source $visualSource --output $newEmptyContextTemp --require-observed `
  --visual-evaluator $visualCheckBinary --maps-root backend/data/maps
```

Añadir `--validate` para releer. `--require-observed` exige candidatos y positivos de offset y FOV de referencia; no exige ni permite contactos inventados. La captura usa `--ai-coach-support-visual-contact-candidate`, sin cambios Go. Productor/materializador rechazan destinos no temporales, no vacíos y sobrescrituras antes de procesar fuentes. Cada cierre exige suites, una pasada nueva de la misma única demo, materializador/lector, recálculo independiente, ocho contextos y canónico intactos, con salidas/inventarios retenidos.

Todos los gates permanecen falsos; entrada física y compromiso son `not_accredited`. S2–S3 siguen abiertos. Contacto legal requiere fuentes positivas de oclusión/materiales/transformaciones, obstáculos dinámicos, humo volumétrico, flash actual y FOV efectivo cuando haya default/transición/scoped. Economía, NAV físico, velocidad instantánea, attachment transportado/C4 caída, ticks/fracciones e intención/cooperación mantienen sus dependencias.
