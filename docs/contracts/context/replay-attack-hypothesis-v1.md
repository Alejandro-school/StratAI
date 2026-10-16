# Hipótesis de ataque B — instrumento local v1

Contrato optativo `stratai.replay_attack_hypothesis@1`, adaptador `1.4.0-local.1`. Se apoya en la entrada factual protegida @6–@9, conserva la proyección y utiliza el transporte reversible @3. No sustituye formatos históricos ni habilita recomendaciones de producto. Estado y siguiente acción: [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md).

## Encuadre experto

El 15/10 el usuario concreta: «En el punto 2 se puede considerar si hay 2-3 jugadores en esa zona». El [mapa revisado](../../../outputs/stratai-attack-depth-framing-20261015/revision-b-apps.html) sitúa el punto 2 en XY `(-1500, 750)` del juego. La referencia local de Apartments sigue siendo un único volumen. No se crea un nuevo volumen ni un radio táctico.

Dos o tres T agrupados cerca de esa referencia pueden apoyar **posible ataque a B**. Estar en otra parte de Apartments, llevar C4 o una coincidencia entre utilidad inicial y recarga no bastan. La cercanía y agrupación son una estimación espacial del modelo que requiere revisión; una distancia euclídea calculable no es distancia transitable ni prueba táctica. La compatibilidad exacta de la geometría local con la build sigue sin acreditarse.

## Entrada y salida

`AttackHypothesisSession(packet)` valida primero el paquete mediante `ReplayReadingSession`. Añade sólo `hypothesis_contract`: criterio, punto aprobado, los otros dos puntos de orientación, referencias elegibles, distancias XY en unidades del juego, soportes de ronda/objetivo, alternativas y límites. El recibo conserva el hash del paquete original, cambia el hash de la entrada suministrada y declara el contrato de lectura nuevo. Los hechos originales permanecen idénticos.

Son elegibles referencias **actuales** de T vivos, con coordenadas finitas y zona Apartments única en el mismo corte. La ronda y el objetivo deben ser actuales, estar en preplant activo y tener C4 llevado o caído; una pausa acreditada excluye la hipótesis. Ronda terminada, plantado, intento de plantado, objetivo ausente, altura incompatible, frontera/ambigüedad o posiciones antiguas no habilitan la regla. La elegibilidad no clasifica cercanía ni selecciona automáticamente ataque.

La selección nueva dentro de `interpretation` es:

```json
{"hypothesis_id":"possible_attack_b_advanced_apps","evidence":["ID_de_T_actual_1","ID_de_T_actual_2"]}
```

Admite exactamente dos o tres referencias distintas, una hipótesis como máximo y ningún texto libre, cuenta o posición aportada por el modelo. Las ocho reglas condicionales previas siguen disponibles. Una lista vacía es una abstención válida.

La aplicación adjunta y verifica los hechos de jugador, zona, ronda y objetivo. Reconstruye nombres, número y fecha, con `kind: inferred`, texto cerrado y un campo `hypothesis` que guarda el ID y las referencias seleccionadas. Se explicitan alternativas —preparación, repliegue o cambio de destino— y ausencia de confirmación de plantado o apoyo desde Medio. La verificación vuelve a renderizar y compara toda la salida: cambiar texto, soportes, cuentas o hipótesis invalida la lectura.

## Límites y compatibilidad

La integridad factual y de citas se comprueba mecánicamente. La decisión del modelo de que el grupo está suficientemente cerca **no se acredita como verdad táctica**; no hay una frontera métrica experta aprobada. Se prohíbe presentar la elegibilidad o los tests como evaluación independiente de esa decisión.

El adaptador nuevo exige la sesión nueva y viceversa. Las sesiones, mensajes, recibos y lecturas previas conservan su comportamiento. El modelo sigue siendo Qwen3.5:9b con digest original; no se amplían contexto, salida, consultas o reintentos. La función `interpret` conserva su versión por defecto histórica: el experimento nuevo requiere selección explícita.

Las posiciones nativas ocultas de sonidos siguen fuera de la entrada de interpretación. El instrumento usa posiciones propias/aliadas y los hechos entregados, sin nuevas coordenadas rivales. No interpreta intención de plantar, fake, rotación, eficacia, organización defensiva ni calidad de una decisión.

## Comprobación

Casos sintéticos cubren grupos de dos/tres, conteo por identidad, exclusión de hechos antiguos/muertos/ambiguos, cierre de ronda, abstención, reglas anteriores, corrupción de texto/soportes y replay del adaptador. La mutación que admite un solo jugador debe fallar. El cierre funcional exige suites públicas, AI Coach completo y una demo aprobada en TEMP nuevo; el productor cambiado se ejecuta sobre sus proyecciones reales y sus distancias se recalculan independientemente.

La comparación fijada conserva N01/N04/N10. Un corte real adicional de la misma demo se elige por geometría de entrada antes de inferencia para cubrir el criterio, sin convertirlo en test reservado ni etiqueta experta. Cada caso tiene un intento; se conservan abstenciones y fallos. Medición de entrada completa antes de generación, originales y comentarios preservados, sin entrenamiento o promoción.

La integración encontró repetición de hipótesis en la generación; las respuestas afectadas se rechazaron. Este contrato permanece como referencia reproducible. La versión final del instrumento usa el [contrato acotado v2](replay-attack-hypothesis-v2.md), que conserva el criterio experto y alinea gramática y verificación.
