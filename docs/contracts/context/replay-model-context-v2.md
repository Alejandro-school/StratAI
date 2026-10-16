# Contexto compacto para el modelo local — v2

`stratai.replay_model_context@2` conserva todos los hechos, datos y archivo
causal de la entrada @8. Comparte valores idénticos con el diccionario de v1
y nombres de campos repetidos con `columns`. No resume ni agrega acontecimientos.

`columns.C0` es una lista ordenada de claves originales. Una fila
`{"$row":"C0","values":[...]}` reconstruye un objeto emparejando cada valor
con la columna de la misma posición. Tipos, nulos, precisión numérica, orden,
fuentes, fechas, antigüedad, observadores y receptores se mantienen exactos.
Se usa sólo para formas de objeto repetidas al menos tres veces y cuya lista
de claves ocupa al menos 64 bytes. Es un criterio técnico de tamaño, sin
significado de juego. Los IDs C0/D0 no son citas factuales.

`payload`, `definitions` y `$ref` siguen las reglas de v1. `$row_literal`
escapa objetos originales con claves `$row`/`$row_literal`; sus claves son
literales y sus valores se expanden. `$literal` sigue escapando las claves
reservadas del diccionario de valores. El orden de expansión es diccionario,
después columnas. Referencias ausentes, ciclos, columnas no usadas, claves
duplicadas o cantidad de valores distinta de columnas se rechazan.

`source_sha256` enlaza los bytes UTF-8 JSON compacto originales. Antes de
enviar se exige reconstrucción idéntica byte a byte. Si no ahorra bytes se
entrega el original, salvo que su discriminador requiera envoltura. Cada
consulta usa diccionarios independientes y mantiene la contabilidad original.

Experimento `1.2.0-local.1`. Los recibos `1.0.0-local.1` se reproducen sin
compresión y los `1.1.0-local.1` con v1. Se conserva la primera tanda completa,
incluido N10 agotado a 240 s. La segunda vuelve a fijar N01/N04/N10 antes de
generar. Qwen, digest, think=false, timeout, contexto y generación no cambian.
La entrega factual no acredita interpretación táctica ni percepción humana.
