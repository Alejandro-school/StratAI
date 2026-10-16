# Tenencia causal de C4 propia o aliada v1

Contrato candidato `1.0.0-candidate.1`; schemas `stratai.support_c4_observations@1` y `stratai.support_c4_context@1`. Extiende el [contexto del objetivo](canonical-support-objective-v1.md) en los mismos [anclajes](canonical-support-anchors-v1.md). El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) registra cobertura, estado y siguiente acción.

## Captura y procedencia

El selector optativo `--ai-coach-support-c4-candidate` ejecuta un único parseo y produce también los sidecars de anclajes, contexto físico y objetivo. Los contratos previos, IDs, T0, selección e instancias preliminares se conservan. No modifica `DecisionExample`, `DEC-TRADE-001` ni el bundle canónico `3.8.0`.

El productor lee las propiedades nativas después de la muestra support y del corte del objetivo en el mismo callback `FrameDone`. Cada corte conserva secuencia support, frame, tick, instancia, época, orden exacto del corte del objetivo y referencia al estado. Se comparte almacenamiento sólo entre payloads recién leídos e idénticos, dentro de una misma instancia/época; `first_cut` impide referenciar un estado antes de su primera captura. Un cambio de frame dentro del mismo tick puede tener otro propietario. El corte acredita disponibilidad de réplica, no el subtick físico de la transferencia.

Fuentes contrastadas con `demoinfocs-golang/v5@v5.2.0`:

| Evidencia | Fuente y límite |
|---|---|
| Inventario actual | `m_pWeaponServices.m_hMyWeapons`, forma nativa `[]any`; longitud y propiedades individuales `.0000`–`.0063`. No se usa el mapa `Player.Inventory` ni `Player.Weapons()`. |
| Objeto actual | `GameState.Entities()`, índice y serie completos del handle de 32 bits; clase de servidor, vida asignada en creación y callback de destrucción. Una nueva vida no hereda propiedad. |
| C4 | Concordancia de clase nativa `CC4` y `m_iItemDefinitionIndex=49`, contrastada con `bindBomb` y `EquipmentIndexMapping`. La definición identifica tipo, no una compra ni una instancia. No se equiparan todos los valores SDK `EqBomb` con C4. |
| Propietario | `m_hOwnerEntity` recién leído, resolución de handle completo en entidades actuales y vida de pawn del mismo tracker support. No se usa `Equipment.Owner`, `GameState.Bomb().Carrier`, último portador ni `hasC4` de presentación. |
| Actor | Alias opaco enlazado al jugador support exacto, conectado y vivo, con identidad, vida de pawn y posición observadas; se comprueba que el pawn sigue siendo el objeto actual del mapa nativo. |

El handle cero es un valor comprobable; no se convierte en ausencia. Índice `16383` significa sentinel inválido, incluido `0xffffffff`. Tipo no admitido, ausencia, serie incorrecta, entidad no disponible, vida/clase no acreditada y destrucción son estados distintos. La definición conserva presencia y tipo nativo; cero o definición ausente no acreditan un objeto distinto de C4. Una discrepancia clase/definición conserva conflicto.

Se retienen los objetos referenciados por inventarios utilizables y los objetos nativos `CC4` vivos para contrastar reclamaciones de propiedad. No se captura ninguna coordenada global de bomba, sitio, intención o comunicación. Los handles y vidas sólo son evidencia de enlace; no son features.

## Semántica de tenencia

`initial.self.has_c4` y `initial.ally.has_c4` son hechos en el corte exacto:

| Resultado | Prueba requerida |
|---|---|
| `observed: true` | Una C4 en un slot resuelto, clase/definición concordantes y propietario actual con handle y vida del pawn coincidentes, sin contradicción observada. Otra ranura parcial no invalida esa prueba positiva. |
| `observed: false` | Forma de inventario observada y dentro del límite, todas sus ranuras acreditadas como sentinel o entidad resuelta de otro tipo, sin duplicados ni reclamación C4 contradictoria del propietario. Una longitud nativa cero puede acreditar inventario vacío. Significa que ese jugador no porta C4; no que no exista una bomba en la partida. |
| `unavailable` | Pawn no utilizable, corte exacto ausente, cobertura parcial, C4 sin corroboración bilateral o conflicto. Un único canal sin el otro no se convierte en prueba positiva. |

