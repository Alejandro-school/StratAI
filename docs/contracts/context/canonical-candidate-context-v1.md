# Contexto factual candidato para interpretación

Contrato candidato de la iteración autorizada el 1 de octubre de 2026. El
[plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y siguiente
acción. La incorporación de hechos no acredita interpretación táctica.

La entrada `stratai.candidate_context@1` reúne el packet factual v3 y los
suplementos de revisión ya verificados. Mantiene el contexto original, los
estados del objetivo, utilidad, elegibilidad, relojes, recuentos parciales y
eventos de combate/C4 sin mezclar respuestas humanas ni predicciones previas.

Dos instrumentos explícitos: @5 incorpora los hechos de bajas, conservando
ausentes los suplementos posteriores; @9 incorpora también daño positivo,
drops/recogidas y posición nativa exacta de C4 caído. En TP10, E03 corresponde a
@5. Un experimento @9 sobre ese caso debe declarar información adicional y no
presentarse como igualdad de contexto con la respuesta original de Alex.

Los archivos de entrada se verifican contra hashes esperados antes de usarlos.
Fuente, lado, caso, ronda/instancia/época, intervalo, identidad, índices de muestra
y disponibilidad deben corresponder. Los campos desconocidos permanecen
desconocidos. Un recuento parcial de rivales nunca se completa contando bajas.

El snapshot distingue estado actual propio/aliado y eventos anteriores. Las
posiciones rivales pertenecen al evento fechado: no son posición actual, visión
universal ni comunicación. La edad se calcula contra el corte causal. La
ausencia de portador no se transforma por sí sola en bomba caída o plantada.
La planta acreditada pertenece a la fase posplantado; la elegibilidad desconocida
no prueba cuchillos. Estas distinciones delimitan el acceso factual a una futura
regla, no evalúan la conveniencia de una jugada.

Las salidas se escriben exclusivamente en un directorio nuevo y vacío dentro
de TEMP, con rechazo de enlaces, destinos ocupados y sobrescrituras. Se retienen
artefactos, manifiesto, descripción factual y sus hashes. El validador vuelve a
leer entradas verificadas y reconstruye el resultado, además de validar esquema.

La configuración congelada `collective_c4_approach@2-temporal-provisional` y
sus predicciones permanecen intactas. La nueva inferencia de preparación exige
concretar señales positivas de organización con el experto. El contexto
candidato, por sí solo, no devuelve un destino táctico nuevo ni habilita consejo,
entrenamiento o producto.
