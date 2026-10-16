# Puntos sonoros de la demo para revisión del mapa

15/10/2026. Corrección de alcance recibida del usuario durante la revisión de interpretación. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado de entrega. El [recibo literal](../../outputs/stratai-interpretation-framing-20261015/expert-criteria.json) distingue sus palabras de las decisiones técnicas.

El usuario autoriza aprovechar la posición real del CT para marcarlo cuando una emisión sonora tiene recepción posible por un T. Excluye usar esa ubicación oculta para interpretar la situación o valorar el ataque T. Se mantiene la recepción espacial estimada y los radios ya acordados.

El mapa obtiene una capa independiente mediante `createReplaySoundReviewProjector`. Reutiliza la validación de fuente, perfil, observador vivo y distancia 3D. Después enlaza el emisor nativo con su propia posición capturada entre los participantes del callback de sonido. No confunde el origen de una explosión o un arma con la posición del jugador; si falta ese enlace o su posición, no añade el punto. La misma operación se aplica bajo las perspectivas T/CT.

Cada emisor conserva un marcador local distinto. Varias emisiones del mismo jugador actualizan su punto y varios receptores de una emisión no duplican al emisor. Esto aprovecha atribución de la demo; no deduce personas contando pasos o separando sus ritmos. No acredita identidad reconocida por un jugador.

El punto conserva la posición **al producir el sonido**. Se muestra su antigüedad y se retira después de 1,5 segundos, una decisión de presentación que no impone caducidad a la memoria factual ni modela la duración audible. Sólo otra emisión habilitada lo actualiza. Retroceder en la replay reconstruye los puntos correspondientes a aquel instante; nunca usa posición futura. La altura permanece en el dato y una flecha distingue un punto situado en otra planta.

La capa «Sonido · puntos de demo» puede ocultarse. Su leyenda la identifica como posición de demo con alcance estimado. No garantiza escucha efectiva, dirección percibida o posición actual después del evento.

Estos puntos se pasan directamente del visor al canvas, fuera de `onPerspectiveContext`, del JSON factual exportable, de la memoria del equipo y de la entrada del modelo. Los contratos de proyección @11 y entrada @9 mantienen sus valores. El modelo continúa recibiendo regiones y recepción posible, sin la identidad oculta o las coordenadas de este marcador.

Ejemplo: un CT emite pasos dentro del radio de un T en el tick 100. El mapa puede mostrar la posición nativa de ese CT en el tick 100. Si después camina sin una nueva emisión admitida, el punto anterior permanece fechado y estático hasta retirarse. Dos emisiones del mismo CT no crean dos rivales; dos emisores nativos diferentes conservan dos puntos de revisión.

El segundo criterio recibido corrige el ejemplo de granadas y recarga: la utilidad al inicio de ronda puede ser la acción habitual contra posibles rushes. La coincidencia temporal no demuestra que reaccione a una entrada T. Esta alternativa debe conservarse al desarrollar la interpretación; el criterio no es una etiqueta automática de cada lanzamiento ni una prohibición absoluta de reconocer una reacción respaldada por evidencia adicional.

La medición de entrada se conserva aparte. Detectar una limitación de tokens no autoriza cortar hechos, cambiar Qwen, ampliar su contexto o escoger sólo los casos que caben. Antes de generar se debe resolver y verificar esa dependencia técnica, conservando las lecturas originales y las cuatro partes acordadas.
