# Auditoría del vínculo exacto entre demo y assets — 3D.2E

Fecha: 6 de septiembre de 2026. **Auditoría documental terminada; compatibilidad exacta pendiente de material externo identificable.** No se cambió productor, schema, comprobador ni hechos derivados. [Evidencia](../../evidence/AI_COACH_ECONOMY_EXACT_ASSET_BINDING_AUDIT_2026-09-06.json), [dependencia y aceptación](../../ai-coach/AI_COACH_ECONOMY_EXACT_ASSET_DEPENDENCY.md) y [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md).

## Resultado en lenguaje sencillo

Sabemos leer los dos bloques de sesión. Esta fase comprobó si sus datos y las fuentes históricas permiten saber qué archivos económicos exactos utilizó la partida. **Todavía falta esa prueba.** Se ha localizado el hueco con mayor precisión: las referencias históricas conservan versiones, textos transformados y asociaciones declaradas con actualizaciones de Steam, pero no la cadena completa desde la build observada hasta los archivos originales.

Se entregó una especificación de los registros, manifests, binarios/recursos y datos de extracción que permitirían acreditarla, con proveedores posibles y nueve casos de aceptación/rechazo. Esos criterios no se presentan como tests de un paquete inexistente. Ninguna de las dos referencias históricas se elige por coincidencia de patch, nombres o precios.

## Hallazgos nuevos y verificables

Los dos `steam.inf` descargados desde sus revisiones inmutables coinciden exactamente en tamaño y SHA-256 con el registro existente. Se conservan cliente/servidor **2000888** y **2000891**, ambos con patch **1.41.7.7**. Sus `SourceRevision` son **10936007** y **10937988**, mientras `game/bin/built_from_cl.txt` contiene **10935971** y **10937911**. Estos marcadores no se equiparan con la build nativa 10896.

Los mensajes de commit citan respectivamente las actualizaciones **24916958** y **24934554**. Son asociaciones declaradas por el tracker, no manifests de depot ni un vínculo acreditado con la demo. Fuentes: [fd04856b](https://github.com/SteamTracking/GameTracking-CS2/commit/fd04856b8668f11cb31ed2b22574607dd9c365e0) y [acfe24d5](https://github.com/SteamTracking/GameTracking-CS2/commit/acfe24d588d2df0a26da0f964e44d780bd3070eb).

La selección de descarga del tracker enumera cuatro depots —2347770, 2347771, 2347773 y 2347779— con filtros, sin acreditar sus manifests históricos. Su proceso utiliza descompilación VPK y excluye del Git binarios, VPK, manifests VPK y `steam_buildid.txt`. Esto obliga a distinguir **hash del texto publicado**, **hash del recurso original** y **procedencia de distribución**. La documentación contractual se ha aclarado; el registro de fuentes y sus hashes no cambian. [Selección fijada](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/files.json), [proceso fijado](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/update.sh), [exclusiones](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/.gitignore).

Se inventariaron árboles de **8.871 y 8.886 entradas**, ambos sin truncado, y se comprobaron las categorías concretas de archivos originales ausentes. Las sondas de identidad en `module_metadata/engine2.kv3` y `server.kv3` no aportaron una relación con 10896. No se afirma que todas las fuentes públicas o locales carezcan de ella: el alcance queda limitado a los materiales identificados y consultados. La búsqueda web acotada no produjo una fuente pertinente que completara el enlace; no se usaron resultados ajenos al objeto de la auditoría.

Se recontrastaron los cinco hashes locales anteriores de `steam.inf`, appmanifest y tres binarios, todos iguales. La instalación sigue representada por cliente/servidor **2000899**, patch **1.41.7.8** y Steam app build **25000182**; su marcador `built_from_cl.txt` es **10948893**. No se ejecutó ningún binario ni se extrajeron assets de ese paquete para aplicarlos a la demo.

La documentación oficial explica que Steam genera un manifest por versión de depot y un BuildID global para la build de aplicación; no suministra la correspondencia de esta partida. [SteamPipe](https://partner.steamgames.com/doc/sdk/uploading). Además, el SDK sólo declara `CDemoFileHeader.build_num` como campo opcional: es necesario acreditar qué componente emitió esa versión y cómo se relaciona con los binarios pertinentes, sin atribuir al encabezado una identidad completa del servidor.

## Lo que permanece ausente

No se obtuvo una relación verificada de 10896 con la distribución correspondiente, ni manifests históricos exactos vinculados a ella, ni los originales necesarios y la extracción reproducible que permita conectar los textos consumidos. Los campos de identidad exacta permanecen nulos/no acreditados. La instalación actual y las dos referencias históricas conservan sus categorías separadas.

La dependencia especifica un registro de captura con procedencia, componente emisor, plataforma, arquitectura y hashes; metadatos app/depot/manifest; archivos originales; y herramienta/argumentos de transformación. Pueden aportarlos el custodio de esa instalación histórica, el distribuidor o un archivo público que conserve la misma cadena. Los mantenedores del tracker podrían aportar registros originales si los conservan. No se contactó con nadie ni se adquirieron paquetes privados.

Tampoco se acreditaron nuevas reglas monetarias, loadout, bonus, precios condicionados, límites o devoluciones. Incluso un paquete exacto necesitaría contrastarse con los overrides y hechos de sesión. Las puertas de comparación, entrenamiento y producto siguen cerradas.

## Verificación y conservación

La auditoría verificó hashes y metadatos de las fuentes; todas las fuentes públicas utilizadas quedan fijadas por URL/revisión cuando existe, bytes y SHA-256. Las páginas de documentación dinámica conservan URL y hash de la captura, sin inventar una revisión Git. Se registran también los límites de las búsquedas, los nulos y los scripts de auditoría.

Pasó `backend/venv/Scripts/python.exe -B -m ai_coach.contracts.validate_contract`: **51 schemas, 19 motivos**, catálogo `1.7.0` intacto. Al no modificar código funcional, no se repitieron las suites ni el smoke. Las **510 pruebas y una demo de 3D.2D son evidencia anterior**, no una ejecución atribuida a esta fase. Se procesaron **cero demos**, se descargaron cero demos/paquetes privados y no se promovió ni entrenó nada.

El inventario protegido antes/después coincide: **4.356 archivos / 181 hashes**, 44 bundles y 50 fuentes. Los temporales anteriores fueron de sólo lectura y sus limpiezas rechazadas no se intentaron de nuevo. No se generó otra salida grande de demo que limpiar. El cierre documental verificó **334 enlaces locales en 54 documentos**, sin destinos ausentes, un único prompt activo y `git diff --check` correcto. Las 23 evidencias previas y las 308 fuentes de la build validada mantienen sus hashes. De 989 archivos previos, 982 permanecen iguales y siete cambios son documentales dentro del alcance.

## Próximo paso

La compatibilidad exacta queda condicionada a recibir el material documentado. La siguiente acción seleccionada es **3D.3: distinguir transacciones económicas observables** mediante eventos, mensajes y actualizaciones nativas, conservando lo ambiguo. Este trabajo puede avanzar sin precios históricos: que una compra, recogida o devolución esté observada es una cuestión distinta de determinar qué alternativas habrían sido legales o mejores.

Seguimos en el **paso 3, completar datos verificables**. Después quedan la comparación respaldada de alternativas, el ciclo de perfil/prioridades/práctica y su validación con coaches y jugadores. Esta auditoría no declara completada la fase económica 3D ni habilita recomendaciones.
