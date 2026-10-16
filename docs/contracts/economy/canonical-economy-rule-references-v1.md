# Referencias históricas y comprobación económica

Contrato `1.0.0-candidate.1`, 6 de septiembre de 2026. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) fija estado y siguiente acción. Esta extensión consume las [fuentes nativas](canonical-economy-native-sources-v1.md) y su origen económico v2. No modifica sus contratos ni habilita recomendaciones.

## Qué contiene

El [registro de fuentes](../../../ai_coach/economy/rule_source_registry.json) fija seis archivos por URL, revisión Git, ruta, tamaño y SHA-256: identidad `steam.inf`, definiciones `weapons.vdata` y configuración `gamemode_competitive.cfg`, para dos compilaciones históricas. No descarga archivos al leer un paquete. El constructor exige las copias locales con los hashes registrados.

Las revisiones [fd04856b](https://github.com/SteamTracking/GameTracking-CS2/commit/fd04856b8668f11cb31ed2b22574607dd9c365e0) y [acfe24d5](https://github.com/SteamTracking/GameTracking-CS2/commit/acfe24d588d2df0a26da0f964e44d780bd3070eb) conservan archivos extraídos del juego. Sus `steam.inf` identifican respectivamente cliente/servidor `2000888` y `2000891`, con distintas revisiones fuente y la misma etiqueta de patch `1.41.7.7`. Son identidades distintas; el número nativo de build, la versión cliente/servidor y la revisión fuente no son intercambiables.

Cada definición numérica de objeto conserva su nombre, precio y premio por baja explícitos en `weapons.vdata`, con puntero al campo. Ausencia no significa cero; cero sólo se conserva si está escrito. Se comprueban tipos, duplicados, identidad numérica y cobertura. No se heredan campos, no se ejecuta código y no se deduce que un objeto con precio cero esté disponible para comprar.

La configuración conserva valores numéricos de economía y su línea en el archivo. Los importes negativos de penalización se mantienen. No se ejecutan instrucciones `exec`, no se resuelven overrides y no se finge que se ha leído toda la configuración efectiva del servidor. Una tabla de premios tampoco prueba el bonus previo de un equipo ni implementa por sí sola el cálculo completo de recompensas.

## Vínculo con la demo

El comprobador mantiene separadas identidad nativa observada, identidad de los archivos y compatibilidad. Eliminar puntos de la etiqueta del patch sólo localiza **candidatos para el diagnóstico**; no acredita equivalencia con el build de la demo. Si falta patch, hay conflicto de versiones o no hay candidato, no se usa el protocolo como sustituto. Si dos referencias discrepan, no se elige la que coincida con el coste observado.

La coincidencia de hashes entre las copias textuales registradas acredita igualdad de esos bytes publicados. El proceso del tracker incluye descompilación VPK: esos hashes no sustituyen a los de contenedores o recursos compilados originales. Tampoco prueban todos los cambios intermedios, todos los archivos económicos ni la configuración de una partida. La [dependencia documentada en 3D.2E](../../ai-coach/AI_COACH_ECONOMY_EXACT_ASSET_DEPENDENCY.md) exige separar origen de distribución, bytes originales y transformación reproducible. Este contrato conserva `build_binding=unverified`, `compatible_for_comparison=false` y cero paquetes de reglas aprobados. No tiene un mecanismo implícito para cambiar estos estados a partir de coincidencias de precios.

## Comparaciones permitidas

- **Costes de entradas nativas:** compara cada entrada completa y conectada con el precio de referencia unánime de los candidatos. Conserva identidad de observación, índice de entrada, actor, instancia y tick. Distingue coincidencia, diferencia, ausencia y conflicto. No cuenta compras únicas ni interpreta una diferencia como error del jugador. Una compra de casco con armadura conservada puede tener un coste distinto del precio de referencia del objeto completo; el informe no inventa su explicación ni reglas de reembolso.
- **Configuración inicial:** conserva únicamente valores nativos disponibles al inicio exacto de cada instancia. Distingue propiedad no capturada por el contrato de origen, valor nativo ausente y dato posterior al inicio. Los defaults no rellenan esos huecos. Una diferencia describe configuración; no es un consejo ni un error.
- **Cobertura:** las capturas desconectadas o con listas incompletas quedan contadas expresamente. Los contrastes de precio pertenecen a ejecución y nunca completan las features de T0. Los resultados posteriores y la economía rival no participan en la comparación de defaults iniciales.

`effective_shop_price=null`, `player_error=false`, `default_applied_to_demo=false`, `comparison_allowed=false`, `training_allowed=false` y `product_supported=false` son obligatorios. La validación independiente vuelve a leer campos y recalcular correspondencias sin llamar a los constructores. Una igualdad con una referencia no constituye aprobación prescriptiva.

## Artefactos y uso

| Artefacto | Schema |
|---|---|
| Registro de fuentes fijadas | [economy_rule_source_registry@1](../../../ai_coach/contracts/economy_rule_source_registry.schema.json) |
| `economy_rule_package.json` | [economy_rule_package@1](../../../ai_coach/contracts/economy_rule_package.schema.json) |
| `economy_rule_assessment.json` | [economy_rule_assessment@1](../../../ai_coach/contracts/economy_rule_assessment.schema.json) |

Registro runtime: **46 schemas**, 19 motivos. Catálogo `1.7.0`, IDs y elegibilidad conservados. Export `3.8.0`, parser `v16`, quality `12` y manifest base `@3` no cambian. El paquete vincula el registro y las fuentes; el informe vincula por SHA-256 paquete, registro, datos nativos, manifest nativo y manifest v2. Sus dos destinos deben existir, ser nuevos, vacíos y estar bajo TEMP; los prechecks ocurren antes de cargar entradas.

Preparar `$ruleSources` con los seis archivos del registro, usando el nombre `file` y contenido de su URL inmutable. Pueden obtenerse en un directorio TEMP dedicado; el lector rechazará un solo byte distinto. No guardar esos archivos como datos runtime del proyecto. Desde la raíz, con `$nativeManifest` generado por el selector económico `3`:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.rule_assets --sources $ruleSources --output $rulePackageOutput
backend/venv/Scripts/python.exe -B -m ai_coach.economy.rule_assessment --native $nativeManifest --package "$rulePackageOutput/economy_rule_package.json" --sources $ruleSources --output $ruleAssessmentOutput
```

El materializador del informe exige registros afectados no vacíos. La evidencia del smoke registra separadamente coincidencias, diferencias, ausencias, tipos de fuente, cobertura y bloqueos; un código cero o una lista no vacía no acreditan reglas compatibles. Las salidas del smoke no se promueven ni sustituyen los 44 bundles.
