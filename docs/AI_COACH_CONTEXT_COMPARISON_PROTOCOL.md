# Protocolo experimental de comparación contextual CS3

Versión 1.0.0-experiment.1, 05/10/2026. Aplicación técnica del [alcance aceptado](AI_COACH_CONTEXT_COMPARISON_REVIEW.md). El [plan](AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado, gates y siguiente acción. No añade un contrato runtime al producto, un dataset de entrenamiento ni un consumidor del chat.

## Métodos e igualdad de información

**Continuación local del 05/10/2026:** C usa Qwen 3.5 9B Q4_K_M en Ollama 0.35.1, digest `56671c2ab9385f9cfcb404638e32cd62d88e3501d44822208363c010179a3c90`. HTTP loopback, sin proxies/redirecciones/proveedor externo; sin descargas, entrenamiento o promoción. Prompt/banco/entrada A/B congelados. El envoltorio técnico de C permite consultar muestras y entregar las cuatro partes, con citas limitadas a lo consultado. Parámetros y salidas se registran; no se corrige automáticamente una respuesta rechazada. Recorte de entrada y desplazamiento del contexto desactivados; exceso de contexto/generación/budget queda como límite explícito. La mecánica no certifica calidad táctica. Sellar la configuración completa antes de gameplay de reserva.

| Variante | Implementación y límite |
|---|---|
| A | Referencia factual: hechos al corte, cambios entre extremos y acontecimientos fechados. Explicita que no interpreta tácticamente. Una lista de datos no satisface por sí sola CS1. |
| B | A más recuperación determinista de ocho principios aprobados y tres ejemplos condicionales seleccionados. Recuperar un criterio no acredita su premisa en el caso ni prueba transferencia. |
| C | Intérprete preentrenado aislado con el prompt del paquete. Configuración local elegida y fijada; ejecución y comprobación registradas por intento. Evaluación independiente pendiente. No sustituirlo por una lectura del agente que ya conoce el desarrollo. |

Las tres variantes tienen la misma entrada inicial, archivo causal consultable y banco de principios/ejemplos. A deja sin utilizar el banco; B lo consulta. El prompt C exige las cuatro partes CS1 y afirmaciones con citas. Cada variante dispone de 32 consultas como máximo y 512.000 bytes de respuestas. `sample(index)` entrega una muestra completa, nunca un recorte silencioso. Si excede el presupuesto, devuelve un límite explícito. A/B consultan el inicio y el corte final; no pretenden haber interpretado los pasos intermedios. Una consulta en un índice futuro o una cita no consultada se rechaza.

La vista inicial conserva hechos, estado local, unidades, perspectiva, pertenencia al instrumento original, índices de disponibilidad, relojes observados, antigüedad de eventos, continuidad y duración del prefijo. Cada muestra anterior conserva los mismos campos. Identidad/vida se sustituyen por alias locales; no se incluyen IDs de caso, hashes de demo, paths de captura, handles nativos, textos humanos, predicciones ni desenlaces posteriores. Índices y ticks de evidencia se expresan relativos al prefijo, conservando antigüedad. Las referencias originales y el valor completo permanecen en `private_citations.json`, vinculado por hash y fuera de la entrada del modelo. La entrada original completa queda retenida como evidencia privada, nunca como prompt.

Esta entrega usa el perfil original de cada candidato: E03 @5, otros casos @9. No incorpora nombres nativos posteriores ni los atribuye retrospectivamente al instrumento original. El banco es redacción de principios ya documentados; no una extracción automática de etiquetas de las respuestas de Alex. El ejemplo de reunión en A es condicional y no sitúa automáticamente jugadores en A usando coordenadas sin fuente espacial acreditada.

## Selección, desarrollo y reserva

[Selección congelada](../outputs/stratai-context-comparison-20261005/selection.json): desarrollo E03/E04/E06/E13/E17/E24/E25/E30. Son ocho episodios conocidos de T; no son evaluación independiente. Las ventanas T/CT del smoke verifican funcionamiento técnico y no incrementan ese número.

La reserva propone dos partidas aprobadas, `demo-ad48bf05c36fef00f8d4` y `demo-cab5667cb4c8e0e77c34`, elegidas por SHA ascendente, sin participantes comunes entre sí ni con el desarrollo conocido y TP10. La selección de metadatos precede a lectura de gameplay nuevo y generación. Por partida: cuatro ventanas continuas `playing`, en cuatro rondas distintas, espaciadas por orden temporal y con prefijo continuo hasta el corte de hasta 4.097 muestras; perspectivas T/CT/T/CT. La selección usa reloj, fase, continuidad e identidad de ronda, nunca posiciones, resultado o predicción. Si faltan cuatro rondas elegibles, registrar déficit y no cambiar la partida para mejorar el resultado.

Los sidecars support/physical/C4 faltaban en esos dos bundles. Se extraen exclusivamente en TEMP nuevo para el experimento posterior al cierre técnico del transporte: 8 episodios materializados de ocho previstos, déficit 0, sin modificar corpus. A/B/C, configuración y presentación estaban sellados antes del gameplay reservado. Son entradas generadas y comprobadas técnicamente, todavía sin revisión táctica independiente. La selección por jugadores no acredita independencia de organizaciones/estilos o ausencia de exposición privada no registrada. No es test de partidas posteriores.

Máximo dos iteraciones usando sólo desarrollo. Congelar código, banco, prompt, configuración, consultas, presupuesto y selección antes de la reserva. No ajustar un método tras conocer sus salidas reservadas; si se usa esa información para ajustar, el material pasa a desarrollo y requiere otra reserva admisible. El [protocolo independiente](ai-coach/AI_COACH_INDEPENDENT_EVALUATION_PROTOCOL.md) conserva sus límites de exposición y separación. C queda sellado con Qwen/Ollama local y `think=false` tras un límite de generación en smoke conocido; el intento previo se conserva. No se modifica el método tras ver reserva.

## Criterios de juego y revisión

Conservar E06/E17/E24 y unknown en E13/E25/E30. Integrar planta acreditada; recuentos parciales no prueban 1v1; movilización requiere evolución pertinente sin mínimo universal; no inferir cuchillos por ausencia de C4. Posiciones históricas de eventos no prueban contactos actuales o percepción.

Para CT conservar efecto defensivo, patrones T y adaptación. El archivo de una ronda no aporta historial entre rondas. Puede describir la defensa observable y sus límites, pero no certificar repetición, adaptación o error por derrota. Los criterios operativos de valoración/corrección se encuadrarán sobre material real; no se implementan aquí por inversión de reglas T.

La comprobación automática valida formato, origen, causalidad, presupuestos, igualdad de entradas y existencia/consulta de citas. **No valida el significado de una frase ni su suficiencia táctica.** La rúbrica humana pendiente revisará cada afirmación, inferencia, alternativa, omisión relevante e incertidumbre, con las cuatro partes de CS1. No convertir texto libre en etiquetas. Revisores independientes recibirán las mismas evidencias, método oculto y orden aleatorio reproducible; no salidas ajenas antes de su lectura. El contraste posterior sólo verifica lo ocurrido y permanece fuera de la entrada causal.

No computar tasa A/B, elegir ganador o declarar intérprete general antes de salidas C reales y revisión independiente. Ocho casos de desarrollo u ocho reservados no garantizan suficiencia estadística. Las ausencias factuales limitan afirmaciones concretas y no borran los hechos disponibles.

## Verificación y conservación

Pruebas públicas Python y lint de módulos afectados, después smoke sobre exactamente una demo aprobada en TEMP nuevo. Recalcular valores expuestos, tiempos relativos, alias, eventos y paridad de inputs con acceso directo a las fuentes; validación semántica canónica y hashes separados de código y datos. Reutilizar ocho candidatos históricos seleccionados, sin reauditar los 34. Guardar manifests, comandos, duración, conteos, abstenciones y todos los inventarios temporales, incluso intentos fallidos.

Conservar el detector, predicciones, respuestas originales, libros anteriores, demos y 44 bundles. No adquirir datos, entrenar, promocionar, publicar ni servir estas salidas como coaching. Registro runtime y catálogo permanecen intactos.

## Ejecución local comprobada

156 pruebas; exactamente una demo aprobada verifica el productor/transporte, con ocho ventanas factuales y dos llamadas C finales T/CT. El experimento posterior usa dos partidas reservadas, separado del smoke y sin cambios del productor. Las salidas, peticiones, respuestas y consultas quedan vinculadas por hash y replay. [Resultado](../outputs/stratai-context-comparison-ollama-20261005/RESULTADO.md) y [fichas Markdown](../outputs/stratai-context-comparison-ollama-20261005/review/REVISION.md). Ninguna revisión humana recibida: formato/citas válidos no son puntuación táctica ni cierre completo CS3.
