# Entrega de contexto por plantillas sin pérdida

Contrato local `stratai.replay_model_context@3`. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado de ejecución. Amplía el [formato @2](replay-model-context-v2.md) para entregar las entradas sonoras ampliadas al mismo Qwen y dentro de sus límites originales. No cambia los hechos, el archivo causal o el bundle publicado.

El formato es opcional. `pack(value)` y los experimentos anteriores mantienen @2; `pack(value, version=TEMPLATE_VERSION)` puede producir @3 cuando ahorra bytes. `unpack` conserva lectura de @1/@2 y valores planos. La nueva versión del adaptador local se selecciona explícitamente. Los recibos históricos no se regeneran ni cambian sus mensajes.

El sobre @3 contiene exactamente `wire_schema`, `source_sha256` y `body`. El hash corresponde al JSON original completo, con orden, tipos y precisión. `body` utiliza el formato anterior y, una vez restaurado, contiene exactamente `templates` y `payload`.

Una instancia `{"$template":"T0","args":[...]}` representa un objeto completo de `templates.T0`. Dentro de su plantilla, `{"$arg":0}` se sustituye por el argumento escalar situado en ese índice. Si existe `$prefix`, éste debe ser texto y se antepone al argumento, también texto. Cada posición de argumento debe utilizarse; no se permiten campos o argumentos sobrantes, índices booleanos, referencias ausentes o ciclos.

`{"$prefix_list":"sound-observation:","suffixes":["1","2"]}` conserva exactamente la lista original de cadenas, en ese orden. No agrupa señales, receptores o personas. Los marcadores de control que aparezcan en datos originales se protegen mediante `$template_literal`; sus claves se conservan literalmente.

Se comparten sólo estructuras o fragmentos de texto idénticos. Valores dinámicos repetidos dentro de una fila pueden referirse al mismo argumento; esto no fusiona filas. Fechas, antigüedad, IDs, incertidumbre y receptores siguen presentes. El decoder rechaza cualquier reconstrucción cuyo hash no coincida con el original. La aplicación verifica esa reconstrucción antes de enviar el mensaje.

Los IDs T0/D0/C0 son nombres de transporte, no evidencias citables. El modelo recibe instrucciones para restaurar primero el cuerpo anterior y después las plantillas. Sólo puede citar IDs factuales efectivamente entregados. Las consultas conservan el presupuesto original calculado sobre sus respuestas factuales, sin obtener consultas extra por ahorrar bytes.

El modelo, digest, contexto 65.536, salida máxima 8.192, ocho llamadas, 240 segundos, seed y configuración sellada permanecen intactos. Una entrada admitida no garantiza que futuras consultas o toda la salida máxima quepan. El exceso debe conservarse como límite explícito, sin cortar entrada ni desplazar contexto. La reducción de tamaño y la reconstrucción matemática no acreditan comprensión táctica de la representación.

Las posiciones nativas de los nuevos puntos de revisión del mapa continúan fuera de la entrada del modelo. Se conserva el criterio recibido sobre utilidad inicial habitual y ausencia de reacción demostrada por mera coincidencia temporal. Las pruebas no autorizan entrenamiento, promoción, corpus masivo o etiquetas tácticas nuevas.
