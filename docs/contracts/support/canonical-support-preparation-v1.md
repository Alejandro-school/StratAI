# Censo prospectivo de preparación de apoyo C0 v1

Contrato optativo `1.0.0-candidate.1`, protocolo `support_preparation_census/1.0.0-candidate.1`. Implementa el [diseño congelado 3E.11](../../ai-coach/AI_COACH_SUPPORT_PRE_ENTRY_PROTOCOL.md) como población factual de investigación. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina su estado de validación. No acredita entrada física, compromiso táctico, intención ni una decisión elegible S2–S3.

## Fuente y alcance

Reutiliza `support_observations.json`, `stratai.support_observations@1`, captura `fresh_replay_16hz_FrameDone_with_full_pawn_life@1`. No añade captura Go ni modifica los contratos 3E.1–3E.10. La fuente conserva su comprobación de schema, posiciones finitas, identidad, orden de entrega y manifest canónico enlazado por alias/checksum/hash. El catálogo `1.7.0`, `DecisionExample` y el comienzo de `DEC-TRADE-001` permanecen intactos.

C0 registra cada corte capturado y selecciona todos sus pares dirigidos propios/aliados utilizables. No exige historia, proximidad, aumento de distancia ni un desenlace futuro. La selección se mantiene aunque la ejecución posterior termine incompleta. `playing` es fase del capturador; no acredita fin de freeze ni ronda competitiva. El censo no afirma que el roster esté completo ni que contenga todas las oportunidades físicas.

La admisión requiere fase `playing`, reloj finito positivo y paso entero `int(tick_rate/16)>=1`, además de referencias temporales enteras. Cada actor debe tener alias único, lado CT/T, conexión positiva, handle completo de 32 bits sin sentinel, vida local e identidad observadas, estado vivo observado positivo y posición XYZ observada finita. Alias ambiguos o referencias de pawn compartidas entre participantes del mismo lado se excluyen. La precedencia de motivos coincide con el protocolo congelado.

La validación estricta de la fuente ocurre antes de materializar. Los casos sintéticos con números no finitos, tipos inválidos o estados contradictorios prueban que el selector los excluye; no amplían los tipos aceptados por el schema de captura. Una fuente inválida se rechaza. Un reloj ausente representado legalmente en la fuente se cuenta como exclusión del corte.

## Artefactos y schemas

Se añaden cuatro schemas estrictos, con propiedades explícitas y autorización de entrenamiento en `false`:

| Archivo | Schema | Contenido |
|---|---|---|
| `support_preparation_protocol.json` | `stratai.support_preparation_protocol@1` | Versión, selección, unidad, límites temporales, motivos y gates congelados |
| `support_preparation_cuts.jsonl` | `stratai.support_preparation_cut@1`, por línea | Todos los cortes, admisión, aliases/lados y disponibilidad de actores, duplicados, denominadores y exclusiones |
| `support_preparation_pairs.jsonl` | `stratai.support_preparation_pair@1`, por línea | Pares seleccionados, ID/T0, referencias nativas, hechos iniciales y descriptores posteriores separados |
| `support_preparation_manifest.json` | `stratai.support_preparation_manifest@1` | Fuente/protocolo por hash, tamaños/conteos, matriz, reconciliación y límites |

El registro pasa de 82 a **86 schemas**, conservando **19 razones generales**. Los motivos de C0 son locales al contrato, no nuevas razones de elegibilidad ni IDs del catálogo. Los JSONL permiten escribir y recomputar por corte sin reunir los cientos de miles de pares en memoria. El manifest se escribe después de los artefactos y sólo en un directorio temporal nuevo y vacío.

Los pares excluidos quedan reconstruibles desde los actores del corte y la precedencia actor/aliado; sus conteos se conservan por corte y en la matriz dirigida. No se fabrican hechos iniciales para ellos. Los motivos de actor se registran también en cortes excluidos, pero los denominadores agregados de actor/par sólo suman cortes admisibles.

## Identidad y frontera causal

El ID se deriva de `partida / versión de protocolo / época / instancia / sequence / actor / aliado`. Frame y tick son referencias exactas. Ni el hash del archivo completo, ni el desenlace, ni el total final de candidatos intervienen. Dos FrameDone distintos con el mismo tick mantienen IDs distintos. El orden del roster no cambia la selección.

`preparation.initial_features(data, candidate_id, actor, ally, sequence, tick, frame)` corta por ordinal de entrega **antes de validar**, exige igualdad exacta de todos los argumentos y recalcula la pertenencia del par en el corte solicitado. Un futuro malformado fuera de ese prefijo no se consulta. La integridad del archivo completo se valida por separado en el lector de artefactos.

