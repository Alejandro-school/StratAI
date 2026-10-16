# Interpretación contextual con hipótesis justificadas

Fecha: 16/10/2026. Versión de alcance: 0.1, encuadre aceptado tras aclaración. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente acción; este documento especifica una capacidad, no otra hoja de ruta.

## Objetivo y acuerdo recibido

El usuario rechaza interpretar CS2 como un catálogo «si pasa X, significa Y» y acepta reorientar el trabajo hacia comprender la situación, conservar interpretaciones posibles y, posteriormente, comparar acciones según su valor esperado. Respuesta recibida: «Hazlo mejor así si».

La primera capacidad es **explicar la situación y su evolución desde la información disponible, con las hipótesis que esa evidencia permita**. La comparación de acciones es posterior. Se recupera el [criterio contextual CS1 ya aceptado](../AI_COACH_CONTEXTUAL_REPRESENTATION_CRITERION.md) y la [definición de producto](../AI_COACH_PRODUCT_DEFINITION.md): acertar un destino omitiendo lo que cambia el significado no basta.

Ejemplo: dos jugadores adelantados en B Apps son un dato espacial. Un avance reciente hacia la salida puede dar más fundamento a preparación de entrada; permanecer allí también puede ser compatible con preparación, espera o presión. Ninguna de esas observaciones demuestra por sí sola la intención final. Tampoco bastaría añadir azar a la respuesta para comprender mejor.

## Alcance y límites

- Mantener el modelo inicial, su digest y los presupuestos vigentes. No atribuir el problema a los pesos sin evidencia ni aprobar otro proveedor, descargas o entrenamiento.
- Conservar hechos y referencias protegidos: un único C4 en cada corte, identidad, cantidades, vida, zonas, estado, relojes, inventario fechado y acontecimientos. Las correcciones factuales existentes siguen siendo necesarias.
- Usar situación **y antecedentes pertinentes hasta el corte**, con fuentes, fecha, perspectiva y disponibilidad. No mezclar hechos históricos con posiciones actuales. Los desconocidos de un aspecto no deben borrar la interpretación de otro que sí tenga respaldo.
- Usar la región aprobada como una medición identificada y aproximada, junto con el resto del contexto. No usar pertenencia ni el catálogo `possible_attack_b_advanced_apps` como frontera de todas las lecturas que el modelo puede ofrecer. El prototipo @4/1.7 queda reproducible como referencia histórica; esta especificación aún no cambia su runtime.
- Admitir una hipótesis, varias o insuficiencia localizada. No exigir siempre dos alternativas, favorecer ataque, seleccionar un destino ni llenar un campo por obligación. Las hipótesis pueden ser compatibles: esperar coordinación y preparar una entrada no son necesariamente excluyentes.
- Explicar qué apoya una lectura, qué la debilita, qué alternativa sigue abierta y qué nueva información podría cambiar la preferencia. Una explicación debe ser breve y comprensible, sin inventar planes internos o comunicaciones efectivas.
- No recomendar todavía una jugada ni declarar un error táctico. No deducir apoyo efectivo desde Medio, defensa organizada o reacción a presión sólo por coincidencia temporal. La utilidad inicial puede ser habitual contra posibles rushes.

El acuerdo general y el contraste ilustrativo están recibidos. Tras aclarar que se trata de dos situaciones en B Apps, el usuario responde «Me parece correcto ese enfoque». Acepta considerar el movimiento anterior sin traducir avance o espera automáticamente en intención. No se convierten estos ejemplos en etiquetas aprobadas de partidas.

## Entrada, perspectiva y causalidad

El primer contraste mantiene perspectiva colectiva T, como el material disponible. Reutiliza hechos propios/aliados, C4, recursos, fase, tiempos, acontecimientos y señales visuales, de utilidad o sonoras ya admitidas. Conserva comunicación inmediata **posible y simulada**, no recepción real acreditada. Una perspectiva CT futura sólo recibe lo disponible para CT; conocer las posiciones nativas T no autoriza exponerlas al intérprete CT.

El historial necesita muestras fechadas, intervalo cubierto, lagunas y continuidad de identidad. Sólo puede describirse un recorrido observado con soporte suficiente; dos extremos no prueban todos los puntos intermedios. Deben distinguirse desplazamiento, permanencia observada y ausencia de muestras. No se crea una regla universal de segundos o velocidad para convertirlos en intención.

Las señales conservan sus estimaciones y límites. Los puntos sonoros nativos autorizados siguen sólo en el mapa; no se filtran coordenadas enemigas ocultas a la interpretación. Tres lanzamientos observables del mismo tipo HE/humo/molotov-incendiaria en cinco segundos conservan el criterio acordado, sus excepciones y exclusión de flashes. Un recuento de registros no es un recuento de enemigos o granadas.

