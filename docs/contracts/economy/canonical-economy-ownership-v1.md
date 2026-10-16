# Tenencia económica observada v1

Contratos candidatos `stratai.economy_ownership_observations@1` y `stratai.economy_ownership_reconciliation@1`, versión `1.0.0-candidate.1`. El selector **9** conserva los productores anteriores y añade una lectura fresca de dueño e inventario al cerrar cada fotograma seleccionado. El registro tiene 55 schemas; catálogo `1.7.0`, export `3.8.0`, parser `v16` y elegibilidad mantienen su significado.

## Corte de observación

En el SDK fijado `demoinfocs-golang/v5@v5.2.0`, `parseFrame` encola el token de cierre después de los mensajes de ese comando; `handleFrameParsed` procesa los eventos diferidos y después emite `FrameDone`. El selector conserva el orden explícito entre consumidores del proyecto de 3D.5. La nueva lectura ocurre dentro de ese callback, antes de incrementar el contador local del observador. Es un estado del SDK al cierre de **ese comando/fotograma de demo**, no un instante subtick del servidor ni una garantía de que todos los comandos con el mismo tick ya se hayan entregado. [Fuente primaria](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/parsing.go).

Se seleccionan únicamente callbacks con warmup falso, freeze nativo verdadero y tick estrictamente posterior al inicio de instancia y anterior al fin de freeze, cuando ya se ha observado ese límite. El lector contrasta de nuevo cada corte con las fronteras finales verificadas del candidato v2. Esas fronteras se usan para validar evidencia retrospectiva, nunca para rellenar T0 o representar conocimiento del jugador.

Cada corte conserva secuencia, fotograma, tick, instancia, última secuencia de identidad, última muestra de reloj y referencia al estado leído. El prefijo de identidad debe contener todas las entregas observadas hasta ese fotograma y ninguna posterior. Se guarda un marcador por cada `FrameDone` seleccionado. Sólo se comparte almacenamiento cuando dos lecturas frescas consecutivas producen exactamente el mismo payload; se abre estado nuevo al cambiar de instancia. No se arrastra una lectura sin volver a consultar propiedades ni se referencia un estado antes de su primer corte.

## Lectura de propiedades

La fuente enumera todas las entidades vivas de las clases consultadas en [identidad v1](canonical-economy-native-identity-v1.md). Por cada equipo lee nuevamente `m_hOwnerEntity`. Por cada pawn conserva alias y lado disponibles, presencia y longitud de `m_pWeaponServices.m_hMyWeapons`, y sólo los slots dentro de esa longitud. Una longitud mayor de 64 queda fuera de la sonda; una propiedad ausente o de tipo no admitido no se convierte en inventario vacío. Los slots inactivos no cuentan como tenencia.

Cada referencia vuelve a comprobar índice, serie completa, clase y disponibilidad **en ese corte**. Una referencia que no podía resolverse en un callback anterior puede resolverse en una nueva lectura posterior: ambas observaciones quedan separadas, sin corregir retrospectivamente la primera. Las vidas sustituidas sin destrucción observada desaparecen del conjunto actual por su alias; no se inventa un callback de destrucción. Los actores siguen requiriendo vínculo controlador–pawn por serie completa y un alias ya disponible.

El lector verifica la cobertura exacta de entidades contra el prefijo de identidad, las referencias, el número y orden de slots, la presencia, la secuencia y las fronteras. Una compresión de estado no puede ocultar un cambio de vida ni cruzar instancias.

## Reconciliación y abstenciones

Los registros corresponden a **equipo por estado distinto**, no a compras ni objetos únicos del dataset. Se conserva una matriz de razones:

| Estado | Significado |
|---|---|
| `consistent` | Dueño y una única referencia activa de inventario señalan el mismo pawn; inventarios del conjunto observado completos y actor local único disponible. |
| `pair_with_incomplete_inventory` | La pareja dueño/slot coincide, pero otro inventario no permite comprobar exclusividad dentro de la sonda. |
| `actor_unavailable` / `actor_binding_nonunique` | Coincidencia por pawn sin identidad de actor suficiente o única. |
| `owner_inventory_conflict` | Dueño y slot activo resuelto señalan pawns diferentes. |
| `duplicate_inventory_claim` | Hay varias referencias activas al mismo objeto, incluso si repiten un slot del mismo pawn. |
| `owner_only` / `inventory_only` | Sólo una de las fuentes aporta la relación resuelta. |
| `unassigned_in_observed_scope` | Handle de dueño con sentinel inválido, ninguna referencia activa resuelta al objeto e inventarios completos dentro de las clases consultadas. No es una afirmación global de ausencia física. |
| `no_resolved_claim` | No hay relación resuelta suficiente; no equivale a objeto abandonado o inexistente. |

El acceso por oportunidad exige además instancia competitiva verificada y actor/lado en su roster. No exige ni concede estado inicial precompra: el dato sigue siendo de ejecución. La matriz incluye las 50 oportunidades cuando el candidato contiene 25 instancias; sus contadores cuentan estados distintos con coincidencia, nunca oportunidades elegibles para aconsejar.

Una transición sólo compara el **mismo alias de vida** en cortes consecutivos, fotogramas adyacentes de la misma instancia y muestras de red presentes del propio fotograma dentro del mismo segmento de reloj. Ambos extremos deben ser `consistent` o `unassigned_in_observed_scope`, y cambiar el actor observado. Se distingue cambio de tenedor, aparición y desaparición de tenencia resuelta. Se conserva `cause=not_accredited`: no demuestra compra, drop voluntario, regalo, cooperación, devolución ni coste. Un estado intermedio incierto, reloj ausente, salto de fotograma, cambio de segmento o nueva vida impide unir los extremos.

Los eventos y callbacks SDK permanecen en la traza y su extensión de identidad. No se unen por tipo o proximidad para explicar estas transiciones. Los desacuerdos de reloj del evento no se corrigen con offsets; la comparación expresa únicamente cambios entre cortes de entrega, sin interpolar ejecución física.

## Comandos y validación

Desde `backend/go-service`, el productor usa `--ai-coach-economy-candidate --economy-candidate-version 9` junto con fuente aprobada, alias, checksum, build y destino TEMP nuevo. Desde la raíz:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.ownership --source $ownershipSource --output $newEmptyTemp --require-observed
```

Se rechaza salida no temporal, no vacía o con enlaces antes de leer la fuente. `economy_ownership_observations.json` enlaza la identidad por nombre fijo, bytes y SHA-256; la cadena llega a la traza, candidato v2 y canónico. `economy_ownership_reconciliation.json` fija a su vez el hash de esa fuente. El validador del informe recalcula registros, transiciones y matriz; modificar una causa, alias, serie, corte o resultado invalida la comparación con su fuente.

El gate real exige registros afectados, cobertura de variantes observadas, recálculo independiente y revisión canónica completa después de las pruebas. Fixtures de conflictos o resets no acreditan su presencia en la demo. `comparison_allowed=false`, `training_allowed=false` y `product_supported=false` se exigen en ambos contratos; transacciones itemizadas, reglas exactas y cooperación mantienen sus dependencias separadas.
