# Evaluar entrada y apoyo en Mirage

Rúbrica documental v0.1, 17 de septiembre de 2026. Propuesta del agente basada en las [respuestas Q1 y sus aclaraciones](../evidence/AI_COACH_Q1_EXPERT_RESPONSES_2026-09-17.json), aplicada de forma documental a [F01-K1 v0.2](AI_COACH_F01_K1_REVIEW_v0.2.md). No es un detector implementado, una nueva versión de contrato ni una evaluación táctica independiente.

El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción. Esta rúbrica desarrolla el [diseño colectivo ICT-D1](AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md) y respeta la [definición de producto](../AI_COACH_PRODUCT_DEFINITION.md), el [contrato de capacidades](../AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) y las [dependencias C0](AI_COACH_SUPPORT_C0_DEPENDENCIES.md).

## 1. Pregunta que debe resolver

**Con la información disponible en ese momento, ¿qué parece estar intentando hacer el equipo, cómo contribuye este jugador y hay una alternativa realizable que merezca recomendar?**

Se contestan por separado esas tres partes. Se puede interpretar un plan sin poder juzgar la calidad de una decisión. Una carencia que impide afirmar «podía cubrir ese duelo» no impide describir el movimiento propio ya observado.

El ámbito es entrada y apoyo en Mirage (`CAP-ENTRY-001`, `CAP-TEAM-001`). Utilidad, posición, información, objetivo y tiempo aportan contexto; no se acredita aquí la implementación completa de esas otras familias. Economía sólo entra como recursos e historial disponibles, sin reconstruir compras. R1–R8 son referencias internas de este documento, no nuevos IDs del catálogo.

## 2. Unidad de revisión y tratamiento de la evidencia

La unidad es **un actor, un equipo, un corte y una pregunta concreta**, dentro de una secuencia de la misma ronda. El corte descriptivo no se convierte automáticamente en el instante anterior a una decisión. Para corregir una decisión habría que identificar además cuándo se tomó y qué información precedía a esa acción.

Registrar:

- Mapa, lado, ronda e instancia, corte exacto, reloj, perspectiva y alias locales.
- Hechos y fuente/versión, momento de ocurrencia y de disponibilidad, incluida la antigüedad de información rival usada.
- Lectura del plan y función del actor, señales que la respaldan, alternativas todavía compatibles y qué hecho las distinguiría.
- Qué cambió frente al corte anterior, sin reescribir la lectura que era razonable con el prefijo anterior.
- Juicio sobre decisión y juicio sobre ejecución por separado, límites y motivo de abstención cuando corresponda.

### Cuatro tipos de afirmación

| Tipo | Qué permite decir | Qué necesita |
|---|---|---|
| Hecho | «T4 porta C4 en este corte». | Registro atribuible y comprobado, con tiempo y alcance. |
| Interpretación | «El despliegue sugiere entrada B». | Señales conjuntas y contexto; alternativas y procedencia de la lectura. No acredita intención mental. |
| Capacidad física | «Podía cubrir ese ángulo o llegar al apoyo». | Prueba de los componentes físicos y temporales pertinentes. La plausibilidad humana no los sustituye. |
| Recomendación | «Convenía realizar esta otra acción». | Alternativa parametrizada, viable y mejor respaldada, información legal y gates de comparación. |

En una ficha de interpretación usar **respaldada en este contexto**, **varias lecturas compatibles**, **contradicha por los datos mostrados** o **información insuficiente**, con razones. Son estados documentales propuestos, no probabilidades calibradas. «Respaldada» en una interpretación no significa capacidad física o decisión correcta. Una contradicción sólo se declara cuando la fuente mostrada puede demostrarla; un dato ausente no contradice por sí solo una hipótesis.

No hay suma de puntos ni una nota global que convierta varias señales débiles en una recomendación. El juicio prescriptivo conserva las tres salidas del producto: alternativa superior respaldada, opciones equivalentes/condicionadas o abstención. Si falta evidencia, no se inventa un porcentaje.

## 3. Criterios de evaluación

### R1. Interpretar el plan que se está construyendo