El retorno contiene exclusivamente dos copias, `self` y `ally`, de posición, estado vivo y tick/frame de disponibilidad. Identidades, handles, vidas, fuente/build, geometría enemiga, ronda final, estadísticas agregadas y ejecución posterior quedan fuera. La posición aliada mantiene su alcance de proxy de información del equipo; no prueba atención o comunicación.

Este contrato base conserva sus features iniciales mínimas. La [extensión física C0 3E.13](canonical-support-preparation-physical-v1.md) incorpora por separado los hechos físicos básicos de 3E.2 en el T0 exacto, sin modificar estos artefactos. Las demás capas requieren su propia unión exacta; no se reutiliza el contexto de un anclaje anterior cercano.

## Ejecución y censura

La ventana es `(T0, T0+ceil(2*tick_rate)]`, con cadencia y horizonte exactos. La duración es una convención de investigación, no un tiempo humano de reacción o plazo táctico óptimo. Se comprueba primero si existe el número de muestras necesario para ese horizonte; después, el primer fallo de continuidad con precedencia de frontera/reloj, identidad/lado y disponibilidad del actor.

Cada rol tiene su descriptor: `sampled_displacement`, `no_sampled_displacement` o un motivo explícito `censored:*`. Las ventanas completas conservan un booleano de cambio; las censuradas conservan `null`. La secuencia/frame/tick que acredita el resultado o primer fallo pertenece a ejecución. Un horizonte sin entrega exacta mantiene esas referencias en `null`.

Cualquier diferencia exacta de coordenadas exportadas consecutivas cuenta como cambio; no se introduce un umbral táctico para ocultar jitter. La precisión cuantizada X/Y y la precisión exportada Z limitan su interpretación. Sin cambio muestreado no significa inmovilidad continua, espera deliberada, ausencia de entrada entre muestras ni mantener correctamente. Una muestra no viva tampoco identifica por sí sola un evento de muerte.

T0 precede a esta ventana futura, pero puede caer dentro de un movimiento ya iniciado. En todos los pares se conserva `decision_id=null`, `decision_eligible=false`, `physical_entry_status=not_accredited` y `tactical_commitment_status=not_accredited`. Comparación, entrenamiento y producto permanecen deshabilitados.

## Cobertura y lectura

El manifest concilia por separado:

- cortes totales, admisibles y excluidos por fase/reloj;
- filas de roster y aliases actor/corte, utilizables y no utilizables por causa;
- pares del mismo lado examinados, excluidos por actor/aliado y seleccionados;
- actor/cortes con al menos un aliado utilizable;
- ventanas posteriores por rol y combinación de roles;
- matriz actor/aliado/lado/instancia/época, con motivos y ceros explícitos.

Pares inversos, aliados múltiples y ventanas superpuestas están correlacionados. Estos conteos no son un tamaño efectivo de muestra ni una cantidad de decisiones tácticas independientes.

`validate_artifact` verifica rutas seguras, schema, protocolo exacto, fuente, hashes, bytes y número de registros; recorre y recalcula cada corte/par, sus hechos y ejecución, y la cobertura. Rechaza filas ausentes, adicionales, reordenadas o alteradas. Esta recomputación del lector comparte la derivación de aplicación; la evidencia del smoke añade una vía independiente para selección, posiciones, referencias, descriptores y matriz.

## Ejecución aislada

Tras superar las pruebas, el cierre funcional usa la misma única demo aprobada y una salida nueva bajo TEMP. El productor Go existente `--ai-coach-support-candidate` genera la fuente suficiente para C0 y el bundle canónico. No se necesita otro flag ni volver a capturar todas las capas nativas cuando sus productores permanecen intactos.

Desde la raíz del repositorio, con la fuente recién producida y otro subdirectorio TEMP vacío:

```powershell
backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation `
  --source <support_observations.json> `
  --output <directorio-nuevo-vacio-bajo-TEMP> `
  --require-observed

backend\venv\Scripts\python.exe -B -m ai_coach.support.preparation `
  --source <support_observations.json> `
  --output <mismo-directorio-materializado> `
  --validate --require-observed
```

El preflight rechaza salida fuera de TEMP, enlaces/reparse points, directorios no vacíos y sobrescrituras antes de cargar la fuente. Fuente, corpus y temporales anteriores son de sólo lectura. Se retienen las nuevas salidas y sus inventarios, sin promoción, limpieza o migración 44/44.
