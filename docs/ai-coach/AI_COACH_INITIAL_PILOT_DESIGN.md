# Diseño del primer piloto de coaching personalizado

Diseño v1.0, 5 de septiembre de 2026. Concreta el paso 2 del [plan de implementación](../AI_COACH_IMPLEMENTATION_PLAN.md). Es una propuesta revisable: no contiene partidas seleccionadas, respuestas de coaches, resultados de usuarios ni capacidades recién implementadas. El plan mantiene los pendientes y la siguiente acción; este documento define el experimento y sus instrumentos.

## 1. Decisión de diseño

Empezar por **compras y reparto de equipamiento en Mirage**, añadir **apoyo al compañero** y **ejecución del disparo** conforme se acrediten sus datos. Mantener peek/hold/reposición como prueba técnica espacial. Desde el primer prototipo habrá perfil previo, una o dos prioridades y misiones con seguimiento.

La primera pieza de software será un **registro económico de lo que estaba disponible antes de comprar**, con procedencia e instante verificables. La economía permite comprobar recursos y opciones legales sin esperar a completar la geometría del mapa. Eso facilita verificar hechos; no demuestra que una compra legal sea tácticamente superior.

La [auditoría Mirage 44](../archive/ai-coach/AI_COACH_MIRAGE_44_DATA_AUDIT_2026-08-27.md) registra 9.308 filas jugador/ronda con dinero e inventario al final de freeze, junto con avisos de reconciliación e itemización. Es un punto de partida útil, pero lo comprado al final no describe lo que el jugador sabía al principio. Las otras familias también tienen carencias materiales; ninguna se declara lista por este diseño.

El ciclo visible será: ver el momento de la partida, entender la comparación, elegir una prioridad, practicarla y revisar oportunidades posteriores. El sistema podrá decir «todavía no hay suficiente información». No necesita inventar una corrección nueva en cada partida.

## 2. Situaciones que debe contemplar el piloto

Las etiquetas E1–P1 sólo organizan este documento; no son nuevos IDs del catálogo. Son situaciones que buscar y revisar, no errores ya confirmados. En cada una se registrará también una acción positiva, equivalencia o abstención cuando corresponda.

| Situación | Instante de decisión y opciones | Qué permite comparar; cuándo abstenerse | Práctica relacionada |
|---|---|---|---|
| **E1. Repartir equipamiento** | Antes de comprometer el presupuesto: cesta propia, cesta más económica con drop o conservar. | Dinero e inventario de los cinco aliados, compras legales, coste, destinatario y cooperación. No afirmar que el compañero aceptaría ni que el AWP causó la derrota. | Revisar recursos del equipo antes de comprar. |
| **E2. Coordinar eco, parcial o compra** | Inicio de freeze: guardar, compra limitada o cesta completa viable. | Escenarios de recursos actuales y rondas posteriores, inventario conservado, bonus previo y reglas versionadas. Si el orden cambia según supuestos plausibles, explicar condiciones. | Comparar dos planes y su coste de perder la ronda. |
| **E3. Evitar una compra que rompe el plan siguiente** | Antes de gastar: comprar ese elemento, elegir otro o conservar dinero. | Saldo e inventario por jugador, margen en escenarios y beneficio táctico revisado. «Tener menos dinero después» no basta para concluir error. | Hacer una pausa de planificación económica. |
| **E4. Conservar una compra o drop razonable** | Mismo marco que E1–E3, incluyendo decisiones con desenlace malo. | Reconocer una decisión respaldada pese a perder; opciones equivalentes cuando haya compensaciones. Falta de compras detalladas puede obligar a no juzgar. | Mantener el hábito sin convertirlo en obligación de dropear. |
| **S1. Responder a una baja con apoyo viable** | Primera decisión legal después de la baja: apoyar, reagrupar, mantener o esperar. | Estado aliado, contacto legal, ruta, orientación, arma, objetivo y margen. Un camino alcanzable no prueba un trade seguro. | Comprobar conexión antes de comprometerse. |
| **S2. Preparar apoyo antes de entrar** | Antes del avance o pérdida de conexión: esperar, seguir, ajustar separación o mantener. | Requiere nuevo anclaje de oportunidad anterior a la entrada; el ejemplo actual en el tick de muerte no lo acredita. Sin ese anclaje, sólo candidato de investigación. | Coordinarse antes de una entrada, con excepciones por objetivo. |
| **S3. Mantener una separación justificada** | Antes de reagruparse o entrar: conservar posición, conectar o cambiar ruta. | Comparar apoyo con función de la posición, tiempo y riesgo de abandonar otra responsabilidad. No premiar juntarse siempre. | Reconocer cuándo el apoyo exige conservar otra posición. |
| **M1. Disparar con movimiento relevante** | Estado legal inmediatamente anterior al tiro: disparar, estabilizar antes o continuar desplazamiento. | Arma, tiro y movimiento alineados; contacto y coste de detenerse. No inferir una tecla ni atribuir un fallo de bala sólo a la velocidad. | Separar desplazamiento y disparo en situaciones compatibles. |
| **M2. Continuar, resetear o retrasar recarga** | Antes de ráfaga/recarga: continuar, resetear, recargar o esperar. | Munición, estado de arma, contacto, tiempo y opción viable. Sin estado de arma fiable o sin contexto del duelo, abstención. | Preparar la siguiente acción sin recargar por costumbre. |
| **M3. Preparar la mira** | Antes del contacto/tiro: mantener o ajustar el ángulo acreditado. | Ángulos, contacto visible y referencia geométrica comprobables. La cabeza real de un enemigo oculto no define dónde debía mirar. | Preparar ángulos revisados en un ejercicio acotado. |
| **P1. Asomar, mantener o recolocarse** | Antes de exponerse o repetir: candidatas con origen, destino, orientación y margen. | Prueba técnica separada de cobertura, visibilidad y ruta exactas en T0. Los paquetes parciales y el muestreo actual no permiten generalizar. | Sólo diseñar la práctica hasta acreditar contexto y alternativa. |

