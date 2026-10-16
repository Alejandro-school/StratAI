# Cierre: ETA contextual y validación real con una demo

Fecha: 28 de agosto de 2026  
Estado: cerrado en código y validado con una única demo aprobada  
Contrato candidato: `stratai.observed_navigation_path@2` (`2.0.0`)  
Evidencia: [AI_COACH_ETA_SINGLE_DEMO_SMOKE_2026-08-28.json](../../evidence/AI_COACH_ETA_SINGLE_DEMO_SMOKE_2026-08-28.json)

## Objetivo

Corregir dos problemas del primer cálculo de ETA:

1. no tratar `path_distance / 250` como si fuera el tiempo real de llegada;
2. demostrar el cambio procesando exactamente una demo real antes de continuar con otra fase.

ETA significa *estimated time of arrival*, «tiempo estimado de llegada». En StratAI deja de ser una cifra única. Los tiempos exactos quedan reservados para cálculos internos y no se pueden convertir directamente en un mensaje como «tardarás 15 segundos» o «por tanto debías guardar».

## Qué se ha cambiado

El contrato anterior `stratai.observed_navigation_path@1` queda sustituido antes de materializarse por `@2`. El nuevo resultado conserva reachability y distancia de camino, pero separa tres escenarios internos:

- referencia optimista continua a 250 unidades por segundo;
- referencia continua según la velocidad máxima sin mira del arma activa, con búsqueda cerrada: un arma desconocida queda `unavailable`;
- proyección del ritmo horizontal observado causalmente en `T0`.

El estado de caminar, agacharse y usar mira se conserva como contexto observado. No se han inventado multiplicadores para estas posturas: su efecto real ya aparece en la velocidad observada. La velocidad cero se conserva como observación, pero no produce infinito ni una llegada falsa. Los tres escenarios indican de forma explícita sus supuestos y ninguno se declara predicción de llegada.

El contrato obliga a:

- `exact_user_facing_time_allowed=false`;
- `arrival_prediction=false`;
- uso interno como restricción, no como recomendación;
- comunicación al jugador sólo cualitativa, condicional y con incertidumbre.

También se mejoró la asociación al NAV: primero se busca el polígono que contiene la posición y la capa vertical correcta; sólo después se usa un fallback acotado al centro más cercano.

## Defecto real encontrado por el gate

El primer intento con la demo real falló correctamente antes de publicar salida. El lector NAV v35 de CS2 interpretaba mal el formato:

- leía flags de 64 bits con tamaños equivocados;
- cambiaba sin motivo a big-endian desde el área 1;
- consumía cada conexión de 8 bytes (`AreaId` y `EdgeId`) como si tuviera 4;
- desalineaba los datos legacy y leía campos que no existen en v35.

Por eso terminaba creyendo que había 95.289 conexiones y alcanzaba EOF. Se corrigió el layout v31-v35 en little-endian, se separaron conexiones de áreas y ladders, se validaron límites, IDs e índices y v36 se rechaza de forma explícita. El NAV real de Mirage carga ahora:

- 3.613 vértices;
- 2.544 polígonos;
- 2.544 áreas;
- 6.381 conexiones/aristas válidas.

Este hallazgo demuestra la utilidad del nuevo gate: las pruebas sintéticas anteriores eran verdes, pero la integración real reveló un defecto que habría dejado ETA sin rutas en producción.

## Smoke de una única demo

Se utilizó un único match distinto en todos los intentos:

- alias opaco: `demo-a11eb0d600ee702c9edb`;
- SHA-256: `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`;
- tamaño: 330.945.844 bytes;
- mapa: Mirage.

Hubo dos intentos sobre ese mismo archivo. El primero falló por el lector NAV y no publicó ningún fichero. Después de corregirlo se repitió la misma demo, no se eligió otra. El segundo intento terminó en 91,484 segundos de pared y publicó sólo dentro de una carpeta nueva bajo `%TEMP%`.

