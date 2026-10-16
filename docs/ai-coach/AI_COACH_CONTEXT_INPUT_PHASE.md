# CS2: entrada factual común

Encuadre recibido el 05/10/2026. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción. Este documento conserva el alcance aceptado y su cierre funcional; el [contrato común v1](../canonical-common-factual-context-v1.md) especifica la interfaz runtime.

## Aportación experta

El usuario acepta la propuesta presentada: «El resto de información lo veo todo correcto». Para CT pide interpretar los patrones de ataque de T, explicar si la defensa está dando efecto y si se adapta, y corregirla cuando corresponda. El [recibo](../evidence/AI_COACH_CONTEXT_INPUT_FRAMING_2026-10-05.json) conserva el mensaje completo, su hash, atribución y límites. IC40 se añade al [Excel v30](../../outputs/stratai-context-input-framing-20261005/StratAI_Plan_del_coach_TP10_v30.xlsx), sin cambiar las filas anteriores.

La frase sobre la menor estrategia CT se conserva como opinión del usuario; no se convierte en una taxonomía automática ni elimina defensa, distribución, información, recursos, rotación o retake del objetivo del producto. El encuadre general CT queda recibido. Sus casos tácticos y criterios operativos se revisarán sobre material causal real en las fases pertinentes; no es necesario volver a preguntar por este foco general.

## Alcance aceptado

Una entrada común por instante analizado reúne:

| Bloque | Hechos a conservar | Ausencias y límites |
|---|---|---|
| Marco | Mapa/build, lado, ronda/instancia/época, corte, perspectiva y elegibilidad | Unknown permanece unknown; no deducir cuchillos por ausencia de C4 |
| Objetivo | Estado acreditado del C4, tenencia, planta y relojes disponibles | Falta de portador no demuestra caída; posición del portador no equivale a coordenada física del C4 |
| Jugadores y distribución | Identidad reconciliada, vida, posiciones y cambios propios/aliados | Recuentos parciales no se completan; distribución no confirma una función táctica |
| Recursos | Vida, armadura, armas, munición, utilidad y otros recursos disponibles | Inventario histórico y actual separados; ausencia no equivale a cero |
| Acontecimientos | Bajas, daño, drops/recogidas y eventos acreditados del objetivo | Posición de evento fechada, sin seguimiento rival ni percepción universal |
| Historial | Prefijo disponible, cambios, continuidad y discontinuidades | Sin futuro ni truncado silencioso; ningún mínimo universal de segundos |

Cada dato conserva valor y estado, fuente, referencia, unidad cuando corresponda, momento observado, disponibilidad causal y perspectiva. Unknown, no aplicable, ausencia y conflicto se mantienen separados. La incertidumbre limita afirmaciones dependientes, no vacía toda descripción.

La salida organiza situación factual, evolución anterior e incertidumbre. El detalle de evidencia permanece consultable. Las derivaciones identifican su método y referencias. Las interpretaciones tácticas y recomendaciones no se fabrican como parte del adaptador.

E03 conserva @5; los demás casos usan @9. Un suplemento extraído después puede acreditar un hecho anterior, pero no formó parte automáticamente de la ficha original: las dos disponibilidades deben distinguirse. Los nombres nativos posteriores quedan separados. Respuestas, IDs de caso y predicciones no son señales para interpretar.

La estructura sirve para T y CT donde sus fuentes/perspectiva estén acreditadas. No se revela a CT una posición global del C4 sólo porque la demo la conserve. Información legal del equipo/jugador, descripción retrospectiva y datos futuros quedan separados. No se añaden agrupaciones espaciales, umbrales de grupos, apoyo eficaz o reglas automáticas de movilización/execute/fake.

## Base existente y entrega técnica

La lectura del [contrato candidato](../contracts/context/canonical-candidate-context-v1.md) y de su productor actual confirma que ya reúne packet v3, snapshot y eventos causales, objetivo con conflictos, recuentos parciales y separación @5/@9. CS2 debe reutilizarlo y comprobar las diferencias respecto al alcance aceptado antes de extender nada. Esta inspección no vuelve a ejecutar ni certificar todo el productor.

La implementación debe entregar una interfaz factual verificable, situación/evolución con referencias y acceso al historial conservado, incertidumbre local y un lector/validador que recompute los hechos derivados. Se resolverán de forma autónoma formato, módulos, versiones y pruebas dentro del alcance. Si aparece una carencia nueva que exige fuente o juicio de juego adicional, se delimitará por afirmación y se detendrá sólo el trabajo dependiente.

