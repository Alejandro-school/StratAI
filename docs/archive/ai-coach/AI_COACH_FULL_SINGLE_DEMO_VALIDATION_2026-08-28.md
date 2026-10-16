# Validación integral del AI Coach con una sola demo

Fecha: 28 de agosto de 2026.

> Nota de alcance: este cierre integró todas las piezas, pero su contacto visual real estaba limitado a un prefijo de 256 muestras. La validación posterior de las 24 rondas completas está en [AI_COACH_FULL_MATCH_VALIDATION_2026-08-28.md](AI_COACH_FULL_MATCH_VALIDATION_2026-08-28.md). En este documento, `1.796` significa intersecciones línea–humo por pareja y muestra dentro de aquella ventana, no 1.796 bloqueos humanos ni una partida completa.

## Resultado

La validación integral ha pasado sobre una única partida aprobada de Mirage. Se usó el alias local `demo-a11eb0d600ee702c9edb`; la demo, los 44 bundles aprobados y el resto del corpus se mantuvieron en solo lectura.

La misma lectura del replay alimentó el export canónico, navegación/ETA y contacto visual. Después, sin volver a parsear la demo, se validaron el bundle completo, los `DecisionExample` y los 17 resultados de `TaskEligibility`.

Gate 1B sigue pendiente y `training_allowed=false`. Este smoke demuestra que lo implementado hasta ahora funciona junto sobre una partida real; no materializa los candidatos dentro de los 44 bundles ni autoriza entrenamiento o promoción.

## Qué se comprobó

- Bundle canónico `3.8.0`, parser `v16`, 72 artefactos y checksum de la demo correcto.
- 64 rutas NAV reales y alcanzables, con 2.544 nodos y 6.381 aristas.
- ETA interna contextual: arma activa, ritmo observado, caminar, movimiento normal, cuchillo, agachado y mira.
- Siete velocidades base distintas entre 200 y 250 unidades por segundo; no existe una única velocidad fija.
- Cero usos de segundos exactos como mensaje para el jugador y ninguna ETA tratada como predicción segura de llegada.
- 12.800 evaluaciones de contacto visual con raycast, FOV y humo.
- 1.796 intersecciones línea–humo por pareja y muestra en la ventana acotada, 10 episodios direccionales iniciados y cerrados en esa ventana, y cero identidades Steam.
- 918 `DecisionExample`: 672 de spacing/trade y 246 de peek/hold/reposición.
- 918 joins correctos, cero fallidos, cero violaciones temporales y cero oracle dentro de observed.
- 17 capacidades evaluadas: 0 elegibles, 3 inelegibles y 14 aún no materializadas.
- Dos ejecuciones del auditor directo produjeron el mismo SHA-256, sin persistir ejemplos individuales.

## Defecto descubierto por la demo

El primer intento produjo correctamente los datos, pero el control posterior de `DecisionExample` falló cerrado. El auditor existente sólo aceptaba bundles ya publicados con un `manifest.json` raíz; una salida temporal directa, correctamente no publicada, sólo contiene `canonical/manifest.json`.

Se corrigió con una API separada para exactamente un bundle directo. Ese modo:

- no relaja ni cambia el descubrimiento normal de los 44 bundles;
- exige alias y SHA aprobados, el nombre `match_<alias>` y las versiones exactas;
- recalcula bytes y SHA-256 de los 72 artefactos;
- rechaza rutas inseguras, enlaces, reparse points, ficheros extra y descriptores desordenados;
- exige ambas familias con registros, un join correcto por ejemplo y cero fallos;
- declara con claridad 1 demo fuente aprobada, 1 bundle temporal validado y 0 bundles de salida aprobados.

Después de las pruebas se repitió la misma demo, no se buscó otra. El segundo intento pasó todos los controles. Los dos intentos generaron las mismas filas NAV, el mismo ledger visual, los mismos resúmenes y los mismos descriptores de los 72 artefactos canónicos.

## Otras defensas cerradas antes del ensayo

- El contacto visual usa un prefijo causal desde el inicio real de la ronda; no inventa un inicio de contacto por recortar a mitad de acción.
- Si la primera ronda con humo no cubre todos los casos, se prueban rondas posteriores de forma determinista.
- `smoke_occluded=true` requiere al menos un humo activo.
- El validador visual exige `maps_root` y vuelve a leer y hashear el GLTF y su buffer.
- El output smoke debe existir, estar vacío y ser un descendiente físico de `%TEMP%`; se rechazan symlinks, junctions y reparse points antes del parseo.

## Resultados principales

| Área | Resultado |
|---|---:|
| Duración registrada del productor final | 68.884 ms |
| Validación semántica canónica | 0 errores; 254.116 ms |
| Rutas NAV | 64/64 alcanzables |
| Proyecciones con ritmo observado | 51 |
| Contextos walking / moving non-walking | 13 / 39 |
| Contextos Knife / ducking / scoped | 19 / 4 / 2 |
| Evaluaciones visuales | 12.800 |
| Intersecciones línea–humo por pareja/muestra en 256 muestras | 1.796 |
| Episodios direccionales cerrados / abiertos en esa ventana | 10 / 0 |
| `DecisionExample` | 918 |
| Joins correctos / fallidos | 918 / 0 |
| Violaciones temporales / oracle en observed | 0 / 0 |
| `TaskEligibility` | 0 eligible / 3 ineligible / 14 not_materialized |

La evidencia completa, incluidos comandos, hashes, conteos, intentos e invariantes, está en [AI_COACH_FULL_SINGLE_DEMO_VALIDATION_2026-08-28.json](../../evidence/AI_COACH_FULL_SINGLE_DEMO_VALIDATION_2026-08-28.json).

## Seguridad del corpus

Antes y después se mantuvieron exactamente:

- demos: 100 archivos y 13.589.310.951 bytes;
- bundles aprobados: 3.013 archivos y 27.264.064.316 bytes;
- `failed`: 727 archivos y 5.245.908.800 bytes;
- `staging`: 0 archivos y 0 bytes;
- mismo SHA-256 de la demo y del manifest aprobado;
- estado Git limpio en `Faceit-Demos`.

No se descargaron demos, no se entrenaron modelos, no se reprocesó el corpus, no se creó una release y no se publicó ni promovió ningún bundle.

Las dos carpetas de los intentos se comprobaron como hijas directas de `%TEMP%`, sin enlaces ni junctions, y se eliminaron después de guardar esta evidencia. La eliminación no es recuperable.

## Siguiente fase

La siguiente acción canónica no cambia: materializar ammo de cargador, reserva y estado de recarga a 16 Hz con sentinels y ausencias explícitos. Esa fase deberá volver a pasar el gate de una única demo aprobada.
