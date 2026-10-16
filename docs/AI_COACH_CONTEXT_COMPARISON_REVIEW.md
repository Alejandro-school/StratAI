# CS3: alcance de viabilidad comparada aceptado

El usuario indica **«Aplica CS3»** el 05/10/2026. Se aplica el alcance A/B/C, 16 episodios exploratorios —8 desarrollo y 8 reserva— y dos iteraciones como máximo. [Recibo y atribución](../outputs/stratai-context-comparison-20261005/framing.json). No se vuelve a solicitar este encuadre. El [plan canónico](AI_COACH_IMPLEMENTATION_PLAN.md) determina estado y siguiente acción. CS1 fija qué debe expresar una lectura y CS2 proporciona hechos consultables; CS3 debe comprobar qué método integra esos hechos y transfiere criterios a situaciones diferentes. La continuación local gratuita queda recibida el 05/10, como se detalla debajo. La interpretación y su calidad todavía deben comprobarse.

## Alcance propuesto

**Decisión de ejecución recibida el 05/10/2026:** después de explicar la opción gratuita, el usuario indica «Listo. Lo tenemos. Implementa CS». Se continúa CS3 con Qwen 3.5 9B mediante Ollama local, comprobado en el equipo; coste de proveedor cero, sin envío externo ni descarga por el agente. [Literal y atribución](../outputs/stratai-context-comparison-ollama-20261005/framing.json). La opción anterior de OpenAI/10 USD no se elige ni se ejecuta. No cambia el criterio experto, el banco, las particiones, el máximo de dos iteraciones ni la necesidad de revisión independiente.

Comparar tres maneras de producir lecturas bajo exactamente la misma entrada factual, corte, perspectiva y suplementos:

| Variante | Qué comprobar |
|---|---|
| A: descripción y reglas acotadas | Referencia técnica de lo que ya puede acreditarse con hechos y criterios explícitos; no basta acertar A/B |
| B: principios y ejemplos seleccionados | Reutilización de criterios generales y recuperación de pocos ejemplos relevantes con procedencia; sin convertir respuestas libres en etiquetas |
| C: intérprete semántico preentrenado aislado | Si integra situación, evolución, hipótesis justificadas e incertidumbre usando la misma evidencia, con citas y sin inventar información |

La C depende de seleccionar mecanismo/modelo, tratamiento de datos y coste antes de cualquier llamada externa. Primero podemos preparar entradas, rúbrica, selección y protocolo local. No se entrenan modelos ni se conecta el experimento al coach del producto. El detector y sus salidas originales continúan congelados como referencia histórica; no se comparan aciertos frente a un método que recibió más información.

Se mantiene la propuesta acotada de 16 episodios: 8 de desarrollo y 8 reservados, como máximo dos iteraciones. La reserva se separa por partida y participantes y se sella antes de ajustar métodos. No llamar independientes a casos ya vistos o empleados para diseñar reglas. Comprobar primero si el material existente satisface esas condiciones; si no, declarar el déficit y pasar la adquisición/evaluación por su gate propio. Dieciséis episodios sirven para explorar viabilidad, sin garantizar suficiencia estadística. No implica reauditar los 34 casos.

## Criterio de juego y ejemplos

Cada lectura debe presentar situación, evolución, interpretación e incertidumbre. Evaluar por afirmación: evidencia citada, disponibilidad, información pertinente omitida, hipótesis alternativas y prudencia. Acertar destino no compensa omitir un hecho que cambia el significado. El resultado posterior sólo verifica lo ocurrido y nunca entra como información previa de la decisión.

- E06 es una regresión de desarrollo: compañero en A y posibilidad de reunirse justifican preferencia A, con B posible. El C4 no fundamenta esa preferencia.
- E17 es movilización inicial con evolución todavía insuficiente; no establecer un mínimo universal de segundos.
- E24 requiere integrar planta acreditada y limitar el recuento rival parcial; no comparar comprensión de postplant con la abstención de un detector de preparación.
- E13/E25/E30 mantienen elegibilidad unknown; no etiquetar cuchillos.
- Para CT, conservar el foco recibido: patrones de T, respuesta defensiva, efecto y adaptación, con explicación. Una derrota o una entrada conseguida no prueban automáticamente error defensivo. Repetición y adaptación requieren antecedentes comparables disponibles; corregir exige alternativa viable mejor respaldada. El encuadre operativo se hará sobre casos causales reales, sin pedir de nuevo este foco general.

Estos casos ya conocidos no forman la reserva independiente. Los ejemplos seleccionados sirven para mostrar principios y límites, no para enseñar manualmente todas las combinaciones. Ningún criterio convierte distribución en intención, proximidad en apoyo eficaz ni ausencia de dato en cero.

## Dependencias y excepciones

CS2 entrega un intervalo de ronda. No suministra historial entre rondas ni percepción/comunicación acreditadas. Un CT sin esos antecedentes puede describir hechos actuales; no puede certificar un patrón repetido de T o una adaptación de varias rondas. No invertir reglas T para fabricar criterio CT.

La comparación exploratoria puede limitarse a familias con datos suficientes, identificando otras como no evaluables. La calidad de recomendaciones y la valoración de defensa CT requieren casos y criterio experto adicional antes de implementación dependiente. La decisión posterior del usuario fija Qwen/Ollama local, alias y cero coste de API; no autoriza adquirir nuevas fuentes, entrenar, promover o publicar. La valoración táctica independiente y CT operativo conservan sus gates.

## Verificación propuesta

1. Sellar selección, particiones, entradas y suplementos por variante antes de generar salidas. Evitar fuga de respuestas humanas, IDs de casos o resultados futuros.
2. Dar igual información y presupuesto de consulta a A/B/C; registrar referencias realmente consultadas, coste y duración. Si una variante añade datos, tratarla como otro experimento.
3. Verificar mecánicamente citas, causalidad y conflictos. Revisar con rúbrica las afirmaciones y omisiones; no inferir etiquetas tácticas de textos libres.
4. Hacer como máximo dos ajustes usando sólo desarrollo; sellar salidas reservadas sin retocar el método tras verlas. Evaluación humana independiente y separación por partida/jugador conforme al protocolo vigente.
5. Explicar mejoras, fallos y límites por familia. No promover un método por una tasa global de A/B ni declarar intérprete general a partir de este piloto.

**Alcance recibido:** incluir el intérprete semántico preentrenado en el experimento exploratorio, sujeto a elegir modelo, coste y datos antes de su ejecución. Los criterios CS1 y el foco general CT permanecen aceptados. La referencia A describe hechos; B recupera principios y ejemplos condicionales, sin asignar etiquetas a respuestas libres. Su verificación mecánica no demuestra suficiencia CS1 ni calidad táctica. [Protocolo local](AI_COACH_CONTEXT_COMPARISON_PROTOCOL.md).

**Ejecución local posterior:** C y las entradas de reserva ya están generados; se conservan originales y fallos, sin puntuación táctica. [Resultado](../outputs/stratai-context-comparison-ollama-20261005/RESULTADO.md). La siguiente acción del plan es la revisión humana independiente, sin repetir estos criterios de diseño.