## 3. Ficha de decisión y comparación

Cada caso debe poder revisarse sin una narración que revele el futuro. La ficha propuesta tendrá estos bloques; todavía no es un schema runtime:

| Bloque | Contenido obligatorio |
|---|---|
| Identificación | Alias opacos de partida y actor, mapa/lado/ronda, versión de fuente, localizador del replay, criterio de selección y conjunto de desarrollo/calibración/test. |
| Momento | T0 y motivo del anclaje; instante de observación y de disponibilidad por dato. Orden dentro del tick si importa; si no puede resolverse, ausencia explícita. |
| Información legal | Estado propio y aliado, objetivo, recursos y contactos disponibles hasta T0; historial previo permitido. Contexto declarado aparte. |
| Acción observada | Acción y parámetros, fuente y ventana usada para identificarla. Puede usar ejecución posterior como etiqueta; no retroalimentar el estado de T0. |
| Opciones | Acción observada y hasta tres alternativas iniciales, con parámetros y precondiciones. Registrar también las rechazadas y la razón. Si ninguna sirve, admitir una nueva propuesta antes de concluir. |
| Comparación | Recursos, apoyo, tiempo, objetivo, riesgo y dificultad; ventajas y costes por opción. Preferencia, equivalencia o abstención, con razón e incertidumbre. |
| Evidencia posterior | Desenlace y horizontes identificados como resultados observados, separados del juicio inicial. No son resultados de las alternativas. |
| Coaching | Corrección o fortaleza respaldada, prioridad, misión, historial previo aplicable y próxima revisión. Si falta aprobación prescriptiva, sólo candidato de revisión. |

La acción observada supera el mismo filtro físico que las alternativas. «Parar», «apoyar» o «comprar mejor» sin parámetros no bastan. Los parámetros mínimos se toman del [catálogo](../../ai_coach/contracts/capability_catalog.json); si el piloto necesita uno nuevo, primero se diseña y versiona su contrato.

En economía se comparan cestas completas por jugador y posibles transferencias, con coste, saldo y recursos conservados. `minimum_money_next_round` será un resultado condicionado a reglas y escenarios explícitos, no dinero futuro conocido. El beneficio de una configuración se evalúa con contexto y revisión táctica; no se reduce al precio agregado de las armas.

Una acción colectiva se presenta como «proponer/acordar este reparto si el compañero acepta». Si no consta un acuerdo previo, se comparan escenarios de aceptación y no aceptación. Las compras futuras reales de otros jugadores no se introducen en T0 como si ya estuvieran decididas.

### Ejemplos ilustrativos de salida objetivo

Estos cuatro ejemplos son guiones condicionales para revisar el diseño. No describen demos reales, no tienen valoración humana y no se pueden publicar como findings validados.