- **Señales:** distribución, dirección y secuencia del movimiento, C4, recursos y acciones propias; rival conocido e historial sólo cuando estén disponibles. Q1–Q3 y Q35 sustentan el enfoque.
- **Lectura:** entrada probable, presión, búsqueda de información, espera o cambio de objetivo son hipótesis contextuales. En el ejemplo de Q1, avance conjunto, disparos y bomba refuerzan entrada; uno o dos mirando sin avanzar pueden buscar información. No se exigen esos tres indicios en toda entrada ni un mínimo de jugadores.
- **Alternativas y cambios:** un split puede mantener opciones y construirse durante la ronda. Nueva información puede justificar adaptación; una pausa no acredita abandono y un rush no exige continuidad incondicional. Comodidad o intención subjetiva quedan desconocidas si no se han aportado.
- **Abstención:** si el prefijo permite varias lecturas, conservarlas e identificar qué dato falta para distinguirlas. No usar un lanzamiento, muerte o victoria posterior para hacer cierta una intención anterior.
- **Verificación:** congelar cada prefijo, justificar cada hipótesis con fuentes de ese prefijo y revisar de nuevo al añadir un cambio. No exigir la misma etiqueta tras nueva información ni cambiar la etiqueta anterior sólo por conocer el desenlace.

### R2. Reconocer avance, control y continuidad

- **Señales:** quién avanza, qué espacio ocupa y deja libre, orientación/movimiento y amenazas disponibles. La entrada puede abarcar una zona o recorrido; Q1 y Q4 no fijan una frontera geométrica universal.
- **Lectura:** distinguir intención probable de entrar, avance observado y control que pueda justificarse. Ocupar una zona usual del rival puede aportar control e información y, a la vez, dejar una defensa expuesta.
- **Alternativas y cambios:** mantener, ceder, explorar o reagruparse pueden ser razonables según objetivo y riesgo. Ocupar una coordenada no prueba que la zona esté limpia ni que el intento haya terminado.
- **Abstención:** si no consta continuidad suficiente, no fechar inicio, consolidación o fin como hechos exactos. Los criterios finales de consolidación/fin siguen parciales; no se rellenan con una distancia o un reloj elegidos por conveniencia.
- **Verificación:** contrastar posiciones y tiempos, separar ocupación observada de control interpretado y comprobar cobertura/amenazas cuando una afirmación las requiera. La anterioridad física de S2 conserva su dependencia; esta definición contextual no la cierra.

### R3. Relacionar funciones, equipamiento y C4

- **Señales:** posición relativa, trayectoria, arma, utilidad, portador y contribución al despliegue. Q5–Q6 explican por qué puede tener sentido que un jugador quede separado.
- **Lectura:** proponer funciones de apoyo con utilidad, avance, protección del portador o separación con finalidad táctica. Una persona puede cambiar de función durante la ronda.
- **Alternativas y cambios:** un jugador aislado puede aportar presión, información o cortar rotaciones, pero estar solo no lo convierte en lurker. Un arma sugiere posibilidades y límites, no una posición obligatoria.
- **Abstención:** no atribuir orden, rol permanente, regreso obligatorio al grupo o abandono sin evidencia suficiente. La necesidad de reintegrarse sigue dependiente del caso.
- **Verificación:** asociar cada función inferida a señales y alternativas; revisar si la secuencia posterior la mantiene o modifica sin atribuir conocimiento previo. C4 en T4 es un hecho distinto de «el equipo lo protege», que requiere su propia justificación.

### R4. Respetar la información disponible

- **Señales:** contactos acreditados, reloj/antigüedad, información del objetivo e historial previo. Q7, Q10–Q12 y Q14 destacan contexto, comunicación y tiempo.
- **Lectura:** describir información disponible según la fuente y perspectiva. Distribución aliada vista en una revisión de equipo no acredita qué conocía personalmente un actor.
- **Alternativas y cambios:** valorar decisiones bajo los escenarios aún compatibles cuando comunicación o conocimiento no consten. Una presión sin C4 conocido puede justificar una respuesta distinta de un ataque confirmado.
- **Abstención:** no inferir silencio por falta de voz registrada, ausencia rival por un hueco de datos ni conocimiento exacto a partir de la visión retrospectiva. No fijar una caducidad táctica universal para un contacto.
- **Verificación:** comprobar disponibilidad antes del corte, separar rival oculto y desenlace, y registrar qué parte del juicio depende de información desconocida. La descripción del equipo puede continuar aunque se abstenga la valoración individual.

