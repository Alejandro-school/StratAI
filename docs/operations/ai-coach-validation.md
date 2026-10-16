# Validación real aislada del AI Coach

Para [entradas visuales C0](../contracts/support/canonical-support-preparation-visual-contact-v1.md), usar `--ai-coach-support-visual-contact-candidate` y `ai_coach.support.preparation_visual_contact`, con `--require-observed`, `--visual-evaluator`, `--maps-root` y relectura `--validate`. Productor/materializador/lector, recálculo independiente de cinco hechos/referencias/matriz y segmentos con assets identificados; el motor compartido no acredita suficiencia visual. Preservar ocho contextos y 72 artefactos canónicos; una demo tras suites en TEMP nuevo/vacío retenido.

Para [temporización nativa C0](../contracts/support/canonical-support-preparation-weapon-timing-v1.md), usar `--ai-coach-support-weapon-timing-candidate` y `ai_coach.support.preparation_weapon_timing`, con `--require-observed` y relectura `--validate`. Recalcular reloj/plazos/estados/propiedad/físico actual/abstenciones/matriz, preservando siete contextos y 72 artefactos canónicos. Los positivos del pawn no resuelven ticks/fracciones. Una demo tras suites a TEMP nuevo/vacío retenido.

Para [movimiento nativo C0](../contracts/support/canonical-support-preparation-native-motion-v1.md), usar `--ai-coach-support-native-motion-candidate` y `ai_coach.support.preparation_native_motion`, con `--require-observed` y relectura `--validate`. Recalcular entregas/vidas/referencias/flags/códigos/abstenciones/matriz independientemente, preservando seis contextos C0 y 72 artefactos canónicos. Velocidad ausente no impide el gate de positivos de suelo/modo. Una demo tras suites a TEMP nuevo/vacío retenido.

Para [posición legal del objetivo C0](../contracts/support/canonical-support-preparation-objective-position-v1.md), usar `--ai-coach-support-objective-position-candidate` y `ai_coach.support.preparation_objective_position`, con `--require-observed` y relectura `--validate`. Recalcular geometría/continuidad/disponibilidad/referencias/abstenciones/matriz por vía independiente; conservar cinco contextos C0 y 72 artefactos canónicos. Una demo después de suites a TEMP nuevo/vacío retenido.

Para [origen/progreso de plantado C0](../contracts/support/canonical-support-preparation-attempt-origin-v1.md), usar `--ai-coach-support-attempt-origin-candidate` y `ai_coach.support.preparation_attempt_origin`, con `--require-observed` y relectura `--validate`. Exigir transición/deadline/origen causal y continuidad nativa/C0, recálculo independiente de hechos/abstenciones/referencias/matriz, conservación C0 físico/objetivo/C4/reloj y 72 artefactos canónicos. Una demo después de suites en TEMP nuevo/vacío retenido, separada por fase.

Para [relojes del objetivo C0](../contracts/support/canonical-support-preparation-objective-clock-v1.md), usar `--ai-coach-support-objective-clock-candidate` y `ai_coach.support.preparation_objective_clock`, con `--require-observed` y relectura `--validate`. Exigir origen/tipos/dominio/pausas y entrega exactos, recálculo independiente de valores/abstenciones/matriz, conservación C0 físico/objetivo/C4 y 72 artefactos canónicos. Una demo después de suites en TEMP nuevo/vacío retenido.

Para [tenencia C4 C0 v1](../contracts/support/canonical-support-preparation-c4-v1.md), procesar la misma única demo con `--ai-coach-support-c4-candidate` y materializar con `ai_coach.support.preparation_c4 --source <support_c4_observations.json> --output <TEMP nuevo/vacío> --require-observed`; releer con `--validate`. Exigir corte/estado/objetivo y vida/handle exactos, inventario completo para ausencia y corroboración bilateral para presencia. Recalcular tenencia, posición actual del portador y matriz independientemente; conservar C0 físico/objetivo, candidatos/IDs/T0/exclusiones/ejecución y 72 artefactos canónicos. Separar C4 física, variantes reales/fixtures y retener salidas/inventarios. No acreditar valor táctico ni añadir relojes o progreso de intentos.

