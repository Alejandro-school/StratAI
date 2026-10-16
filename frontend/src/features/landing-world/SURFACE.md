# Landing experimental de Mirage

Ámbito actual: landing oficial en `/`, mediante `frontend/src/App.jsx` y la entrada de feature `index.js`. Modo: Persuade. Sustituye la landing anterior por instrucción del usuario del 16 de octubre. `landing-world.html` queda como entrada opcional de revisión. No modifica la definición ni el estado de las capacidades del producto.

## Direction contract

THESIS: descubrir StratAI entrando físicamente en una ronda. El mapa es el soporte de la narración y de las funciones.

OWN-WORLD: geometría tridimensional nativa de Mirage, piedra cálida, fachadas azul y salmón, horizonte verde petróleo; controles compactos oscuros, acentos menta, ámbar, lavanda y azul según función. Rajdhani para titulares, Inter para lectura y controles.

STORY: Mirage aéreo presenta el mensaje de producto; «Estoy listo» inicia un descenso cinematográfico a A. Al llegar, arrastre, rueda, giro limitado y controles accesibles permiten explorar el punto. Cinco destinos abren replay, mapa, estadísticas, Coach IA y retos. Capturas, vídeo, planes y FAQ conservan su contenido original y sus límites de propuesta.

FIRST VIEWPORT: titular grande a la izquierda, Mirage en perspectiva a la derecha, marca y navegación arriba, acceso a funciones abajo. La acción principal invita a entrar; la cámara responde suavemente al cursor.

FORM: dirección fijada por el usuario: entorno 3D profundo, entrada por botón y cámara suave con libertad limitada a A. Rueda y arrastre pertenecen a la exploración; los paneles conservan su scroll nativo. Ejecución por código, sin comp aprobado ni selección aleatoria.

SIGNATURE: vuelo hasta el encuadre abierto de A, seguido de cámara amortiguada con desplazamiento, giro y zoom. A conserva geometría original y mayor detalle de texturas; el fondo y la vista distante usan menor detalle. El T permanece en escaleras y el CT en punto. Abrir un destino acomoda la cámara; cerrarlo recupera la exploración anterior. Al detenerse vuelve la nitidez y cesa el render. Movimiento reducido conserva todos los destinos y controles.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance.

## Movimiento aprobado — 2026-10-16

La migración oficial mantiene este movimiento, el replay y sus anotaciones, los
recursos nativos y el contenido. Todos los estilos se limitan a `.landing-world`;
no bloquean el scroll global al pasar a una ruta privada. El HTML oficial carga
las tipografías necesarias. Durante la carga del mapa siguen disponibles el
acceso real a Steam y las funciones del producto. La sesión y el retorno al
historial o dashboard pertenecen al flujo existente del backend.

El usuario aprueba aplicar la salida hacia Steam y las transiciones entre
herramientas sobre esta misma base. El momento principal es un recorrido de
1,1 segundos desde la cámara actual hacia A/CT: desaparece la interfaz,
la escena se oscurece y aparece «Conectando con Steam» antes de seguir el
enlace de acceso existente. Escape cancela y devuelve el foco; al volver
desde la navegación se recupera la landing. No se simula una sesión iniciada.

Cambiar de herramienta coordina una salida de panel de 120 ms, entrada de
360 ms y arco de cámara de 560 ms, conservando los encuadres finales y el
contenido existente. Las selecciones rápidas conservan el último destino.
El movimiento reducido usa únicamente un fundido de 180 ms al acceder a
Steam y elimina estos desplazamientos de cámara. El presupuesto mantiene
el Canvas y los recursos existentes, sin bucles permanentes nuevos.

## Veracidad y fuentes

Refinamiento solicitado el 16 de octubre: paneles y capturas con mayor tamaño;
replay protagonista en un panel ancho, con vídeo y Coach lado a lado en escritorio.
Al abrirlo se reproduce el vídeo original silenciado. Se detiene en los fotogramas exactos de 1,3 y 6 segundos,
presenta los comentarios originales progresivamente y continúa automáticamente
con los tiempos del landing previo. Al finalizar conserva la conversación preparada.
En móvil se apilan vídeo y análisis, manteniendo accesibles los controles y el cierre.
La cabecera global no lleva línea inferior.

Los círculos rojo y amarillo se centran sobre las bolas del vídeo con posiciones
calibradas para cada pausa, incluida la posición distinta de corta en el segundo
análisis. El estilo y pulso originales se conservan; en móvil la etiqueta del
compañero permanece dentro del encuadre. Son anotaciones de esta demo preparada.

AI Coach continúa en desarrollo y evaluación. No se acredita funcionalidad nueva del producto. El mapa texturizado se ha extraído de la instalación local de CS2 mediante Source 2 Viewer y comprimido para navegador: geometría, UV y texturas nativas, con iluminación web propia. La fuente y transformación se registran en `frontend/public/landing-world/ATTRIBUTION.md`. Los GLB de agentes, capturas y vídeo ya existían en el proyecto. No se han generado imágenes.
