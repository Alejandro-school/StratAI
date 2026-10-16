# Contexto NAV propio/aliado — candidato v1

Contrato `1.0.0-candidate.1`, fase 3E.3. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y continuación. Conserva los [anclajes v1](canonical-support-anchors-v1.md), el [contexto físico v1](canonical-support-physical-v1.md), catálogo `1.7.0`, parser `v16`, export `3.8.0`, quality `12`, `DecisionExample` y `DEC-TRADE-001`.

## Unidad y corte causal

Una fila por **actor → aliado / instancia / anclaje**, con igualdad exacta de fuente, checksum, ID, `source_sequence`, tick, FrameDone, época, lado y vidas de ambos pawns. El selector original exige tres muestras contiguas con identidad, posición y vida compatibles. Se reutilizan exclusivamente las dos posiciones de su última muestra. No se mueve T0 ni se usa posición futura del aliado, enemigo oculto, muerte o desenlace para seleccionar o calcular.

Las posiciones X/Y conservan la cuantización del replay; Z conserva su precisión exportada. El aliado es un proxy de información de equipo en la demo, no prueba de percepción o cooperación. La acreditación competitiva se copia del control 3E.2 y se revalida con intervalos canónicos; las instancias preliminares no se promueven. Ambos sentidos del par siguen correlacionados.

## Productor y artefactos optativos

La demo se procesa con el selector existente `--ai-coach-support-physical-candidate`. El nuevo materializador `ai_coach.support.navigation` ejecuta el materializador 3E.2 y conserva su serialización de `support_anchors.json` y `support_physical_context.json`. Después envía sólo los extremos seleccionados al ejecutable `cmd/support-navigation`, que reutiliza `MapManager.NavigationGraphSnapshot` y `navigation.PreparedEvaluator.EvaluatePath`. No aumenta ninguna captura ni carga o genera un mesh físico.

Se añaden dos schemas estrictos:

- [support_navigation_observations@1](../../../ai_coach/contracts/support_navigation_observations.schema.json): fuente, referencias por bytes/SHA-256, cabecera y hash del NAV, grafo candidato, claves causales, extremos y resultados del productor Go.
- [support_navigation_context@1](../../../ai_coach/contracts/support_navigation_context.schema.json): una fila por anclaje, diagnóstico candidato separado de hechos iniciales, abstenciones, matriz actor/aliado/instancia y resumen. El registro runtime tiene 66 schemas y 19 reason codes; las razones locales de abstención no cambian los IDs estables del catálogo.

El materializador comprueba TEMP nuevo, vacío y sin enlaces/reparse points antes de leer fuentes o ejecutar procesos. Las escrituras exclusivas rechazan sobrescrituras. La CLI Go usa stdin/stdout, no acepta destinos ni interpreta identidades, relojes o outcomes. Rechaza campos desconocidos, endpoints incompletos, mapas con traversal, hashes inválidos y datos concatenados. Un cero debe estar presente: componentes ausentes no se convierten en coordenadas cero.

## Procedencia y cobertura del asset

Se admite como diagnóstico la combinación local auditada **Source 2 NAV v35 / subversión 1**, con magic `0xFEEDFACE`. El mapa procede del manifiesto canónico verificado; referencia y hash deben coincidir exactamente con el input `nav_mesh` del parser. Se verifica el archivo antes y después de cargarlo. La relectura reconstruye el grafo y resultados desde el archivo y ejecutable fijados por hash; copiar un hash sobre un grafo alterado no acredita procedencia.

Estas comprobaciones acreditan qué archivo local produjo el diagnóstico. **No acreditan que ese archivo perteneciera al mapa exacto usado por la demo.** `demo_binding=not_accredited` es constante. Un nombre `de_mirage`, instalación local, fecha o hash calculado no sustituyen una cadena de distribución/extracción y vínculo con el recurso de la partida.

El lector conserva polígonos y enlaces dirigidos de área. En Source 2 descarta el ID de borde destino, atributos/hull, referencias de escalera y transformaciones móviles; tampoco interpreta todo el sufijo del archivo. El snapshot usa sólo los enlaces retenidos y no representa condiciones de paso, colisiones dinámicas o todos los tipos de conexión. La auditoría independiente del asset real, con conteos y hash del sufijo, se retiene en la evidencia de fase. No se modifican los lectores históricos para fingir cobertura adicional.

La dependencia queda concreta: material original del mapa con una cadena verificable hasta esta demo —identidad del recurso/distribución, manifiesto de contenido, hash del archivo original y extracción reproducible del NAV—; interpretación acreditada de los tipos de enlace y atributos que use cada ruta; y asociación sobre superficies/portales/hull del mismo sistema de coordenadas, con cuantización y alturas justificadas. Para afirmar ausencia física se requiere cobertura completa de las conexiones relevantes, incluidas escaleras y datos hoy no interpretados. Ni un hash local coincidente entre dos ejecuciones, ni la cercanía al centro, ni el mismo nombre de mapa satisfacen estas condiciones. La falta se reabre con ese material positivo; no con otro inventario o smoke de la misma ausencia.

## Significado del cálculo