Para el [contexto causal del objetivo C0 v1](../contracts/support/canonical-support-preparation-objective-v1.md), procesar la misma única demo con `--ai-coach-support-objective-candidate` y materializar mediante `ai_coach.support.preparation_objective --source <support_objective_observations.json> --output <TEMP nuevo/vacío> --require-observed`; releer con `--validate`. Comprobar entrega/prefijo público y vida/handle/instancia/época exactos, también con otro FrameDone o evento posterior en el mismo tick. Recalcular hechos y matriz independientemente; conservar todos los candidatos/IDs/T0/exclusiones/ejecución, contexto físico, rutas reutilizadas y 72 artefactos canónicos. Separar variantes reales y fixtures y retener salidas/inventarios. No incorporar capas posteriores ni acreditar valor táctico de mantener.

El [contexto físico C0 v1](../contracts/support/canonical-support-preparation-physical-v1.md) reutiliza `--ai-coach-support-physical-candidate` sobre la misma única demo aprobada después de las suites. Ejecutar `python -B -m ai_coach.support.preparation_physical --source <support_physical_observations.json> --output <TEMP-nuevo-vacío> --require-observed` y releer con `--validate`. Comprobar todos los cortes y pares C0, identidad exacta, estados físicos y ausencias, velocidad media sólo con historia continua, matriz independiente, futuro malformado fuera del prefijo, replay y 72 artefactos canónicos. Conservar el censo y los contextos anteriores; retener salidas con inventario. No acredita readiness, entrada física o apoyo viable.

El [censo prospectivo C0 v1](../contracts/support/canonical-support-preparation-v1.md) se materializa con `python -B -m ai_coach.support.preparation --source <support_observations.json> --output <TEMP-nuevo-vacío> --require-observed`; el mismo comando con `--validate` lee y recomputa la salida existente. Reutiliza la captura `--ai-coach-support-candidate` y exige el gate de la misma única demo aprobada después de las suites. Sus [31 casos congelados](../references/support-preparation-selection-cases-v1.json) y las pruebas runtime mantienen separación de selección/ejecución, identidad, prefijos y denominadores. C0 no acredita entrada física ni compromiso.

Para las [entradas visuales en el corte support v1](../canonical-support-visual-contact-v1.md), usar `--ai-coach-support-visual-contact-candidate` y `ai_coach.support.visual_contact --source $visualSource --output $newEmptyContextTemp --evaluator $navigationBinary --visual-evaluator $visualCheckBinary --maps-root backend/data/maps`, con TEMP nuevo/vacío y relectura completa `--validate`. Compilar el verificador de sólo lectura desde `backend/go-service/cmd/support-visual-check`; con `--source` y `--maps-root` recompone todos los rayos sin parsear la demo. Contrastar entregas, vida/serie, coordenadas actuales, mismo corte, tipos/ceros, FOV y referencias de geometría; separar diagnóstico de malla/FOV de contacto legal. Humo/flash/oclusión insuficientes conservan abstención, incluso con rayo libre o flash nativo cero. Verificar nueve fuentes y once contextos anteriores, 72 artefactos canónicos, matrices y acceso inicial sin futuro/identidad/geometría enemiga; distinguir variantes reales y fixtures. Retener salidas, helpers, binarios, logs e inventarios.

Para la [temporización nativa del arma v1](../contracts/support/canonical-support-weapon-timing-v1.md), usar `--ai-coach-support-weapon-timing-candidate` y `ai_coach.support.weapon_timing`, con TEMP nuevo/vacío y relectura completa `--validate`. Comprobar identidad actual de pawn, arma y propietario por vida/serie, contexto físico del mismo corte, entregas y referencias retenidas. Recalcular el plazo GameTime del pawn sólo con reloj y pausa acreditados; ticks/fracciones conservan abstención sin prueba del codec/dominio. Contrastar ocho fuentes y diez contextos anteriores, los 72 artefactos canónicos, la matriz y el acceso inicial sin futuro/identidad/enemigo oculto. Retener salidas, helpers, binarios, logs e inventarios.

Para el [movimiento nativo v1](../contracts/support/canonical-support-native-motion-v1.md), usar `--ai-coach-support-native-motion-candidate` y `ai_coach.support.native_motion`, con TEMP nuevo/vacío y relectura completa `--validate`. Comprobar declaración, presencia, variantes de vectores, identidad actual de pawn/suelo/parent y el corte común; reconstruir cada estado desde sus entregas. Corroborar contacto con suelo mediante flags y referencia actual, conservando la distinción mundo/entidad/ausencia. Derivar magnitudes sólo de un vector actual y sistema espacial acreditados; velocidad base, histórica o media entre posiciones no completan una fuente ausente. Contrastar las siete fuentes y nueve contextos anteriores, canónico, matrices y acceso inicial sin futuro ni información ajena. Retener salida, binarios, logs e inventarios y distinguir cobertura real de fixtures.

