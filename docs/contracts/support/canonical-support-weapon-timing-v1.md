# Temporización nativa del arma propia y aliada v1

Extensión optativa `1.0.0-candidate.1`: `stratai.support_weapon_timing_observations@1` y `stratai.support_weapon_timing_context@1`. Conserva los [anclajes](canonical-support-anchors-v1.md), el [contexto físico](canonical-support-physical-v1.md), los [relojes](canonical-support-objective-clock-v1.md) y las capas hasta [movimiento nativo](canonical-support-native-motion-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción.

## Fuente actual y procedencia

El selector `--ai-coach-support-weapon-timing-candidate` conserva todos los productores previos y añade observaciones en el mismo callback support. La lectura física actual se pasa directamente al observador antes de su serialización. No se usa la muestra física del frame anterior.

| Objeto | Rutas consultadas |
|---|---|
| Pawn actual | `m_pWeaponServices.m_hActiveWeapon`, `m_pWeaponServices.m_flNextAttack`, `m_flSimulationTime`, `m_nTickBase` |
| Arma activa | `m_hOwnerEntity`, `m_nNextPrimaryAttackTick`, `m_flNextPrimaryAttackTickRatio`, `m_nNextSecondaryAttackTick`, `m_flNextSecondaryAttackTickRatio`, `m_nPostponeFireReadyTicks`, `m_flPostponeFireReadyFrac` |

La declaración se resuelve en los serializers de la propia demo antes de consultar la propiedad. Cada escalar conserva presencia, valor, tipo nativo, tipo declarado y estado de decodificación. Cero, ausencia, tipo no admitido y valor no finito permanecen separados. La declaración por sí sola no acredita un valor replicado ni su uso temporal.

Las entregas distinguen creación, callbacks en el frame de creación (`creation_replay`), actualización posterior y destrucción. `creation_replay` puede incluir reemisiones del SDK y otros callbacks de ese mismo frame; no afirma que cada entrega sea un cambio nativo nuevo. Se retienen siempre valor previo y valor recibido. Cada objeto SDK tiene una vida nativa local. Los estados compartidos conservan la última entrega de cada propiedad y no atraviesan instancia/época. Cada corte contiene fronteras de entregas/estados, frame, tick de demo y orden común de reloj. Ese orden pertenece al observador, no demuestra orden físico subtick.

El pawn debe estar vivo, disponible y seguir en el mapa actual de entidades con la misma vida support. El arma se resuelve mediante el handle completo, incluido su número de serie; la entidad debe seguir actual y viva. Para habilitar sus hechos se exige además dueño nativo apuntando al mismo pawn y coincidencia con handle, vida y observación de arma del contexto físico actual. El registro físico debe asignar esa vida a la misma entidad nativa actual y mantenerla viva; un handle completo reutilizado no basta para enlazar dos vidas. Una referencia cacheada, soltada, destruida o reutilizada no hereda plazos. Una identidad o tenencia insuficiente conserva el motivo de abstención.

## Dominios temporales y hechos

Cada rol `self` y `ally` aporta ocho hechos: tiempo restante y estado del plazo para ataque primario, secundario, aplazamiento de disparo y servicio de armas del pawn. Los valores nativos absolutos permanecen en la fuente y sus referencias; no se exportan como features iniciales.

**Servicio de armas del pawn.** `m_flNextAttack` debe estar presente, ser flotante finito positivo y estar declarado `GameTime_t`. Se reutiliza la prueba causal de correspondencia de relojes 3E.6, dentro de su instancia, época, vida de rules, origen e intervalo vigentes, o su vía directa de simulación nativa acreditada. La pausa debe estar observada. Entonces:

`restante = max(0, deadline_nativo − ahora_nativo_acreditado)`.

El estado es `pending`, `elapsed` o `paused`. Son segundos del reloj nativo, no tiempo de pared ni lectura exacta del HUD. El cero nativo se retiene y provoca `native_pawn_zero_deadline_semantics_unaccredited`: no se presupone que signifique plazo vencido. Ausencia, tipo incompatible, valor fuera de rango y pérdida de la prueba de reloj conservan abstención. El estado `elapsed` sólo describe ese plazo y no identifica la causa de un bloqueo ni acredita capacidad de disparar.

**Ticks y fracciones del arma.** Se conservan los tres pares nativos con tipo y procedencia. Los enteros son los valores entregados por el SDK: su decoder por defecto lee `VarUint32`; no se presupone que ya sean deadlines `GameTick_t` semánticamente decodificados. La declaración `GameTick_t`, nombres de campos y ratios numéricamente plausibles no prueban por sí solos el codec de ese wrapper, una época compatible con red/pawn, la composición tick–fracción ni sus sentinels. Este contrato **no aplica `(tick + ratio) × intervalo`**, divisiones por un factor ajustado a los datos ni reinterpretaciones de signo, y no convierte esos pares en tiempo restante o readiness. Distingue ausencia, tipo no acreditado, cero/valor grande potencialmente sentinel y fracción fuera del intervalo candidato `[0, 1)`. Incluso un par que supera esas comprobaciones conserva `native_weapon_tick_fraction_clock_unaccredited`; el intervalo candidato no es una validación de semántica subtick.

Resolver esta dependencia requiere una fuente positiva de la build correspondiente que vincule los ticks de ataque al reloj utilizado, defina composición y actualización de fracciones, sentinels y aplicación por categoría de arma, con evidencia disponible en el prefijo. Una correlación con disparos posteriores o una fórmula de otra build no acredita esa prueba. El aplazamiento puede ser específico de una categoría; no se interpreta un valor de una clase distinta como restricción universal.

Para Knife, C4 y utilidad que el contrato físico clasifica sin munición de cargador, los seis hechos del arma son `not_applicable`, con valor nulo. No significan cero espera ni que ninguna acción sea posible. El servicio de armas del pawn conserva su comprobación independiente.

## Materialización y acceso inicial

`support_weapon_timing_observations.json` referencia por tamaño y SHA-256 las ocho fuentes anteriores pertinentes, incluida la física actual. El contexto conserva selección, IDs/T0, instancias preliminares, matriz por actor/aliado/instancia y testigos del reloj por separado. El lector recompone las capas previas y contrasta el informe completo.

`initial_features` recorta support, movimiento, físico, objetivo, tenencia, reloj y temporización a sus fronteras exactas de T0 antes de validar y derivar. El prefijo de eventos públicos usa su secuencia/orden. Identidades, handles, timestamps absolutos, enemigo oculto, desenlace, voz e intención no forman parte de las features.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-weapon-timing-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.weapon_timing `
  --source $timingSource --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

Añadir `--validate` al segundo comando para la relectura y el recálculo completo. El productor rechaza destinos no temporales, no vacíos y sobrescrituras antes del parser. El materializador exige TEMP vacío; los archivos se crean de forma exclusiva. El cierre requiere pruebas, la misma única demo aprobada, registros no vacíos, auditoría independiente, comparación de capas/canónico e inventarios retenidos.

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` siguen en `false`. No cambia `DecisionExample`, `DEC-TRADE-001`, elegibilidad ni recomendaciones. Contacto, entrada/compromiso, velocidad instantánea, precisión, readiness completa, cooperación, objetivo transportado, C4 caída, NAV físico y economía conservan sus dependencias.

La validación real, variantes, cifras y dependencia temporal pendiente se registran en el [cierre 3E.9](../../archive/ai-coach/AI_COACH_SUPPORT_WEAPON_TIMING_PHASE_2026-09-11.md) y su [evidencia](../../evidence/AI_COACH_SUPPORT_WEAPON_TIMING_SINGLE_DEMO_SMOKE_2026-09-11.json).
