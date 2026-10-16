# Intérprete contextual desde replay — ICT-I2

Primera versión del 23/09 y actualización temporal del 24/09/2026. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md)
conserva estado, evidencias y siguiente acción.

## Criterio acordado

El usuario acepta «Aplicar este criterio provisional»: organización compatible
con preparar una entrada cuando varios aliados y el portador de C4 progresan
hacia la misma referencia espacial. Un jugador retrasado con utilidad equipada
puede preparar utilidad. Una pausa no obliga a abandonar la lectura; retroceso,
dispersión o insuficiencia de datos exigen revisión o abstención. No acredita
lanzamiento, cobertura, intención ni superioridad táctica. Se conserva en IC19
del [Excel v9](../../../outputs/stratai-contextual-20260923/StratAI_Plan_del_coach_ICT_I2_v9.xlsx).

## Entrada causal

`stratai.contextual_input@1`, `1.0.0-candidate.1`, incorpora **cada muestra** del
intervalo explícito solicitado, con reloj, vida, identidad, posiciones, los siete
hechos físicos disponibles, C4 y referencias originales. Es el schema 108 del
registro, con las 19 razones existentes conservadas. El catálogo `1.7.0` y el
bundle base `3.8.0` mantienen su significado.

La API `context.from_sources` proyecta antes de validar mediante el registro
temporal existente. El detector no recibe respuestas humanas, nombres K1/K2,
desenlace ni rivales ocultos. Los IDs son enlaces de procedencia, no predictores.
La integridad completa del artefacto se verifica en el lector offline.

Los tramos no se interpolan. La trayectoria utilizada es el sufijo continuo por
actor: un cambio de vida, lado, instancia, reloj, muestra ausente o exclusión
corta la historia. Cero y falso mantienen su valor. El intervalo de análisis es
explícito, no una duración universal de intención. Se retienen munición y recarga
como hechos de su contrato original, pero no intervienen en la inferencia de
readiness. El contexto actual tampoco acredita inventario completo, disparos,
posición mundial de C4 ni conocimiento compartido.

## Referencia espacial

`spatial_reference.extract_vmap` extrae los orígenes de entidades `env_cs_place`
del VMAP local. Conserva nombre nativo, nodeID, coordenadas y SHA-256 del archivo.
Sólo normaliza `BombsiteA`, `BombsiteB` y `Apartments` a A, B y apartamentos.
Rechaza transformaciones no soportadas y no usa el callout por vecino más próximo.

Esta versión ofrece **referencias de desarrollo**, no polígonos de zonas.
Incluso situado exactamente sobre un origen, `resolve` devuelve `zone=null`.
La etiqueta «hacia la referencia B del mapa local» se distingue de «está en B»
o «entra por apartamentos». La compatibilidad del asset con la build de la demo,
la región ocupada, NAV, tránsito y LOS siguen sin acreditarse. La referencia se
verifica contra su fuente al recomponer el resultado; no se toma de la respuesta
del experto ni de una etiqueta incrustada en el caso.

## Primera regla automática

`collective_c4_approach@1-provisional` está acotada a preparación T. Compara el
desplazamiento horizontal con las referencias de los dos sites, sin convertirlo
en distancia de ruta. Para un candidato exige:

1. Un único portador actual, con C4 corroborada a lo largo de su tramo utilizado.
2. Más de un actor, incluido el portador, con reducción neta de distancia y
   último cambio de distancia resuelto hacia la referencia. Las muestras de
   posición repetida no se convierten en retroceso.
3. La referencia es la más alineada con el vector inicial→final de cada actor
   participante. Se retienen empates con tolerancia numérica de `1e-12` en el
   coseno, sin usar nombres o IDs para desempatar. No es una probabilidad.
4. Un aliado en desplazamiento resuelto hacia otra referencia impide presentar
   el patrón como una preparación conjunta respaldada.

Estos son suficientes provisionales y conservadores, no definiciones universales
de entrada. Rutas curvas, desplazamientos mínimos, cambios de portador y planes
distribuidos pueden generar abstenciones o errores. El tamaño del grupo no prueba
cooperación ni que todo el equipo esté entrando.

La función posible de utilidad combina el patrón colectivo con posición más
retrasada respecto a la referencia, utilidad equipada y un sufijo de al menos dos
muestras horizontales iguales con esa arma. La duración exacta queda en la
evidencia, no se convierte en un umbral táctico universal. No afirma lanzamiento
ni utilidad efectiva. No asigna entry, lurker o apoyo eficaz por orden espacial.

Cada corte recalcula la lectura. La comparación con la anterior registra inicio,
continuidad, revisión, abstención o contexto anterior indisponible. No conserva
una hipótesis si perdió su evidencia. Puede emitir varias hipótesis compatibles
sin escoger una arbitrariamente. E01–E04 conservan sus dependencias.

## Actualización temporal v2

El [criterio recibido el 24/09](../../evidence/AI_COACH_TEMPORAL_PERSISTENCE_CRITERION_2026-09-24.json)
es «Mantener posibilidad hasta contradicción». IC24/Excel v14 conserva el literal.
`collective_c4_approach@2-temporal-provisional`, en `temporal_detector.py`, separa
la creación de una posibilidad y su mantenimiento. `detector.py` conserva v1 y
sus resultados anteriores pueden reproducirse con su configuración explícita.

