# ICT-I1.R1 — informe descriptivo desde la replay real

22 de septiembre de 2026. Cierre técnico de la salida legible del alcance
descriptivo acordado. La interpretación táctica sigue pendiente.

`ai_coach.interpretation.replay_report` carga las fuentes verificadas, proyecta
un lado hasta los cortes solicitados y presenta los registros existentes sin
modificar sus hechos. Muestra posición, vida, recursos físicos, C4 y diferencias
entre muestras adyacentes, conservando ausencia y procedencia.

La selección explícita no usa desenlaces. No se inventa una correspondencia de
alias con T1–T5 ni se atribuye a jugadores conocimiento de los datos de sus aliados.
El manifiesto `replay_description_es@1` documenta una presentación offline, no un
nuevo contrato de runtime. Parser, contratos, catálogo, web y gates no cambian.

## Verificación

- `npm run check:all`: correcto, 61,316 s. Suite afectada: 262 pruebas correctas.
  Tras ajustes de estilo: 60 pruebas correctas y Ruff correcto.
- 17 pruebas nuevas: cero/falso/ausencia, no aplicabilidad, texto inerte, selección,
  corrupción, mezcla de partidas/lados, referencias, aislamiento de futuro/enemigos/
  respuestas, guardas de salida y conservación de evidencia.
- Una demo aprobada: `demo-a11eb0d600ee702c9edb`; SHA-256
  `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`.
  Parsing: 77,346 s. Generación del informe con validación de fuentes: 14,316 s.
- Cortes 1389 y 1417, ticks 8524 y 8636: dos registros y diez actor-cortes.
  Hechos iguales al registro completo previamente verificado, excluyendo IDs
  derivados del nuevo build. Ocho hipótesis abstendidas y cero recomendaciones.
- Contraste directo: 10 posiciones, 10 estados de vida, 70 hechos físicos,
  10 diferencias recalculadas y 30 referencias. Hashes correctos; 44 manifiestos
  aprobados intactos. Esta selección conocida no es evaluación independiente.
- El primer intento del comprobador independiente utilizó una ruta de campo
  incorrecta. Se corrigió únicamente ese comprobador y se auditó la misma salida,
  sin repetir parsing ni cambiar el productor para acomodar el resultado.

Los comandos, versiones, hashes, tiempos e inventario están en la
[evidencia](../../evidence/AI_COACH_REPLAY_DESCRIPTION_REPORT_2026-09-22.json)
y en `outputs/stratai-replay-report-20260922/`. El smoke permanece en un nuevo
TEMP; no hubo promoción, sobrescritura, eliminación, adquisición ni entrenamiento.
No se usaron grabaciones ni nuevas partidas generadas.

Excel v8/IC18 permanece intacto: no se reciben nuevos criterios expertos en esta
ejecución. E01–E04 conservan sus dependencias. La siguiente acción pertenece al
[plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md).