## Regresiones de desarrollo

- E06: conservar distribución/evolución; A preferida y B posible es criterio experto para interpretación posterior, no salida nueva de CS2; C4 no fundamenta esa preferencia.
- E17: movimiento inicial e historia disponibles; movilización inicial recibida, sin umbral temporal universal ni etiqueta automática.
- E24: planta acreditada integrada; recuento CT parcial impide certificar 1v1, no describir la bomba plantada.
- E13/E25/E30: elegibilidad unknown; ninguna clasificación positiva de cuchillos.
- E03: no introducir retroactivamente extra/bomb de @9 o nombre nativo posterior en @5.

Estos casos comprueban el diseño y futuras regresiones. No son test táctico independiente ni requieren repetir la auditoría completa de los 34 casos.

## Aplicación posterior del foco CT

Interpretación técnica del agente: distinguir ataque T observado, respuesta defensiva CT, cambios y resultados retrospectivos. Un patrón repetido exige antecedentes disponibles y comparables. No fijar ahora un número universal de rondas, caducidad de patrones ni obligación de cambiar después de perder. Para explicar una adaptación hay que conservar qué información existía en ese momento; el resultado posterior no puede convertirse en entrada de la decisión anterior.

Una defensa puede producir un resultado favorable sin que eso demuestre su calidad, o perder sin que la decisión fuera errónea. Describir eficacia y adaptación, interpretar sus razones y recomendar una corrección son afirmaciones diferentes. El producto sigue exigiendo una alternativa viable mejor respaldada antes de llamar error a una decisión. Estas son limitaciones operativas del agente bajo el contrato vigente, no nuevas respuestas literales del usuario ni etiquetas CT adjudicadas.

El historial entre rondas se incluirá sólo donde exista material previo acreditado. CS2 no crea un perfil longitudinal ni acredita por sí solo lectura de patrones T o calidad defensiva CT; el lote actual de revisión es T. Las carencias de cobertura se documentarán.

## Verificación y condiciones de cierre

1. Fuentes/hashes y referencias reproducibles; identidad, corte y continuidad correctos.
2. Invariancia al futuro y separación de perspectivas; disponibilidad observada diferenciada de exposición al instrumento original.
3. Exactitud de objetivo/recursos y desconocidos/conflictos conservados; resumen contrastado con evidencia e historial.
4. Retirar un dato limita únicamente dependencias; no rellena ausencias ni convierte todo el episodio en indescriptible.
5. Versiones @5/@9 y suplementos explícitos; ninguna respuesta humana ni predicción en la entrada factual.
6. Pruebas pertinentes por los scripts públicos. Si cambia parsing, exportador, hechos derivados, schema o contrato runtime, exactamente una demo aprobada después de las pruebas, en TEMP nuevo/aislado, con semántica, disponibilidad, procedencia, recomputación e inventario retenido. Repetir esa misma demo ante fallo.

Primero cerrar/documentar CS2 y después presentar CS3: comparar maneras de interpretar con las mismas entradas y criterios. El especialista semántico, proveedor/modelo, presupuesto y evaluación independiente conservan su revisión/gates propios. La aprobación de CS2 no autoriza entrenamiento, adquisición, publicación, promoción ni reprocess 44/44.

## Entrega funcional y límites comprobados — 05/10/2026

Implementados adaptador, schema, consultas causales, materializador y lector/validador. Reutiliza el contexto candidato; no cambia sus bytes, productores originales ni detector. La entrada integra recursos acreditados y declara salud/armadura actuales no suministradas. Las derivaciones nombran método y referencias. La evolución conserva cambios de estado; discontinuidades o cambios de identidad/roster no prueban un recorrido continuo.

132 pruebas y lint correctos. Una demo aprobada distinta en TEMP nuevo, con repetición de la misma tras retirar validaciones redundantes, ocho ventanas T/CT y contraste independiente. Siete regresiones completas conservan los criterios E03/E06/E17/E24/E13/E25/E30. El nombre nativo positivo se prueba como suplemento posterior separado; el smoke de la entrada original no añade esa capa. [Resultado, comandos, inventarios y límites](../../outputs/stratai-common-context-20261005/RESULTADO.md).

El historial entre rondas permanece no suministrado. CS2 no acredita interpretación de patrones T ni buena/mala defensa CT. Fuente retrospectiva de eventos tampoco prueba percepción. No se entrena ni conecta un modelo general. El siguiente paso del plan es la [revisión experta CS3](../AI_COACH_CONTEXT_COMPARISON_REVIEW.md).