Para la [posición nativa del objetivo v1](../contracts/support/canonical-support-objective-position-v1.md), usar `--ai-coach-support-objective-position-candidate` y `ai_coach.support.objective_position`, con TEMP nuevo/vacío y relectura `--validate`. Recalcular celdas/origen y acreditar por separado parent/sistema mundial y disponibilidad del rol; comprobar creación, estado retenido, actualizaciones, destrucción, vida y orden exacto. Tenencia actual, contacto nativo de desactivación y plantado propio corroborado tienen pruebas distintas. Ni portador pasado, evento global o spotted autorizan coordenadas exactas. Contrastar las seis fuentes anteriores, todos los contextos 3E.1–3E.6B, matrices y canónico; retener cada intento con hashes, incluidas ejecuciones fallidas, y separar variantes reales de fixtures.

Para el [origen nativo del plantado v1](../contracts/support/canonical-support-attempt-origin-v1.md), usar `--ai-coach-support-attempt-origin-candidate` y `ai_coach.support.attempt_origin`, con destino TEMP nuevo/vacío y relectura `--validate`. Contrastar creación y emisiones iniciales, transición posterior, asignación de deadline y todos los frames activos; verificar ticks nativos, dominio, vidas, propietario/inventario, pausas y continuidad. Recalcular duración y fracción sin default ni primer sample, distinguir resolución de tick replicado y desenlace, preservar 3E.1–3E.6 y retener inventarios/hashes.

Para los [relojes causales del objetivo v1](../contracts/support/canonical-support-objective-clock-v1.md), usar `--ai-coach-support-objective-clock-candidate` y `ai_coach.support.objective_clock`, con salida TEMP nueva/vacía y relectura `--validate`. Verificar tipos declarados, ambos ticks nativos e intervalo, prueba causal de origen, pausas, vidas y referencias; recalcular plazos/progreso/margen desde fuentes nativas. Una duración ausente no se rellena por default ni con el primer sample. Contrastar todas las capas 3E.1–3E.5, matrices y canónico completo. Retener cada salida e inventario.

Para la [tenencia causal de C4 v1](../contracts/support/canonical-support-c4-v1.md), usar `--ai-coach-support-c4-candidate` y `ai_coach.support.c4` con `--source`, `--output` TEMP nuevo/vacío, `--evaluator` y `--maps-root`; releer con `--validate`. Comprobar inventario nativo y cada slot, handles/series/vidas actuales, clase `CC4` y definición `49`, propietario bilateral, cobertura para ausencia y conflictos explícitos. Recalcular tenencia y posición del portador desde las muestras exactas; no acreditar coordenadas físicas de C4 ni información enemiga. Contrastar observaciones y hechos 3E.1–3E.4, matrices y canónico completo. Retener salidas e inventarios.

Para el [contexto causal del objetivo v1](../contracts/support/canonical-support-objective-v1.md), usar `--ai-coach-support-objective-candidate` y `ai_coach.support.objective` con `--source`, `--output` TEMP nuevo/vacío, `--evaluator` y `--maps-root`; releer con `--validate`. Comprobar reglas Source 2 bajo `m_pGameRules`, tipos/presencia nativos, mismo corte y vida de pawn, y frontera de eventos públicos por orden de entrega. Recalcular fases y tiempos transcurridos sin defaults ni futuro del mismo tick. Contrastar contextos 3E.1–3E.3 y canónico completo. Mantener separados estado público, hechos propios/aliados, datos ausentes y tenencia/ubicación aún no implementadas; conservar TEMP e intentos rechazados con inventario.

Para [contexto NAV de apoyo v1](../contracts/support/canonical-support-navigation-v1.md), procesar la misma demo con `--ai-coach-support-physical-candidate`, compilar `cmd/support-navigation` en TEMP y ejecutar `ai_coach.support.navigation` con `--source`, `--output` TEMP nuevo/vacío, `--evaluator` y `--maps-root`. Releer con los mismos parámetros y `--validate`. Conservar artefactos 3E.1–3E.2, fuente/hash/vidas/T0 y matriz. Exigir reconstrucción del grafo desde el NAV local y recálculo independiente de asociaciones, caminos dirigidos, pesos y polilíneas; contrastar nativamente áreas/enlaces y documentar partes omitidas. Diferenciar candidato conectado, sin camino en el subgrafo y fuera de cobertura de la conexión física no acreditada. No inferir ausencia física, seguridad, ETA ni apoyo viable; retener TEMP con inventario y canónico completo.

