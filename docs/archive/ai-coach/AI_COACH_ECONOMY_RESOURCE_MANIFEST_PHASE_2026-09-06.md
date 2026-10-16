# Manifiesto nativo de recursos de sesión — fase 3D.2D

Fecha: 6 de septiembre de 2026. Resultado funcional: lector y exportación optativa implementados y validados; limpieza temporal pendiente por rechazo automático. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) decide el siguiente trabajo. [Contrato](../../contracts/economy/canonical-economy-resource-manifest-v1.md) y [evidencia reproducible](../../evidence/AI_COACH_ECONOMY_RESOURCE_MANIFEST_SINGLE_DEMO_SMOKE_2026-09-06.json).

## Qué hemos conseguido

El segundo bloque de sesión ya tiene un formato demostrado para la demo observada. El manifiesto comprimido de **7.160 bytes** contiene una tabla de **463 referencias de recursos**, organizada mediante **cinco tipos y 119 directorios**. El codec se obtiene de una implementación pública identificable y se comprueba desde el primer bit hasta el último relleno. Los dos bloques investigados están ahora interpretados dentro de límites explícitos: la configuración VBKV de 3D.2C y este manifiesto de recursos.

Se añadió el selector **6**, el artefacto `economy_resource_manifest.json`, su schema @1, el observador Go y el lector/validador Python. Conservan orden, índices, tick, secuencia, presencia y hashes; ninguna cadena arbitraria ni bloque opaco se incorpora al repositorio. Los selectores anteriores mantienen sus contratos. El registro runtime pasa a **51 schemas**, con 19 motivos; export `3.8.0`, parser `v16`, quality `12`, catálogo `1.7.0` e IDs permanecen iguales.

Esto mejora la trazabilidad. Todavía no permite recomendar qué comprar: la lista dice qué nombres de recursos referencia la sesión, pero no acredita los bytes de esos archivos, la distribución exacta que los suministró ni las reglas económicas efectivas.

## Fuente y demostración del codec

Se inventariaron los metadatos locales de Steam/CS2 y los árboles públicos fijados de HL2SDK, CounterStrikeSharp, ValveResourceFormat, Clarity y Manta. La selección inicial fue por nombres de archivo relacionados con recursos/manifiestos/sesión; ese filtro no demuestra ausencia de código en los repositorios restantes. Al encontrar el lector pertinente se detuvo la búsqueda genérica. El grafo local devolvió `Transport closed`; se usó descubrimiento localizado de código.