- **Reparto:** si se acredita que ambas cestas cabían, eran legales y el reparto resultaba preferible bajo condiciones claras: «Había una opción para equiparos ambos con rifles, si tu compañero aceptaba el reparto. Revisaría esa opción antes de concentrar el presupuesto en tu arma. No podemos afirmar que cambiara el resultado de la ronda». Una misión posible es practicar esa comparación antes de comprar.
- **Eco o compra limitada:** «Los dos planes dejan recursos distintos si perdéis. Conservar tiene sentido bajo este objetivo; la compra limitada gana interés bajo este otro. Falta el plan del equipo para escoger uno con confianza». No inventar la regla «siempre fuerza para evitar dos ecos».
- **Apoyo:** si ruta, arma, información y contexto lo permiten: «Tras la baja existía una opción de conectar con el compañero. Su ventaja dependía de mantener ese contacto y de no abandonar el objetivo». No convertir ese episodio posterior en «debías haber estado más cerca antes».
- **Movimiento:** si el tiro y la velocidad están acreditados: «El disparo coincide con movimiento. Para recomendar una pausa necesitamos comprobar que detenerte era razonable en ese duelo». Con datos parciales se describe el hecho o se propone práctica general identificada como tal; no se etiqueta automáticamente un error.

## 4. Diseño del lote de desarrollo: 40 situaciones

Se elige **40** dentro del intervalo de diseño 30–50. Es un objetivo de selección futura, no un lote creado ni el test final.

| Familia | Selección dirigida por contexto | Control aleatorio dentro de la familia | Total |
|---|---:|---:|---:|
| Economía E1–E4 | 12 | 4 | 16 |
| Apoyo S1–S3 | 9 | 3 | 12 |
| Mecánica M1–M3 | 6 | 2 | 8 |
| Prueba espacial P1 | 3 | 1 | 4 |
| **Total** | **30** | **10** | **40** |

Propuesta de distribución dirigida: economía 4/4/2/2, apoyo 3/3/3, mecánica 2/2/2 y tres pruebas espaciales de contextos distintos. La selección usa contexto y disponibilidad, no una etiqueta de «error» ni sólo muertes. Se incluyen situaciones donde mantener o no intervenir sea plausible, pero la revisión decide su clasificación real.

Objetivo de diversidad: al menos 16 partidas y 12 actores, máximo tres casos por partida y cuatro por actor, con ambos lados y contextos económicos distintos. Comprobar la viabilidad sobre el corpus autorizado antes de seleccionar; si no se alcanza, registrar el déficit, sin inventar casos ni descargar para completar la tabla. No trasladar los 4 casos espaciales a economía para ocultar falta de cobertura.

El 25 % aleatorio se extrae con semilla y población definidas previamente, incluyendo oportunidades sin hallazgo y con datos ausentes. Es control **estratificado**, no una muestra representativa de todas las partidas. La cobertura del producto se medirá después sobre un registro completo de oportunidades, sin usar las proporciones artificiales de esta tabla.

Todos los casos de una misma partida pertenecen al mismo conjunto. Las versiones usadas para ajustar rúbrica y reglas permanecen en desarrollo. Un caso al que le falten hechos puede servir para probar abstención, pero no recibe una preferencia táctica forzada. La rúbrica se ensaya primero con ocho casos del propio lote y se revisan de nuevo si cambia; esos ocho no son una validación independiente.

## 5. Instrumento de revisión humana

Los antiguos protocolos espaciales v1/v2 siguen excluidos como ground truth. Este diseño propone una revisión de decisiones completas, todavía sin respuestas ni aprobación experta.

**Presentación en dos momentos.** Primero se muestra la información legal hasta T0 y un clip terminado allí, sin x-ray ni desenlace. Después se muestran acciones parametrizadas con nombres neutros y orden aleatorio, incluida la observada sin destacarla; sólo se revela ejecución posterior necesaria para etiquetarla en un compartimento separado, sin resultados de combate. Si el revisor conoce la partida o el desenlace, registra esa exposición y el caso no cuenta como juicio ciego.

En diez casos del lote, distribuidos entre familias, pedir una propuesta antes de mostrar alternativas. Así se detecta si el generador omite una opción importante. No puntuar sólo la calidad de las opciones que el propio sistema eligió.

**Formulario por revisor, para copiar a la herramienta futura:**

