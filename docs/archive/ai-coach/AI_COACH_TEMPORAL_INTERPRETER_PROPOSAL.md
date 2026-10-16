# Primer intérprete descriptivo temporal — propuesta ICT-I1

20 de septiembre de 2026. Versión 0.1 para encuadre experto. **Propuesta preparada; motor sin implementar.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) es la fuente del estado y de la siguiente acción. Esta propuesta concreta el registro y el baseline previstos en las secciones 6 y 7 del [diseño colectivo](../../ai-coach/AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md), tras la [recepción de K2](AI_COACH_F01_K2_EXPERT_REVIEW_2026-09-17.md).

## Alcance que se propone revisar

Un primer módulo de desarrollo que lea dos o más prefijos causales del mismo episodio y produzca un registro de organización observable, hipótesis compatibles, posibles funciones y motivos de actualización. Debe explicar qué sostiene cada lectura y qué no puede saberse. Todavía no calificará decisiones como buenas o malas ni generará prioridades o ejercicios de entrenamiento.

Se propone comenzar por **continuidad y ambigüedad de la interpretación colectiva**. K1/K2 aportan el ejemplo conocido de continuidad. La generalización a cambios de plan requiere criterios expertos y material contrastivo adicionales: no se deduce una regla de cambio de dos cortes que mantienen la misma lectura. La selección de un caso nuevo vendrá después de acordar qué contraste debe aportar; no se prepara K3 por rutina ni se elige una escena para confirmar B.

## Entradas y límites

| Entrada propuesta | Condición para utilizarla |
|---|---|
| Identidad, ronda, lado, actor, vida, época y secuencia | Coincidencia explícita entre cortes. Un hueco o cambio de identidad rompe la continuidad de la afirmación afectada. No se arrastra automáticamente el estado anterior. |
| Posiciones, orientación y movimiento propios/aliados | Fuente, precisión y disponibilidad hasta el corte. Desplazamiento muestreado no equivale a velocidad instantánea, intención o tránsito físicamente posible. |
| Recursos, C4, acciones y relojes | Sólo los componentes acreditados para ese actor, variante y corte. No completar munición, readiness, comunicación o conocimiento del jugador. |
| Contexto previo | Hechos anteriores filtrados por perspectiva; ausencia de historia queda explícita. El resultado posterior de la ronda actual nunca entra. |
| Zonas y nombres comprensibles | Enlace espacial documentado. Un callout ambiguo o un radar de compatibilidad pendiente no habilita inferir automáticamente «B» desde coordenadas. Las etiquetas de las fichas conocidas se conservan como referencia de revisión. |
| Hipótesis previa del propio módulo | Conservada con sus señales y versión. Es una inferencia previa, no un hecho que confirme por sí solo la siguiente salida. |

La entrada factual tendrá una lista explícita de campos admitidos. Respuestas expertas, títulos interpretativos de las fichas, posiciones rivales ocultas y desenlace quedan en materiales de revisión separados. El acceso analítico a posiciones aliadas tampoco demuestra que cada jugador conociera esas posiciones.

El diagnóstico NAV de AUD-C01 no entra como evidencia física. FOV y proxies históricos de humo/contacto tampoco acreditan percepción legal. Cuando una hipótesis necesite información sin fuente positiva, se registrará el requisito y la abstención correspondiente; una revisión experta no lo sustituye. Rigen el [contrato humano](../../AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) y la [matriz C0](../../ai-coach/AI_COACH_SUPPORT_C0_DEPENDENCIES.md).

## Salida propuesta

Cada registro contendrá identidad y corte, versión del módulo/configuración, perspectiva, hechos con referencias, hipótesis compatibles, señales a favor y en contra, funciones posibles, desconocidos y enlace al registro anterior. La revisión humana estará separada de la salida del motor.

La actualización explicará una de estas operaciones propuestas: mantener, precisar, abrir alternativas, retirar una hipótesis o abstenerse por datos insuficientes. Puede mantener el plan colectivo y dejar sin resolver una función individual. No se asignarán porcentajes sin calibración ni se reducirá «sin evidencia» a «falso». Los nombres de estas operaciones son una propuesta técnica de representación, no etiquetas ya aprobadas por el experto.

**Ejemplo conocido K1→K2, como objetivo de desarrollo separado de la entrada:** T1, T2, T3 y T4 presentan avance entre cortes; T1 repite posición horizontal al final durante 0,125 s; T5 no presenta desplazamiento horizontal resuelto. C4 en T4 y recursos de T5 conservan sus comprobaciones. El experto mantiene la lectura de push hacia B y las funciones anteriores por referencia. Esto enseña que esos detalles no refutan por sí solos esa lectura colectiva; no enseña que cualquier pausa mantenga cualquier plan. «El grupo protege a T4» sigue siendo una interpretación, sin certificar cobertura física.

El registro esperado debe poder representar esa continuidad y sus límites. Que el programa emita exactamente la lectura conocida no demostraría inferencia válida: habrá que verificar que no recibe la respuesta humana, que usa señales documentadas y que sus criterios también admiten resultados distintos o abstención en contrastes pertinentes.

## Casos y excepciones que delimitan el trabajo