La fuente suficiente fue [Resources.java de Clarity, revisión 7fb3f1d07564a12efa99194d45cfbf5762ba5910](https://github.com/skadistats/clarity/blob/7fb3f1d07564a12efa99194d45cfbf5762ba5910/src/main/java/skadistats/clarity/processor/resources/Resources.java). Registra `CSVCMsg_ServerInfo` para `CSGO_S2` y llama al lector del manifiesto. Se fijaron también [LZSS.java](https://github.com/skadistats/clarity/blob/7fb3f1d07564a12efa99194d45cfbf5762ba5910/src/main/java/skadistats/clarity/util/LZSS.java), BitStream, BitStream32, Util y licencia; la evidencia contiene URL, bytes y SHA-256 de cada archivo. No se ejecutó el runtime completo de Clarity ni se presenta su implementación como especificación oficial de Valve.

El formato empieza con un bit de compresión y 24 bits de tamaño; el payload comprimido empieza en el bit **25**, según el lector fuente. No se buscaron offsets por ensayo. La descompresión LZSS produce las tablas de tipos, directorios y entradas. Clarity ignora el tamaño externo en la rama comprimida: su interpretación como bytes codificados se acredita en este fixture y se impone como restricción del dialecto local. No se equipara este bloque de red con archivos `.vrman_c`.

| Comprobación independiente | Resultado |
|---|---|
| Bloque original | 7.160 bytes; SHA-256 `6f2854ab5b7551145c8fec8e8f9ae1db942e8d1f96c91595c4c3a90e8d1539b2` |
| Payload codificado | 7.156 bytes, LZSS |
| Tabla descomprimida | 13.146 bytes; SHA-256 `d91ebccf392ce106364c5f0d836b4e965f91772d1d5777eda6c2363e084fd810` |
| Consumo externo | 57.273 bits más siete bits cero |
| Consumo interno | 105.166 bits más dos bits cero |
| Tipos/directorios/entradas | 5 / 119 / 463 |
| Entradas por índice de tipo 0–4 | 334 / 65 / 60 / 3 / 1 |
| Nombres reconstruidos únicos | 463; el lector conserva duplicados si existen |
| Diferencias de campos y hashes frente al nuevo export | 0 |
| Disponibilidad | Tick 0, secuencia 1; no disponible antes del tick o callback |

Un lector Python independiente lee bit a bit el fixture privado anterior, de sólo lectura, siguiendo el formato fijado. Se contrasta su SHA-256 con el bloque capturado por la nueva pasada completa y se comparan todas las tablas, índices, ordinales y hashes. La compatibilidad demostrada corresponde al fixture de la build nativa **10896**; no a todas las versiones del motor. Los fixtures sintéticos cubren además la rama sin compresión, duplicados y fallos.

## Identidad y economía: lo que sigue sin demostrarse

- **Identidad exacta demo–assets:** no acreditada. El formato no contiene campo de versión ni hashes del contenido de los assets. El hash de la tabla descomprimida tampoco es un hash de esos archivos.
- **Referencias históricas candidatas:** siguen siendo cliente/servidor **2000888** y **2000891**; el conflicto no se resuelve escogiendo una. La demo conserva build nativa **10896** y patch/protocolo **14177**.
- **Instalación local:** dos bibliotecas Steam inventariadas, una instalación CS2. Cliente/servidor **2000899**, patch **1.41.7.8** y Steam app build **25000182**, con manifests de seis depots y hashes de `steam.inf` y tres binarios conocidos. Son namespaces diferentes; ese paquete no se selecciona para esta demo. No se ejecutaron binarios ni se descargaron paquetes privados. No se encontraron PDB en los dos directorios win64 consultados; no se afirma ausencia global de símbolos.
- **Referencias de los dos bloques:** ninguna de las doce apariciones de nombres de modelos de configuración coincide exactamente por SHA-256 con las 463 referencias reconstruidas. No se aplicó normalización de sufijos/rutas; el resultado no demuestra ausencia de esos modelos ni incompatibilidad de contenidos.
- **Reglas económicas nuevas:** cero acreditadas. La tabla no lleva parámetros monetarios tipados. Recompensas, fórmula de bonus por derrotas, precios condicionados, loadout, límites, compra y devolución siguen pendientes de sus propias fuentes. No se afirma ausencia de reglas en toda la demo.

Las puertas `comparison_allowed`, `training_allowed` y `product_supported` siguen cerradas. Las capturas nativas no se convierten en compras únicas; los contadores no se convierten en dinero ni los defaults históricos en valores efectivos.

## Verificación y conservación

Pasaron **`npm run check:all`**, **510 pruebas del AI Coach** —27 nuevas—, Ruff de archivos Python afectados y validación global del contrato. El fuzz del lector pasó **102.148 ejecuciones** en 5,221 s. Los tests incluyen truncado en cada byte, índices inválidos, tamaños, copias LZSS solapadas, datos sobrantes, UTF-8 inválido, ausencia frente a vacío y rechazo de destinos no temporales/no vacíos antes del parser. Un test nuevo utilizó inicialmente una firma incorrecta del helper existente; se corrigió antes de la suite completa y del smoke.

Se procesó **una sola demo aprobada**, alias `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, una pasada completa en `C:/Users/kerch/AppData/Local/Temp/stratai-3d2d-smoke-sm09pvug`, creado nuevo y vacío. Parser: **85,655 s**; reconstrucción nativa: **5,528 s**; validación de candidatos y recálculo: **30,983 s**. La validación semántica canónica completa terminó con **cero errores en 292,764 s**.

Se conservaron los cinco enteros y doce referencias de configuración anteriores, 77 actores inicialmente acreditados, 173 abstenciones, tres equipos competitivos completos y 50 fichas de oportunidad, con 60 diferencias numéricas y 15 transiciones. El inventario protegido antes/después coincide: **4.356 archivos, 181 hashes**, 44 bundles y 50 fuentes. No se promovió ninguna salida ni se tocó el corpus, releases, runtime o `CURRENT`.

La evidencia registró **84 archivos / 743.913.051 bytes** antes de intentar la limpieza. La revisión automática rechazó `Remove-Item -LiteralPath $newSmokePath -Recurse -Force` antes de crear el proceso, motivo `blocked by policy`. Una comprobación posterior de sólo lectura confirmó que el destino sigue presente y todos los hashes coinciden. No se reintentó ni eludió el rechazo. Las limpiezas antiguas tampoco se repitieron. Las 22 evidencias anteriores y las 308 fuentes de la build validada conservan sus hashes.

El cierre documental comprobó **317 enlaces locales en 52 documentos**, sin destinos ausentes, un único prompt activo y `git diff --check` correcto. De los 977 archivos previos inventariados, 962 mantienen su hash y 15 cambios pertenecen al alcance de la fase, incluida la retirada del prompt completado. Las fuentes funcionales coinciden con la build validada.

## Dependencia que queda

La siguiente fase **3D.2E** debe acreditar la cadena **build nativa → distribución exacta → contenido de los archivos económicos**. Se necesita un paquete o registro de procedencia correspondiente a build nativa 10896 que conecte esa build con versiones cliente/servidor, app/depot/manifest IDs y hashes de los archivos económicos. Puede proceder de quien custodie esa instalación histórica, del distribuidor o de fuentes públicas que demuestren la misma cadena; no basta con un paquete de patch parecido o una fecha coincidente.

La aceptación exige comprobar cada eslabón, rechazar metadatos o contenidos alterados, distinguir candidato/conflicto/ausencia y mantener aparte los overrides efectivos de sesión. Si ese material no está disponible, se entrega la dependencia concreta y se condiciona el trabajo que depende de ella; no se repiten el codec, defaults o una demo para volver a constatar el mismo hueco.

Seguimos en el **paso 3: completar datos verificables**. Después quedan las transacciones y reglas necesarias para comparar alternativas, el prototipo que conecte análisis, perfil, prioridades y práctica, la revisión con coaches y jugadores, y el seguimiento entre partidas. Este cierre no habilita entrenamiento, comparación prescriptiva ni la futura migración 44/44.
