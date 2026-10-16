# Contexto físico propio/aliado — candidato v1

Contrato `1.0.0-candidate.1`, fase 3E.2. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y continuación. Extiende los [anclajes factuales v1](canonical-support-anchors-v1.md) con observaciones físicas en el mismo corte, sin cambiar su protocolo, selección, ID o T0. No acredita una decisión anterior a la entrada, apoyo seguro ni preparación completa del arma.

## Artefactos y procedencia

El modo optativo y exclusivo `--ai-coach-support-physical-candidate` produce el bundle base `3.8.0`, el sidecar de anclajes `support_observations.json` conservado y un sidecar propio `support_physical_observations.json`. El contexto se captura dentro del mismo callback de replay/support, sin avanzar el parser. `source_sequence` referencia una muestra exacta del sidecar v1 y hereda tick, fotograma, época, instancia y vida de pawn de ese corte.

Cada fila usa el mismo alias local opaco de actor que v1. Un actor no vivo, desconectado o sin identidad/posición acreditada tiene contexto `null`; nunca se rellena con otra muestra. Los hechos nuevos no se serializan en replay ni reinterpretan los 44 bundles actuales. Parser `v16`, quality `12`, catálogo `1.7.0`, IDs y `DEC-TRADE-001` conservan sus contratos.

Los schemas estrictos son [observaciones físicas](../../../ai_coach/contracts/support_physical_observations.schema.json) y [contexto físico](../../../ai_coach/contracts/support_physical_context.schema.json). Fuente física, fuente v1, informe de anclajes y rondas canónicas se enlazan por bytes/SHA-256. La build y el checksum de demo deben coincidir. El lector valida el manifiesto y los hashes del bundle completo. Cualquier referencia incoherente se rechaza.

## Hechos del corte exacto

| Campo | Fuente / transformación | Ausencia y límite |
|---|---|---|
| `view_angles` | `m_angEyeAngles`, vector nativo completo y finito: yaw = Y, pitch = X, grados | Cero presente es válido. Campo ausente, tipo inesperado, vector parcial o no finito queda indisponible. No demuestra atención, FOV ni contacto. |
| `walking` | Bool nativo `m_bIsWalking` | No se deduce una tecla ni se aplica un multiplicador de velocidad. Un entero cero no se convierte en bool observado. |
| `ducking` | `m_fFlags` uint64 y accessor `PlayerFlags.Ducking()` del SDK fijado | Es la bandera de postura, no intención ni tecla pulsada; falta/tipo incorrecto permanece ausente. |
| `active_weapon` | Observación actual del mismo replay, reconciliada con el arma resuelta por handle completo y entidad actual | Un nombre coincidente o una referencia por índice sin serie no bastan. |
| `ammo_in_magazine` | Observación actual del replay contrastada con `m_iClip1` uint32; el SDK fijado representa munición como valor codificado menos uno | Se retiene el valor codificado para recalcular. Codificado 1 acredita cero balas; codificado 0, tipo incorrecto, ausencia o desacuerdo no acreditan munición. |
| `ammo_reserve` | `m_pReserveAmmo.0000`, o `m_iPrimaryReserveAmmoCount` cuando la primera está ausente; int32 no negativo, contrastado con replay | Cero presente se conserva; negativo o tipo incorrecto no se normaliza. La propiedad seleccionada queda en procedencia. |
| `reloading` | Tracker causal existente, inicializado por callback acreditado, refrescado al tick actual y contrastado con la observación actual del replay | Sin señal inicializada permanece ausente. Se conserva señal y tick originales separados del tick de disponibilidad actual. No equivale a cooldown, precisión o readiness completa. |

Cada hecho contiene `status=observed/unavailable/not_applicable`, `value` y `reason`. `not_applicable` sólo corresponde a munición/recarga de Knife, C4 y utilidad sin munición de cargador; su valor es `null`, no cero. Una clase desconocida mantiene ausencia. `false` observado y dato no disponible son distintos.

