# Manual de revisión espacial situada

Versión: `2.0.0`  
Protocolo: `stratai.spatial_context_review_protocol@2`

## Qué se revisa

Cada caso procede de una única demo Mirage aprobada y muestra sólo información
disponible hasta el instante T0. La tarea es comprobar si la conclusión
geométrica presentada coincide con la evidencia situada en el radar. No se
valora si la jugada ganó, si era óptima ni qué habría ocurrido con otra acción.

El radar debe mostrar la posición observada del participante mediante un alias
opaco. Exposure superpone el sector y sus rayos; cover superpone la dirección de
entrada y acompaña el mapa con los nueve probes del volumen; exit-route muestra
los extremos y la polilínea completa de centros NAV usada para calcular la
distancia. Una flecha recta sin recorrido NAV no es evidencia válida.

## Cómo responder

- `supported`: la evidencia visible basta y coincide con la afirmación.
- `not_supported`: la evidencia visible basta pero contradice la afirmación.
- `missing_context`: falta un hecho necesario que debería haberse mostrado.
- `not_evaluable`: el mapa, la superposición o los datos son ilegibles o están
  rotos.
- `abstain`: la persona no puede emitir el juicio con la confianza requerida.

No se debe imaginar información ausente. `not_supported` no significa “no me
gusta la jugada”; exige una contradicción visible. `missing_context` no debe
usarse por costumbre: el constructor impide entregar unidades sin radar,
posición T0 y evidencia específica de su familia.

## Límites causales

Se permiten ronda, tick, posición, orientación, postura, arma y velocidad cuando
proceden del sidecar causal de T0. No se permiten nombres públicos, Steam IDs,
voz, intención, futuro, outcome, ganador ni posiciones enemigas obtenidas por
oracle. La revisión no convierte conectividad en seguridad ni geometría en una
recomendación.

## Independencia y adjudicación

Dos personas revisan por separado y reciben únicamente `reviewer_view`. Sus
pseudónimos no contienen identidad pública y ninguna ve la respuesta de la
otra. Un desacuerdo, una abstención material o una disputa de evidencia requiere
una tercera persona distinta. La adjudicación nunca reescribe geometría ni
concede soporte de producto o autorización de entrenamiento.

## Gate de evidencia

Una unidad no se entrega si falta contexto observado, registro radar o evidencia
específica. Para exit-route se exige una polilínea NAV con al menos origen y
destino y una medición que coincida con el sidecar. Cualquier cambio de demo,
sidecar, paquete, NAV, radar, protocolo o evidencia visible invalida las
respuestas anteriores y requiere un lote nuevo.
