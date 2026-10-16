# Contexto compacto para el modelo local — v1

Contrato de transporte `stratai.replay_model_context@1`. La entrada factual
@8, proyección @10, lectura @1, archivo causal, IDs y criterios de juego se
conservan. No es un resumen ni una selección de hechos.

`payload` contiene el contexto original. `definitions` comparte únicamente
objetos o listas JSON idénticos, de al menos 96 bytes y repetidos. Un objeto
`{"$ref":"D0"}` se sustituye por el valor completo de `definitions.D0`,
recursivamente. D0 es una dirección del diccionario, nunca una cita factual.
Los campos originales `$ref`/`$literal` se escapan con `$literal`, cuyas claves
son literales y cuyos valores siguen las mismas reglas. `source_sha256` enlaza
los bytes UTF-8 JSON compacto originales, incluyendo precisión, tipos y orden.

La reconstrucción debe coincidir byte a byte antes de enviar el mensaje.
Referencias ausentes, ciclos, definiciones no usadas, contratos desconocidos
o hashes distintos se rechazan. Si no ahorra bytes, se entrega el JSON
original, salvo que su discriminador `wire_schema` necesite envoltura.
Cada resultado de consulta usa un diccionario independiente. La contabilidad
de consultas/bytes sigue basada en los hechos originales, sin ampliar límites.

El modelo recibe las reglas de expansión explícitas. Recibe todas las
definiciones usadas, hechos, fuentes, fechas, antigüedad, observadores,
receptores posibles, incertidumbre e identidad local. Sólo las muestras ya
consultadas pueden citarse; el archivo conserva los demás cortes y la frontera
causal original. Compactar no confirma recepción ni percepción humana.

Experimento local `1.1.0-local.1`: formato nuevo para selección de referencias.
Los recibos históricos `1.0.0-local.1` se reproducen con sus mensajes originales.
La validación de `finish` con índices vacíos continúa intacta; una respuesta
inválida se conserva y rechaza. Qwen, digest, think=false y presupuesto vigente
permanecen iguales. Menos bytes no garantiza una lectura táctica mejor.
