# Arquitectura de modelos del AI Coach

Estado: referencia técnica v1.0, 2026-08-27. No mantiene progreso ni autoriza entrenamiento. El estado y la siguiente acción viven únicamente en el [plan de implementación](AI_COACH_IMPLEMENTATION_PLAN.md). Esta referencia se apoya en el [contrato de capacidades y datos](AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) y en su [representación máquina](../ai_coach/contracts/capability_catalog.json).

## Objetivo

El sistema debe separar tres preguntas que no son equivalentes:

1. ¿Qué ocurrió y qué podía saber el jugador en T0?
2. ¿Qué acciones alternativas eran físicamente posibles?
3. ¿Hay evidencia suficiente para recomendar una de ellas?

La arquitectura evita un modelo monolítico. Usa proveedores de hechos, especialistas por familia, modelos transversales, un generador de candidatos, un ranking común, perfil histórico y una capa final de evidencia. El LLM sólo redacta el resultado estructurado.

## Flujo de inferencia

```text
bundle + manifest
  -> reconciliación y validación de procedencia
  -> selector de partición observed y snapshot legal T0
  -> detectores de decisión/episodio
  -> elegibilidad por capacidad
  -> generador de candidatos parametrizados
  -> filtro de física, reglas y soporte
  -> especialistas + modelos transversales
  -> estimadores de outcome e incertidumbre
  -> ranking común y política de abstención
  -> agregación con perfil de partidas anteriores
  -> Evidence Pack verificable
  -> redacción limitada del LLM
```

Oracle y outcomes entran después del snapshot T0 y sólo durante construcción de etiquetas/evaluación. El servicio de inferencia no debe aceptar esos campos en el payload observado.

## Componentes

| Componente | Responsabilidad | Entrada | Salida | No debe hacer |
|---|---|---|---|---|
| Proveedor canónico | Resolver manifest, schema, IDs, unidades y artefactos | bundle 3.8 | registros tipados + procedencia | inferir táctica |
| Proveedor temporal | Construir historial cerrado en T0 y disponibilidad | observed | ventanas y máscara de disponibilidad | leer futuro |
| Proveedor espacial | Raycast, nav/path y clasificaciones versionadas | posiciones + assets | LOS, alcance, ruta, ETA, zona/cobertura | convertir LOS en intención o corrección |
| Proveedor de episodios | Detectar inicio, T0, final, solape y acción observada | eventos/estados | `DecisionEpisode` | etiquetar por victoria sin contexto |
| Motor de elegibilidad | Aplicar blockers por tarea | episodio + campos | eligible/reason codes | imputar faltantes críticos |
| Generador de candidatos | Proponer acciones concretas y parametrizadas | episodio elegible | `ActionCandidate[]` | proponer acciones imposibles o genéricas |
| Filtro físico | Comprobar dinero, inventario, ruta, clocks, trayectoria y estado | candidato | valid/reasons | rankear utilidad táctica |
| Especialistas | Estimar outcomes relevantes por familia | observed T0 + candidato | distribución/intervalo/soporte | redactar consejo final |
| Modelos transversales | Tiempo, ventaja y riesgo comunes | salidas especialistas | features/ajustes calibrados | duplicar labels futuros como inputs |
| Ranker común | Comparar sólo candidatos válidos y soportados | estimaciones calibradas | orden, equivalencia o abstención | ocultar desacuerdo/incertidumbre |
| Perfil histórico | Detectar patrones en matches anteriores | hallazgos cerrados previos | recurrencia/tendencia | usar el match actual como prior |
| Evidence Pack | Unir hechos, candidato, outcomes, soporte y replay refs | decisión final | JSON verificable | crear hechos nuevos |
| LLM | Convertir Evidence Pack en explicación clara | Evidence Pack | texto | acceder al bundle/oracle o cambiar el ranking |

## Especialistas iniciales

