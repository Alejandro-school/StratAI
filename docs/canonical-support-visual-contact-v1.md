# Entradas visuales en el corte support — candidato v1

Extensión optativa `1.0.0-candidate.1`: `stratai.support_visual_contact_observations@1` y `stratai.support_visual_contact_context@1`. El [plan](AI_COACH_IMPLEMENTATION_PLAN.md) fija el estado, la evidencia y la siguiente acción. Conserva los [anclajes](contracts/support/canonical-support-anchors-v1.md), el [contexto físico](contracts/support/canonical-support-physical-v1.md) y todas las capas hasta [temporización del arma](contracts/support/canonical-support-weapon-timing-v1.md).

## Fuente y resolución

`--ai-coach-support-visual-contact-candidate` activa la captura dentro del mismo callback support, después de 3E.9 y sin avanzar el parser. Conserva frame, tick, instancia, época, orden común y fronteras de entregas/estados. Cada observador debe corresponder al pawn actual, vivo y conectado, con handle completo, serie, vida y posición support utilizables. Una entidad destruida, cacheada o sustituida no hereda el estado anterior.

Se capturan las rutas que el SDK fijado usa como offset de ojos (`m_vecX/Y/Z`), FOV de cámara (`m_pCameraServices.m_iFOV`, `m_iFOVStart`, `m_flFOVTime`, `m_flFOVRate`), `m_bIsScoped`, `m_flFlashDuration` y `m_flFlashMaxAlpha`. Las declaraciones se resuelven antes de consultar propiedades. Cada valor conserva presencia, tipo declarado, tipo nativo y estado; cero no es ausencia. Se retienen creación, reemisiones del frame de creación, actualizaciones y destrucción, con valor previo y número de entrega. Los estados sólo se comparten dentro de la misma instancia/época cuando campos y últimas entregas coinciden.

Cada corte relee además los seis componentes `CBodyComponent.m_cellX/Y/Z` y `m_vecX/Y/Z`, con sus tipos. Se contrasta la decodificación `cell * 512 - 16384 + offset` contra la posición support, respetando el redondeo de X/Y del replay. Esta captura exacta del estado replicado no acredita percepción, input subtick ni interpolación de cámara. Los campos de coordenadas son snapshots actuales; no se presentan como una traza independiente de todas sus entregas.

## Diagnósticos físicos separados

El productor evalúa segmentos entre el origen decodificado más el offset nativo de cada pawn utilizable y los de los oponentes actuales. Registra la geometría cargada y las referencias SHA-256 del GLTF y de sus buffers. No lee ni arrastra el ledger histórico de contacto, `enemy_los`, radar/spotted, una muerte posterior o un contacto de otra muestra. Los índices de observador/objetivo sólo localizan el censo interno del corte.

El rayo describe **la malla estática seleccionada por el cargador existente**. Ese cargador filtra grupos `clip`, `grenadeclip`, `passbullets` y `window`; la selección no acredita por sí misma oclusión visual, materiales transparentes, transformaciones de cámara/pawn ni obstáculos dinámicos. Un rayo libre o bloqueado en esa malla sigue siendo diagnóstico. Geometría o endpoints ausentes dan `null`, nunca una ausencia física de obstrucción.

El consumidor calcula por separado el frustum rectangular canónico 16:9 desde la referencia 4:3 del proyecto. Sólo usa FOV nativo positivo, menor que 180, con origen y destino del FOV iguales y estado scoped explícitamente falso. Cero conserva dependencia de default; endpoints diferentes conservan dependencia de interpolación; óptica scoped o estado scoped ausente conservan abstención. Se admiten ángulos nativos equivalentes por vuelta, con pitch normalizado en `[-90,90]`; los rangos no admitidos se abstienen. Es un proxy de referencia, no una grabación del viewport del jugador.

La fuente no dispone de un volumen de humo evolutivo y una cobertura de entrega acreditados. La lista histórica de cilindros X/Y no suple esa ausencia. Tampoco `m_flFlashDuration` y el máximo de alpha demuestran la opacidad renderizada actual: el SDK puede conservar duración y derivar tiempo desde otra entrega. Ni siquiera el cero de esos valores desbloquea automáticamente contacto legal.

Por estas razones, esta versión **no publica contactos positivos ni negativos como información inicial legal**. Los schemas fijan `flash_clear`, `smoke_clear` y `visual_contact` a `unavailable/null`, con motivos distintos. No se infiere que el jugador no tuviera contacto.

## Acceso inicial y lector

Los cinco hechos por rol `self` y `ally` son `native_eye_offset`, `reference_fov_degrees`, `flash_clear`, `smoke_clear` y `visual_contact`. El offset describe las componentes consultadas por el accessor del SDK; no afirma posición mundial de la cámara. Los dos primeros pueden estar observados; los tres últimos mantienen las limitaciones anteriores.

`initial_features` recorta support, temporización y la nueva fuente a la frontera exacta de T0 antes de validar. Devuelve copias de esos hechos propios/aliados. No devuelve censo enemigo, posiciones objetivo, rayos, identidades, índices, desenlaces ni conteos dependientes de enemigos. `visual_reference` y `evaluator_diagnostics` permanecen separados: los conteos de pares son procedencia interna y nunca features. Visión aliada no implica conocimiento subjetivo propio, voz, atención o intención.

La fuente enlaza las nueve fuentes anteriores por bytes/SHA-256. El materializador reconstruye las capas previas y conserva selección, IDs/T0, clasificación competitiva o preliminar y matrices. El lector contrasta todos los informes y recalcula los rayos desde los assets identificados mediante `cmd/support-visual-check`, que sólo lee fuentes y mapas. El verificador reconstruye los endpoints con otra expresión aritmética, pero reutiliza el motor de intersección del proyecto; no constituye una validación independiente de ese motor ni de la suficiencia visual de la malla. Los schemas son estrictos y rechazan permisos o contactos positivos inventados.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-visual-contact-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.visual_contact `
  --source $visualSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --visual-evaluator $visualCheckBinary `
  --maps-root backend/data/maps
```

Añadir `--validate` para releer y recomponer el informe completo. Los binarios se compilan desde `cmd/canonical-export`, `cmd/support-navigation` y `cmd/support-visual-check`. Se rechaza un destino no temporal/no vacío antes del parser o materializador; no se sobrescriben artefactos. Cada cierre funcional exige suites y la misma única demo aprobada a TEMP nuevo, con inventarios retenidos.

## Material positivo pendiente

Para acreditar contacto hace falta una fuente física/temporal que vincule a esta demo: geometría y materiales de oclusión visual con cobertura y transformaciones comprobables; vida/estado actual de los obstáculos dinámicos; volumen efectivo de humo y cobertura de sus entregas; estado de deterioro visual por flash en el mismo reloj; y FOV/óptica/transformación efectivos cuando exista transición o un default. Cada vía debe quedar disponible en el prefijo y tener una comprobación independiente. Más muestras del mismo proxy, ausencia de eventos, una fórmula de otra build o juicios humanos sobre un rayo no sustituyen esos datos.

No cambia el significado histórico de `enemy_los` en 3.8.0, el ledger visual anterior ni `DecisionExample`, `DEC-TRADE-001`, elegibilidad, comparaciones, recomendaciones o entrenamiento. `decision_eligible`, `comparison_allowed`, `training_allowed` y `product_supported` siguen en `false`. Contacto aislado tampoco acreditaría apoyo viable S2–S3, atención o superioridad táctica. Las dependencias de economía, NAV físico, velocidad instantánea, C4 transportada/caída y codec/temporización del arma se conservan.
