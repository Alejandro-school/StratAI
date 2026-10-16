# Manifiesto nativo de recursos de sesión v1

Contrato optativo de desarrollo `stratai.economy_resource_manifest@1`, versión `1.0.0-candidate.1`, selector **6**. Extiende la observación de [contexto y configuración](canonical-economy-rule-context-v1.md), conserva los artefactos de los selectores 1–5 y añade `economy_resource_manifest.json`. El registro runtime contiene **51 schemas**. Export base `3.8.0`, parser `v16`, quality `12`, catálogo `1.7.0` e IDs conservados. Ningún consumidor de producto utiliza este artefacto.

## Formato y alcance acreditados

La fuente del formato es [Resources.addManifestData de Clarity](https://github.com/skadistats/clarity/blob/7fb3f1d07564a12efa99194d45cfbf5762ba5910/src/main/java/skadistats/clarity/processor/resources/Resources.java), que registra `CSVCMsg_ServerInfo` para `CSGO_S2`, junto con su [lector LZSS](https://github.com/skadistats/clarity/blob/7fb3f1d07564a12efa99194d45cfbf5762ba5910/src/main/java/skadistats/clarity/util/LZSS.java) y lectores de bits. Es una implementación pública identificable, no una especificación oficial de Valve. Revisión y SHA-256 por archivo quedan en la evidencia; se conserva el [aviso BSD](../../references/clarity-manifest-codec.txt).

El [lector Go](../../../backend/go-service/handlers/economy_manifest.go) lee bits de menor a mayor peso: un bit de compresión, 24 bits de tamaño del payload codificado y ese payload. Si está comprimido, exige firma `LZSS`, tamaño descomprimido de 32 bits little-endian, comandos de ocho tokens, literales o referencias anteriores con distancia de 12 bits más uno y longitud de cuatro bits más uno. Longitud uno termina el flujo; las copias solapadas están permitidas. Se comprueban distancias, tamaños y terminación antes de publicar resultados. Clarity no usa el tamaño externo en su rama comprimida; la interpretación de ese tamaño como bytes codificados se contrasta con el fixture de la build nativa 10896 y es una restricción explícita del dialecto local.

La estructura descomprimida contiene tres contadores de 16 bits: tipos/extensiones, directorios y entradas. Siguen las tablas de cadenas UTF-8 terminadas en NUL. Cada entrada lleva índice de directorio, nombre terminado en NUL e índice de tipo. Cada índice ocupa `max(1, ceil(log2(tamaño_tabla)))` bits; se rechazan índices fuera de tabla. La referencia se reconstruye exactamente como `directorio + nombre + "." + extensión`, sin normalización de rutas ni lectura de archivos.

No hay campo de versión de formato, build, hash de contenido del asset ni parámetro monetario tipado en esta estructura. `native_format_version=null` expresa el dato ausente. Que el lector funcione con el fixture de build **10896**, patch/protocolo **14177**, no demuestra compatibilidad universal con otras builds. Los nombres de recursos tampoco prueban qué contenido se cargó o qué reglas fueron efectivas.

Se exige consumo completo de ambos flujos, salvo un máximo de siete bits finales de valor cero en cada uno. El fixture de 7.160 bytes contiene 7.156 bytes codificados, produce 13.146 bytes, cinco tipos, 119 directorios y 463 entradas; consume 57.273 bits externos más siete de relleno y 105.166 internos más dos de relleno. La rama sin compresión y las entradas repetidas se verifican mediante fixtures sintéticos; en la demo sólo se observó la rama LZSS.

Los límites locales son 1 MiB de entrada y salida descomprimida, 4.096 elementos por tabla y entradas, y 4.096 bytes por cadena. No son máximos oficiales del motor. Ausencia del campo, contenido malformado o dialecto no soportado, exceso del límite de entrada y decodificación válida se publican como estados distintos. No se filtran valores de una decodificación parcial.

## Procedencia y privacidad

Cada sesión conserva tick y secuencia compartidos con el contexto nativo, ruta protobuf, presencia, tamaño y SHA-256 del bloque. El resultado conserva compresión, tamaños, consumo, relleno, SHA-256 de bytes descomprimidos y orden de tablas. El SHA de los bytes descomprimidos corresponde a la tabla, no a sus assets. Cada entrada conserva ordinal desde cero, índices nativos y SHA-256 de la referencia UTF-8 reconstruida. Las tablas conservan SHA-256 de cada nombre, también en orden. Las entradas repetidas no se deduplican. No se publican bloques crudos ni cadenas arbitrarias.

El [schema](../../../ai_coach/contracts/economy_resource_manifest.schema.json) es cerrado. El [lector Python](../../../ai_coach/economy/resource_manifest.py) comprueba esquema, identidad, cobertura de sesiones, fingerprints, ordinales, índices y coherencia de tamaños. Enlaza por ruta fija, tamaño y SHA-256 el manifest de contexto y toda su cadena hasta el bundle base. `available_sessions` exige tick y secuencia disponibles; el orden del callback no acredita conocimiento del jugador. Un hash permite contrastar bytes disponibles, pero no reconstruirlos: el smoke compara cada campo y cada hash contra un lector Python independiente del bloque privado cuya huella coincide con la nueva captura.

`exact_build_asset_binding=not_accredited`, `economic_rules=not_accredited`, `comparison_allowed=false`, `training_allowed=false` y `product_supported=false` son obligatorios. Identidad exacta, referencias históricas candidatas, conflictos y datos ausentes se mantienen separados. No se deducen precios, fórmulas de bonus, compras únicas, loadout, límites o devoluciones a partir de nombres, índices, contadores o defaults.

## Ejecución

Después de las pruebas, ejecutar el exportador desde `backend/go-service` con los argumentos habituales de demo aprobada, alias, SHA-256, mapas, build y TEMP nuevo/vacío, añadiendo `--ai-coach-economy-candidate --economy-candidate-version 6`. Los prechecks rechazan destino no temporal, no vacío y sobreescritura antes de analizar la demo.

Desde Python, importar `Path` y `ai_coach.economy.resource_manifest` y llamar a `resource_manifest.load_candidate(Path(artifact), require_observed=True)`. El loader es de sólo lectura. No se promueve el resultado ni se reescriben los 44 bundles. Un único match aprobado por cierre sigue siendo obligatorio.
