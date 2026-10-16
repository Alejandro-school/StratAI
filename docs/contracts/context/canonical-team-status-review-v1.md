# Recuento de vivos en fichas de revisión v3

Contrato candidato `five_per_side_current_native_alive@1`, 25/09/2026.
La [ficha v3](../../../ai_coach/interpretation/review_packet_v3.py) añade recuentos de
aliados y rivales por muestra y un resumen de hechos. La interpretación táctica
permanece separada: este módulo no reconoce defensa posplantado ni califica posiciones.

## Fuente y alcance

`support_observations@1` aporta alias opaco, lado, conexión, identidad de pawn y
estado vivo/muerto. `team_status.sequence` proyecta únicamente esos campos.
No consume posiciones, armas, salud, inventario, comunicaciones ni información
posterior al corte. Los datos completos se validan por el lector offline; la
proyección factual no depende de posiciones o recursos ocultos del rival.

El ámbito es **5v5**. Un recuento exacto requiere cinco registros actuales únicos
por lado, conectados, identidad resuelta, handle/life no ambiguos y booleano
vivo/muerto observado. Las identidades vistas en el prefijo establecen el censo
de la ronda. Los cambios de lado o conflictos dentro de esa ronda impiden dar
un total exacto. El censo se reinicia al cambiar epoch, instancia o ronda.
No se conserva el valor vivo/muerto anterior para rellenar una ausencia.

Con datos parciales, `alive` es nulo: `observed_alive` y `observed_dead` son
recuentos confirmados y `unknown` conserva los estados sin acreditar hasta los
cinco participantes del ámbito. Más de cinco identidades/registros por lado
produce `unavailable`, sin forzar el caso a cinco. Fuera de `playing` tampoco
se publica un total exacto. Otros formatos de partida requieren otro contrato.

Cada fila contiene `source_sequence`, `roster_start_sequence`, `allies` y
`opponents`. La fila coincide con la frontera completa de su muestra original.
El manifiesto conserva hashes de fuentes, radar y las tres plantillas. El
validador recalcula JSON y HTML desde sus fuentes; rechaza divergencias.

## Presentación y límites

El resumen sigue el control temporal y combina plantado observado, vivos y reloj
nativo de bomba, si está disponible. «Bomba plantada» describe un hecho; no equivale
a inferir «están defendiendo bien». Los datos desconocidos se muestran como tales.
El resumen no muestra predicciones durante la revisión humana y desaparece junto
con la ficha táctica cuando la ronda está excluida por el filtro existente.

La v3 incorpora los campos de v2 sin alterarlos. Las fichas v1/v2, las lecturas
expertas y las predicciones congeladas conservan sus bytes e IDs. Las respuestas
del experto no entran en el cómputo. El nuevo recuento se entrega al revisor, no
se introduce silenciosamente en el detector congelado. No implica entrenamiento.

## Verificación prevista

Pruebas de muerte, datos parciales, desconexión, identidad duplicada, cambio de
lado/ronda/epoch, conflictos, futuro y campos ocultos irrelevantes. Materialización
y validación en una demo aprobada después de las pruebas, con salida TEMP vacía,
contraste de cada recuento contra bajas públicas anteriores y conservación de
originales. El cierre y los resultados efectivos constan en el plan canónico.
