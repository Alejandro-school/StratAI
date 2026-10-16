# Sonido e información posible — encuadre v0.2

Fecha: 12/10/2026. Estado: **encuadre experto recibido; implementación autorizada
dentro del alcance, con gates factuales pendientes**. Conserva la propuesta
[v0.1](AI_COACH_SOUND_INFORMATION_SCOPE_v0.1.md) y sus fuentes.

El usuario acepta «Sí, con esas estimaciones y límites» y concreta sonido
espacial: arriba, abajo, detrás, izquierda, derecha y otras alturas. Autoriza
aprovechar las posiciones reales de los enemigos: si el sonido de un CT entra
en el alcance de un T, marcarlo para recabar información. El literal, incluidas
sus grafías originales, se conserva en
[respuestas](../../outputs/stratai-sound-framing-20261012/expert-review.json).

Se acuerda una recepción **estimada en 3D**, usando posiciones nativas del
emisor y receptor y un alcance defendible. No depende de que el receptor mire
al emisor ni de un rayo visual libre. La geometría visual no se empleará para
bloquear sonidos detrás de paredes o en otro nivel. No se implementa una
simulación acústica exacta de materiales, ecos o enmascaramiento real.

La identidad y el bando nativos pueden usarse dentro del cálculo y auditoría.
Por esta instrucción se permite marcar la procedencia CT de la señal como
**estimación derivada de datos de demo**, distinguiéndola de reconocimiento
real del jugador. No se convierte en visión de ese CT ni se concede identidad
reconocida, número de rivales o coordenadas percibidas exactas por sonido solo.
Las coordenadas nativas sirven para el cálculo y la procedencia; la información
auditiva publicada conserva su carácter estimado y su fecha.

Familias candidatas: pasos, disparos, recargas, granadas y C4. Hace falta
emisión positiva y alcance suficiente; un acontecimiento de acción no basta
para atribuir escucha. Eventos genéricos mantienen tipo desconocido cuando
no haya vínculo causal acreditado. Movimiento, velocidad, recogidas o el ID
oculto no generan pasos, sonidos o reconocimiento por sí solos.

El alcance nativo se conservará con fuente y unidades. Su semántica y la
cobertura positiva deben contrastarse antes de usarlo; faltas de alcance,
emisión o posición producen abstención. La aceptación experta no acredita
valores nativos ni permite una tabla de distancias inventada. El solapamiento
puede anotarse cuando haya señales coincidentes, sin asegurar que enmascara
una recepción o habilita una salida segura. No se fija un umbral ficticio.

Se conserva comunicación inmediata posible, con observador vivo y posibles
receptores presentes incluidos muertos; causalidad, memorias y antigüedad.
Tres sonidos no se transforman en tres rivales ni en tres lanzadores. La regla
visual acordada de cinco segundos y exclusión de flashes permanece intacta.

Implementar y verificar dentro de este encuadre; resolver detalles técnicos
rutinarios sin volver a consultar estos criterios. Una fuente ausente o un
cambio material del alcance no queda autorizado por esta revisión. Una demo
aprobada por cierre, TEMP nuevo retenido, variantes y conteos positivos,
regresiones y fuentes verificadas. No adelantar interpretación general ni
atribuir mejora táctica a hechos nuevos. Modelo Qwen y límites originales.