La [tenencia observada v1](../contracts/economy/canonical-economy-ownership-v1.md) usa `--economy-candidate-version 9` y captura una lectura fresca al final de cada fotograma seleccionado del freeze. Después del único parse, ejecuta `backend/venv/Scripts/python.exe -B -m ai_coach.economy.ownership --source $ownershipSource --output $newEmptyTemp --require-observed`. Comprueba la cadena de hashes, todos los cortes y estados, serie y vida, longitud activa, conflictos y transiciones; contrasta el informe completo con un recálculo independiente. Las apariciones o desapariciones de tenencia no acreditan compras, drops voluntarios o devoluciones. Los controles anteriores y el canónico completo siguen siendo obligatorios.

Para [entradas y objetos v1](../contracts/economy/canonical-economy-entry-identity-v1.md), usar selector **10** y `ai_coach.economy.entry_identity --require-observed`. Verificar cobertura exacta de cortes/pawns, ordinales y duplicados, tipos/presencia, serie completa y vida, definición, vínculo con tenencia y matriz por oportunidad. Recalcular todos los registros de entrada/estado por un método independiente; coste y vínculo no acreditan transacción. Conservar controles 3D.1–3D.6 y canónico completo. Por indicación del usuario de 2026-09-08, retener TEMP inventariado sin intentar borrado ni pedir revisión de limpieza.

Para [contexto físico de apoyo v1](../contracts/support/canonical-support-physical-v1.md), usar `--ai-coach-support-physical-candidate` y `backend/venv/Scripts/python.exe -B -m ai_coach.support.physical --source $physicalSource --output $newEmptyTemp --require-observed`. Conservar observaciones/anclajes/matriz v1 y hashes del canónico. Recalcular velocidad media desde posiciones/ticks, munición desde valores nativos y cobertura conjunta; verificar handle completo, vida, señal original de recarga y disponibilidad actual. Contrastar ángulos float32, walking, ducking y arma con replay. No confundir componentes disponibles con readiness total o apoyo viable, ni promover instancias preliminares. Exigir controles de futuro/oracle, ausencias/sentinels/Knife/utilidad, canónico completo y recálculo independiente. Retener TEMP inventariado.

Para [anclajes de apoyo v1](../contracts/support/canonical-support-anchors-v1.md), usar `--ai-coach-support-candidate` y `backend/venv/Scripts/python.exe -B -m ai_coach.support.anchors --source $supportSource --output $newEmptyTemp --require-observed`. Recalcular todos los pares, selección, distancias, historia, matriz y ventanas posteriores; contrastar el acceso inicial exacto y los campos presentes con replay. Comprobar invariancia al futuro y enemigos, vidas/series, ceros frente a ausencia nativa, desconexión, cambios de lado/época, ticks repetidos y cadencia. Auditar casos sin muertes con eventos posteriores sólo después de seleccionar; diferenciar muestra no viva de kill. Exigir canónico completo y variantes reales separadas de fixtures. Una separación creciente no demuestra entrada ni apoyo viable. Retener TEMP inventariado.

Para [conciliación monetaria v1](../contracts/economy/canonical-economy-cashflow-v1.md), usar selector **11** y `backend/venv/Scripts/python.exe -B -m ai_coach.economy.cashflow --source $cashflowSource --output $newEmptyTemp --require-observed`. Comprobar lectura fresca común, prefijo de traza completo, `trace_next_frame = cut.frame + 1`, disponibilidad de controlador/pawn/actor en cada entrega, baselines y creación/reenvío. Recalcular por separado multiconjuntos, movimientos individuales, sumas positivas/negativas, netos y residuos; conservar compensaciones y ceros. Los intervalos omitidos por payload idéntico no acreditan ausencia de actividad cuando falta información. Exigir matriz de abstenciones, controles 3D.1–3D.7 y canónico completo. Igualdad contable no equivale a compra/refund efectivo. Retener TEMP con inventario y hashes.

La [identidad y relojes nativos v1](../contracts/economy/canonical-economy-native-identity-v1.md) usa `--economy-candidate-version 8` y añade `economy_native_identity.json`. Ejecuta su lector con `backend/venv/Scripts/python.exe -B -m ai_coach.economy.native_identity --source $identityFile --output $newEmptyTemp --require-observed`. Valida la cadena completa de hashes, vidas y series de entidad, secuencia causal de reloj y fronteras competitivas. El selector ordena explícitamente los consumidores del proyecto; no garantiza el orden interno del SDK ni el instante de conocimiento del jugador. Registra ceros de mensajes observados, callbacks de creación ambiguos, referencias sin resolver y cambios de orden respecto a controles históricos. No interpreta esos datos como transacciones o recomendaciones.