| Pregunta | Respuesta a registrar |
|---|---|
| ¿El contexto basta para comparar? | Sí / no; dato crítico ausente y por qué cambiaría el juicio. |
| ¿Hay un problema en hechos o presentación? | Referencia concreta; remitir a validación automática. El coach no certifica rayos ni NAV. |
| ¿Cuál opción prefieres? | A / B / C / D, varias equivalentes, ninguna o contexto insuficiente. |
| ¿Por qué y con qué condiciones? | Ventaja, coste, información legal usada, cooperación necesaria y supuesto que cambiaría la elección. |
| ¿Qué confianza tienes? | Baja / media / alta, con motivo; opinión humana, no probabilidad calibrada. |
| ¿Es ejecutable por este jugador? | Sí / condicionada / no / historial insuficiente. Identificar la capacidad concreta que limita. |
| ¿Falta una opción? | Propuesta con parámetros; vuelve al filtro físico antes de entrar en otra comparación. |
| ¿El consejo sería correcto y accionable? | Correcto / parcialmente / incorrecto / no evaluable, más claridad, prudencia y acción concreta. Se pregunta después del juicio ciego. |
| ¿La misión ayuda a practicar esa habilidad? | Adecuación, dificultad, excepciones, oportunidad y riesgo de incentivar una mala conducta. |

Dos coaches revisarán independientemente los 40 casos de desarrollo; al menos uno no habrá diseñado sus alternativas. Un tercero distinto adjudicará los desacuerdos que lo permitan. Mantener tanto respuestas originales como adjudicación; un desacuerdo razonable puede terminar en equivalencia o abstención. Sin un tercero disponible, el caso queda pendiente, no decidido por el LLM.

Cada respuesta conservará alias del revisor, caso y conjunto asignado, versión de protocolo, hechos y candidatas mostradas, orden de presentación, fecha y exposición previa al desenlace. Cambiar una premisa o una alternativa genera una revisión identificable; no sobrescribe la respuesta anterior.

La prueba formal futura seguirá los tamaños, revisión externa ciega y umbrales del plan. El lote de 40 no sustituye los 100 casos prescriptivos por familia frecuente ni el conjunto de abstención. Los mínimos de soporte para recomendaciones aprendidas siguen separados. Ningún consejo experimental evita esos requisitos por llamarse «asistido».

## 6. Perfil, prioridades y memoria

El perfil será una memoria de evidencias y preferencias revisables, no una etiqueta fija de estilo. Sus registros propuestos se enlazan con `CAP-PROFILE-001`, `DEC-PROFILE-001`, `FLD-PRIOR-MATCH-FINDINGS`, `FLD-PRIOR-MATCH-CONTEXT` y `FLD-PRIOR-EXECUTION`. Estos IDs existen, pero los campos históricos siguen pendientes de extracción; el detalle pedagógico aún necesita contrato propio.

| Registro propuesto | Qué guarda y cómo se usa |
|---|---|
| Preferencias | Objetivos, disponibilidad, dificultad percibida y contexto declarado, con autor y fecha; no prueba capacidad real. |
| Evidencia de habilidad | Oportunidades comparables, ejecución, incertidumbre, contexto y referencias; primeras observaciones separadas de recurrencias. |
| Fortalezas y dificultades | Hechos aprobados por capacidad y situación; no usar K/D como diagnóstico global ni llamar hábito a tres rondas aisladas. |
| Plan personal | Una o dos prioridades, razón, misión y versión, aceptación/ajuste y fecha o condición de revisión. |
| Práctica | Ofrecida, aceptada, declarada realizada, verificada o no observable; evidencia y duración cuando consten. |
| Seguimiento | Oportunidades elegibles, aplicación, excepción válida, falta de datos y tendencia con muestra; separado del cumplimiento de ejercicios. |

Para analizar una partida se congela el perfil que sólo incluye partidas anteriores a ella. Si se procesan fuera de orden, usar orden de juego y reconstruir las referencias; la fecha de carga no convierte una partida posterior en pasado. El historial de la partida actual se incorpora después de cerrar su evaluación. Una corrección posterior invalida o recalcula los resúmenes derivados, sin reescribir la evidencia original.

Con la primera partida: una observación concreta y práctica revisada, sin afirmar recurrencia. Como regla de presentación **provisional**, no usar «se repite» hasta observar al menos tres oportunidades relevantes en tres partidas distintas; tampoco ese mínimo demuestra un hábito estable. Mostrar siempre numerador, denominador y contexto; estimación personal incierta se acerca a una referencia de desarrollo en vez de volverse extrema.

