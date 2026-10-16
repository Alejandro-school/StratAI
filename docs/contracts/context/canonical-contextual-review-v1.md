# Inventario y fichas de revisión contextual

Contrato candidato de revisión sobre replays. Amplía la presentación y el
contraste experto del [intérprete contextual](canonical-contextual-interpreter-v1.md).
El plan canónico conserva el estado y la siguiente acción.

## Fuentes y vigencia

`support_c4_observations.json` ya captura las listas nativas de armas en cada
corte `FrameDone` de soporte. La compresión reutiliza estados iguales después
de leerlos de nuevo. No se consume la lista `weapons` del replay de presentación,
que puede conservar una lectura anterior.

La posesión actual de una granada exige el mismo pawn, vida, handle completo,
slot resuelto, entidad vigente, definición conocida y owner corroborado.
Las referencias ausentes o ambiguas impiden afirmar ausencia. Un jugador muerto
queda como no aplicable. Una lista completa vacía acredita «sin granadas»;
una lista incompleta se presenta como parcial o sin dato.

El nuevo `support_inventory_observations.json`,
`stratai.support_inventory_observations@1`, conserva las propiedades
`m_pReserveAmmo.0000` y `m_iPrimaryReserveAmmoCount` de las entidades actuales
en el mismo corte. Guarda presencia, tipo y valor nativo, vida/handle, reloj y
frontera del estado C4. Se vincula por SHA-256 a la captura de posesión.
Se produce con el opt-in existente `--ai-coach-support-c4-candidate` y no cambia
los bundles aprobados ni el formato del sidecar C4 anterior.

Para una granada poseída y corroborada, se aplica la semántica documentada en
`Equipment.AmmoInMagazine/AmmoReserve` del SDK fijado `demoinfocs-golang/v5 v5.2.0`:
una unidad en mano más la reserva nativa. El valor debe ser `int32`, entero y
no negativo. Se consulta primero `m_pReserveAmmo.0000`; sólo su ausencia permite
consultar la segunda propiedad. Un valor inválido no se sustituye por cero ni
por otra propiedad. Sin cantidad verificable, se muestra presencia mínima de
una unidad y cantidad sin confirmar. Varias entidades del mismo tipo no se suman
sin resolver la ambigüedad.

Las listas conocidas anteriores se etiquetan separadamente y conservan corte e
identidad. Se interrumpen por muerte/exclusión, cambio de vida/lado, discontinuidad,
cambio de ronda/instancia/epoch o desaparición del actor. No completan el
inventario actual y nunca usan una muestra posterior.

## Ficha reproducible

`stratai.contextual_review_packet@1` incorpora el contexto causal completo,
inventario por actor y muestra, plantado nativo y portadores aliados acreditados.
La posición física de C4 continúa sin dato en esta ficha.

`ai_coach.interpretation.review_packet` materializa JSON y HTML autónomo con:

- Vivo, muerto o sin dato, arma activa, munición, granadas y tenencia de C4.
- Radar de Mirage rotulado y trayectorias muestreadas hasta el instante elegido.
- Control de tiempo y reproducción del intervalo; no interpola rutas.
- Ausencias explícitas y listas anteriores separadas del inventario vigente.

El radar local usa la proyección `(x + 3230) / 5`, `(1713 - y) / 5`.
Sus nombres son orientación visual. No asignan zonas automáticas, prueban
contención geométrica, viabilidad de ruta ni compatibilidad de build.
Otros mapas conservan sus hechos sin una proyección inventada.

La ficha excluye predicciones, juicios humanos y desenlace. El manifiesto registra
intervalos, fuentes, radar, plantilla y hashes de salida. `validate` vuelve a
calcular los hechos y el HTML, además de comprobar sus hashes. La materialización
exige un directorio nuevo vacío dentro de TEMP y rechaza sobrescrituras.

Interfaz de comandos, usando rutas explícitas a la captura y al TEMP vacío:

```text
python -B -m ai_coach.interpretation.review_packet materialize --source CAPTURA --output TEMP_VACIO --side T --intervals 6318:6382 --radar frontend/public/maps/de_mirage_radar_psd.webp
python -B -m ai_coach.interpretation.review_packet validate --source CAPTURA --output TEMP_SALIDA --radar frontend/public/maps/de_mirage_radar_psd.webp
```

## Revisión humana y contraste

`stratai.contextual_review@2` conserva literal, autor, episodio/contexto,
exposición previa y tres dimensiones separadas:

- Estrategia: sin revisar, indeterminada o interpretable con planes compatibles.
- Destinos: posibles, respaldados o rechazados.
- Funciones valoradas, sin completar las que el revisor no menciona.

`review.compare(manifest, predictions, reviews)` verifica el manifiesto congelado,
la configuración, el hash de cada predicción y el vínculo de la revisión al
contexto. Rechaza revisiones duplicadas del mismo autor/episodio y cuenta
episodios únicos aunque haya varios revisores. Agrupa resultados por partida.

«Estrategia indeterminada y B posible» conserva ambos matices. Una emisión de
plan no queda confirmada por ese juicio, incluso si propone B. Una abstención
se registra junto a la indeterminación, sin convertir el destino posible en
acierto del plan. Las funciones humanas se conservan para contraste posterior;
no se puntúan automáticamente. La concordancia global permanece nula y no se
acredita generalización independiente. El evaluador v1 y sus métricas históricas
se conservan; no se reinterpretan sus plantillas vacías.

Ninguna respuesta entra en el detector. `training_allowed=false`; no hay
entrenamiento, ajuste de umbrales ni aprendizaje automático al revisar más demos.
Consejos, perfil y práctica dependen de alternativas viables y evaluación
respaldada, según el plan canónico.
