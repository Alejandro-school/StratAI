# Lectura de la hipótesis B — contrato local v3

`stratai.replay_attack_hypothesis@3`, `FocusedAttackHypothesisSession` y adaptador optativo `1.6.0-local.1`. Mantienen el criterio del [contrato v2](replay-attack-hypothesis-v2.md), el Qwen original y sus presupuestos. Esta revisión repara la selección de salida y la presentación del contexto; no acredita que el modelo aplique bien el criterio táctico. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente acción.

## Hechos completos y selección breve

El productor conserva todos los hechos originales, el archivo causal, sus fechas, las restricciones y el transporte reversible @3. `required_reading_facts` ya obliga al presentador a incluir la situación actual, la evolución esencial del objetivo y la incertidumbre protegida, aunque el modelo entregue listas vacías. Pedir al modelo que repita esas referencias consumía salida sin añadir cobertura: el rastro de N10 reproduce la enumeración hasta el límite de 240 segundos.

La gramática nueva excluye esas referencias obligatorias de la selección del modelo. Cada sección factual permite hasta ocho referencias **adicionales**, disponibles y válidas para esa sección: actuales en situación, históricas o acontecimientos en evolución, estados desconocidos/estimados/simulados en incertidumbre. Una sección sin referencias opcionales se genera vacía. El render sigue añadiendo íntegramente todos los hechos obligatorios; el límite de ocho no recorta la lectura ni la entrada. El verificador permite reconstruir esa lectura completa y limita a ocho sólo las referencias adicionales.

La interpretación mantiene una hipótesis opcional con dos/tres identidades actuales distintas, o hasta ocho principios condicionales, o una lista vacía. Se conservan los sobres coherentes de consulta/fin, las consultas causales, los límites de bytes y el rechazo sin reintentos ocultos. La salida cerrada sigue impidiendo que el modelo escriba números, estados del C4, zonas o explicaciones factuales libres.

## Vista espacial legible

El suplemento incluye `spatial_review`, una copia exacta y vinculada de los hechos actuales de cada jugador propio elegible y su zona, sus distancias ya calculadas al punto 2 y las combinaciones canónicas. El adaptador repite ese objeto como un tercer mensaje en JSON legible. Permanece también dentro del contexto completo cuyo hash figura en el recibo. No aporta observaciones nuevas, coordenadas rivales ocultas, una clasificación de cercanía ni una respuesta de ataque.

La consulta espacial de desarrollo reveló sustitución de una zona actual por otra y confusión entre todos los jugadores elegibles y una pareja concreta. La vista legible evita exigir al modelo reconstruir esos vínculos desde el transporte compactado. El prompt distingue las parejas y tríos, mantiene las alternativas y no exige C4 dentro del grupo, intención confirmada o plantado para una posibilidad. No añade un radio métrico ni convierte los puntos de orientación en límites.

## Límite demostrado

Los controles de desarrollo conservados muestran abstenciones y confusión entre grupos al evaluar ataque. Preguntar sólo cercanía produce una respuesta distinta, pero no demuestra aplicación táctica fiable. N01 está cerca del punto 3: no sirve de control retrasado hacia el punto 1. Se corrige expresamente esa clasificación inicial del diagnóstico y se conserva el borrador anterior. Dos fixtures ilustrativos, con los demás datos constantes y la pareja en el punto 1 o en el 2, reciben ambos `near_reference`; no acreditan discriminación espacial. No se transforma esa respuesta en una etiqueta automática de ataque. Una explicación libre de diagnóstico no es una lectura aceptada, una medición geométrica ni una explicación verificable de los mecanismos internos del modelo.

Se mantienen @1/@2 y sus adaptadores para replay histórico. @3 es optativo, local y ajeno a una ruta de producto. Las regresiones y el smoke verifican hechos, referencias, causalidad y representación; la comparación registra la respuesta real del modelo. No se deduce calidad táctica de esas comprobaciones, ni se usa el render manual de cobertura como inferencia. Antes de ampliar el alcance o sustituir el criterio por una región determinista se necesita la revisión experta que exige el plan.