Dependencias antes de ejecutar: comprobar qué antecedentes contiene realmente la entrada actual, con qué cobertura y presupuesto; versionar de forma optativa cualquier ampliación; separar información ausente de información descartada al serializar. Una carencia se documenta y limita sólo las afirmaciones dependientes. No rellenarla con el futuro de la demo.

## Casos concretos para revisión experta

Son ejemplos construidos para discutir el criterio, **no extracciones adicionales de W01 ni predicciones de Qwen**. Comparten una distribución final inspirada en el [caso real conservado](../../outputs/stratai-abstention-diagnosis-20261016/caso-W01.html): P03/P05 adelantados dentro del tramo de B Apps, P04 antes del paso, C4 único con P02 en BackAlley y P08 en Catwalk; ronda activa antes de plantar. En ambos se mantienen iguales recursos, información recibida y corte actual. No se supone contacto visual CT ni apoyo efectivo de P08.

| Caso | Antecedente disponible que cambia | Lectura propuesta por el agente | Alternativas y límite |
|---|---|---|---|
| Situación 1: avance | Muestras previas fechadas muestran a P03/P05 progresando desde más atrás hacia la puerta/ventana que desemboca en el punto B; en el corte están adelantados. | «El grupo ha avanzado hacia el acceso B; la preparación de una entrada gana fundamento». | Destino definitivo abierto: puede detenerse o cambiar el plan. La progresión no demuestra compromiso irreversible, plantado ni coordinación efectiva. |
| Situación 2: espera | Muestras previas fechadas muestran a esos jugadores manteniéndose en la zona adelantada, sin progresión observada durante el intervalo cubierto. | «Grupo adelantado en espera; preparación, presión y espera de coordinación siguen siendo compatibles». | La quietud no demuestra fake ni descarta un ataque inminente. No ver avance no demuestra ausencia de intención. |

**Aclaración y aceptación:** las antiguas letras A/B nombraban ejemplos, no puntos del mapa ni dos hipótesis. Ambos se refieren a B; «salida» es el acceso desde B Apps al punto B. El usuario acepta el enfoque aclarado. No queda una pregunta pendiente sobre estos casos ni se añade un umbral temporal. La aceptación no acredita que las demos conocidas tengan muestras suficientes para observar ese avance o espera.

Si lo disponible, incluidos los antecedentes, fuese idéntico en ambos, el sistema no podría conocer cuál de dos intenciones ocultas es la real. Debe conservar esa incertidumbre, aunque la partida completa revele después el desenlace.

## Representación propuesta y estilo de explicación

Separar semánticamente hechos y evolución observada, hipótesis, evidencia/contraevidencia, alternativas y desconocidos. La forma ejecutable exacta es una decisión técnica posterior al encuadre; no se fija aquí un nuevo esquema JSON ni una lista universal de etiquetas tácticas.

Ejemplo de salida **ilustrativa, no lectura generada ni etiqueta de una partida**:

> Dos T están adelantados en B Apps y sus muestras anteriores muestran avance hacia el acceso. Esto respalda preparación de entrada en B, aunque todavía podrían detenerse o cambiar el plan. El C4 sigue en BackAlley: su posición no acredita que vayan a plantar en B. No hay evidencia suficiente de apoyo efectivo desde Catwalk.

Cada afirmación factual debe poder volver a sus referencias y corte. Los hechos seleccionados se reconstruyen con las protecciones existentes; la interpretación debe identificarse como inferencia. Permitir prosa más expresiva introduce nuevas afirmaciones posibles: referencias válidas no bastan para demostrar que el texto no contradice los hechos, ni un validador factual demuestra comprensión táctica. La verificación debe revisar ambos aspectos y rechazar o abstener sólo la parte afectada, sin corregir silenciosamente lo que decidió el modelo.

La valoración y su representación deben ser coherentes: no declarar apoyo y ocultarlo en otra lista vacía, como en D6/D7. Esa reparación sigue siendo necesaria, pero pasa a servir a esta capacidad contextual; el éxito ya no será emitir una candidatura espacial prefijada.

## Probabilidad y acciones posteriores

La dirección es probabilística: estimar qué explicaciones son más plausibles con información parcial y actualizar esa valoración al llegar evidencia. Primero se admitirán preferencias relativas justificadas o indeterminación. Un «70 %» autodeclarado por el LLM no es una probabilidad calibrada; la repetición de respuestas tampoco demuestra calibración.