El selector del smoke no se conforma con una sola arma rápida. Busca diversidad real y el validador exige cuchillo, otra velocidad de arma, caminar y movimiento sin walk. El resultado fue:

| Comprobación real | Resultado |
|---|---:|
| Consultas ETA | 64 |
| Rutas alcanzables | 64 |
| Baselines optimistas a 250 | 64 |
| Baselines de arma conocida | 60 |
| Proyecciones de ritmo observado | 51 |
| Contextos caminando | 13 |
| Contextos en movimiento sin walk | 39 |
| Contextos agachados | 4 |
| Contextos con mira | 2 |
| Contextos con cuchillo | 19 |
| Armas distintas con baseline | 15 |
| Velocidades máximas distintas | 7 |
| Violaciones de tiempo exacto para usuario | 0 |

Las cuatro ausencias de baseline de arma se explican como arma desconocida y trece ritmos como jugador parado. No se sustituyeron por 250 ni por infinito.

Los IDs reales de Steam no se guardan en el informe: se sustituyen por alias locales `participant-NNN`.

## Validación

Pasaron:

- tests Go de maps, navegación, parser y CLI;
- 32 tests del AI Coach y el validador de contratos;
- `npm run check:all`: 88 tests frontend, 14 Node correctos y 3 omitidos, todos los paquetes Go, 43 tests backend Python y build frontend;
- validador independiente del informe ETA, que recalcula `distancia / velocidad`, cruza arma con la tabla versionada, cruza ritmo con la velocidad causal y vuelve a calcular todos los conteos;
- `golden_demo_semantic_runner.validate_direct_bundle`: cero errores en el bundle temporal completo.

El informe ETA tiene SHA-256 `75256c80ba0c510be90c3f46cb68274e18b81fe6301a21a3b22626eafe26b2e8`. El bundle temporal contenía 73 ficheros y 735.286.361 bytes; su manifest canónico tenía SHA-256 `b6d56ef50e4ecd6b394634d46125f6ec7bb9fa126b4e04d8dbabff8beff0efb2`.

Después de guardar esos hashes y conteos se retiraron de `%TEMP%` las carpetas de ambos intentos. La eliminación recursiva directa fue bloqueada por la protección del entorno, por lo que se enviaron a la Papelera de reciclaje de Windows; ya no existen en su ubicación temporal y todavía son recuperables hasta vaciar la Papelera.

## Qué no se ha hecho

- No se entrenó ningún modelo.
- No se descargó ninguna demo.
- No se procesó ningún segundo match.
- No se reprocesaron las 44 demos.
- No se escribió en bundles aprobados, `backend/data/exports`, staging, failed, releases ni `CURRENT`.
- No se publicó ni promovió la salida temporal.
- El contrato candidato no se añadió silenciosamente al schema canónico `3.8.0`.

Antes y después se conservaron exactamente 100 ficheros fuente/13.589.310.951 bytes, 3.013 ficheros de bundles/27.264.064.316 bytes, 727 ficheros excluidos/5.245.908.800 bytes y staging 0/0. El manifest aprobado de la demo y su SHA-256 no cambiaron; el repositorio `Faceit-Demos` permaneció limpio.

## Nueva regla permanente

`AGENTS.md`, el plan, el contrato humano, README y el prompt sucesor exigen desde ahora un smoke real después de cada cierre funcional que afecte parsing, exportación o hechos derivados:

1. primero pasan los tests;
2. se procesa exactamente una demo aprobada en una carpeta temporal nueva;
3. si falla, se corrige y se repite esa misma demo;
4. se validan el productor afectado y el bundle, no sólo el código de salida;
5. se guarda evidencia pequeña y se elimina la salida grande;
6. esto no autoriza el reproceso 44/44, descargas, entrenamiento ni publicación.

Gate 1B sigue pendiente y `training_allowed=false`. La siguiente fase continúa siendo ammo, reserva y reload causales a 16 Hz.