### R5. Comprobar qué apoyo era posible

- **Señales:** acción a apoyar, estado y recursos del compañero, posición/orientación, caminos/ángulos, obstáculos, reloj y amenaza pertinente. Q5–Q8 y Q14 contextualizan el apoyo.
- **Lectura:** distinguir intención probable de acompañar, cobertura de otra amenaza, oportunidad de intercambio y apoyo físicamente viable. Cercanía, compañero vivo o flecha de orientación no demuestran capacidad de ayuda.
- **Alternativas y cambios:** acompañar, mantener una función diferente, usar utilidad o reagruparse sólo son candidatos. Conservar la función actual también debe compararse; no se presupone que acercarse sea mejor.
- **Abstención:** sin los componentes físicos necesarios, no afirmar «estaba demasiado lejos», «llegaba a tiempo», «podía hacer trade» o «debía esperar». Un plazo vencido del arma, NAV candidato o munición ambigua no completan disponibilidad efectiva.
- **Verificación:** acreditar por separado geometría, conectividad, contacto, estado del arma y margen pertinente, usando las fuentes que exija esa afirmación. Recalcular los componentes posibles por una vía independiente. Expresar cooperación futura como condición, no como acción garantizada.

### R6. Valorar utilidad y coordinación

- **Señales:** inventario, lanzamiento, trayectoria/explosión cuando consten, posición de aliados y rivales conocidos, efecto observado y acción apoyada. Q7–Q8 y Q17–Q20, incluida la aclaración 18.
- **Lectura:** separar disponer de una granada, lanzarla, producir un efecto y contribuir a la acción. Una flash que explota después del asomo puede seguir ayudando; el lugar y el efecto sobre el compañero importan.
- **Alternativas y cambios:** entrar en sigilo o cubrir otro ángulo pueden justificar no lanzar. Dos granadas del mismo tipo necesitan contraste de espacio, tiempo y función antes de llamarlas redundantes.
- **Abstención:** no declarar una flash mala sólo por llegar después, buena sólo por ganar el duelo ni eficaz por haberse lanzado. Cegar al aliado es un efecto perjudicial señalado por el experto; un veredicto sobre la decisión completa requiere el resto del contexto y alternativas.
- **Verificación:** reconstruir orden y disponibilidad, contrastar efectos pertinentes y distinguir ejecución de decisión de usarla. Si falta fuente visual/humo/flash suficiente, limitarse a lanzamiento o inventario observado; no simular eficacia a partir de una lectura táctica.

### R7. Decidir si existe una corrección justificable

- **Señales:** acción concreta, plan probable, bando, reloj, recursos, información, alternativas y coste de abandonar la función actual. Q8, Q14, Q29, Q33–Q35.
- **Lectura:** separar decisión, ejecución, mérito rival y resultado. Entradas sucesivas o utilidad duplicada pueden motivar revisión; no son un error automáticamente. Perder no obliga a encontrar un culpable.
- **Alternativas y cambios:** comparar la acción observada con mantenerla y con alternativas físicamente acreditadas. Especificar actor, lugar, momento/condición y cooperación necesaria. Conservar equivalencias cuando no haya ventaja respaldada.
- **Abstención:** no llamar error sin una alternativa disponible y mejor sustentada, ni asignar ventaja numérica inventada. Porcentajes de éxito brutos, una victoria o la preferencia del revisor no sustituyen comparación con soporte e incertidumbre.
- **Verificación:** revisión de alternativas y razones bajo la misma información, comprobación física independiente y evaluación separada de desarrollo. Se mantienen los gates y umbrales del plan; no se rebajan por disponer de una rúbrica.

