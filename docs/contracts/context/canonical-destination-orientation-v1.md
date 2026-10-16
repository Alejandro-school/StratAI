# Hipótesis candidata de orientación A/B

La [aceptación del 01/10](../../evidence/AI_COACH_ORIENTATION_SCOPE_ACCEPTANCE_2026-10-01.json)
autoriza añadir orientación probable separada de preparar una entrada. En E06,
el experto considera A más probable y B todavía posible. El estado de entrega,
evidencia y siguiente acción pertenecen al [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md).

## Significado

`stratai.destination_orientation@1` representa una hipótesis ordinal. Una salida
positiva contiene `preferred_site`, calificado como `more_likely`, y conserva
el otro site como `possible_not_excluded`. No asigna porcentajes ni afirma que
el equipo haya decidido, pueda recorrer una ruta, consiga apoyo eficaz o esté
preparando una entrada. `prepare_entry_assessed=false` es obligatorio.

`ally_site_middle_orientation@1-provisional` es la primera codificación acotada
del principio de reunión/apoyo recibido. Requiere:

- Perspectiva T en Mirage, elegibilidad positiva `bomb_objective` y fase
  preplantada acreditada. El posplantado queda fuera de este alcance.
- Recuento propio completo: exactamente dos T vivos y tres muertos observados,
  sin completar desconocidos ni deducir muertes por ausencia.
- Identidades, conexión y posiciones actuales verificadas para los vivos.
- Una [lectura nominal nativa](canonical-native-place-context-v1.md) actual de
  `BombsiteA` o `BombsiteB` en un aliado y `Middle` en el otro, ligada al corte.

El site del aliado recibe la preferencia. La simetría A/B es la codificación
técnica del patrón de desarrollo; no es una segunda lectura humana de una
partida diferente. La regla no usa portador C4, puntuaciones de aproximación,
nombre del caso, identidad del revisor, respuesta libre ni predicción anterior
para ordenar destinos. La fase del objetivo sigue siendo una precondición.

El valor nativo es el nombre que expone la propiedad en ese instante. No se
transforma en una región geométrica acreditada. No se impone caducidad por la
edad de la última entrega de esa propiedad.

## Abstención y alcance

`abstain` conserva `preferred_site=null` y alternativas vacías. Sus motivos
distinguen elegibilidad desconocida, fase incierta/conflictiva, recuento propio
parcial, configuración no cubierta, fuente nativa no materializada y valor
actual no disponible. `out_of_scope` distingue posplantado, lado/mapa fuera de
alcance o tipo de ronda excluido positivamente acreditado.

E13/E25/E30 con elegibilidad `unknown` no se convierten en cuchillos. E24 conserva
su posplantado. Cinco T movilizándose al inicio, como E17, no cumplen este patrón
de dos supervivientes. Esa abstención no demuestra reconocer spawn ni haber
resuelto el criterio de madurez temporal. No hay veto universal de cuatro
segundos ni una duración que cree evidencia por sí sola.

Las pausas y las reglas de preparación anteriores conservan su propio alcance.
La nueva orientación no sobrescribe una predicción de preparación congelada.
Compararlas requiere mostrar ambas salidas y sus diferencias de información.

## Entrada, procedencia y conservación

`infer(candidate, native_places=None)` valida `stratai.candidate_context@1` y
el contexto nativo separado. Sin ese enriquecimiento se conserva explícitamente
su ausencia. La salida contiene vínculos hash, corte, hechos usados, inferencia,
motivos y límites. `validate(output)` comprueba estructura y coherencia interna;
el hash de contenido no sustituye la autenticación de sus entradas.

`destination_orientation_io` verifica hashes esperados de archivos, escribe
exclusivamente en TEMP nuevo/vacío y conserva cuatro artefactos: contexto nativo,
predicción, informe y manifiesto. Este último registra entradas, código,
artefactos y `information_delta`. Su validador reconstruye la salida desde las
fuentes retenidas. Se rechazan sobrescrituras y destinos enlazados.

La comparación de desarrollo mantiene una ejecución sin enriquecimiento nativo
y otra con él cuando existe. Los valores nativos extraídos después no se atribuyen
retroactivamente al detector primario ni al instrumento respondido. No se calcula
una tasa de acierto con información desigual ni se convierten las 34 respuestas
libres en etiquetas automáticas.

`training_allowed=false` y `product_supported=false` permanecen obligatorios.
Una evaluación independiente necesita otros episodios y nuevas lecturas bajo
el [protocolo](../../ai-coach/AI_COACH_INDEPENDENT_EVALUATION_PROTOCOL.md).
