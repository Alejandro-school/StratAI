# Cierre documental — Diseño del primer piloto

Fecha: 5 de septiembre de 2026. Alcance: ejecución del prompt del paso 2 y entrega del [diseño del piloto v1.0](../../ai-coach/AI_COACH_INITIAL_PILOT_DESIGN.md). El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y siguiente acción. Este informe no acredita implementación ni validación humana del piloto.

## Resultado entregado

- Selección de economía como primera pieza factual: acreditar dinero, inventario y contexto disponibles antes de comprar, separando ejecución y reconciliación posterior.
- Once situaciones de diseño en economía, apoyo, mecánica y peek, con información legal, alternativas, límites y práctica asociada.
- Lote propuesto de 40 casos: 16 económicos, 12 de apoyo, 8 mecánicos y 4 espaciales; diez son control aleatorio estratificado. No se han seleccionado ni creado casos de demo.
- Instrumento de revisión independiente de decisiones completas; dos coaches y adjudicación propuesta, sin respuestas humanas inventadas.
- Perfil previo, una o dos prioridades y cinco misiones con oportunidad, excepciones y separación entre realizar, aplicar y aprender.
- Evaluación exploratoria propuesta, umbrales formales conservados en el plan y escenario de coste ilustrativo de 2.150 EUR; no es cotización ni gasto autorizado.

## Hallazgos y decisiones

La lectura de los contratos y productores confirmó que los registros económicos agregados mezclan observaciones de distintos momentos. La captura de freeze y la reconciliación al final de ronda deben conservar su disponibilidad real, sin completar retrospectivamente el estado anterior a comprar.

Se registraron tres desajustes que requieren diseño/versionado futuro: ventana genérica de acción frente a toda la compra de freeze, anclaje de trade en una baja frente al apoyo anterior y ventana genérica de diez segundos frente a perfil entre partidas. El catálogo y los schemas no se han modificado en esta fase.

El grafo de código devolvió `Transport closed`; se consultaron de forma localizada `handlers/economy.go`, `models/block6_models.go` y `parser/block6_export.go`, además del contrato, catálogo y auditoría Mirage. No se ejecutaron productores ni se leyó una demo para generar datos nuevos.

## Pendientes preservados

Revisión real de coaches, materialización del lote, participación de jugadores, aprobación de consejos, contratos de perfil/práctica y prototipo funcional. Gate 1B, entrenamiento y soporte de producto permanecen bloqueados. La deuda de limpieza espacial 1B.12 conserva su estado; esta fase no intentó borrarla ni cerrarla.

El resultado se trasladó al plan v2.1 y al índice documental. El prompt ejecutado se retira al cerrar esta fase; la única siguiente instrucción es el paso 3A, mantenido por el plan. No se ha ejecutado esa implementación dentro del diseño.

## Verificación documental

Se verificaron los destinos locales Markdown del README y de `docs/`, sin enlaces ausentes; 45 IDs citados por el diseño y siguiente prompt existen en el catálogo. `docs/prompts/` contiene únicamente `AI_COACH_ECONOMY_CAUSAL_STATE_PHASE_PROMPT.md`.

Los SHA-256 de las 15 evidencias JSON y del catálogo coinciden antes y después. `git -c core.safecrlf=false diff --check -- README.md docs` terminó con código cero y sin errores. Las comprobaciones adicionales se hicieron con scripts efímeros de Python en lectura; no se añadieron tests ni artefactos runtime.

| Artefacto | SHA-256 de cierre |
|---|---|
| Diseño v1.0 | `4e236d55c40018a3315ef11add7607b0e0e2ff7bfd6dd7cedb8c6311c208c6f0` |
| Prompt de 3A | `5d975fbc5d0a093fc97211895a941d066ffc7e5f3f8cec7be577e0a74ddb60f7` |
| Catálogo preservado | `50ce21b1920de85816feb4895a6475ed151db04f9326277718e53aef0f74b380` |

El cierre es documental: no se ejecutaron suites funcionales ni smoke, ni se modificaron productores, schemas, datos, modelos o releases. Las cifras de cobertura del corpus siguen siendo las de las evidencias históricas, no una medición nueva.
