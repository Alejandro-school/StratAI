# Ampliación del mapa y aprovechamiento de la ventana — 28/09/2026

Petición: ampliar el mapa, ocupar el margen disponible y mejorar la distribución.
Se corrige el CSS de la web existente, dentro del alcance de interfaz autorizado.
La cuadrícula elimina el máximo de 1760 px y usa el ancho de ventana con márgenes
de 12–24 px. El radar llena su columna y conserva su relación 1:1; la columna se
dimensiona por ancho y altura de pantalla para mostrar el terreno completo.
Estado y formulario se mantienen alineados y accesibles al desplazar la página.
Se aumenta la tipografía de recursos en pantallas grandes y se conserva la
adaptación a dos columnas y móvil. C4 y formulario de tres campos no cambian.

Inspección local con Playwright a 2131×1083, 1366×768 y 390×844. A 2131 px el
contenedor ocupa 2131 px, el mapa empieza a 21,30 px del borde y su SVG mide
851×851 px. Las tres columnas empiezan a y=182,30 px; no hay desbordamiento
horizontal. Capturas de radar, terreno completo y móvil revisadas. Consola sin
errores. No se añaden tests que repliquen reglas de CSS: build y 23 pruebas
existentes pasan. Se conserva el hash de 106 entradas y los cinco JSON por caso.

Presentación @8, versión publicada 11. La revisión 10 fue intermedia; @8 añade
el marcador de versión para que una pestaña anterior solicite recarga al enviar,
recupere el borrador y no asocie su respuesta a otro HTML. Se comprobaron 34/34
fichas alojadas por hash, la invitación sin cuenta, el rechazo de @7 y las 36
respuestas previas por hash. No se crean respuestas técnicas ni expertas alojadas
en este cierre. El esquema D1 y claves de acceso/sesión se conservan.

No cambian productores, hechos, contratos de datos ni detector; no aplica smoke
de demo. Excel v23/IC32 se mantiene: no se ha añadido un criterio táctico.
Artefactos y pruebas SQLite se retienen. Continúa la recogida de lecturas humanas
con su instrumento/hash, sin entrenamiento automático.

[Evidencia](../../evidence/AI_COACH_FULL_WIDTH_MAP_2026-09-28.json).
