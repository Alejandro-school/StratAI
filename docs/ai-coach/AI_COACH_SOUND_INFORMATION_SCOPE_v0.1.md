# Sonido e información posible — propuesta v0.1

Fecha: 12/10/2026. Estado: **propuesta para encuadre experto, sin implementación**.
El plan canónico mantiene el estado y la siguiente acción. Esta propuesta
continúa la comparación de contexto compacto, con Qwen y límites originales.

## Resultado previo y propósito

N01/N04/N10 tienen lecturas factuales válidas mecánicamente; interpretación
vacía en los tres. N04 es idéntico al par anterior. La revisión del agente
no sustituye la aceptación experta o la evaluación humana independiente.
[Revisión y ejemplos](../../outputs/stratai-sound-framing-20261012/REVISION_LECTURAS.md).

Añadir qué sonidos **podría recibir cada jugador vivo** y qué información
podría compartir el equipo. Mantener fuente, fecha, antigüedad, observador y
receptores posibles. La emisión nativa, la recepción estimada y la deducción
táctica deben seguir separadas. El modelo actual de comunicación inmediata
posible, incluidos receptores muertos, permanece acordado.

## Datos comprobados y carencias

La captura final retenida del cierre anterior contiene 20 rondas. Su
`event_context` registra 2.660 disparos, 347 lanzamientos, 489 daños,
450 inicios y 164 finales de utilidad. Estos conteos son acontecimientos
registrados, no sonidos escuchados. Los marcadores de objetivo registran
intentos y transiciones de C4. Se ha leído esta captura sin volver a parsear.
[Conteos y hashes](../../outputs/stratai-sound-framing-20261012/retained-capture-survey.json).

El código actual registra callbacks de disparo y recarga en combate; el
inicio de recarga no acredita finalización ni audibilidad. La entrada actual
expresa que no entrega sonido. Las colecciones inspeccionadas no contienen
eventos de pasos ni observaciones auditivas por jugador. Esta afirmación se
limita a esas colecciones y a las rutas inspeccionadas; no prueba ausencia
de sonido en la demo ni revisión exhaustiva de todo el proyecto.

La dependencia fijada es demoinfocs v5.2.0. Sus tipos `Footstep`, `PlayerSound`,
`WeaponFire` y `WeaponReload` ofrecen fuentes candidatas. `PlayerSound`
incluye jugador, radio y duración, pero no una clase de sonido. Un evento
genérico no se puede llamar «paso» por defecto. El radio publicado tampoco
se puede presentar como distancia real de escucha sin comprobar su semántica.
[Fuente primaria fijada a v5.2.0](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/v5.2.0/pkg/demoinfocs/events/events.go).

El grafo orientó las rutas; su metadata de varios archivos está desactualizada.
Se contrastaron los archivos locales vigentes. No se actualizó ni se sustituyó
el índice. Las fuentes y su cobertura se conservan en la evidencia de preparación.

## Alcance propuesto

| Fuente candidata | Qué se podría representar | Condición y excepción |
|---|---|---|
| Pasos nativos | Señal compatible con pasos desde una región aproximada | Callback positivo; no crear pasos desde velocidad, cambio de posición o un sonido genérico |
| Emisión genérica | Ruido de clase desconocida | Conservar radio/duración publicados como datos del evento, sin inventar causa |
| Disparo y recarga | Señal compatible con disparo o inicio de recarga | El acontecimiento necesita un criterio separado de recepción; no garantiza acierto, cargador vacío o recarga terminada |
| Granadas | Señal de impacto/explosión/efecto cuando la fuente y el tipo lo permitan | Un lanzamiento, rebote o fin de efecto no se transforma automáticamente en explosión audible |
| C4 | Señal compatible con intento de plantado/desactivación u otro sonido del objetivo acreditado | Separar aviso global del juego, acción propia/aliada y audición local; no revelar intentos ocultos desde el estado omnisciente |

Se proponen estas familias en la misma fase. La elegibilidad de cada familia
depende de fuentes positivas; una familia sin evidencia suficiente queda
desconocida y se informa su cobertura, sin completar huecos con suposiciones.
No se incorporan voces, atención, volumen personal, auriculares o escucha real.

## Criterios de recepción propuestos para revisar

1. Emisión y posición 3D causales, con fecha y procedencia. Posición del
   receptor disponible al emitir, receptor vivo y mismo intervalo de ronda.
   No usar posiciones futuras ni inventar una posición exacta entre muestras.
2. Calcular distancia 3D. Utilizar un alcance sólo si procede de una fuente
   cuya semántica sea comprobada o de un perfil de estimación explícito,
   verificable y revisable. El radio nativo puede conservarse como dato de
   emisión; no basta por sí solo para declarar escucha. **No se fija aquí
   una cifra universal ni una tabla de distancias inventada.** Sin alcance
   defendible, conservar la emisión y abstenerse sobre recepción.
