# Registro descriptivo temporal ICT-I1.A

Contrato candidato `1.0.0-candidate.1`; registro `stratai.temporal_description@1`.
El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente acción.
Implementa la frontera descriptiva del [alcance acordado](../../ai-coach/AI_COACH_TEMPORAL_INTERPRETER_SCOPE_v0.2.md).

## Entrada y perspectiva

`ai_coach.interpretation.temporal.iter_records` recibe observaciones support, lado
analítico y secuencia final exacta. Antes de validar, recorta el prefijo y proyecta
una lista cerrada de campos propios/aliados. No consume pares C0 ni sus descriptores
posteriores, respuestas humanas, títulos, etiquetas de zona, resultados o posiciones
rivales. CT y T tienen cadenas independientes; cambiar de lado o de vida nunca
justifica arrastrar una trayectoria individual.

La perspectiva es analítica: disponer de datos de un aliado no demuestra que cada
jugador los conociera. No se supone una plantilla completa de cinco jugadores.
Las ausencias respecto al corte adyacente se registran como falta de muestra, sin
convertirlas en muerte o retirada.

El adaptador usa posiciones/vida de support, los siete hechos físicos acreditados
(orientación, caminar, agacharse, arma, cargador, reserva y recarga) y tenencia C4
corroborada por inventario/propietario. Los órdenes del contexto de objetivo sirven
sólo para validar la unión C4. No se proyectan sus eventos como conocimiento.
Los demás contextos C0 conservan sus contratos y no se incorporan automáticamente
a este registro: no se amplía su cobertura ni se sustituyen sus consumidores.

## Salida y continuidad

Se emite un registro por corte y lado con versión, configuración, identidad de la
demo, reloj exacto, identidades locales, hechos, exclusiones, referencias y enlace
hash al registro anterior. El hash se calcula sobre la proyección permitida; los
hashes de archivos completos pertenecen al manifiesto externo y no condicionan
una descripción anterior al corte.

El desplazamiento es la resta de coordenadas entre muestras adyacentes válidas
con vida, handle, lado, instancia, época y muestreo compatibles. Conserva la
precisión de la fuente. Un vector cero significa que esas muestras no resuelven
desplazamiento; no afirma inmovilidad durante todo el intervalo. No se atribuye
velocidad instantánea, tránsito viable, zona semántica ni intención.

Un hueco, cambio de identidad, actor excluido o límite de ronda produce una razón
de indisponibilidad. Un estado `alive=false` observado describe ese estado; no
acredita que la muerte fuera percibida ni que el actor buscara información.

Los cuatro criterios ICT-I1-E01–E04 figuran con requisitos pendientes y sin señales
positivas. El contrato fija `unavailable`; no existe en esta versión un canal
acreditado que active sus reglas. E02 conserva expresamente la excepción de todo
el equipo en B. Los contrastes sintéticos comprueban que etiquetas positivas,
ausencia de prueba y precondiciones incumplidas no habilitan hipótesis. No se
presentan esos tests como detección positiva o evaluación táctica.

`comparison_allowed`, `training_allowed` y `product_supported` son falsos, y
`recommendations` está vacío. El catálogo y los gates S2–S3 no cambian.

## Artefactos y verificación

El materializador exige las cuatro fuentes originales support, physical, objective
y C4, comprueba sus schemas, semántica, hashes y vínculo con el bundle. Esta
comprobación de integridad del archivo completo se separa de la API causal: un
archivo corrupto se rechaza aunque el defecto esté después del corte solicitado.
La API causal sí puede recibir un futuro truncado o modificado sin alterar su
prefijo. La ausencia de recursos está representada explícitamente en dicha API.

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.ledger_io materialize --source <captura> --output <TEMP-vacio>
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.ledger_io validate --source <captura> --output <salida-retenida>
```

El escritor rechaza destinos fuera de TEMP, enlaces, destinos ocupados y
sobrescrituras antes de cargar fuentes. Produce `temporal_descriptions.jsonl` y
`temporal_description_manifest.json`. El lector recompone cada registro y todos
los conteos y compara el manifiesto completo; no basta con validar el hash.
El esquema del registro se incorpora al verificador público de contratos.

La optimización técnica de validación conserva el contrato y todos los controles.
Cinco schemas usan compilación Draft 2020-12 con `jsonschema-rs==0.57.1`;
la comprobación del schema y los errores siguen contrastados con `jsonschema`.
Sólo se cachea la compilación por contenido, nunca el resultado de validar una
instancia. Los valores Python que no son JSON simple usan el motor original.
El [protocolo de evaluación](../../ai-coach/AI_COACH_TECHNICAL_EVALUATION_PROTOCOL.md) especifica
equivalencia, pruebas diferenciales y evaluación técnica en varias partidas.

La validación independiente del smoke contrasta coordenadas, diferencias,
identidades, munición nativa y tenencia C4 directamente con las fuentes. K1/K2
se verifican desde un auditor separado; sus respuestas humanas no entran al
productor y siguen siendo desarrollo conocido. Las salidas temporales se retienen.

## Presentación offline en español

`ai_coach.interpretation.replay_report` presenta cortes explícitos de un único
lado, sin cambiar registros ni inferir tácticas. Reutiliza la validación de fuentes
y la proyección causal. Rechaza destinos ocupados o fuera de TEMP antes de leer
entradas. El directorio de salida debe existir y estar vacío.

```powershell
python -B -m ai_coach.interpretation.replay_report --source <support-directory> --output <empty-TEMP-directory> --side T --sequences 1389 1417
```

Produce `replay_report.md`, `replay_report_records.jsonl` y
`replay_report_manifest.json`. Este manifiesto de presentación
`replay_description_es@1` conserva hashes de fuentes y salidas fuera de los
registros causales; no constituye un contrato de API ni habilita producto.
Selección: 1–100 secuencias positivas, únicas y crecientes, todas presentes.
La diferencia mostrada sigue siendo la de la muestra inmediatamente anterior,
no la diferencia entre cortes seleccionados. El [cierre técnico](../../archive/ai-coach/AI_COACH_REPLAY_DESCRIPTION_REPORT_2026-09-22.md)
conserva la verificación sobre una replay real.