La versión nueva recorre todas las muestras causales. La señal inicial conserva
el criterio de aproximación conjunta de v1. Después conserva la posibilidad
durante una pausa, incluso de todo el grupo, mientras haya soporte previo,
identidades/posiciones/C4 actuales verificadas y continuidad. No existe un plazo
de caducidad por inmovilidad. Una imagen estática sin soporte previo no crea un
plan. Las funciones de utilidad se recalculan con el arma y los supervivientes
actuales, sin presentar una función histórica como vigente.

La contradicción utiliza un filtro técnico provisional de **1 segundo de
movimiento contrario acumulado sin avance intermedio**, medido con el reloj
nativo. Una muestra estacionaria no suma tiempo contrario ni borra lo observado.
Un paso hacia la referencia reinicia el contador de alejamiento; la alineación
con la referencia reinicia el contador de divergencia. También se retira la
lectura si el portador deshace toda la aproximación que la respaldó, sin esperar
ese segundo. La duración es una decisión de ingeniería documentada antes de
probar R2, no un umbral experto ni una definición universal de retirada. Su
sensibilidad se contrasta por separado con 0,5 y 2 segundos.

Una retirada registrada revisa la hipótesis. Se conserva la trayectoria factual
de aliados no afectados; se reinicia la del actor que aporta la contradicción.
Para una nueva lectura tras esa revisión se exige movimiento fresco del portador
hacia la referencia. De esta forma una pausa posterior no recupera automáticamente
el plan retirado. La muerte excluye al fallecido y vuelve a comprobar el soporte
de los supervivientes; no equivale a datos ausentes ni prueba qué comunicó.
Pérdida de C4, identidad o continuidad invalida el soporte dependiente. Un destino
alternativo requiere su propia evidencia positiva.

`stratai.contextual_interpretation@2` añade un schema ejecutable: **112 schemas
y 19 razones** en el registro. Conserva el límite prescriptivo de v1 y añade
`support_origin_cut`, `support_current_cut`, supervivientes que aportan soporte,
señales históricas y actuales diferenciadas, política técnica, eventos de revisión
y estado temporal. Este estado enlaza la muestra anterior, identidades, origen y
contadores; no incorpora respuestas humanas. Se valida estructura, hash, cortes
y referencias no futuras. La recomputación offline contrasta el contenido con
las fuentes originales, además del hash.

Para reutilizar una lectura anterior se exige fuente, lado, build, referencia de
mapa y configuración compatibles. El corte anterior debe coincidir exactamente
con una muestra incluida o ser el inmediatamente anterior con reloj continuo.
Un hueco no se completa con posiciones antiguas. La evidencia de origen permanece
enlazada cuando sale de la ventana actual; no se anida recursivamente la lectura
anterior. Los eventos del resultado describen las muestras procesadas en esa
actualización, no una historia de intenciones confirmadas.

R2 y los episodios ya expuestos siguen siendo desarrollo. Esta implementación
no cierra TP10 ni la evaluación independiente. Las referencias espaciales siguen
siendo landmarks de desarrollo y la información rival permanece desconocida.

## Evaluación y consejo

El evaluador congela episodios por partida y versión antes de leer predicciones
o etiquetas. Recibe las revisiones en una entrada separada. Registra emisiones,
abstenciones, desconocidos, errores y acuerdos por partida. Sin respuestas reales,
`agreement=null`: ninguna abstención se contabiliza como acierto por defecto.
Una coincidencia en K1/K2 es desarrollo conocido. Ni fuentes nuevas para esta
regla ni filas nuevas acreditan por sí solas un test independiente: requieren
exposición previa, cruces temporales/de participantes y revisión ciega.

`coaching.assess_readiness` conecta la interpretación con una salida explícita de
abstención prescriptiva y enumera las dependencias. **No implementa todavía un
comparador de alternativas, perfil personal ni generador de práctica.** Las
hipótesis no generan prioridades, errores, misiones ni cambios del perfil. Faltan
alternativas parametrizadas físicamente viables, comparación respaldada,
evaluación independiente, identidad segura/perfil previo y criterios revisados
de práctica y oportunidad. No hay entrenamiento ni aprendizaje online.

## Uso y verificación

El escritor exige un directorio vacío bajo TEMP antes de leer fuentes. Usa
escrituras exclusivas y retiene todos los artefactos. El lector recompone entradas,
mapa, interpretaciones, límites, conteos e informe desde las fuentes originales.
El materializador usa v2 por defecto. `--configuration collective_c4_approach@1-provisional`
selecciona la versión anterior; el validador utiliza la versión registrada en
el manifiesto y rechaza configuraciones desconocidas.

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.contextual_io materialize --source <captura> --output <TEMP-vacio> --side T --intervals 1325:1389 1353:1417 --map-vmap backend/data/maps/de_mirage/de_mirage_texto.vmap
backend/venv/Scripts/python.exe -B -m ai_coach.interpretation.contextual_io validate --source <captura> --output <salida> --map-vmap backend/data/maps/de_mirage/de_mirage_texto.vmap
```

Los números anteriores son sólo el ejemplo de desarrollo conocido. Produce
`contexts.jsonl`, `interpretations.jsonl`, `coaching_readiness.jsonl`, informe en
español y manifiesto con hashes. No añade endpoint web ni cambia el modo temporal
de interfaz. Las pruebas cubren causalidad, mapas incompatibles, ausencias,
ambigüedad, continuidad, alteraciones de archivos y límites de evaluación.