Duplicados de slot, múltiples C4 reclamadas, propietario distinto del inventario, sentinel de dueño contra una C4 inventariada o definición/clase contradictorias conservan `inventory_owner_or_type_conflict`. Una reclamación nativa C4 excluida por un inventario completo también conserva conflicto. Un objeto `CC4` de definición ausente reclamado por el pawn impide declarar ausencia. No se corrige una fuente usando un corte posterior ni se elige una de dos fuentes contradictorias.

El lector inicial usa únicamente el prefijo hasta T0. El estado del corte anterior no rellena un corte exacto ausente. Muerte, desconexión, vida nueva, instancia y época se comprueban de nuevo. La conclusión de tenencia no depende de quién ganará la ronda, del movimiento futuro, de la economía rival ni de la posición de un enemigo oculto.

## Ubicación y alcance legal

Con tenencia positiva, `carrier_player_position` enlaza la posición exacta del jugador propio o aliado ya verificada por support. Con ausencia acreditada es `not_applicable:not_a_carrier`; con tenencia insuficiente es `unavailable:ownership_unavailable`. La posición aliada conserva el significado de proxy de información del equipo establecido por support, no acredita atención, comunicación ni conocimiento subjetivo.

`physical_c4_position` permanece `unavailable:not_observed_in_legal_view`. La posición del portador no es una medición del objeto físico, de su punto de anclaje, de la bomba caída ni de la bomba plantada. Esta entrega no interpreta coordenadas de `GameState.Bomb().Position()` ni decide un sitio de planta.

## Artefactos y verificaciones

`support_c4_observations.json` referencia por nombre, tamaño y SHA-256 los sidecars support y objetivo. `support_c4_context.json` añade referencias a anclajes y contextos físico, NAV y objetivo. Ambos conservan alias de demo, SHA-256 de la fuente y build del productor. Los schemas son estrictos y el lector comprueba además integridad de fuentes, identidades, relojes, frontera de estados y consistencia de ranuras. La relectura recalcula los contextos anteriores y el informe nuevo.

La matriz por actor/aliado/instancia conserva denominadores, intervalo competitivo separado, conteos de tenencia positiva/negativa/insuficiente y estados/motivos de los seis hechos. Los anclajes preliminares no se promueven por tener C4 acreditada.

```powershell
& $supportBinary --demo $approvedDemo --output $newEmptyTemp `
  --alias $opaqueAlias --checksum $demoSha256 --maps-root ../data/maps `
  --build-id $buildId --ai-coach-support-c4-candidate

backend/venv/Scripts/python.exe -B -m ai_coach.support.c4 `
  --source $c4Source --output $newEmptyContextTemp `
  --evaluator $navigationBinary --maps-root backend/data/maps
```

Añadir `--validate` al segundo comando relee la salida. El productor y materializador rechazan destinos no temporales, no vacíos y sobrescrituras antes de procesar. Fuente, bundles y temporales anteriores son de sólo lectura; las nuevas salidas se retienen inventariadas. La validación real usa una única demo aprobada después de las pruebas; si requiere corrección se repite la misma partida en otro TEMP.

## Límites

`comparison_allowed`, `training_allowed`, `product_supported` y `decision_eligible` permanecen en `false`; cero decisiones elegibles. Tenencia y posición del portador no acreditan responsabilidad táctica, apoyo viable, ruta física, LOS/contacto, superioridad de una alternativa, cuenta atrás efectiva ni progreso de un intento. Las limitaciones de contratos anteriores se conservan como alcance de aquellas capas; esta extensión añade sólo los hechos definidos aquí.

La [evidencia de la fase](../../evidence/AI_COACH_SUPPORT_C4_SINGLE_DEMO_SMOKE_2026-09-09.json) y el [informe de cierre](../../archive/ai-coach/AI_COACH_SUPPORT_C4_PHASE_2026-09-09.md) distinguen variantes reales y fixtures. El cierre factual no cierra la capacidad completa S2–S3.