| Elemento | Contrato y límite |
|---|---|
| Grafo | `nav_area_connections_with_polygons@2`; dirigido, copia defensiva, nodos y aristas ordenados. |
| Asociación | `containing_nav_polygon_xy_then_nearest_center_3d@2`: contención XY y diferencia Z respecto al centro hasta 128 unidades; si no hay contención admisible, centro 3D más cercano hasta 96 unidades. Son supuestos heredados del diagnóstico, sin acreditación de superficie/hull. |
| Desempates | Menor diferencia vertical, luego menor ID; fallback por menor distancia y orden de ID. |
| Distancia | `directed_dijkstra_stable_ties@1`, pesos positivos euclídeos entre centros, más distancias de origen y destino a sus centros asociados. Unidades del mundo. |
| Ruta | Polilínea de origen, centros del camino dirigido y destino, sin puntos consecutivos duplicados. No es un funnel por portales, camino físico mínimo ni trayectoria comprobada contra colisiones. |
| Mismo polígono | Conserva el coste de ambos extremos al centro, incluso si la línea directa sería más corta. No se sustituye silenciosamente el algoritmo. |
| Sin camino | Ausencia de conexión **en el subgrafo retenido**; no prueba inexistencia de ruta física porque el grafo es parcial. |
| Fuera de cobertura | `Reachable=null`, distancia NAV `null`, sin puntos; distinto de `Reachable=false` en el grafo candidato. |

No se introducen velocidades nominales, multiplicadores, ETA, umbrales de buen apoyo, nombres tácticos de rutas o recomendaciones. Una ruta candidata conectada no acredita seguridad, LOS, contacto visual, llegada a tiempo, apoyo, intención o superioridad táctica.

## Estados, ausencia y acceso inicial

Se distinguen conceptualmente **conexión física observada**, **ausencia física acreditada** e **información insuficiente**. Para observar conexión harían falta asset exacto, asociación física acreditada y enlaces/condiciones completos para la ruta. Para acreditar ausencia haría falta además cobertura exhaustiva del espacio de conexión relevante. Ninguna de esas pruebas existe en este candidato: el schema v1 sólo permite `initial.status=insufficient_information`, con reachability, distancia y ruta físicos `null`. No tiene un interruptor para declarar acreditación manualmente.

Cada fila conserva seis motivos: falta del vínculo exacto demo–asset; asociación física no acreditada; atributos/hull/transformaciones no modelados; escaleras no modeladas; bordes destino no retenidos; polilínea por centros sin validación de colisiones. El diagnóstico separado distingue `candidate_connected`, `candidate_no_directed_path`, `candidate_outside_coverage` y `unavailable`. Una falta de archivo o formato compatible conserva todas las filas con la razón concreta. Un desacuerdo de mapa/hash/fuente se rechaza, no se normaliza como observación.

`initial_features(support, anchor_id, actor, tick, frame)` valida el prefijo hasta T0 y exige el anclaje exacto. Devuelve únicamente disponibilidad y abstención física; excluye IDs, assets candidatos, polilíneas provisionales, enemigos, resultados y clasificación competitiva. Los diagnósticos siguen accesibles como investigación bajo el lector validado del artefacto. Una futura habilitación factual necesita otro contrato y evidencia positiva; ningún consumidor inicial puede asumirla.

La matriz conserva denominador de anclajes por actor/aliado/instancia, estados candidatos y abstenciones físicas. No cuenta direcciones correlacionadas como decisiones independientes. `decision_eligible`, `comparison_allowed`, `training_allowed` y `product_supported` permanecen en `false`.

## Ejecución y verificación

Desde `backend/go-service`, compilar ambos ejecutables en el temporal de trabajo:

```powershell
go build -o $supportBinary ./cmd/canonical-export
go build -o $navigationBinary ./cmd/support-navigation
& $supportBinary --demo $approvedDemo --output $supportRunRoot --alias demo-a11eb0d600ee702c9edb --checksum a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262 --maps-root ../data/maps --build-id $buildID --ai-coach-support-physical-candidate
```

Desde la raíz, con `$navigationOutput` como TEMP nuevo y vacío:

```powershell
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.navigation --source $physicalSource --output $navigationOutput --evaluator $navigationBinary --maps-root backend/data/maps
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.navigation --source $physicalSource --output $navigationOutput --evaluator $navigationBinary --maps-root backend/data/maps --validate
```

El lector valida las fuentes y sus hashes, reconstruye los anclajes y el contexto físico v1, reconstruye Go desde el NAV y contrasta todos los resultados con un verificador independiente Python: centros, pesos, asociaciones, caminos mínimos, dirección de cada segmento y longitud de polilíneas. El smoke también decodifica de forma independiente el prefijo nativo del NAV y compara sus polígonos/enlaces con el snapshot, además del canónico completo y controles 3E.1–3E.2.

Sólo se procesa la demo aprobada fijada por alias/checksum, tras pasar las suites automáticas. Las variantes ausentes del caso real se distinguen como fixtures. Los temporales se conservan inventariados con hashes; no se borran ni se pide revisión de limpieza. Este contrato no autoriza 44/44, descargas, promoción, publicación ni entrenamiento.