| Contraste | Tratamiento propuesto y material necesario |
|---|---|
| Pausa local dentro de avance colectivo | Referencia conocida K1/K2; no pedir de nuevo la interpretación recibida ni convertirla en una regla universal. |
| Misma organización, historial ausente | Conservar la descripción y retirar motivos que dependían de ese historial. El contraste F02/F03 sirve como material ficticio de desarrollo, con sus premisas y exposición declaradas. |
| Parte del grupo cambia de dirección | La observación no fija por sí sola «rotación», «cancelación» o «engaño». Antes de elegir un caso real hacen falta posiciones localizables de cada actor, orden y duración, C4/recursos relevantes y límites de contacto/comunicación. El criterio de actualización está pendiente de encuadre experto. |
| Cambio aparente con hueco de fuente o identidad | Separar discontinuidad del dato de cambio táctico. No inventar el movimiento ni convertir el hueco en evidencia de un nuevo plan. |
| Posible función sin apoyo físico acreditado | Puede revisarse la hipótesis de función; no afirmar viabilidad de ayuda, cobertura, llegada o alternativa superior. |

No se fijan por cuenta del agente cantidades de jugadores, segundos o distancias para decidir un cambio. Las 18 valoraciones pendientes siguen pendientes; las premisas rechazadas de F04-K3/F05-K3 no se reutilizan como pruebas válidas.

## Implementación acotada tras el encuadre

Se propone un módulo de dominio en `ai_coach/interpretation/`, separado de rutas y UI, con adaptador de prefijos acreditados, registro versionado y actualización determinista según criterios revisados. Una entrada de desarrollo reproducible permitirá inspeccionar cada salida y su procedencia. La ubicación es una propuesta; todavía no se crea el paquete ni un contrato runtime nuevo.

Primero se implementará la frontera de entradas y el registro; después las reglas de interpretación expresamente acordadas. Una etiqueta semántica sin derivación factual acreditada quedará desconocida: no se codificarán IDs de caso, ticks o respuestas K1/K2 para forzar «B». La redacción inicial será trazable y determinista, sin llamadas nuevas a modelos ni entrenamiento. Las decisiones técnicas rutinarias podrán resolverse sin otra consulta una vez acordado el alcance.

La entrega inicial no integra frontend/API ni habilita capacidades del catálogo. Tampoco completa rutas, contacto legal, apoyo físico, economía itemizada, S2–S3, prescripción o migración 44/44. Sus gates conservan el estado del plan.

## Verificación prevista para la implementación

1. Verificar schemas del registro y trazabilidad de cada señal; rechazar identidad incompatible, fuente futura o campo no permitido.
2. Mantener idéntica salida para el mismo prefijo/configuración al truncar o alterar el futuro, respuestas humanas, posiciones rivales ocultas y desenlace.
3. Contrastar ausencias, huecos y alternativas sin transformarlos en negativos; comprobar que NAV y proxies visuales no habilitan afirmaciones físicas o perceptivas.
4. Reproducir K1/K2 como desarrollo conocido. Contrastar ambigüedad/cambio con material suficiente y criterios revisados antes de declarar que el motor cubre esas operaciones.
5. Revisar que las transiciones citan señales nuevas y no reescriben registros pasados; separar lectura defendible de recomendación superior.
6. Ejecutar los checks públicos y pruebas afectadas. Si el cierre cambia hechos derivados, schemas o contratos runtime, procesar exactamente una demo aprobada tras las pruebas, en TEMP nuevo, con validación semántica/causal y recálculo independiente aplicable. Retener salida e inventario. El smoke no será evaluación táctica independiente.

La aceptación del primer módulo exige controles técnicos aprobados, criterios de alcance recogidos y límites explícitos. K1/K2 y las fichas ya vistas son desarrollo abierto; no acreditan generalización ni evaluación ciega. Un conjunto independiente, con protocolo y soporte suficientes, sigue siendo una dependencia posterior.

## Encuadre experto pendiente

La propuesta del agente es que **un cambio parcial de dirección pueda abrir ambigüedad sin que el motor declare automáticamente un cambio de plan**. Antes de implementar esa regla, falta concretar con el experto qué señales deben influir, qué excepciones importan y qué información mínima necesita la revisión. No se pide reinterpretar K1/K2 ni volver a responder Q1/Q18/Q26.

Pregunta de encuadre: para este primer módulo, cuando una parte del grupo deja de avanzar y otra retrocede, ¿qué exigirías observar para mantener el plan anterior con dudas, abrir una alternativa o retirar esa lectura? Interesa tu criterio contextual y los datos imprescindibles; no hace falta fijar umbrales universales ni interpretar una escena sin posiciones. A partir de esa respuesta se delimitará el contraste concreto y se comprobará si las fuentes existentes bastan.

Tras recibir el encuadre se conservará el literal, se separará la síntesis del agente y se actualizarán sólo las filas relevantes de una nueva versión del Excel y la documentación de fase. Los cinco libros existentes permanecen intactos mientras no haya criterios nuevos acordados. No se presupone aprobación por silencio. Esta preparación documental no implementa el motor ni satisface sus dependencias factuales.