### R8. Convertir una mejora respaldada en aprendizaje

- **Señales:** impacto del problema, recurrencia por oportunidad comparable, dificultad y evolución documentada. Q29–Q32 y Q35 priorizan decisiones/contexto, ejemplo y misión.
- **Lectura:** elegir una conducta concreta para trabajar, explicar su relación con el plan y usar un ejemplo comprensible. Las una o dos prioridades activas del producto no se llenan con correcciones sin soporte.
- **Alternativas y cambios:** conservar una prioridad, ajustar la dificultad o practicar antes fuera de partida depende de evidencia de ejecución y aplicación; no se fija aquí un número de repeticiones que el usuario no haya acordado.
- **Abstención:** sin mejora respaldada, no crear una misión correctiva para completar el formato. Sin oportunidades posteriores, registrar «sin oportunidad»; sin datos suficientes, no afirmar aprendizaje.
- **Verificación:** enlazar cada misión al consejo, la oportunidad y la conducta observable; contrastar aplicación, retención y transferencia por separado. Completar la tarea o ganar rondas no demuestra por sí solo progreso ni efecto causal de la práctica.

## 4. Aplicación documental a F01-K1

Fuentes: [ficha presentada](AI_COACH_F01_K1_REVIEW_v0.2.md), [evidencia de preparación](../evidence/AI_COACH_F01_K1_REVIEW_PREPARATION_2026-09-17.json) y [lectura literal del experto](../evidence/AI_COACH_F01_K1_EXPERT_REVIEW_2026-09-17.json). Se reutilizan las comprobaciones retenidas; no se vuelve a extraer la demo ni se reconstruye K2.

**Mirage, primera ronda, T, 0–0 y 5v5. Corte tick 8524, frame 8542, secuencia 1389, instancia 2, época 0. Quedan 103 segundos y han transcurrido 12 desde el fin de freeze.** La perspectiva es descripción retrospectiva del equipo propio; no acredita todo el conocimiento individual.

| Afirmación del caso | Nivel y resultado de revisión | Límite que conserva |
|---|---|---|
| T1, T2, T3 y T4 avanzan en la dirección mostrada durante la ventana anterior. | Hechos descritos en la ficha; posiciones y movimiento previamente corroborados. | No prueban ruta física ni instante de compromiso. No se importan los disparos del ejemplo hipotético de Q1. |
| T4 porta la bomba. | Hecho de tenencia corroborado. | La posición del portador no reemplaza la transformación mundial del objeto adjunto. |
| T5 queda detrás, tiene flash y humo, lleva humo equipado y está prácticamente quieto el último segundo. | Hechos del corte previamente revisado. | Quietud no prueba espera deliberada. Tener utilidad no demuestra que pudiera producir un efecto concreto. |
| El plan es entrar en B; T5 se encarga de la utilidad y el grupo protege a T4. | Interpretación contextual ya expresada por el experto. Compatible con los hechos presentados. | No se rebaja ni se pide de nuevo esta lectura. Tampoco se eleva a intención cierta, cobertura física o coordinación comunicada. |
| T5 está mal colocado o demasiado lejos; el grupo debería esperar. | Información insuficiente para ese juicio prescriptivo. | No hay alternativa viable superior acreditada, efecto de utilidad ni apoyo físico completo. No se acusa un error. |
| La jugada está bien ejecutada, ganará espacio o terminará en B. | No acreditado por este corte. | No se añade futuro ni se convierte la interpretación del plan en evaluación de ejecución o desenlace. |

### Texto de demostración para mostrar la lectura

Texto redactado por el agente con la lectura ya recibida, no salida de un modelo:

> El despliegue sugiere una entrada en B: el grupo avanza con T4 llevando la bomba y T5 concentra la utilidad por detrás. Su separación puede encajar con esa función. Este corte todavía no permite juzgar si podía apoyar un duelo concreto o si alguno debía actuar de otra manera.

En este caso **no se presenta una corrección, acción positiva evaluada ni misión correctiva**. Sí se muestra la interpretación y su evidencia. La lectura de plan no ocupa una prioridad de mejora como si se hubiera identificado un error.

