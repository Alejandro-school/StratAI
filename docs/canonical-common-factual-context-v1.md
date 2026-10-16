# Entrada factual común v1

Contrato `stratai.common_factual_context@1`, versión 1.0.0. Adaptador CS2 sobre el [contexto candidato](contracts/context/canonical-candidate-context-v1.md). El [plan](AI_COACH_IMPLEMENTATION_PLAN.md) determina el estado y orden de trabajo. No cambia el detector congelado ni acredita un intérprete general.

## Contenido y lectura

`common_context.build(candidate, through_index=..., native=...)` produce una entrada limitada a una muestra exacta del intervalo conservado. Omitir `through_index` selecciona la última. Rechaza índices negativos, flotantes, booleanos y fuera del intervalo. La entrada tiene cinco bloques factuales, evolución entre muestras, acontecimientos fechados, incertidumbres locales e historial disponible.

| Bloque | Contenido |
|---|---|
| Marco | Mapa/build, perspectiva, corte, continuidad, elegibilidad, relojes y alcance original de recursos |
| Objetivo | Tenencia propia/aliada, estado del C4, fase reconciliada y reloj de bomba disponibles |
| Jugadores | Identidad, conexión, vida, posición, desplazamiento muestreado, exclusiones y recuentos propios/rivales |
| Recursos | Inventario actual e histórico separado, arma activa, munición, recarga y propiedades físicas acreditadas |
| Información | Percepción no acreditada y acontecimientos históricos con primera disponibilidad y edad al corte |

Cada hecho conserva valor, estado, motivo, unidad cuando procede, perspectiva, exposición al instrumento original, corte de disponibilidad e índice, referencias JSON y método de derivación cuando lo hay. Las referencias apuntan al candidato conservado, que mantiene las referencias/hashes de sus fuentes originales. El detalle crudo contiene el momento de observación: corte del sample, tick del evento o entrega nativa. Estos momentos no se confunden con disponibilidad ni con exposición original.

`sample(result, candidate, index)` consulta cualquier muestra hasta el corte elegido. `evidence(result, candidate, reference)` sólo resuelve referencias citadas en esa entrada o su evolución. Ambas funciones recomputan la entrada antes de servir detalle: modificar el límite o el resumen no abre acceso al futuro. El candidato completo se retiene como evidencia y no debe enviarse directamente al intérprete. Los hashes de linaje describen el archivo completo y no son señales predictivas.

La evolución conserva todas las transiciones, con referencias anteriores/posteriores para cada cambio. Ignora cambios exclusivos de `source_sequence` al comparar el significado del dato. Una discontinuidad temporal, cambio de identidad o cambio de roster permite contrastar extremos (`boundary_only`), sin afirmar una trayectoria continua. No crea clusters, roles, apoyo eficaz ni intención. Los eventos entran sólo desde su `available_index`; su edad se recalcula en el corte solicitado y su posición sigue siendo histórica.

## Desconocidos, conflictos y suplementos

No disponible, unknown, no aplicable, parcial, no suministrado y conflicto mantienen los estados originales. Cero y false observados son valores válidos. Inventario incompleto sigue siendo parcial y conserva cantidades y `last_known`. Un recuento rival parcial no se convierte en un número exacto de vivos. Elegibilidad unknown no declara cuchillos ni impide describir hechos observados.

La reconciliación del objetivo reutiliza `candidate_context._objective@1`, en el instante exacto: planta acreditada permite describir postplant; contradicción de planta/portadores o suelo mantiene conflicto; falta de portador no demuestra caída. Las posiciones de portador y C4 físico permanecen distintas. CT no recibe coordenadas globales del C4 en el suelo: conserva la abstención `review_side_not_eligible` de su fuente.

Las fichas originales no contienen salud y armadura actuales. El adaptador añade hechos explícitos `not_supplied`, referidos al alcance del instrumento y al actor; no calcula esos valores desde daño histórico. Otros recursos sólo se exponen donde la fuente los aporta. No añade parsing ni una captura nueva de propiedades. La retirada de un recurso limita ese hecho y las afirmaciones que dependan de él, sin vaciar el resto de situación.

E03 conserva @5, con kills y sin extra/bomb de @9; los demás casos históricos usan @9. `later_supplements` admite la [capa de lugar nativo](contracts/context/canonical-native-place-context-v1.md) previamente validada contra identidad, fuente y corte final. Sus hechos llevan `original_instrument=false`; no alteran `situation` ni enriquecen retroactivamente un corte anterior. Los estados nativos desconocidos también aparecen en incertidumbre local. La validación aquí comprueba el vínculo semántico del suplemento y sus bytes; la acreditación de la extracción nativa corresponde al lector upstream y sus recibos originales.

`original_instrument` identifica pertenencia a la versión declarada del candidato. Acreditar qué vio una persona requiere además el recibo de su ficha original; el indicador por sí solo no prueba exposición humana. Las ventanas técnicas T/CT del smoke son reconstrucciones para verificar el contrato en la versión declarada, con cortes derivados. No representan nuevas lecturas humanas ni atribuyen información CT a las respuestas históricas. Las siete regresiones históricas utilizan los candidatos originales, cuyos recibos y bytes se conservan.

El historial de rondas anteriores queda `not_supplied`. La estructura sirve para T y CT en el intervalo acreditado; no acredita por sí sola patrones de ataque repetidos o eficacia/adaptación defensiva. Las interpretaciones y recomendaciones permanecen vacías. `training_allowed=false` y `product_supported=false` son obligatorios.

## Archivos y verificación

`common_context_io.materialize()` exige una carpeta TEMP dedicada, existente y vacía antes de leer las fuentes. Rechaza destinos inseguros y escrituras sobre archivos existentes. Conserva `common_context.json`, candidato original, suplemento nativo opcional, informe legible y manifiesto `stratai.common_factual_manifest@1`. Registra hashes de productor/contratos, bytes de fuentes, corte e inventario; no crea predicciones.

`common_context_io.validate()` vuelve a leer las fuentes originales mediante sus hashes, verifica que coincidan con los datos embebidos, recomputa la entrada y el informe y comprueba inventario y bytes exactos. Actualizar sólo un checksum después de manipular un resultado no supera la validación. La validación depende de conservar las fuentes originales; los outputs no son una release autónoma.

Ejemplo PowerShell, con hashes reales acreditados y salida TEMP nueva/vacía:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.common_context_io materialize --output $salidaCS2 --candidate-path $candidatoCS2 --candidate-sha256 $hashCS2
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.common_context_io validate --output $salidaCS2
```

El [schema](../ai_coach/contracts/common_factual_context.schema.json) se registra en la validación runtime. Las pruebas comprueban límites causales, false/cero, incertidumbre local, fuentes alteradas, resúmenes falsificados, @5/@9, CT y suplementos nativos separados. El contraste real recomputa posiciones cambiadas, estados de objetivo y edades/disponibilidad de eventos desde referencias conservadas mediante un comprobador que no llama a los métodos del adaptador. Estos controles son funcionales; la evaluación táctica independiente tiene su gate propio.