Antes de presentar porcentajes fiables habrá que definir el evento y horizonte que predicen, reunir evaluación independiente representativa, medir calibración y cobertura y examinar casos fuera de distribución. No exigir que hipótesis compatibles sumen cien. Esta especificación no autoriza reunir nuevos datos o entrenar.

La fase posterior comparará acciones viables bajo varios escenarios y su incertidumbre, con tiempo, geometría, recursos y capacidad del jugador. La explicación rival más probable no determina automáticamente la mejor acción: una alternativa menos probable pero costosa puede cambiar la decisión. Un resultado retrospectivo favorable o desfavorable no demuestra el valor contrafactual de otra jugada. Faltan el encuadre experto y los datos para ese comparador; no se declara implementado.

## Estructura y comandos de verificación

Referencias actuales: `ai_coach/interpretation/` para lógica de lectura y adaptadores optativos; `ai_coach/tests/` para regresiones; `backend/tests/` y `frontend/src/features/replay2d/` para sus consumidores; `docs/contracts/context/` para contratos versionados; `outputs/` para evidencia aislada. Mantener módulos focalizados y las convenciones existentes; no llevar lógica de dominio a rutas o componentes. El encuadre original fue documental; tras su aceptación se implementa el [contrato optativo](../contracts/context/replay-contextual-reading-v1.md), según el plan canónico.

Comprobación documental actual:

```powershell
npm run check:docs
```

Para una implementación posterior, ejecutar las comprobaciones pertinentes al cambio; la interfaz pública disponible es:

```powershell
npm run lint:python
npm run test:python -- ai_coach/tests
npm run test:node
npm run test:go
npm run test:frontend
npm run build:frontend
```

La verificación de una implementación debe incluir:

1. **Hechos y causalidad:** contradicciones de C4/cifras/tiempo, corte sin futuro, perspectiva, fuentes, historial con lagunas y ausencia de filtración de posiciones rivales. Conservar regresiones y contratos históricos.
2. **Coherencia de salida:** apoyo, no apoyo e insuficiencia representados sin contradicciones; hipótesis compatibles permitidas; ninguna prueba obliga una etiqueta de ataque por entrar en un polígono.
3. **Contrastes de desarrollo:** misma fotografía con historias distintas y misma evidencia con varias realidades posibles. Comprobar sensibilidad justificada, sin exigir diferencia artificial de texto o de preferencia en cada par. Las propuestas A/B son para encuadre, no verdad independiente para puntuar al modelo.
4. **Evaluación táctica separada:** W01/N01/N04/N10, D0–D7 y casos ya discutidos son material conocido. La calidad general exige nuevos casos y revisión humana independiente, con separación previa entre desarrollo y evaluación, errores/omisiones/abstenciones conservados y sin reintentos selectivos. No afirmar porcentaje de aciertos con este encuadre.
5. **Integración y presupuesto:** entrada completa, medición real de tokens/tiempo, mismo modelo y límites, sin recorte silencioso. Si no cabe, resolverlo explícitamente sin sacrificar información decisiva para mejorar una lectura concreta.
6. **Smoke funcional:** si cambia parsing, exportación, hechos derivados o contratos runtime, después de las pruebas procesar exactamente una demo aprobada en TEMP nuevo, vacío y aislado; repetir esa misma si falla, recomputar lo afectado y registrar hashes, versiones, causalidad, conteos, abstenciones y duración. No sustituir este gate por una llamada al modelo o un exit code cero. Retener todos los temporales sin sobrescribir, promover ni tocar los 44 bundles.

## Condiciones de aceptación y límites de autorización

El diseño es suficiente si distingue hechos de inferencias, admite explicaciones compatibles sin etiquetas obligatorias, integra antecedentes disponibles y hace revisables sus preferencias y límites. La capacidad funcional sólo podrá declararse tras la implementación y la evidencia correspondientes. La calidad táctica y las probabilidades fiables conservan evaluación independiente propia; no se acreditan con JSON válido o ejemplos favorables.

Siempre: preservar cambios, comentarios, análisis y libros originales; mantener procedencia, versiones, causalidad y abstenciones locales; resolver decisiones técnicas rutinarias dentro del alcance revisado. Volver al experto sólo por criterios nuevos de juego/producto o ampliaciones materiales. No entrenar, adquirir demos, cambiar corpus/release/CURRENT, publicar, promover ni limpiar temporales con esta autorización.

La revisión de las situaciones 1/2 está recibida. Se registra sin repetir acuerdos anteriores y se implementa esta capacidad según el plan canónico. No continuar por inercia la antigua reparación centrada en una sola etiqueta. El diseño previo y su estado pendiente permanecen íntegros en la evidencia de encuadre.
