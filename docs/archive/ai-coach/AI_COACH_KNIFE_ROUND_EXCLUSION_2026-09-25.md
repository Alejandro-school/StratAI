# E01: exclusión de la ronda de cuchillos

El 25/09/2026 el usuario identifica E01 como ronda de cuchillos y pide omitir ese
tipo de ronda del análisis táctico. El [recibo](../../evidence/AI_COACH_KNIFE_ROUND_EXCLUSION_2026-09-25.json)
conserva el literal completo y su contexto. Es una decisión de alcance asistida:
el experto ya había recibido el diagnóstico del inventario y la abstención.
La primera lectura de execute A no se reescribe ni se cuenta una segunda lectura.

La [cola activa](../../../outputs/stratai-knife-exclusion-20260925/active-review-scope.json)
aplica la exclusión por SHA de partida, epoch, instancia y ronda. El lote original
conserva 36 casos. E01 queda fuera de análisis/puntuación tácticos; hay 35 pendientes
de contexto/revisión y cero lecturas activas puntuadas. E01 no se convierte en
acierto, error o abstención correcta, ni se sustituye por otro caso favorable.
La precisión continúa sin calcular. Se declara el cambio posterior a la selección.

La [política registrada](../../../outputs/stratai-knife-exclusion-20260925/round-exclusion-policy.json)
separa clasificación experta y automática. La ausencia de C4, el cuchillo activo
o un número de ronda no acreditan por sí solos ronda de cuchillos. La búsqueda
estructural MCP no estuvo disponible (Transport closed); el barrido de literales
en Python/Go/JSON no localizó un clasificador de producción explícito de cuchillos.
Es una comprobación acotada, no prueba exhaustiva de ausencia de cualquier lógica.
La identificación nativa automática y el reloj general siguen pendientes.

IC27 del [Excel v17](../../../outputs/stratai-knife-exclusion-20260925/StratAI_Plan_del_coach_TP10_v17.xlsx)
añade sólo A32:G32. Se verifican las 14 hojas, valores, fórmulas, estilos, controles,
literal y presentación; v16 se conserva. Se verifican por hash 278 archivos
protegidos, incluidas las fuentes y los artefactos históricos. Detector, renderer,
contratos y corpus no cambian. No hay parsing nuevo ni entrenamiento; pruebas
funcionales y smoke nuevo no aplican a esta recepción y cambio de cola de revisión.

El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija la siguiente acción:
completar elegibilidad de ronda y reloj nativo de las fichas antes de E02, respetando
el criterio ya recibido. Cualquier nueva implementación cumple sus verificaciones
y conserva el test histórico; no se vuelve a preguntar la exclusión aceptada.
