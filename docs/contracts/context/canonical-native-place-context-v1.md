# Lugar nominal nativo como contexto candidato

Esta capa conserva la propiedad `m_szLastPlaceName` del pawn sin pasar por
`MapManager`, semillas de callouts o NAV. Es una fuente nominal, no una definición
de zona física, ruta, percepción individual o intención. El
[plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y siguiente acción.

## Captura separada

El comando Go `cmd/native-place-candidate` recibe una demo, su SHA-256 esperado,
un archivo de targets con su SHA-256 esperado y un directorio TEMP nuevo/vacío.
No descarga, modifica, promueve ni elimina fuentes o salidas.

`stratai.native_place_targets@1` vincula fuente/alias/mapa y cada corte exacto
(frame, tick, instancia, época, ronda y fase) con el hash del contexto factual.
Cada actor tiene alias opaco local, handle, lado, conexión, vida y posición
esperados; los desconocidos se mantienen nulos. El documento rechaza claves
duplicadas, campos desconocidos y metadatos incompletos. Los targets derivan de
los cortes factuales seleccionados, sin respuestas expertas ni destinos deseados.

La lectura ocurre al final del frame, con cola de mensajes desactivada. El
parser se detiene en el último corte solicitado. El vínculo de pawn comprueba
índice y serial del handle. Los campos observados conocidos deben coincidir con
el target; una identidad incompatible no se mezcla con otra vida.

Las entregas de propiedad se fechan en el `FrameDone` correspondiente, después
de los eventos de ese frame. Se excluye el callback inmediato que el SDK ejecuta
al registrar `OnUpdate`: registrarse no acredita una nueva entrega de la demo.
La última entrega se limita a la misma instancia y época. Un valor heredado
puede tener entrega desconocida y continuar visible como propiedad actual.

Los estados distinguen string positivo, vacío, propiedad ausente, valor sin
inicializar, tipo inesperado, entidad no disponible y error de lectura. No se
convierte cualquier tipo a texto. El acceso crudo y la API `LastPlaceName()` se
contrastan cuando hay string. Vida, conexión, lado, posiciones e identidad se
retienen con sus estados, sin nombres de participantes ni SteamID.

El export conserva `snapshots.jsonl`, `deliveries.jsonl`, `state_history.jsonl`
y `manifest.json`. El manifiesto contiene hashes antes/después de demo/targets,
bindings, header de mapa/build/patch, versión del SDK, binario y fuentes Go
embebidas, inventario y conteos. Declara expresamente que no acredita zonas,
no produce interpretación táctica y no permite entrenamiento.

## Unión con la ficha

`stratai.native_place_context@1` se añade como archivo separado a
`stratai.candidate_context@1`; no modifica ese contexto sellado. El adaptador
Python verifica los hashes esperados del manifiesto y sus tres artefactos,
selecciona un target único y verifica fuente, mapa, corte, contexto y roster.
Cada observación se une por alias, handle/vida, lado, conexión, vida y posición.
Los lugares de muertos o actores no vinculables no se usan como lugares de
supervivientes actuales.

El contexto conserva la propiedad, su tipo/estado, el valor actual y su última
entrega causal, además de las referencias hash a la fuente completa. La edad
de esa entrega no es edad de una posición ni un plazo de validez inventado.
Un error actual no se rellena con el último string positivo para emitir una
hipótesis; su entrega histórica queda diferenciada.

Sin export, el estado es `unavailable`, motivo `native_source_not_materialized`,
sin observaciones ni procedencia fabricadas. Ausencia de captura no demuestra
ausencia de una organización de juego.

La incorporación es información causal adicional extraída posteriormente.
Debe declararse como tal al compararla con un detector o una revisión anteriores.
La captura nominal no acredita por sí sola el contorno de «spawn» ni la fase
de movilización inicial. La [orientación candidata](canonical-destination-orientation-v1.md)
puede usar el valor nominal manteniendo explícitos esos límites.

## Verificación

Las pruebas cubren tipos/ausencias, causalidad, identidad, cortes alterados,
seriales reutilizados, hashes y destinos inválidos. El cierre funcional exige
la demo única de AGENTS.md después de las pruebas, con contraste independiente
de targets, valores, entregas, reglas derivadas y artefactos no vacíos. La
aplicación posterior del productor congelado a los restantes cortes TP10 se
registra por separado, sin alterar la selección ni convertir ese lote en otro
smoke o en evaluación táctica independiente.
