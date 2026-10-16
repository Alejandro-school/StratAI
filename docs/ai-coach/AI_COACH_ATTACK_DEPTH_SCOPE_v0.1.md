# Profundidad de B Apps y contrato de hipótesis — encuadre parcial

15/10/2026. Criterio experto recibido; **sin implementación de clasificación o nuevo esquema runtime**. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y siguiente acción.

## Criterio recibido

> Si estan muy dentro de B apps, es decir, apunto de entrar a B, si se podria acreditar como ataque, si estan mas atrás, no tanto

La profundidad importa dentro de una misma zona. Un grupo muy avanzado, próximo a entrar en B, puede respaldar la lectura de ataque aunque todavía esté en B Apps. Una posición más retrasada aporta menos fundamento. No equivale a una prohibición universal de ataque desde atrás ni a que cualquier jugador adelantado pruebe por sí solo un ataque del equipo.

El usuario no ha fijado distancia, tiempo universal, cantidad mínima de jugadores, probabilidad, frontera exacta o criterio suficiente para intención de plantar. Su respuesta ajusta la lectura de ataque del par ilustrativo; no se convierte en aprobación de todas las hipótesis sugeridas. Entrada, ataque, intención de plantar y plantado confirmado permanecen distintos.

## Fuente espacial comprobada

Se han revisado `ai_coach/interpretation/maps/de_mirage_place_volumes.json`, `place_volumes.py`, `replayPlaces.js` y el [contrato de zonas](../contracts/context/replay-places-v1.md). El paquete contiene 23 volúmenes con nombre y un único volumen `Apartments` con un hull. Esa referencia agrupa la zona, sin subregiones de profundidad ni portal aprobado para este criterio. La tolerancia geométrica de 0,0001 es numérica, no táctica.

El [mapa de revisión](../../outputs/stratai-attack-depth-framing-20261015/revision-b-apps.html) superpone la proyección 2D de Apartments y BombsiteB sobre el radar local. Tres puntos sirven sólo para orientar la respuesta experta. No son jugadores ni nuevos datos de una demo, no definen un orden de avance y no son límites aprobados. La vista no acredita altura, transitabilidad o compatibilidad exacta con una demo. Los volúmenes y el radar originales quedan intactos.

## Alcance propuesto y dependencias

La salida futura separará hipótesis tácticas, evidencias entregadas, alternativas e incertidumbre. Conservará los hechos numéricos reconstruidos y las restricciones de C4. Para este criterio, usaría únicamente posición propia/aliada disponible, evolución causal y una referencia explícita del tramo; no posiciones ocultas rivales o el punto sonoro de revisión.

Falta concretar qué tramo se entiende por «a punto de entrar». Una distancia euclídea a cualquier cara del volumen no identifica una salida, y un solape 2D con BombsiteB no acredita proximidad transitable a ella. No se inventa un umbral para hacer operativa la frase. Se prepara una revisión gráfica antes de la implementación dependiente, según AGENTS.md.

Ejemplos y excepciones que deberá conservar el diseño:

- Grupo en el tramo final acordado: puede respaldar hipótesis de ataque; no confirma intención de plantar ni apoyo efectivo desde Medio.
- Mismo grupo más atrás: mantener abierta la lectura y explicar su menor fundamento espacial; no convertirlo automáticamente en fake, rotación o ausencia de ataque.
- Frontera, solape, altura incompatible, referencia ausente o sólo posiciones antiguas: conservar la incertidumbre correspondiente, sin asignar la subregión a la fuerza.
- Ronda terminada o C4 ya plantado/resuelto: integrar ese estado; no describir una entrada o un plantado actual incompatible.

## Verificación prevista

Tras resolver el encuadre: casos de posiciones a ambos lados del límite acordado, alturas, fronteras, retroceso y hechos históricos; evidencias no consultadas rechazadas; números y C4 intactos; formatos anteriores reproducibles. Medir las peticiones finales antes de generación con el mismo Qwen y sus límites. Un cambio funcional de hechos/esquema exige el smoke de una demo del proyecto después de las pruebas; este encuadre documental no lo ejecuta ni acredita.

El mapa se comprueba en Chromium, la fila IC67 se añade en una copia Excel con preservación de las partes anteriores y se verifican los enlaces documentales. No se ejecutan nuevas generaciones, procesamiento de demos, entrenamiento, adquisición, promoción o limpieza de temporales.


## Continuación del mismo día

El usuario concreta el punto 2 del mapa y un grupo de 2–3 jugadores. La dependencia de encuadre anterior queda resuelta para la primera hipótesis local de posible ataque B; no se ha aprobado un radio métrico ni intención de plantar. La implementación conserva esa cercanía como estimación del modelo. Véanse el [contrato implementado](../contracts/context/replay-attack-hypothesis-v2.md) y el [resultado](../../outputs/stratai-attack-hypothesis-20261015/RESULTADO.md). Los apartados anteriores conservan la propuesta y limitaciones originales del encuadre.

## Propuesta de representación espacial — 16/10/2026

El diagnóstico posterior conserva la estimación sin demostrar discriminación táctica fiable. Se presenta para revisión un tramo visible: desde el paso del tabique junto a 2 hasta la salida B, incluido el codo junto a 3. El [mapa concreto](../../outputs/stratai-attack-region-framing-20261016/revision-region-b-apps.html) y su [alcance con ejemplos y verificación](../../outputs/stratai-attack-region-framing-20261016/PROPUESTA.md) distinguen ese borrador del criterio ya aprobado. Contorno y extensión pendientes de respuesta experta; sin cambio de runtime, radio nuevo o etiqueta automática. Excel v59/IC71 apoya la revisión. El plan canónico fija el estado y la siguiente acción.

## Aceptación e implementación del tramo — 16/10/2026

El usuario responde «Sí, ese tramo completo». Se acepta desde el paso del tabique junto a 2 hasta la salida B, incluido el codo junto a 3, manteniendo grupo de 2–3. El [contrato v4](../contracts/context/replay-attack-hypothesis-v4.md) implementa pertenencia calculada sobre el trazo revisado y el interior único del hull 3D original; conserva la condición de aproximación local y build no verificada. No se introduce un radio ni se genera ataque automáticamente. Los apartados previos mantienen la historia del encuadre.

El [resultado](../../outputs/stratai-attack-region-20261016/RESULTADO.md) y la [comparación](../../outputs/stratai-attack-region-20261016/COMPARACION.md) separan cálculo verificado de la selección real de Qwen: W01 tiene P03/P05 dentro y P04 fuera; la interpretación queda vacía en las cuatro lecturas. Excel v60/IC72 registra el criterio aceptado sin sustituir el plan como fuente de estado. Evaluación táctica independiente pendiente.