| Especialista | Capacidades | Unidad de ejemplo | Baseline obligatorio | Challenger inicial | Escalado temporal sólo si |
|---|---|---|---|---|---|
| CombatExecution | combate, crosshair, movement | duelo/disparo | reglas verificadas + frecuencia contextual + regresión regularizada | CatBoost | la secuencia aporta mejora agrupada y estable |
| PeekPosition | peek, posición | peek/ocupación | reglas geométricas | CatBoost con assets espaciales | historial de 0.5–3 s supera baseline sin fuga |
| TeamEntry | equipo, trade, entry | ventana multijugador | reglas de conexión + regresión jerárquica | CatBoost | soporte por acción y dependencia temporal suficientes |
| Utility | utilidad | lanzamiento/efecto | reglas balísticas por tipo | CatBoost por tipo de granada | trayectorias/inventario causales completos |
| Economy | economía | freeze-time decision | reglas exactas de compra | regresión jerárquica/CatBoost | hay secuencias de economía 1–3 rondas sin fuga |
| InformationRotation | información, rotación | contacto/rotate | reglas de frescura + path/ETA | CatBoost | ledger FOV/humo/last-known y rutas están verificados |
| Objective | plant, defuse, retake, save | objective deadline | reglas de reloj/kit/alcance | survival model + CatBoost | secuencias mejoran calibración de deadlines |
| Clutch | clutch 1vN | secuencia clutch | reglas prudentes y abstención alta | modelo temporal ligero | soporte raro prerregistrado y revisión humana pasan |
| Profile | patrones históricos | jugador-match previo | estadística descriptiva jerárquica | modelo de tendencia calibrado | identidad, privacidad y orden temporal están resueltos |

## Comparación de familias de modelo

### Reglas

Primera opción para legalidad, clocks, economía, física y explicaciones de baja varianza. Son auditables y funcionan con poco soporte. No estiman bien interacciones complejas ni contrafactuales.

### Frecuencia contextual

Baseline empírico por side, fase, arma, economía y situación. Debe suavizar grupos pequeños y mostrar soporte. Es útil para saber si un modelo sofisticado realmente añade señal.

### Regresión regularizada y modelos jerárquicos

Baseline aprendido principal cuando importan dirección, calibración e intervalos. Los efectos por match/jugador/mapa reducen falsa confianza por filas correlacionadas. Deben reportar coeficientes o contribuciones comprensibles.

### CatBoost

Challenger tabular para no linealidades, categorías y valores ausentes explícitos. Sólo se promociona si supera reglas/regresión en splits agrupados, calibración, slices, estabilidad y revisión humana; nunca sólo por una métrica global.

### Modelos temporales

No son el punto de partida. TCN/RNN/Transformer ligero se evalúan únicamente cuando el historial ordenado aporta una mejora reproducible sobre features agregadas, hay volumen por acción y los tests de invariancia futura siguen verdes. Su complejidad debe justificarse por familia.

## Contrato de datos de modelo

`DecisionExample` contiene como mínimo:

- IDs de join: versión, bundle, match, ronda, episodio, decisión, actor y T0.
- Procedencia: artefacto, path, productor, schema, release, tick de disponibilidad y hash.
- Observed: historial legal, máscara de disponibilidad y contexto de tarea.
- Acción observada y candidatos con parámetros y validaciones físicas.
- Oracle separado, opcional y bloqueado en serving.
- Outcomes separados con inicio/horizonte y censura.
- Elegibilidad, exclusions, confounders conocidos y episode weight.
- Grupo de split y, si está autorizado, grupo de jugador estable.

El esquema de entrenamiento debe tener listas explícitas de columnas permitidas por tarea. La ausencia de una columna crítica marca `ineligible`; no se rellena con oracle, cero o media global.

## Construcción de candidatos

Cada generador es determinista y versionado antes de incorporar aprendizaje. Produce un conjunto pequeño de acciones con parámetros físicos y una razón de inclusión. Ejemplos:

- peek: origen/destino, yaw, duración, ruta, ángulos expuestos;
- rotación: destino, nodos de ruta, distancia y ETA antes del deadline;
- utilidad: tipo, origen, yaw/pitch/fuerza, trayectoria, rebotes y detonation;
- economía: cesta de compra/drop, coste, dinero restante y legalidad;
- objetivo: sitio/posición, duración de plant/defuse, kit y margen temporal.

