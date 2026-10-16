# Hipótesis B con región revisada — contrato local v4

`stratai.replay_attack_hypothesis@4`, `RegionAttackHypothesisSession`, adaptador optativo `1.7.0-local.1`. Continúa [v3](replay-attack-hypothesis-v3.md) tras la aceptación experta del 16/10/2026: «Sí, ese tramo completo». El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción. No cambia el adaptador por defecto ni las versiones históricas.

## Región y procedencia

`maps/de_mirage_attack_region_v1.json` conserva el contorno revisado, el hash de la imagen aceptada y del borrador, la transformación del radar y el vínculo con el paquete original de zonas. El polígono XY es `(-1500,850), (-1024,850), (-1024,408), (-1208,408), (-1208,675), (-1500,675)`: habitación final desde el paso junto a 2 hasta la salida B, incluido el codo junto a 3. La conversión de los seis vértices del dibujo a coordenadas mundiales es reproducible; no deriva de una distancia, bisectriz o respuesta del modelo.

Es una **aproximación del radar revisada por el experto**. La aceptación del tramo no acredita precisión física de sus paredes ni compatibilidad exacta con la build de la demo. Cada resultado mantiene `status=estimated`, `geometry_status=expert_reviewed_radar_trace_approximation` y `build_compatibility=unverified`, junto con hash de la región y referencia original.

## Pertenencia calculada

La pertenencia intersecta el polígono XY con el interior único del volumen 3D original de Apartments. Se recalcula la zona desde las coordenadas; una etiqueta suministrada incompatible no vale como soporte. Otra planta, otro mapa/referencia, solapes, datos no finitos, precisión ausente o fronteras no se cuentan como interior válido. No se extruye el dibujo a cualquier altura ni se modifica el paquete original.

Las aristas se protegen con la precisión XY ya suministrada por la fuente y el epsilon numérico existente. Esa protección describe la incertidumbre de coordenadas, no la precisión física del trazo ni un radio táctico nuevo. `inside`, `outside`, `boundary` y `unknown` son relaciones calculadas respecto de esta referencia local, siempre con sus límites explícitos.

Se mantienen las guardas actuales de jugador propio T vivo, posición y zona actuales, ronda activa preplant sin pausa y objetivo compatible. `spatial_review@2` entrega copias vinculadas de los hechos actuales que superan esas guardas y su pertenencia derivada. Jugadores muertos, enemigos, coordenadas sonoras ocultas o posiciones históricas no se convierten en miembros actuales. Los hechos iniciales, archivo causal y consultas conservan contenido, fechas y límites.

## Grupos y lectura

Sólo identidades distintas con `relation=inside` forman combinaciones canónicas de dos o tres. Estar en otra parte de Apartments no basta; un tercero fuera no invalida una pareja interior. C4 dentro del grupo, contacto CT o intención confirmada no se añaden como condiciones. W01 mantiene el mismo corte: P03/P05 quedan dentro y P04 antes del paso. N01 sigue hacia la salida, nunca se reutiliza como control retrasado.

La gramática sólo permite seleccionar esas combinaciones para la hipótesis opcional de posible ataque B. El modelo puede abstenerse o elegir los principios condicionales existentes; **no se rellena automáticamente la interpretación**. La pertenencia deja de ser una estimación de cercanía a cargo de Qwen; la selección táctica sigue siendo del modelo y pendiente de evaluación independiente.

El presentador reconstruye el número y los nombres desde hechos protegidos. La hipótesis muestra pertenencia calculada en el tramo revisado, soporte espacial con hashes, alternativas de preparación/repliegue/otro destino y límites de build. No confirma intención de plantar, apoyo desde Medio, reacción defensiva o recomendación. El verificador reconstruye también el soporte espacial y rechaza modificaciones.

Qwen/digest y presupuestos originales, salida factual breve, contexto íntegro reversible @3 y recibos históricos permanecen. Pruebas geométricas, smoke y validación mecánica no sustituyen la comparación real ni una evaluación táctica independiente.