### Qué requeriría un siguiente corte

K1 basta para preparar la rúbrica y comprobar que no se mezclan niveles. No basta para contrastar una actualización temporal. Para esa pregunta, el siguiente material mínimo sería el primer cambio relevante posterior respaldado de la misma ronda, con los mismos alias, reloj, intervalo y perspectiva. No se presupone qué hará T5 ni se busca una confirmación del plan. K2 no se ha preparado ni evaluado en esta entrega.

## 5. Cómo comprobar que la interpretación y el consejo se sostienen

### Ahora: revisión del diseño y del anclaje

Comprobar trazabilidad de R1–R8 a Q1/producto, integridad de las fuentes, separación de hechos/interpretaciones, fidelidad de la aplicación a K1 y límites. Son comprobaciones documentales, no aciertos de una IA ni validación empírica de las 75 casuísticas.

### Cuando exista una implementación: comprobaciones dirigidas

Estos doce casos son especificaciones de prueba. No son doce rondas reales, no se han ejecutado contra un motor y no incorporan nuevas etiquetas humanas.

| ID | Cambio o riesgo que se probará | Comportamiento exigido | Criterio |
|---|---|---|---|
| V01 | Añadir o cambiar desenlace posterior a un corte congelado. | La lectura del prefijo y su evidencia no cambian por ese futuro. | R1, R4 |
| V02 | Añadir información nueva disponible en el siguiente corte. | La lectura puede cambiar; se explica el hecho nuevo sin reescribir la anterior. | R1, R2 |
| V03 | Mantener separación de T5 pero variar contexto de utilidad/función. | No asignar lurker, abandono o error sólo por estar separado. Una nueva lectura requiere señales. | R3, R7 |
| V04 | Retirar traza de comunicación o contacto. | Conocimiento/comunicación quedan desconocidos, sin inventar silencio ni información exclusiva. | R4 |
| V05 | Mostrar cercanía, orientación o NAV sin prueba física suficiente. | No afirmar cobertura, trade o ruta viable. | R5 |
| V06 | Proponer flash posterior al asomo con efecto aliado no acreditado. | No dar veredicto por reloj solo; conservar la dependencia del efecto. | R6 |
| V07 | Variar posición/efecto acreditado de una flash manteniendo el mismo timing. | La valoración del efecto puede cambiar, con fuente explícita; victoria no suple causalidad. | R6, R7 |
| V08 | Mostrar dos humos o granadas del mismo tipo. | Examinar espacio, solapamiento temporal y función antes de declarar redundancia. | R6 |
| V09 | Mantener decisión/prefijo y cambiar victoria por derrota. | No convertir el resultado aislado en error o acierto; ejecución se evalúa aparte. | R7 |
| V10 | Proponer una alternativa sin lugar, tiempo, viabilidad o soporte. | No publicar corrección; conservar la causa concreta de abstención. | R5, R7 |
| V11 | Introducir hueco temporal, cambio de vida/identidad o perspectiva. | No coser una transición como continua; localizar la carencia sin borrar hechos independientes. | R2, R4 |
| V12 | No encontrar oportunidades de aplicar una misión o sólo registrar que se completó. | «Sin oportunidad» o «práctica completada», sin inventar fracaso, dominio o eficacia causal. | R8 |

Para ejecutar V06/V07 no se fabricará una simulación física presentada como verdad: harán falta fixtures de fuente comprobables o episodios acreditados. Las alteraciones V01/V09 son controles del sistema, no nuevas partidas o éxitos tácticos.

### Evaluación humana y soporte antes del uso prescriptivo