El candidato observado debe reconstruirse cuando sea posible. Si no aparece o no pasa el mismo filtro, el episodio se bloquea: eso descubre un error de contrato o extracción.

## Outcomes y estimación prudente

Cada especialista predice distribuciones o probabilidades para outcomes declarados, no una etiqueta única de “bueno/malo”. Se distingue:

- outcome factual de la acción observada;
- estimación condicional para una alternativa;
- incertidumbre aleatoria y por falta de soporte;
- censura por fin de ronda/muerte/datos ausentes;
- límite causal: el replay no revela el contrafactual real.

Cuando se comparen acciones observacionales, se requieren overlap diagnostics, propensity/support por acción, sensibilidad a confusores y lenguaje “en situaciones comparables”. No se usa `round_win` como sustituto universal de calidad.

## Ranking común

El ranker recibe únicamente candidatos físicamente válidos. Normaliza outcomes por capacidad y aplica una utilidad versionada que conserva sus componentes: objetivo, supervivencia, daño/space, tiempo, economía y riesgo. No debe colapsar esos valores sin mostrarlos en el Evidence Pack.

Devuelve una de tres respuestas:

1. **recommend:** delta supera umbral, intervalo/soporte/calibración pasan y no hay blocker;
2. **equivalent:** candidatos dentro del margen de equivalencia; se explican trade-offs;
3. **abstain:** datos, soporte, overlap, OOD, física o revisión humana insuficientes.

El umbral y margen se fijan en evaluación, por familia, antes de mirar el test final.

## Perfil histórico

El perfil no es un modelo de personalidad. Agrega hallazgos ya validados de partidas estrictamente anteriores, ponderados por recencia, contexto comparable y confianza. Exige identidad estable autorizada y políticas de privacidad/retención. Un patrón se muestra sólo si recurre en matches distintos, supera ESS y no depende de una sola fase/mapa. La partida actual nunca alimenta el prior usado para aconsejarla.

## Evidence Pack y LLM

El Evidence Pack debe incluir:

- versión de contratos/modelos/assets;
- decisión, ronda, T0 y replay refs;
- hechos observed citables y sus availability ticks;
- acción observada y candidatos parametrizados;
- validación física y motivos de descarte;
- outcomes factuales claramente posteriores;
- estimaciones por candidato, intervalos, soporte/ESS y slices;
- decisión del ranker, nivel de confianza, abstention reasons y limitaciones;
- referencias a reglas y pruebas humanas aplicables.

El LLM recibe sólo este paquete. Se valida que no cambie números, no introduzca enemigos/acciones inexistentes, diferencie hechos de estimaciones y conserve abstenciones. Ante conflicto, se sirve el paquete estructurado o una plantilla, no texto libre.

## Entrenamiento y evaluación

No se abre entrenamiento hasta que el gate de datos lo autorice. Cuando ocurra:

1. congelar inventario, schema, contrato, assets y hashes;
2. construir ejemplos en destino nuevo e inmutable;
3. validar T0/oracle/eligibilidad y soporte antes de split;
4. split agrupado por match y, si es posible, jugador; reservar test final una sola vez;
5. entrenar reglas/frecuencia/regresión antes de challengers;
6. calibrar sólo en validación;
7. evaluar métricas globales y slices, intervalos bootstrap por match y abstención;
8. challenge set de datos ausentes, OOD, smoke/FOV, rutas, clocks y acciones raras;
9. revisión humana ciega y adjudicación;
10. publicar únicamente artefactos versionados con model card, data card y rollback.

Promoción requiere superar al baseline en métrica primaria agrupada, no degradar calibración ni slices críticos, respetar presupuesto de cobertura/abstención y aprobar fidelidad factual humana. Ningún resultado de esta fase cumple todavía esas puertas.

## Servicios futuros y límites

La construcción de ejemplos, modelos y serving debe residir en módulos Python nuevos bajo `ai_coach/`; el parser Go sólo se modifica si una auditoría demuestra que un hecho de replay no puede derivarse del canónico existente. Node continúa como orquestador. Cualquier cambio Go se versiona, prueba y reprocesa hacia un corpus nuevo; nunca reemplaza silenciosamente los 44 bundles actuales.
