# F01-K1 v0.2 — una situación real para revisar juntos

Estado: preparada para la primera lectura del usuario. Sin respuesta ni interpretación del agente añadidas. Esta versión sustituye la situación ficticia v0.1 por un corte real; las respuestas anteriores siguen vinculadas a sus premisas originales.

## Situación

**Mirage, primera ronda, equipo T. Marcador 0–0, cinco contra cinco. Quedan 1:43; han transcurrido 12 segundos de juego desde el fin de la congelación.** Los cinco T tienen 100 de vida. No hay rondas competitivas anteriores de esta partida que usar como patrón rival.

Observamos el despliegue propio en el replay hasta ese instante. El dibujo muestra la posición de cada T, su orientación horizontal y su movimiento durante los **cuatro segundos anteriores**. Las posiciones CT y el futuro quedan fuera de la ficha. La voz y el conocimiento efectivo de cada jugador no están registrados; eso no significa que no se comunicaran ni que no vieran enemigos.

![Posiciones T y movimiento anterior al corte](../../outputs/stratai-f01-k1-review-20260917-0a53280/F01-K1_v0.2_mapa.png)

## Los cinco jugadores

Los nombres T1, T2, T3, T4 y T5 identifican personas distintas en este caso. No indican roles. El mapa permite situarlos sin depender de un nombre de zona automático.

| Jugador | Posición y movimiento hasta este instante | Equipo disponible | Estado actual |
|---|---|---|---|
| **T1** | Punto rojo, el más adelantado hacia B del grupo del pasillo. Avanza hacia la izquierda del plano; orientación casi horizontal hacia la izquierda. | Glock y cuchillo. Sin granadas en el inventario mostrado. | Glock equipada, caminando; 100 de armadura, $150. |
| **T2** | Punto azul, detrás de T1. Avanza en la misma dirección y mira hacia la izquierda del plano. | Glock y cuchillo. Sin granadas en el inventario mostrado. | Glock equipada, caminando; 100 de armadura, $150. |
| **T3** | Punto violeta, detrás de T2 y delante de T4. Avanza hacia la izquierda y mira en esa dirección. | Glock y cuchillo. Sin granadas en el inventario mostrado. | Glock equipada, caminando; 100 de armadura, $150. |
| **T4** | Punto amarillo, detrás de T3. Avanza hacia la izquierda, mirando algo en diagonal hacia abajo en el plano. | **C4**, Glock y cuchillo. Sin granadas en el inventario mostrado. | Glock equipada, caminando; 100 de armadura, $150. |
| **T5** | Punto verde, separado por detrás del grupo. Tras el recorrido dibujado, permanece prácticamente en la misma posición durante el último segundo. | Glock, **una flash, un humo** y cuchillo. | **Humo equipado**, mirada inclinada hacia el suelo unos 75°; 0 de armadura, $100. |

Altura registrada (`z`, unidades del mapa): T1 −80,6; T2 −79,5; T3 −80,6; T4 −80,6; T5 −69,8. Los cuatro primeros están a altura similar y T5 unos 10–11 puntos por encima. Se conserva esta dimensión porque el plano superior no identifica por sí solo la planta ni demuestra que estén dentro de un recinto; no se adopta una etiqueta de planta automática.

Hasta el corte no aparecen lanzamientos T en los eventos de utilidad revisados. Esto no acredita ausencia de contactos visuales, sonido o comunicación. Las flechas representan orientación horizontal, no visibilidad ni cobertura; en T5 la inclinación hacia el suelo es especialmente relevante. Los círculos huecos son las posiciones de hace cuatro segundos.

## Lo que te pido interpretar

**¿Qué plan te sugiere este reparto y qué función atribuirías a cada jugador?** Indica qué señales usas. Si alguna función no puede inferirse todavía, puedes dejarla abierta.

Primero necesitamos saber si ahora puedes situar a todos e interpretar la organización con esta información, o qué dato concreto sigue faltando. Si la planta o el acceso de alguno siguen siendo ambiguos en el plano y las alturas, indícalo antes de atribuirle una función. Todavía no te pido decidir si la jugada es buena, si debían hacer otra cosa ni cómo acabará.

Puedes responder directamente en la conversación. No tienes que editar de nuevo el Excel ni repetir las otras fichas.

## Alcance de esta revisión

- Es una lectura descriptiva del equipo propio, con material real de desarrollo. No es una evaluación ciega, una etiqueta táctica final ni una prueba de un modelo.
- El inventario y el dinero son los mostrados en ese instante; no se reconstruyen compras ni se presupone la economía rival. No se especifica casco a partir del valor de armadura.
- La munición exacta no se utiliza en esta pregunta: el sidecar físico distingue `raw_clip=20` y el hecho exportado `19`, cuya semántica no se resuelve aquí. No se presume capacidad concreta de disparo ni apoyo viable.
- El radar local orienta la lectura. Su correspondencia exacta con los assets de la build y la geometría de contacto siguen sin acreditarse. Los callouts automáticos `Underpass` y `position.alleyac` no se adoptan como etiquetas espaciales validadas.
- No hay evidencia suficiente de comunicación, contactos legales completos, propósito mental o superioridad táctica. Una interpretación razonable no cierra esas dependencias.

## Procedencia y cambio de versión

Corte de `demo-a11eb0d600ee702c9edb`, ronda 1, tick 8524, frame 8542, secuencia 1389, instancia 2 y época 0. Replay y sidecars ya retenidos de 3E.21; reloj y tenencia corroborados además con los contextos conservados de 3E.16 y 3E.15. No se volvió a parsear la demo.

Frente a v0.1 cambian las premisas: 0–0 en vez de 4–4, 1:43 en vez de 1:30, pistolas en vez de rifles, C4 en T4 en vez de T3 y posiciones/recursos individuales observados en vez de un rango ambiguo de actores. Los alias son locales a esta versión. No se trasladan las respuestas v0.1 como aprobación de este corte.

El [informe de preparación](../archive/ai-coach/AI_COACH_F01_K1_PREPARATION_PHASE_2026-09-17.md) y la [evidencia](../evidence/AI_COACH_F01_K1_REVIEW_PREPARATION_2026-09-17.json) conservan comprobaciones y límites. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) determina la continuación.
