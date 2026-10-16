> **ARCHIVO HISTÓRICO — NO EJECUTABLE.** Esta instrucción fue retirada del catálogo activo y se conserva para trazabilidad. No ejecutes los pasos siguientes desde esta copia: manda el [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md). El archivo no demuestra que se haya realizado la limpieza temporal ni cierra 1B.12.

# Fase 1B.12 — pendiente operativo de limpieza y cierre

La implementación, las pruebas y el smoke de 1B.12 **ya se ejecutaron y pasaron** el 5 de septiembre de 2026. No vuelvas a generar el candidato ni a procesar la demo. El plan canónico es la única fuente de orden y estado.

Consulta:

- `docs/archive/ai-coach/AI_COACH_SPATIAL_FACTS_CANONICAL_MATERIALIZATION_PHASE_2026-09-05.md`;
- `docs/evidence/AI_COACH_SPATIAL_FACTS_CANONICAL_MATERIALIZATION_SINGLE_DEMO_SMOKE_2026-09-05.json`;
- `docs/canonical-spatial-candidate-v3.9.md`.

## Único trabajo restante

La evidencia ya contiene los hashes de los 83 archivos generados, 746.944.298 bytes. La revisión automática rechazó dos órdenes de limpieza con `blocked by policy`, incluso una limitada a artefactos inventariados con comprobación SHA-256. No se ejecutó ninguna orden de borrado. No intentes eludir esa restricción mediante otro intérprete.

Destinos pendientes, ambos creados por esta ejecución bajo `%TEMP%`:

- salida: `C:/Users/kerch/AppData/Local/Temp/stratai-1b12-smoke-hwelcj03`;
- control y logs: `C:/Users/kerch/AppData/Local/Temp/stratai-1b12-control-qj5etbed`.

1. Comprueba si una acción del usuario ya retiró esos destinos. Si todavía existen, sólo límpialos cuando la política lo permita. Verifica primero sus rutas absolutas bajo `%TEMP%`, identidad del directorio, ausencia de reparse points y correspondencia con los artefactos de la evidencia. No toques ninguna demo fuente, bundle aprobado, corpus, staging, exports, release ni `CURRENT`.
2. Confirma la ausencia de la salida y el control, actualiza `temporary_output_cleanup` y el estado de la evidencia, y registra el cierre en informe, plan, contrato, catálogo y README. Mantén Gate 1B pendiente y `training_allowed=false`.
3. Sólo después de cerrar elimina este prompt y deja exactamente un sucesor de fase 1B.13 para el déficit medido de muestreo espacial en T0.

## Frontera del sucesor, todavía no ejecutable

El candidato `3.9.0-candidate.1` compone y verifica 918 ejemplos/2.754 joins. Sólo 10 decisiones tienen muestra exacta a 1 Hz: 1 tiene los tres hechos presentes, 9 están fuera de cobertura y 908 no tienen muestra exacta. Peek tiene 246 decisiones y ningún hecho espacial presente.

El sucesor debe medir qué T0 tienen snapshot fuente exacto y cuáles requieren captura causal; materializar los tres hechos desde estados acreditados; conservar ausencia explícita cuando no pueda acreditarse T0; versionar cualquier cambio de muestreo/reportes/joins; comparar contra `3.9.0-candidate.1` en el mismo match y preservar lectura/rollback de `3.8.0`. No usar nearest, interpolación ni carry-forward no acreditados, no ampliar paquetes geométricos en esa fase y no convertir juicios humanos, oracle u outcomes en features. Los ocho schemas humanos v1/v2 siguen rechazados como ground truth. Requerirá pruebas y smoke de una sola demo en destino temporal nuevo.

Esta instrucción no autoriza descargas, entrenamiento, reproceso 44/44, nuevas releases, publicación, promoción ni ejecución adelantada de 1B.13.
