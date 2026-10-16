# F01-K2 v0.2 — qué cambia respecto al primer corte

Ficha real de desarrollo preparada para la lectura del usuario. Se conservan los mismos cinco jugadores de [F01-K1 v0.2](AI_COACH_F01_K1_REVIEW_v0.2.md). No contiene una nueva interpretación del agente ni una corrección táctica.

## Situación

**Mirage, primera ronda, equipo T. 0–0, cinco contra cinco y todos los T con 100 de vida. Han pasado 1,75 segundos desde K1. Quedan 101,25 segundos de ronda y han transcurrido 13,75 desde el fin de freeze.** Son referencias temporales de la demo, sin atribuir conciencia exacta del reloj a los jugadores.

En K1 interpretaste un plan de entrada en B, con T5 encargado de la utilidad y el grupo entrando mientras protege a T4, portador de la bomba. Aquí se muestra únicamente lo ocurrido hasta el nuevo corte; la lectura anterior permanece vinculada a K1.

![Comparación del despliegue T en K1 y K2](../../outputs/stratai-f01-k2-review-20260917-75f20cf/F01-K2_v0.2_mapa.png)

## Cambios y continuidad

- **T1** ha avanzado hacia la izquierda del plano. Al final reduce ese avance: en las tres últimas muestras ocupa la misma posición horizontal del replay, durante **0,125 segundos**. Esto no demuestra una espera deliberada ni velocidad física cero.
- **T2, T3 y T4** continúan avanzando. T4 sigue llevando la bomba y cambia la dirección de la mirada, como muestra su flecha.
- **T5** mantiene su posición horizontal, el humo equipado y la flash en el inventario. Cambia la inclinación de la mirada: de unos **75° hacia el suelo en K1** a unos **8° por encima de la horizontal en K2**.
- Hasta K2 no aparece ningún lanzamiento propio en los eventos revisados. Esto no acredita ausencia de contactos, sonido o comunicación.

El corte se sitúa en la primera interrupción del avance de T1 que queda corroborada por posiciones horizontales repetidas. Los giros de mirada de T5 y T4 ocurrieron antes y también se muestran; no se afirma que la interrupción sea el primer cambio tácticamente significativo de toda clase.

## Los cinco jugadores en K2

Los colores y alias identifican a las mismas personas que en K1. Las líneas son movimiento registrado y las flechas orientación horizontal; no representan visibilidad ni cobertura.

| Jugador | Movimiento y orientación | Equipo y recursos actuales | Altura z |
|---|---|---|---|
| **T1, rojo** | El más adelantado del grupo del pasillo. Avanza y termina en las tres posiciones horizontales repetidas indicadas arriba. Mira hacia la izquierda y ligeramente hacia arriba del plano. | Glock equipada, cuchillo, sin granadas mostradas; 100 de armadura, $150. | −81,1 |
| **T2, azul** | Sigue avanzando detrás de T1, con la mirada hacia la izquierda y algo hacia arriba del plano. | Glock equipada, cuchillo, sin granadas mostradas; 100 de armadura, $150. | −79,7 |
| **T3, violeta** | Sigue avanzando detrás de T2 y delante de T4; mira casi hacia la izquierda del plano. | Glock equipada, cuchillo, sin granadas mostradas; 100 de armadura, $150. | −80,5 |
| **T4, amarillo** | Sigue avanzando detrás de T3; ahora mira en diagonal hacia abajo y la derecha del plano. | **C4**, Glock equipada y cuchillo; sin granadas mostradas; 100 de armadura, $150. | −79,7 |
| **T5, verde** | Permanece separado por detrás, sin desplazamiento horizontal resuelto desde K1. La mirada se ha elevado y su orientación horizontal es hacia la izquierda y abajo del plano. | **Humo equipado**, una flash, Glock y cuchillo; 0 de armadura, $100. | −69,8 |

Las alturas se conservan porque una vista superior no identifica por sí sola la planta, el recorrido transitable ni los ángulos visibles. No se usan los callouts automáticos ambiguos como etiquetas espaciales verificadas.

## Pregunta

**Respecto a K1, ¿qué se mantiene y qué cambia en tu interpretación del plan y las funciones de los jugadores, y qué señales nuevas te llevan a esa lectura?**

Puedes mantener la lectura anterior si estos cambios no justifican modificarla, o señalar qué sigue sin poder saberse. No hace falta revisar otra vez el formato ni editar el Excel.

## Procedencia y límites

- Misma partida aprobada `demo-a11eb0d600ee702c9edb`, misma primera ronda. K2: tick **8636**, frame **8654**, secuencia **1417**, instancia **2**, época **0**. K1: tick 8524, frame 8542, secuencia 1389.
- Posiciones, orientación, recursos y continuidad proceden de replay/support retenidos de 3E.21. Reloj y C4 se contrastan además con los contextos conservados de 3E.16 y 3E.15. No se vuelve a parsear la demo.
- Replay/support muestrean cada cuatro ticks. `player_states` muestrea cada 32 y no tiene un registro exacto en 8636; no se sustituye por uno anterior o posterior. Las afirmaciones actuales se limitan a las fuentes exactas disponibles.
- La posición horizontal del replay está redondeada. Repetir x/y describe esos registros, no inmovilidad física absoluta ni intención de esperar. La orientación tampoco acredita contacto visual.
- Posiciones CT ocultas y futuro quedan fuera de la ficha. Contacto legal completo, comunicación, conocimiento individual, readiness del arma y apoyo físico no están acreditados. La munición exacta no se usa.
- La correspondencia exacta del radar con los assets de la build continúa pendiente. Se conserva como ayuda de localización ya aceptada, no como prueba geométrica.
- Esta es revisión abierta de desarrollo. Durante la preparación se inspeccionó un prefijo propio hasta tick 8652; las muestras posteriores a 8636 se excluyen de la ficha y del prefijo mostrado. No se declara selección ciega.

La [evidencia](../evidence/AI_COACH_F01_K2_REVIEW_PREPARATION_2026-09-17.json) conserva fuentes, corte, selección y verificaciones. La aplicación documental de la rúbrica está separada en el [informe de preparación](../archive/ai-coach/AI_COACH_F01_K2_PREPARATION_PHASE_2026-09-17.md); no se añade como respuesta humana a esta ficha.
