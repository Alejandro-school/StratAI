# Handbook de revisión humana espacial

Versión: `1.0.0`  
Protocolo: `stratai.spatial_human_review_protocol@1`  
Uso: piloto Mirage de sólo revisión

## Propósito

Este handbook permite valorar candidatos espaciales sin transformar geometría,
NAV, nombres de semillas ni resultados de una demo en verdad táctica. El piloto
sirve para comprobar que las preguntas, la evidencia y el flujo de desacuerdo
son utilizables. No concede soporte de producto ni autoriza entrenamiento.

## Unidad y evidencia

Cada unidad tiene un ID opaco y un join de auditoría separado. La vista del
revisor contiene únicamente:

- geometría objetiva ya congelada: sector angular, perfil de oclusión de volumen
  o conexión NAV dirigida;
- contexto factual observed disponible hasta T0, si existe;
- límites de la evidencia y opciones cerradas de respuesta.

En el lote package-only de 1B.10 no se abre ninguna demo. Por tanto, el contexto
observed aparece como `unavailable`, nunca como un contexto vacío interpretado
como negativo.

Está prohibido usar posición enemiga oracle, outcomes, futuro, voz, intención,
sonido subjetivo, nombres públicos, Steam IDs, rutas locales o la etiqueta de
otro revisor. Los nombres de coordenadas semilla tampoco son etiquetas tácticas.

## Preguntas

Las tres preguntas conservan la misma escala neutral:

1. `supported`: la afirmación táctica concreta está respaldada por la evidencia
   permitida;
2. `not_supported`: la evidencia permitida contradice o no respalda esa
   afirmación;
3. `missing_context`: faltan hechos observed necesarios;
4. `not_evaluable`: la unidad o su presentación no permiten evaluar;
5. `abstain`: el revisor decide no emitir juicio y registra el motivo.

La pregunta de apertura angular se limita a exposición tácticamente relevante;
la de oclusión, a cover tácticamente utilizable; y la de conexión dirigida, a
salida tácticamente viable. Ninguna pregunta solicita seguridad absoluta,
optimalidad, recomendación o resultado contrafactual.

## Blinding e independencia

El coordinador entrega exclusivamente `reviewer_view`. No muestra package paths,
aliases de semillas, hashes de soporte, anotaciones pendientes, nombres de
familia como metadatos de auditoría, respuestas previas ni orden de generación.
Las unidades de las tres familias se intercalan mediante un orden SHA-256
determinista. Las opciones mantienen el mismo orden para todas las preguntas.

Dos revisores responden de forma independiente. Sus pseudónimos son opacos y no
contienen identidad pública. No pueden consultar ni modificar la respuesta del
otro. Cada respuesta registra versión de handbook, timestamp UTC, hash de la
evidencia visible y una declaración de independencia.

## Conflictos y adjudicación

Dos respuestas compatibles permiten el estado `reviewed`. Una discrepancia,
una abstención que impida decidir o un conflicto factual mueve la unidad a
`needs_adjudication`. El adjudicador debe ser una tercera persona pseudonimizada,
distinta de ambos revisores, y debe registrar los dos response IDs, el motivo del
desacuerdo, la evidencia considerada y un timestamp UTC.

La adjudicación puede terminar en `adjudicated`, `rejected` o `unavailable`.
Nunca cambia geometría ni activa soporte, entrenamiento, materialización
canónica o publicación. El protocolo impide que revisores o adjudicador
autoaprueben un paquete o un mapa.

## Política de cambios

Una revisión humana no reescribe geometría. Si la geometría, sus fuentes, el
package hash, el schema o un join cambian, todas las respuestas dependientes se
invalidan y el lote se regenera con una nueva versión. Una corrección meramente
editorial del handbook también crea una versión explícita y queda registrada en
las respuestas posteriores.

## Estados

- `pending_review`: no existen dos respuestas válidas;
- `reviewed`: existen dos respuestas independientes compatibles;
- `needs_adjudication`: existe conflicto o abstención relevante;
- `adjudicated`: una tercera persona resolvió el conflicto;
- `rejected`: la unidad no debe consumirse por un defecto de revisión;
- `unavailable`: faltan geometría, procedencia o hechos mínimos.

Estos estados pertenecen al flujo humano. No sustituyen los estados de los
paquetes espaciales ni cambian `product_supported=false`,
`training_use_authorized=false` o `canonical_materialization=false`.