La [traza económica v1](../contracts/economy/canonical-economy-transaction-trace-v1.md) usa el selector optativo `--economy-candidate-version 7` junto con `--ai-coach-economy-candidate`. Añade `economy_transaction_trace.json` conservando las salidas de los selectores anteriores. Después del único parse, ejecuta `backend/venv/Scripts/python.exe -B -m ai_coach.economy.transaction_trace --source $traceFile --output $newEmptyTemp --require-observed`. El lector comprueba procedencia y cobertura; una diferencia de saldo o un aviso de refund del SDK no se convierte en una transacción acreditada. La validación canónica completa y los controles anteriores siguen siendo necesarios; código de salida cero no cierra el gate.

Guía operativa conservada del README. Los comandos se ejecutan desde la raíz del repositorio y contienen parámetros que deben completarse con la demo aprobada. El [plan de implementación](../AI_COACH_IMPLEMENTATION_PLAN.md) determina qué trabajo está vigente y autorizado; esta guía no habilita por sí sola ninguna operación.

Para la materialización espacial combinada, consulta además el [contrato candidato 3.9](../canonical-spatial-candidate-v3.9.md). Para la captura económica independiente, usa `--ai-coach-economy-candidate` y el [contrato económico v1](../contracts/economy/canonical-economy-candidate-v1.md), que incluye su validador. Ese modo no genera ejemplos ni requiere productores espaciales. Los modos siguientes son procedimientos por productor: cada cierre ejecuta el modo adecuado sobre una única partida, sin lanzar todos los ejemplos indiscriminadamente.

Los cambios de parser, exportadores o hechos derivados no se cierran sólo con fixtures. Después de los tests se usa una única demo aprobada, siempre como entrada de sólo lectura, y un directorio nuevo y vacío bajo `%TEMP%`. El modo `--ai-coach-smoke` lee la demo una sola vez y, a partir de ese mismo replay, genera el bundle canónico temporal y los informes candidatos de navegación/ETA y contacto visual:

```powershell
$runRoot = Join-Path ([IO.Path]::GetTempPath()) "stratai-one-demo-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $runRoot -ErrorAction Stop | Out-Null

Push-Location backend\go-service
try {
  go run ./cmd/canonical-export `
    --demo <demo-aprobada.dem> `
    --output $runRoot `
    --alias demo-<primeros-20-caracteres-del-sha256> `
    --checksum <sha256-completo> `
    --maps-root ..\data\maps `
    --build-id <id-del-working-tree> `
    --ai-coach-smoke `
    --navigation-smoke-queries 64 `
    --visual-contact-smoke-frames 256
} finally {
  Pop-Location
}

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_navigation_smoke `
  "$runRoot\smoke_run_<alias>\observed_navigation_smoke.json"

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_visual_contact_smoke `
  --maps-root backend\data\maps `
  "$runRoot\smoke_run_<alias>\observed_visual_contact_smoke.json"

$bundle = "$runRoot\smoke_run_<alias>\match_<alias>"
$scriptsRoot = (Resolve-Path backend\go-service\scripts).Path
$env:PYTHONDONTWRITEBYTECODE = '1'
backend\venv\Scripts\python.exe -c "import sys; from pathlib import Path; sys.path.insert(0,sys.argv[1]); from golden_demo_semantic_runner import validate_direct_bundle; errors=validate_direct_bundle(Path(sys.argv[2]),sys.argv[3],sys.argv[4]); print(len(errors)); print(errors); raise SystemExit(0 if not errors else 1)" `
  $scriptsRoot $bundle <alias> <sha256-completo>

backend\venv\Scripts\python.exe -c "import sys; from pathlib import Path; from ai_coach.datasets.build_examples import build_direct_bundle_audit_summary; report=build_direct_bundle_audit_summary(Path(sys.argv[1]),expected_match_id=sys.argv[2],expected_demo_checksum=sys.argv[3]); print(report['status'],report['decision_examples']['total'],report['joins'])" `
  $bundle <alias> <sha256-completo>
```

Para recorrer la partida entera se sustituye `--ai-coach-smoke` por `--ai-coach-full-demo-validation`. Ese modo conserva los dos smokes rápidos y añade los reportes y streams gzip completos de navegación, contacto visual y estado de arma. Los tres candidatos siguen fuera del bundle canónico. Después se validan todos los streams completos:

```powershell
backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_full_demo `
  --maps-root backend\data\maps `
  "$runRoot\smoke_run_<alias>\observed_navigation_full_demo.json" `
  "$runRoot\smoke_run_<alias>\observed_visual_contact_full_demo.json"

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_weapon_state_full_demo `
  --expected-source-alias demo-<primeros-20-caracteres-del-sha256> `
  --expected-source-sha256 <sha256-completo> `
  --expected-rounds <rondas-esperadas> `
  "$runRoot\smoke_run_<alias>\observed_weapon_state_full_demo.json"
```

Los modos smoke/full rechazan una salida que no sea una carpeta vacía dedicada bajo `%TEMP%`, así como escapes, enlaces, junctions y sobrescrituras. Los validadores independientes recalculan los escenarios temporales con arma y ritmo reales; comprueban NAV, geometría, FOV, humo, episodios, estado de arma/ammo/reload, causalidad, hashes, abstenciones, privacidad y la prohibición de mostrar segundos exactos al jugador. Después se valida semánticamente el bundle completo y se construyen `DecisionExample`/`TaskEligibility` directamente desde ese bundle temporal, sin fingir que fue publicado. Ningún modo autoriza descargar, entrenar, publicar, promover ni reprocesar 44/44.

Para validar el candidato de exposure sin incorporarlo a `3.8.0`, usa el modo standalone con el registro versionado y después el validador independiente:

```powershell
Push-Location backend\go-service
try {
  go run ./cmd/canonical-export `
    --demo <demo-aprobada.dem> `
    --output <directorio-nuevo-bajo-TEMP> `
    --alias demo-<primeros-20-del-sha256> `
    --checksum <sha256-completo> `
    --maps-root ..\data\maps `
    --exposure-packages-root ..\..\ai_coach\spatial\exposure `
    --build-id <id-del-working-tree> `
    --ai-coach-exposure-angle-validation
} finally {
  Pop-Location
}

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_exposure_angle_full_demo `
  <observed_exposure_angle_full_demo.json> `
  --registry-root ai_coach\spatial\exposure `
  --maps-root backend\data\maps `
  --require-geometry-observed `
  --require-geometry-unavailable `
  --require-tactical-unavailable
```

Para validar cover objetivo sin incorporarlo a `3.8.0`, usa el modo standalone equivalente. El registro resuelve el paquete por mapa/versión/hash y el validador vuelve a calcular selección, postura, yaw y los nueve probes de cada dirección:

```powershell
Push-Location backend\go-service
try {
  go run ./cmd/canonical-export `
    --demo <demo-aprobada.dem> `
    --output <directorio-nuevo-bajo-TEMP> `
    --alias demo-<primeros-20-del-sha256> `
    --checksum <sha256-completo> `
    --maps-root ..\data\maps `
    --cover-packages-root ..\..\ai_coach\spatial\cover `
    --build-id <id-del-working-tree> `
    --ai-coach-cover-validation
} finally {
  Pop-Location
}

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_cover_full_demo `
  <observed_cover_full_demo.json> `
  --registry-root ai_coach\spatial\cover `
  --maps-root backend\data\maps `
  --expected-map-id de_mirage `
  --require-observed
```

Para validar viabilidad física de rutas de salida sin incorporarla a `3.8.0`, usa el modo standalone con el registro de exit-route. El validador recomputa package/assets, selección, destino, distancias y álgebra temporal; la valoración táctica permanece obligatoriamente abstendida:

```powershell
Push-Location backend\go-service
try {
  go run ./cmd/canonical-export `
    --demo <demo-aprobada.dem> `
    --output <directorio-nuevo-bajo-TEMP> `
    --alias demo-<primeros-20-del-sha256> `
    --checksum <sha256-completo> `
    --maps-root ..\data\maps `
    --exit-route-packages-root ..\..\ai_coach\spatial\exit_route `
    --build-id <id-del-working-tree> `
    --ai-coach-exit-route-validation
} finally {
  Pop-Location
}

backend\venv\Scripts\python.exe -m ai_coach.contracts.validate_exit_route_full_demo `
  <observed_exit_route_full_demo.json> `
  --registry-root ai_coach\spatial\exit_route `
  --maps-root backend\data\maps `
  --expected-source-alias demo-<primeros-20-del-sha256> `
  --expected-source-sha256 <sha256-completo> `
  --expected-map-id de_mirage `
  --require-observed
```