La selección de prioridades sigue una regla explicable: primero aprobación/evidencia, después impacto y oportunidades comparables, capacidad de ejecución, recurrencia y carga de práctica. Agrupar una misma causa entre familias. En empates razonables ofrecer al usuario elegir; no inventar diferencias numéricas. Una fortaleza puede ser la prioridad si no hay correcciones respaldadas.

Memoria de conversación, perfil y evidencia original se almacenan por separado, con acceso y retirada definidos. Las respuestas del usuario conservan procedencia; «me dijeron que esperase» no se añade retrospectivamente al estado observado. Ajustar una preferencia de práctica no requiere reentrenar un modelo global.

## 7. Biblioteca inicial de misiones

Todas son propuestas pendientes de revisión de coaches. Se corresponden con `ACT-PRACTICE-DECISION`, `ACT-PRACTICE-MECHANIC` o `ACT-MAINTAIN-STRENGTH`. Una misión basada en un supuesto error sólo se asigna cuando el hallazgo y el consejo están aprobados; un ejercicio general se identifica como tal.

| Misión | Ejercicio y aplicación | Oportunidad y comprobación | Excepciones y revisión |
|---|---|---|---|
| **Revisar el reparto** | Comparar dos cestas en cinco fichas sin desenlace y explicar destinatario, coste y condición. En partida, revisar equipo antes de comprometer una compra. | Freeze con datos completos y una decisión de reparto real. Puede observarse compra/drop; no que miró el marcador o habló. Esas acciones se declaran aparte. | No se exige dropear si empeora el plan o el compañero no acepta. Revisar tras tres partidas o cinco oportunidades, lo que ocurra antes. |
| **Planificar la siguiente compra** | Elegir entre conservar y compra limitada en tres situaciones, explicando recursos en escenarios de perder/ganar/conservar. | Ventana económica con reglas y estado previo válidos. Medir calidad del plan y después coherencia de la conducta, no acertar el resultado. | Última ronda, cambios de lado, overtime o reglas no soportadas se separan. Si falta contexto, practicar la pregunta que falta. |
| **Conectar antes de comprometerse** | Pausar tres clips previos a la acción y escoger apoyar, esperar o mantener con una razón. En partida, buscar apoyo cuando sea viable. | Entrada o respuesta de apoyo con T0 y alternativa acreditados. Tasa de elecciones adecuadas en esas oportunidades. | Objetivo urgente, ruta no viable o función distinta pueden justificar mantener. No premiar perseguir trades ni agruparse siempre. |
| **Estabilizar el disparo adecuado** | Bloque breve de desplazamiento y disparo en entorno de práctica verificado para el arma elegida; dificultad revisada por coach. | En partida, tiros comparables donde estabilizar sea viable y haya estado de movimiento/arma suficiente. Separar ejecución del ejercicio y aplicación real. | Combate forzado, movimiento requerido, arma o muestra no soportados no cuentan como fallo. No diagnosticar teclas ni precisión sub-tick. |
| **Mantener una fortaleza** | Explicar qué hizo bien en dos casos y reconocer una situación donde no convenga repetirlo. | Nuevas oportunidades del mismo contexto; valorar mantener criterio y adaptarse, no repetir una acción mecánicamente. | Una buena decisión puede perder. Retirar la misión si deja de ser prioritaria, sin fabricar un error nuevo. |

Duraciones y cantidades son parámetros iniciales para revisar, no dosis con eficacia demostrada. Empezar con una misión y bloques de 5–10 minutos si encajan con el usuario; añadir una segunda sólo si la carga es razonable.

Cada revisión ofrece: mantener, simplificar, aumentar dificultad, cambiar, pausar o dar por cerrada la prioridad. «Sin oportunidad», «sin datos» y «excepción válida» son estados distintos de fallo. Si el usuario no practicó, preguntar por tiempo/dificultad y ajustar carga; no inventar desinterés ni deterioro de habilidad.

**Medida de aplicación propuesta:** elecciones adecuadas divididas por oportunidades elegibles de esa misión, mostrando aparte excepciones, casos sin datos y oportunidades totales detectadas. Conservar el criterio de elegibilidad versionado para impedir mejorar la tasa eliminando casos difíciles o evitando jugar. También revisar perjuicios al objetivo o al equipo.