3. Propuesta para paredes y alturas: registrar incertidumbre acústica y
   separar espacios conectados de casos con geometría incompleta. Un rayo
   visual bloqueado no decide por sí solo que el sonido sea inaudible.
   No afirmar recepción clara a través de toda pared. La atenuación exacta,
   materiales y distinción entre niveles necesitan datos/calibración.
4. Disparos/explosiones simultáneos: conservar las señales y marcar posible
   enmascaramiento. No borrar automáticamente una recarga ni garantizar
   que se oyó sobre una explosión. Umbrales temporales y perfiles requieren
   evidencia; no se inventan duraciones para rellenarlos.
5. Mostrar clase de señal y región/dirección compatibles sólo cuando estén
   respaldadas. Propuesta inicial: no conceder identidad enemiga, coordenadas
   exactas ni un número de rivales a partir de sonidos aislados. El ID del
   emisor sirve para auditoría, no como reconocimiento concedido al equipo.
   El bando puede seguir desconocido; no se obtiene de la identidad nativa oculta.
6. Un sonido oído de forma estimada puede originar un informe posible bajo
   la comunicación ya acordada. Mantener observador y receptores separados,
   incluidos muertos como posibles receptores compartidos. La ausencia de
   una nueva señal no significa «zona despejada» ni prueba que nadie esté allí.
7. Correlacionar fuentes sólo con evidencia; no duplicar una emisión desde
   varios callbacks ni sumar observadores como emisores. La regla de tres
   granadas en cinco segundos sigue ligada a lanzamientos observables del
   mismo tipo; tres ruidos/explosiones no recuperan esos lanzamientos ocultos.

Los criterios 2–5 contienen decisiones de juego nuevas. Se requiere el encuadre
experto y datos suficientes antes de implementar estimaciones dependientes.
La aceptación de un criterio no acredita una fuente acústica ni una calibración.

## Casos para la revisión experta

| Caso | Propuesta de lectura | Lo que debe quedar abierto |
|---|---|---|
| CT próximo a B Apps; pasos detrás de una pared, sin ver al rival | «Posibles pasos desde la zona compatible con B Apps», si las fuentes y el alcance son elegibles | Identidad, número y coordenadas actuales; pared/altura y alcance pueden mantener la recepción desconocida |
| Recarga cercana coincide con explosión/disparos | «Posible señal de recarga, con enmascaramiento por otros sonidos» | No asegurar escucha, recarga completa, munición o ventana segura para salir |
| Tres explosiones cerca de B | «Varias señales de explosión compatibles» si cada señal es elegible | No afirmar tres CT, tres lanzadores o su origen por la regla visual previa |
| Sonido de arma sin contacto visual | Clase de señal y procedencia aproximada si están acreditadas | El arma puede ser recogida; no identifica automáticamente bando o persona |
| No se registran pasos durante un avance | No hay señal elegible publicada | No prueba avance silencioso real, ausencia de enemigos o zona despejada |
| Rival inicia una desactivación fuera de visión | Sólo informe acústico elegible, fechado y estimado | La transición omnisciente no se entrega como conocimiento T por sí sola |
| Muerto en el tick de una emisión | Aplicar exclusión causal de nueva observación directa | Puede continuar como receptor compartido; no simular escucha de espectador no instrumentada |

## Dependencias y verificación tras el encuadre

- Capturar fuentes de emisión nativas con aliases locales, orden causal,
  posición y fuente; publicar ausencias y abstenciones. Comprobar realmente
  `Footstep`/`PlayerSound`, sus variantes y significado del radio en la demo
  aprobada. Registrar tipo desconocido, actor ausente y entradas corruptas.
- Fijar perfiles acústicos o abstenerse por familia donde falten alcance,
  oclusión/altura o procedencia. Datos positivos primero; la revisión experta
  no elimina estos gates. Una clasificación específica exige evidencia propia.
- Extensión optativa de proyección/input con contratos anteriores reproducibles;
  memoria, consultas y transporte compacto conservados. Compartir datos entre
  mapa y texto protegido sin convertir una señal anónima en contacto enemigo.
- Regresiones: límites de alcance, alturas/paredes, sonidos superpuestos,
  movimientos silenciosos sin callback, faltas de posiciones, alias oculto,
  duplicados, muerte/reinicio/retroceso y consultas futuras; regla visual de
  cinco segundos intacta, flashes siguen fuera de su deducción.
- Suites públicas pertinentes y una única partida aprobada tras pruebas,
  en TEMP nuevo, vacío y retenido. Repetir sólo esa misma partida si aparece
  un defecto. Productor real, conteos no vacíos por familia presente, causalidad,
  fuentes y contraste independiente; no declarar cobertura de variantes ausentes.
- Cerrar y documentar sonido antes de preparar interpretación general. Una nueva
  tanda Qwen, si se acuerda al cerrar, se fija completa antes de generar y se
  conserva con todos sus fallos. No generar ahora para aparentar progreso táctico.

## Preguntas pendientes

Q1: alcance y representación estimada de las familias de sonido, con abstención
cuando falte evidencia. Q2: criterio experto para pared/altura y enmascaramiento,
usando los dos primeros casos de la tabla. No se reabren criterios previos.
