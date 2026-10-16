# C4 causal en el instrumento de replay por bando

Contrato acotado del 09/10/2026. [Cierre y evidencia](../../../outputs/stratai-c4-repair-20261009/RESULTADO.txt).
El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y siguiente acción.

## Versiones y alcance

El replay base permanece en schema 5. Su objeto `bomb` incorpora la extensión
`stratai.replay_objective@2`, con tick de observación y posición Z. Los eventos
objetivo añaden secuencia nativa, bando del actor, estado/fase posteriores,
disponibilidad y referencias. La proyección nueva es
`stratai.team_replay_projection@4`; el instrumento optativo de desarrollo es
`stratai.replay_context_input@2`. El consumidor conserva la vía histórica
input @1/proyección @3. No migra los artefactos sellados ni amplía los 116
schemas runtime registrados, catálogo 1.7.0 o bundle canónico 3.8.

## Significado factual

- Estado físico: `carried`, `dropped`, `planting`, `planted`, `defusing`,
  `defused`, `exploded`, `resolved` o `unknown`, según la fuente causal.
- Fase del objetivo y banderas de plantado se conservan separadas. `resolved`
  con `was_planted_this_round=true` no significa que siga plantado ni acredita
  desactivación/explosión. No resuelve la elegibilidad de la ronda.
- `carrier_id` propio debe corresponder a un jugador vivo único. `has_c4`
  propio refleja esa tenencia; no se deduce de un inventario retenido ni
  demuestra intención. Una muerte sin transición disponible invalida la
  tenencia sin inventar una caída o su ubicación.
- Se conservan caída, recogida, snapshot de portador, inicio/aborto de plantado,
  plantado, inicio/aborto de desactivación, desactivación y explosión. El orden
  dentro del mismo tick sigue la secuencia del tracker, no el nombre del evento.
- Cada evento debe estar ocurrido y disponible antes del corte. Sus coordenadas
  de callback no son posición del C4: se omiten de la proyección y el productor
  las marca indisponibles, aunque pertenezcan al actor del evento.

## Ubicación y perspectiva

Una coordenada actual exige muestra nativa exacta, finita, observada y permitida
por la perspectiva. El replay no recicla como actual la posición anterior retenida
por el tracker. Entre muestras no hay `x/y` actuales: puede conservarse
`last_observed_position`, con antigüedad y fuente explícitas. Una nueva transición
o una muestra sin geometría retira ese dato histórico. La ausencia de ubicación
no elimina un estado acreditado.

Para T se aplica el criterio recibido de conocimiento de C4 caído. Se conserva
la tenencia propia y ubicación nativa permitida; el plantado propio puede acreditar
ubicación. El intento oculto de desactivación CT permanece como plantado para T.
Para CT, los eventos públicos no revelan actor, sitio o coordenadas globales
ocultas. Las transiciones propias, el contacto propio de desactivación y el final
público de ronda conservan su información permitida. CT sin fuente propia no
recibe la ubicación de la bomba transportada/caída en preplantado.

Los hechos de posición del C4 proceden de la observación runtime de demo; esta
extensión no acredita la transformación física de attachment de los contratos
canónicos 3E.7/3E.18 ni sustituye sus gates. Los contactos calculados conservan
`stratai.calculated_vision@2` y productor `go.reaction.world_occlusion_v2` como
estimaciones; no se convierten en visión exacta.

## Consumo, revisión y compatibilidad

Input @2 entrega tenencia propia, objetivo y eventos fechados con alias locales,
fuentes y restricciones. El lector rechaza coordenadas actuales con tick ajeno,
historial sin referencia, eventos futuros y actores ajenos indebidamente revelados.
Las diez ventanas corregidas contienen los mismos 6840 ticks/cortes históricos;
la información posterior de ronda completa sólo sirve para revisión asistida.

Las respuestas del 06/10 permanecen literales: el modelo no recibió la corrección.
El visor indica «LECTURA ORIGINAL» y registra el suplemento nuevo con su hash y
el hash del input previo. Se han hecho cero llamadas nuevas al modelo. Las notas
no se convierten en etiquetas, criterios aprobados o aprendizaje automático.

## Evidencia y límites

Regresiones por estado, frescura, causalidad, orden, muerte del portador y
perspectiva; `check:all` y validaciones del consumidor pasan. Una demo aprobada
aislada, posterior a las pruebas: 20 rondas, 23398 muestras y 94 eventos C4;
6840 vistas nuevas verificadas, incluidas 3365 caídas recuperadas. Aborto de
desactivación, ausencia nativa de geometría y ciertos conflictos sólo tienen
cobertura de regresión: esta demo no contiene todas las variantes posibles.
Compatibilidad semántica exacta de las diez entradas @1 acreditada contra el
lector previo. No acredita calidad táctica general ni evaluación independiente.
QA visual de navegador pendiente por política local de URL; Excel revisado
visualmente y 14 hojas anteriores verificadas.