**Transferencia y retención:** comprobar nuevas oportunidades de otras rondas/contextos, con y sin recordatorio, después de la práctica. Una mejoría de porcentaje con pocas oportunidades se describe con límites. Para atribuir efecto a la misión habrá que prerregistrar una comparación apropiada, como inicio escalonado por jugador; este diseño exploratorio no prueba causalidad.

## 8. Qué automatiza el prototipo y qué revisa una persona

| Parte | Función inicial | Evidencia que debe conservar |
|---|---|---|
| Sistema factual | Leer bundle/candidato aprobado, construir oportunidad y filtrar datos ilegales o ausentes. | Fuente, versión, T0, disponibilidad, motivos de rechazo y replay. |
| Cálculos y alternativas | Recursos, legalidad y escenarios; propuestas acotadas mediante reglas y casos revisados. | Parámetros, costes y condiciones; no inventar probabilidades de victoria. |
| Coach | Comparar decisiones completas, revisar explicación y adecuación de misión. | Respuesta independiente, argumentos, condiciones y desacuerdos. |
| Memoria y plan | Resumir pasado y seleccionar prioridad/misión con reglas aprobadas. | Perfil congelado, razón de selección y seguimiento por oportunidad. |
| Chat o plantilla | Explicar la evidencia aprobada, aclarar condiciones y recoger contexto. | Versión del hallazgo y fuentes; abstenciones y límites intactos. |

La interfaz mostrará la intervención humana cuando exista y distinguirá análisis pendiente, revisado y no evaluable. Antes de la aprobación prescriptiva, los casos se usan para desarrollo y revisión, no como correcciones automáticas del usuario. La evaluación exploratoria puede probar comprensión de guiones y ejercicios claramente identificados.

Propuesta de prueba de uso posterior: seis jugadores, tres sesiones por jugador, sin promesa de coach integral. Primera sesión: objetivos y comprensión; segunda: práctica y primer seguimiento; tercera: nueva oportunidad y ajuste. No es el panel formal de Gate 9B. Si aún faltan hechos o revisión, limitar la sesión a guiones ilustrativos, sin atribuirles errores en una partida real.

## 9. Datos e incompatibilidades que condicionan el diseño

La disponibilidad indicada procede del contrato y evidencia actuales; no de un nuevo barrido de los 44 bundles. «Verificado» significa presencia con el alcance allí documentado, no uso libre en cualquier instante.

| Necesidad del piloto | IDs y fuente existentes | Disponibilidad y carencia relevante |
|---|---|---|
| Compra y reparto E1–E4 | `CAP-ECONOMY-001`, `DEC-ECONOMY-001`; `FLD-MONEY`, `FLD-TEAM-ECONOMY`, `FLD-WEAPON-INVENTORY`, `FLD-UTILITY-INVENTORY`, `FLD-SCORE`. `derived/economy_players.json` y `derived/economy_rounds.json`. | Dinero/economía presentes y armas parciales. Falta snapshot conjunto causal en freeze, compra observada reconciliada y legalidad de cesta/cooperación. El marcador final no se usa como marcador de T0. |
| Resultados económicos | `OUT-ECONOMY-1R`, `OUT-ECONOMY-3R`, `OUT-ROUND-WIN`, `OUT-MATCH-WIN`. | Son resultados/escenarios separados. El dinero de la ronda siguiente observado y el gasto total posterior no son entradas del comprador. |
| Apoyo S1 | `CAP-TEAM-001`, `DEC-TRADE-001`; `FLD-TEAMMATE-STATES`, `FLD-TRADE-POSSIBLE`, `FLD-CONNECTION-TIME`, `FLD-FACING-DELTA`, `FLD-LINE-OF-SIGHT`, `FLD-PATH-DISTANCE`. | Parte avanzada disponible; LOS parcial y NAV candidato sin materializar en corpus. Prescripción exige contexto y acciones físicamente válidas por oportunidad. |
| Apoyo previo S2–S3 | Misma capacidad; futuro anclaje de decisión anterior al compromiso. | `DEC-TRADE-001` arranca hoy en una muerte. No reutilizarlo silenciosamente para juzgar una decisión anterior ni omitir oportunidades sin muerte. |
| Movimiento, tiro y recarga M1–M2 | `CAP-MOVEMENT-001`, `CAP-COMBAT-001`, `DEC-SHOOT-001`; `FLD-SELF-STATE-16HZ`, `FLD-SHOT-STATE`, `FLD-WEAPON-AMMO`, `FLD-VELOCITY-2HZ`, `FLD-STANCE`. | Tiro/arma parciales; candidato de arma fuera de corpus y disponibilidad/resolución de velocidad por comprobar en el instante del tiro. 2 Hz no acredita timing fino. |
| Mira M3 | `CAP-CROSSHAIR-001`; `FLD-VIEW-ANGLES`, `FLD-VISIBLE-CONTACT`, `FLD-CROSSHAIR-ERROR`. | Proyección parcial y contacto causal candidato; bloquear referencias a objetivos ocultos y errores geométricos no verificables. |
| Peek P1 | `CAP-PEEK-001`, `DEC-PEEK-001`; hechos `FLD-GEOMETRIC-CLEARANCE-SECTORS`, `FLD-GEOMETRIC-OCCLUSION`, `FLD-NAV-EXIT-PATH`, contacto/FOV/humo. | Candidato espacial parcial; 0 peek elegibles en el smoke documentado. Cobertura espacial y coincidencia temporal pendientes por separado. |
| Perfil y misiones | `CAP-PROFILE-001`, `DEC-PROFILE-001`, tres campos `FLD-PRIOR-*` citados en la sección 6; `OUT-PATTERN-RECURRENCE`, `OUT-PATTERN-TREND`. | IDs existentes, materialización histórica pendiente. Intervención, cumplimiento y oportunidades necesitan contratos pedagógicos propios. |
| Contexto no observado | `FLD-VOICE`, `FLD-INTENT`, `FLD-SUBJECTIVE-SOUND`, `FLD-COUNTERFACTUAL-FUTURE`. | No observables; conservar desconocido o declaración atribuida. Nunca completarlos desde una respuesta del LLM. |