La identidad del arma exige `m_pWeaponServices.m_hActiveWeapon` completo de 32 bits, índice y serie exactos frente a la entidad actual. La vida local sigue el objeto SDK y su destrucción. Un objeto destruido no se reutiliza; un nuevo objeto con el mismo handle recibe otra vida y su recarga se abstiene por `weapon_handle_reused`, sin heredar una señal antigua. La vida de objeto no demuestra por sí sola una vida física del jugador ni propiedad/intención de transferencia.

El tick original de señal de recarga no puede ser posterior al corte. La comprobación se basa en el tracker existente y su frontera de entrega, con fuente de código fijada; no añade una traza independiente de todos los eventos nativos ni acredita orden físico subtick. La extensión del tracker conserva ese tick como metadato interno y no cambia sus resultados previos.

## Movimiento derivado disponible en T0

Se reutilizan las dos últimas posiciones de la historia v1, sin añadir un stream de velocidad ni heredar un fallback del tracker de movimiento. Para cada participante:

`duración = (tick_actual − tick_anterior) / tick_rate`

`velocidad_media_XYZ = (posición_actual − posición_anterior) / duración`

La rapidez horizontal es la norma de las componentes X/Y. Se conservan extremos temporales, duración, vector, unidades y método `quantized_position_interval_average@1`. El resultado está disponible en T0 y resume ese intervalo, **no una velocidad instantánea en T0**. X/Y proceden de coordenadas redondeadas; los cambios pequeños y cero tienen esa resolución. No mide distancia recorrida por una curva, aceleración, colisiones, walk speed nominal ni viabilidad de una acción futura.

## Informe y acceso inicial

El materializador escribe `support_anchors.json` bajo el contrato v1 y `support_physical_context.json` separado. Reconstruye los anclajes v1 y exige igualdad completa de sus hechos iniciales antes de unir contexto. No usa el contexto añadido para seleccionar o mover anclajes. Cada anclaje conserva contexto propio y del aliado, disponibilidad, cobertura conjunta por componente y ausencias tácticas.

La matriz usa actor/aliado/instancia, con denominador de anclajes, componentes disponibles para ambos y conteos por estado/motivo. `all_components_available` significa que los siete componentes están observados o explícitamente no aplican; **no significa listo para disparar, apoyo viable, decisión elegible o recomendación**. Un campo parcial conserva su ausencia sin eliminar la oportunidad.

`competitive_round_status` sólo informa la unión posterior a un intervalo canónico único que contiene la historia y T0. La instancia preliminar queda `not_accredited`. Esa etiqueta de control no se devuelve como feature inicial ni selecciona anclajes; la unión no acredita intención táctica.

`initial_features(support, physical, anchor_id, actor, tick, frame)` valida y utiliza exclusivamente el prefijo hasta el anclaje exacto. Devuelve copias de hechos propios/aliados, movimiento y disponibilidad; no expone identidades de actor/arma, referencias, clasificación competitiva, enemigos u outcomes. La posición y el estado aliado son un proxy de información de equipo de la demo, no conocimiento subjetivo demostrado.

Entrada/compromiso previo, NAV, LOS/contacto, objetivo, cooperación, intención, velocidad instantánea y readiness completa permanecen ausentes. `comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` siguen en `false`.

## Ejecución y cierre

Desde `backend/go-service`, con la fuente aprobada resuelta por checksum y un destino TEMP nuevo/vacío:

```powershell
& $supportBinary --demo $approvedDemo --output $supportRunRoot --alias demo-a11eb0d600ee702c9edb --checksum a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262 --maps-root ../data/maps --build-id $buildID --ai-coach-support-physical-candidate
```

Desde la raíz, con un segundo directorio temporal vacío para los informes:

```powershell
.\backend\venv\Scripts\python.exe -B -m ai_coach.support.physical --source $physicalSource --output $newEmptyTemp --require-observed
```

El cierre exige suites automáticas, la misma única demo aprobada, validación completa, recálculo independiente de movimiento/munición/matriz, contraste de campos físicos con replay y conservación de v1/base. Los floats de orientación del replay se contrastan a su precisión float32, sin confundir la representación decimal abreviada con un cambio de ángulo. Se distinguen variantes reales y fixtures. Los resultados se conservan con inventario y hashes, sin intentos ni revisión de limpieza; el smoke no autoriza 44/44, promoción ni entrenamiento.
