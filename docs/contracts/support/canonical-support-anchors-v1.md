# Anclajes factuales de apoyo — candidato v1

Contrato `1.0.0-candidate.1`, fase 3E.1. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción. Este contrato permite estudiar cambios observados de separación propia/aliada para S2–S3 del [piloto](../../ai-coach/AI_COACH_INITIAL_PILOT_DESIGN.md). No acredita preparación anterior a una entrada física, apoyo seguro, error ni alternativa superior.

## Unidad y selección

La unidad es **actor → aliado / instancia observada de ronda**. El [protocolo congelado](../../../ai_coach/contracts/support_anchor_protocol.json) selecciona el primer aumento de distancia al cuadrado después de un paso no creciente, sobre tres muestras contiguas:

`d²[anterior] <= d²[anteanterior] AND d²[actual] > d²[anterior]`.

T0 es el tick y ordinal de `FrameDone` de la muestra actual: **el aumento ya ha ocurrido**. No se retrocede T0 ni se escoge con una muerte, daño, resultado o enemigo oculto. No hay umbral de distancia considerado tácticamente correcto. X/Y son coordenadas redondeadas a unidades del replay; Z conserva su precisión exportada. Un cambio de signo puede ser pequeño o deberse a cuantización. Ambos sentidos del par pueden producir registros correlacionados; no son oportunidades estadísticamente independientes ni compromisos tácticos acreditados.

Se exige la misma instancia, época, lado, vida y handle completo de ambos pawns; conexión, posición y estado vivo observados en las tres muestras. La fase debe ser `playing`, con tick rate válido e idéntico y diferencias exactas de `int(tick_rate/16)` ticks. No hay interpolación, arrastre de otra muestra ni reparación con futuro. Un aumento continuo no vuelve a disparar el anclaje; puede empezar otro episodio después de un paso no creciente.

## Captura y procedencia

El selector Go optativo `--ai-coach-support-candidate` produce el bundle base `3.8.0` y `support_observations.json`. Es exclusivo de los otros selectores candidatos. Comprueba destino temporal, vacío y sin sobrescritura antes de parsear. No cambia `DEC-TRADE-001`, `DecisionExample`, los lectores existentes ni el catálogo `1.7.0`.

La captura ocurre inmediatamente después de añadir la muestra del replay, dentro del mismo callback, sin avanzar el parser. Relee participantes y pawn actual. El reloj cuenta **todos** los `FrameDone`, incluidos los no muestreados y los que repiten tick; un retroceso de tick abre una época. `instance` cuenta los inicios observados del handler, no acredita por sí sola una ronda competitiva oficial. Las muestras `postround` se retienen como fronteras, pero no seleccionan anclajes.

El handle conserva sus 32 bits y se contrasta con índice y serie completa de la entidad. La vida local sigue la instancia del objeto SDK y su destrucción; nunca se empareja sólo por índice. Los alias de actor son `support-actor-` más 20 hexadecimales de SHA-256 de checksum de demo, separador y referencia nativa; sólo son identificadores locales de procedencia. La referencia nativa no se exporta en este sidecar.

Se comprueba presencia de los seis componentes nativos de coordenadas y de salud/estado de vida, además de valores finitos. El cero numérico observado se conserva. Si el SDK devuelve su vector cero por falta de campos, el sidecar guarda posición `null` y `native_fields_absent`. Pawn no resuelto, desconexión, ausencia y duplicados provocan exclusiones explícitas. Es una observación del estado SDK al cerrar un comando: no acredita ejecución subtick, percepción, atención ni comunicaciones del jugador.

Los schemas propios son [observaciones](../../../ai_coach/contracts/support_observations.schema.json), [anclajes](../../../ai_coach/contracts/support_anchors.schema.json) y [protocolo](../../../ai_coach/contracts/support_anchor_protocol.schema.json). Todos son estrictos. La fuente conserva alias/checksum, build, contrato y referencia con bytes/SHA-256 al manifiesto base. El informe enlaza fuente y protocolo por bytes/hash. El lector verifica todos los hashes del bundle canónico y contrasta posición y estado vivo presentes con el replay en claves ronda/tick únicas; informa aparte de muestras sin unión única. Esa unión de control no interviene en la selección.

## Lectura inicial y ventana posterior

`initial_features(data, anchor_id, actor, tick, frame)` valida y selecciona únicamente el prefijo hasta ese `frame`, exige coincidencia exacta de actor/T0 y devuelve copias de tres posiciones/estados propios y del aliado elegido, distancias y disponibilidad. No devuelve identidades, enemigos, resultados o una posición interpolada. El acceso a coordenadas aliadas es un proxy de información del equipo disponible en la demo, no una prueba de que el jugador la consultara o conociera con esa precisión.

La ventana posterior, separada de `observed`, termina en `T0 + ceil(2 * tick_rate)` ticks. Exige muestra final exacta y cadencia completa dentro de la misma instancia/época/fase/vidas/lado. Si falta, sus valores quedan `null` con `missing_exact_horizon_sample`, `horizon_gap_or_boundary` o `horizon_pair_unavailable_or_changed`. Si está completa, sólo informa estados vivos al final y si se observó alguna muestra no viva. No atribuye una kill ni una causa. El cruce independiente con eventos canónicos de muerte pertenece a la evidencia posterior, nunca a las features o selección.

## Matriz, ausencias y gates

Cada actor/instancia con muestras `playing` tiene conteos de muestras y pares examinados. Se conservan ceros y ocho razones: `actor_unavailable`, `no_live_ally`, `insufficient_history`, `incompatible_sample_window`, `pair_history_unavailable`, `pair_life_or_side_changed`, `no_increase_onset`, `observed_onset`. Las dos primeras cuentan muestras de actor; las seis restantes clasifican pares examinados. No se suman todas como si compartieran denominador.

Cada anclaje declara ausentes: entrada física, compromiso táctico, ruta NAV, línea de visión, contacto legal acreditado, contexto del objetivo, cooperación, intención, orientación y preparación del arma. Su `decision_id` es `null`, `physical_entry_status=not_accredited` y `decision_eligible=false`. `comparison_allowed`, `training_allowed` y `product_supported` permanecen en `false`.

## Ejecución y validación

Desde `backend/go-service`, con una build y variables que identifiquen la fuente aprobada y un destino TEMP nuevo/vacío:

```powershell
& $supportBinary --demo $approvedDemo --output $supportRunRoot --alias demo-a11eb0d600ee702c9edb --checksum a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262 --maps-root ../data/maps --build-id $buildID --ai-coach-support-candidate
```

Desde la raíz, con otro directorio TEMP vacío para el informe:

```powershell
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.anchors --source $supportSource --output $supportReportRoot --require-observed
```

El cierre exige pruebas completas, schema/semántica/procedencia, recálculo independiente, casos reales sin muerte y validación canónica completa de la misma demo. Las variantes sintéticas se distinguen de las observadas. El informe de cierre y la evidencia enlazados desde el plan registran los conteos. TEMP se conserva con inventario por instrucción del usuario; no se intenta borrar ni se solicita revisión de limpieza. Una demo de desarrollo no sustituye el futuro gate 44/44 ni una evaluación táctica humana.