### Hallazgos concretos que hay que resolver al implementar

La lectura de [economy.go](../../backend/go-service/handlers/economy.go) confirma captura de dinero inicial en `RoundStart`, captura de freeze aplazada a `FrameDone` y reconciliación posterior en `RoundEnd`. [block6_models.go](../../backend/go-service/models/block6_models.go) y [block6_export.go](../../backend/go-service/parser/block6_export.go) exportan snapshots agregados sin un instante de disponibilidad por cada valor; conservan además resultados posteriores en el mismo registro.

Esto obliga a separar el momento al que se refiere un valor de cuándo estuvo realmente disponible. `spent_in_buy` y `after_buy_calculated` pueden consolidarse al terminar la ronda; no retroceder su disponibilidad a T0. Tampoco convertir cada `ItemPickup` en una compra cierta: existen transferencias, adquisiciones tras freeze y coberturas parciales. Una reconstrucción usando información posterior sólo puede ser reconciliación/etiqueta, no observación causal previa.

Tres desajustes del catálogo requieren futura revisión versionada, sin modificarlo en esta fase:

1. Economía define T0 en inicio de freeze y final al terminar freeze, pero hereda una ventana genérica de identificación de acción de 0,25–2 segundos. Una cesta completa exige definir su ventana propia; las compras tardías y decisiones adaptativas serán episodios distintos o quedarán fuera de ese primer alcance.
2. Trade en el tick de muerte no representa preparación previa; S2 necesita otro comienzo y muestreo independiente del desenlace.
3. Perfil hereda historia de hasta diez segundos y una ventana de acción genérica, incompatibles con historia de partidas anteriores y asignación de prácticas. Diseñar ese contrato específico antes de implementarlo.

### Primera implementación elegida

Un candidato económico aislado, compatible con los lectores actuales, debe registrar por ronda/equipo/actor: dinero e inventario previos, lado y contexto legal, instante de observación/disponibilidad, procedencia, reglas/precios versionados, acciones posteriores separadas y reconciliación. Mantener el dinero observado y calculado separados; valores ausentes y cero no son lo mismo.

El primer alcance es acreditar el estado inicial económico y la frontera temporal de sus evidencias. Si el parser no puede demostrar un estado anterior a una compra temprana, conservarla como no evaluable; no desplazar T0 en silencio ni usar el estado de final de frame como si fuera anterior. No se implementan todavía ranking de compras, un nuevo `DecisionExample`, recomendador ni misiones productivas.

Antes de automatizar E1 se necesitarán además cestas legales por jugador, destinatarios, límites de compra/equipamiento, inventario de armadura/kit/utilidad y supuestos de cooperación, así como revisión táctica y elegibilidad por oportunidad. El coste y las reglas del juego se tomarán de fuentes versionadas verificadas para las demos; este diseño no fija precios de armas ni constantes del juego.