1. Separar desarrollo abierto, validación y test bloqueado por partida; los cortes de la misma ronda/partida no son observaciones independientes. F01 y Q1 son desarrollo conocido, nunca test ciego.
2. Seleccionar oportunidades sin filtrar sólo por muerte, resultado o confirmación de hipótesis. Incluir claridad, ambigüedad, contradicción y carencias reales; conservar una parte no seleccionada por el desacuerdo del sistema. Ningún muestreo o adquisición se ejecuta en esta entrega.
3. Presentar al revisor prefijo, perspectiva, acción y alternativas que hayan pasado comprobación física. Recoger interpretación propia, razones, preferencia/equivalencia/abstención y confianza; conservar desacuerdos y adjudicación sin borrar votos.
4. Medir por separado fidelidad factual, atribuciones sin soporte, fuga temporal, consistencia de actualización, calidad de decisión/ejecución y abstenciones con su motivo. Informar cobertura por oportunidad y resultados agrupados por partida. Para varias lecturas defendibles, no imponer una sola etiqueta «correcta».
5. Los porcentajes necesitan denominador, oportunidades comparables, soporte efectivo e incertidumbre. Se aplican íntegros los mínimos y gates del [plan](../AI_COACH_IMPLEMENTATION_PLAN.md#7-condiciones-de-avance-y-calidad); esta rúbrica no los sustituye ni alcanza. Una frecuencia mayor no prueba por sí sola que cambiar una acción cause una mejora.

## 6. Explicación y misión: formato preparado

Cuando R7 permita una corrección, la ficha deberá completar lo siguiente con evidencia, sin rellenar carencias con texto plausible:

| Parte | Contenido requerido |
|---|---|
| Situación | Ronda/corte, actor, plan probable y hechos decisivos. |
| Acción y alternativa | Qué hizo y qué podía hacer, con lugar, momento/condición y coste de abandonar su función actual. |
| Justificación | Diferencia respaldada, alternativas equivalentes cuando proceda, incertidumbre y dato que cambiaría el consejo. |
| Ejemplo | Corte/replay suficiente para comprenderlo, sin convertir futuro oculto en motivo anterior. |
| Misión | Conducta ligada a esa mejora y condición concreta en la que practicarla; no una orden para todas las entradas. |
| Comprobación | Oportunidades posteriores comparables, conducta aplicada, excepción justificada, no aplicada o no observable, con su evidencia. Separar decisión, ejecución y resultado. |
| Ajuste | Mantener o adaptar según evolución y dificultad observadas. Un umbral de dominio requiere concreción posterior. |

Ejemplo **condicional de estructura**, no consejo para F01: si en otro episodio se acredita que un apoyo podía facilitar una acción, que la alternativa coordinada era realizable y que estaba mejor respaldada, la misión puede consistir en practicar esa coordinación cuando vuelva a aparecer la misma oportunidad. Debe especificarse entonces qué apoyo, qué acción y qué condiciones. No se prescribe «esperar siempre la flash» ni se considera ese texto una misión ya asignada.

## 7. Qué queda entregado y qué sigue abierto

- Entregado: ocho criterios con señales, alternativas, abstención y verificación; aplicación documental a K1; doce especificaciones de prueba; formato de explicación/misión.
- Pendiente: contrastar actualización en otro corte, calibrar la rúbrica sobre casos independientes y acreditar las dependencias de cualquier afirmación física/prescriptiva. Consolidación/fin, reintegración y progresión se concretarán sólo cuando el alcance los necesite.
- Conservado: respuestas Q1, tres aclaraciones, anotaciones del Excel v4, lectura K1 y 18 hipótesis v0.1 pendientes. F04-K3/F05-K3 siguen excluidas hasta reparar sus premisas. No se altera el Excel porque no se han recibido nuevas respuestas ni se aprueba esta propuesta como criterio humano.
- Sin cambio funcional: no motor, producer, schema, parsing, corpus, CURRENT, entrenamiento o publicación. S2–S3 siguen abiertos y la elegibilidad no cambia. Suites funcionales y smoke no aplican a esta preparación documental.

El [informe de preparación](../archive/ai-coach/AI_COACH_ENTRY_SUPPORT_RUBRIC_PHASE_2026-09-17.md) y la [evidencia](../evidence/AI_COACH_ENTRY_SUPPORT_RUBRIC_2026-09-17.json) registran las comprobaciones realmente realizadas. El plan fija la continuación; las pruebas futuras de esta rúbrica no se marcan como ejecutadas.
