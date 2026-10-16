# Mirage 3D — procedencia del recurso

Uso: prototipo local de la experiencia de landing de StratAI. El recurso no se utiliza para análisis, generación de hechos ni cambios de contratos del AI Coach.

- Geometría original de Counter-Strike 2: Valve.
- Recurso simplificado anterior (`mirage-native.glb`) preparado por el proyecto público [sliuhaob/mirage-utility-lab](https://github.com/sliuhaob/mirage-utility-lab).
- Modelo: https://raw.githubusercontent.com/sliuhaob/mirage-utility-lab/main/assets/mirage-native.glb
- Metadata: https://raw.githubusercontent.com/sliuhaob/mirage-utility-lab/main/assets/mirage-native.json
- Descarga: 15 de octubre de 2026. Ambos archivos se conservan sin modificar.
- El autor del recurso anterior documenta la extracción con Source 2 Viewer y preparación con Blender, con materiales simplificados. Ese GLB no contiene UV ni texturas y se conserva como referencia; ya no es el mapa que carga la escena.
- No es una web oficial ni una nueva licencia sobre los recursos de Valve. La preparación de esta demostración local no implica publicar el recurso ni la experiencia.

Los modelos CT y T proceden de los recursos existentes de StratAI bajo `public/images/Landing/`. El radar procede de `public/maps/`. No se han sustituido esos archivos.

## Recurso actual: exploración de A

La escena carga `mirage-exploration.glb`, preparado desde la exportación nativa sin compresión. Conserva las 277.445 caras originales de A, sus posiciones, normales, tangentes y UV; una comparación independiente verifica cada cara y atributo tras la compresión Meshopt sin pérdida de ese sector. Las zonas lejanas y la alternativa distante de A tienen menos geometría. Se comparten 500 imágenes KTX2 originales, con un máximo de 1024 px cerca de A y 256 px para las exclusivas del fondo.

Los colores de vértice destinados a variación/viento se excluyen de los materiales de follaje para evitar su interpretación como tintes rojos o negros en el material PBR. Se conservan las imágenes y máscaras alfa originales. La representación web sigue sin reproducir todos los shaders del motor de CS2.

Los scripts de preparación y verificación están en `scripts/landing-world/`; los parámetros, hashes e intentos retenidos están en `output/landing-world-exploration-20261015/PROVENANCE.json`. Este recurso pesa 71.413.376 bytes: recupera fidelidad local con una descarga mayor que la revisión anterior. Los recursos anteriores se conservan como referencias históricas. El VPK, los modelos de agentes y las capturas del producto no se modifican.

## Primera revisión texturizada (histórico)

- Fuente texturizada conservada: `mirage-textured.glb`. La revisión anterior utilizaba su derivado optimizado `mirage-lowpc.glb`. Geometría, UV y texturas originales de Valve, obtenidas de `maps/de_mirage.vpk` de la instalación local de Counter-Strike 2, en modo de lectura.
- Extractor: [Source 2 Viewer 20.1, CLI oficial de ValveResourceFormat](https://github.com/ValveResourceFormat/ValveResourceFormat/releases/tag/20.1). No se han generado texturas ni sustituido los materiales por imágenes de referencia.
- Fuente exportada: `maps/de_mirage/world.vwrld_c`, con materiales y adaptación de texturas a glTF.
- Preparación reproducible: `scripts/landing-world/prepare_textured_map.py`; elimina helpers y luces importadas y aplica el cambio de ejes documentado para esta versión del extractor. La escala de presentación es .02 por unidad Source.
- Optimización: glTF Transform 4.5.0, geometría Draco sin simplificar y texturas WebP de hasta 512 px. Se conservan las coordenadas UV. La iluminación y las sombras se resuelven en Three.js: el render web no reproduce todos los shaders, mezclas y lightmaps del motor de CS2.
- Los hashes, comandos y conteos se conservan en `output/landing-world-assets-20261015/PROVENANCE.json`. El export original y los registros quedan retenidos en ese directorio, fuera de los recursos públicos de la aplicación.

Las capturas de Coach, mapa, replay, estadísticas y retos y el vídeo `ChatIA.mp4` se reutilizan de `public/images/Landing/` y `public/videos/`, sin editar sus imágenes. El texto procede de `components/Landing/i18n/es.json` y el guion de la demo de `chatDemoData.js`.

## Primera optimización para equipos modestos (histórico)

`mirage-lowpc.glb` conserva los materiales y UV con geometría simplificada mediante meshoptimizer, agrupación por material y descarte espacial. Las imágenes originales se convierten a KTX2 (ETC1S color y UASTC normal/ORM, con mipmaps), sin sustituirlas por texturas generadas. Herramientas: glTF Transform 4.5.0 y KTX Software oficial de Khronos 4.4.2. Los scripts están en `scripts/landing-world/`; los hashes, parámetros e intentos retenidos están en `output/landing-world-optimization-20261015/OPTIMIZATION.json`. El VPK y el GLB texturizado de partida permanecen sin modificar.

El decodificador local `public/basis/` procede de Three 0.160.1/Basis Universal. Conserva sus licencias y WASM; su capa Embind utiliza cierres para funcionar con la CSP existente. La comparación de todas las imágenes/mipmaps con el codec original queda registrada en `basis-validation.json`.