## 10. Evaluación y recursos del experimento

Se medirán cosas distintas, con denominadores explícitos:

| Pregunta | Medida y unidad |
|---|---|
| ¿Los datos permiten juzgar? | Oportunidades con todos los requisitos / oportunidades detectadas; por familia, partida y motivo de ausencia. |
| ¿El consejo se sostiene? | Correcciones aceptadas / correcciones evaluables; fidelidad factual y del replay, viabilidad física, preferencia y desacuerdo aparte. |
| ¿Sabe abstenerse? | Abstenciones correctas / casos cuya respuesta esperada es abstención; detectar también correcciones emitidas indebidamente. |
| ¿Aporta al usuario? | Usuarios que consideran al menos un consejo útil y accionable / usuarios evaluados; registrar invitados, ausencias y respuestas faltantes. |
| ¿Se entiende? | Usuario explica la condición y la acción con sus palabras; valoración registrada, no simple clic de aceptación. |
| ¿Se practica? | Plan ofrecido, aceptado, declarado realizado y verificado, con carga y fuente distintas. |
| ¿Se aplica y mantiene? | Conducta adecuada por oportunidad comparable; retención/transferencia y perjuicios al equipo por separado. |
| ¿Es sostenible? | Minutos humanos por caso y partida; latencia de extracción, cálculo y revisión; gasto externo y horas de fundadores. |

Los umbrales de promoción siguen exclusivamente en el plan. El lote de diseño da estimaciones exploratorias y revela fallos; no sirve para declarar precisión del producto. En la evaluación formal se agrupa por partida y se analiza dependencia por jugador; no se calculan intervalos suponiendo independencia de cada tiro.

**Cálculo de esfuerzo propuesto, no cotización:** seis horas de preparación/rúbrica, 40 casos × dos revisores × diez minutos = 13,3 horas, ocho adjudicaciones × diez minutos = 1,3 horas, cuatro horas de revisión de misiones y dos de puesta en común: unas **27 horas de coaches**. Las cantidades y el 20 % de conflictos son supuestos que se sustituirán por mediciones de los ocho primeros casos.

Como ejemplo de sensibilidad presupuestaria, usando una tarifa ficticia de cálculo de 50 EUR/h: 1.350 EUR de revisión, seis participantes × tres sesiones × 15 EUR de compensación por tiempo = 270 EUR, y una provisión de 100 EUR para operación. Con 25 % de margen: **2.150 EUR**. No son tarifas de mercado ni gasto aprobado. La compensación, si se acuerda, no dependerá de mejorar, obedecer o dar una opinión positiva. El escenario ampliable de 10.000 EUR del plan conserva margen para iteraciones; no obliga a gastarlo.

El trabajo de los fundadores se registra aparte: preparación de evidencia, implementación, coordinación y análisis. No se estima su coste total sin medir el primer caso y el primer procesamiento. La CPU existente es la base; este experimento no tiene una necesidad demostrada de GPU ni de entrenar un modelo.

## 11. Alcance de esta entrega y referencias

Se entrega diseño de situaciones, instrumento de revisión, perfil/misiones propuestos, muestreo, evaluación, supuestos de coste y elección de la primera pieza factual. La aprobación de coaches, los 40 casos, los seis jugadores, los contratos runtime nuevos y los resultados del piloto siguen pendientes en el plan.

Fuentes locales: [producto](../AI_COACH_PRODUCT_DEFINITION.md), [contrato humano](../AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md), [catálogo](../../ai_coach/contracts/capability_catalog.json), [arquitectura](AI_COACH_MODEL_ARCHITECTURE.md), [auditoría Mirage](../archive/ai-coach/AI_COACH_MIRAGE_44_DATA_AUDIT_2026-08-27.md), [frontera factual](../archive/ai-coach/AI_COACH_SPATIAL_FACT_BOUNDARY_PHASE_2026-09-04.md) y [materialización espacial](../archive/ai-coach/AI_COACH_SPATIAL_FACTS_CANONICAL_MATERIALIZATION_PHASE_2026-09-05.md). Los hechos del código citados se inspeccionaron en lectura: el grafo no estaba disponible (`Transport closed`), por lo que se usaron búsquedas literales y archivos concretos.

Los pasos siguientes, su prioridad y condiciones de cierre se mantienen sólo en el [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md).
