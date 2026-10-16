# Preparación del contraste con el radar del jugador

22 de septiembre de 2026. **El contraste visual no se ha podido ejecutar:** no
hay una grabación del jugador identificada en el material consultado y no se
ha acreditado un cliente compatible que reproduzca su HUD. El
[plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente
acción. La [evidencia](../../evidence/AI_COACH_RADAR_HUD_CONTRAST_PREPARATION_2026-09-22.json)
delimita las lecturas y conserva hashes.

## Trabajo realizado

- Comprobar la instalación local mediante `steam.inf`: cliente/servidor
  **2000908**, patch **1.41.8.1**, SourceRevision **10981323**. Ha cambiado desde
  la auditoría del 06/09. La demo conserva build 10896/protocolo 14177. No se
  acredita equivalencia entre versiones ni se inicia el juego para presentar
  esa reproducción como el HUD original.
- Buscar vídeos en el ámbito acotado del repositorio, excluyendo dependencias,
  y en los archivos de primer nivel de la carpeta del juego. No se encontró
  una grabación de pantalla identificable como esta partida. No se inspeccionaron
  todas las carpetas personales ni se afirma que el usuario no la tenga.
- Leer la referencia histórica fijada `fd04856b...`. Su árbol completo contiene
  dos archivos con `radar` en la ruta: layout XML y estilos CSS. El layout
  define `EnemyOnMap` y `EnemyGhost` dentro de `CSGOHudRadar`; no contiene un
  script que vincule esos iconos a los cambios de máscara. Los estilos no
  acreditan recepción, condición de selección ni duración de un contacto.
  Es material reconstruido por Source 2 Viewer, no una vinculación exacta
  con los binarios de la demo. [Layout](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/game/csgo/pak01_dir/panorama/layout/hud/hudradar.xml),
  [estilos](https://github.com/SteamTracking/GameTracking-CS2/blob/fd04856b8668f11cb31ed2b22574607dd9c365e0/game/csgo/pak01_dir/panorama/styles/hud/hudradar.css).
- Preparar **seis cortes de revisión** desde la captura existente: activación,
  paso a cero, varios bits, reactivación, cambio de vida y cambio de lado. Se
  eligió el primer ejemplo cronológico que cumple cada condición, antes de ver
  cualquier HUD. Se conservan alias, entidad, reloj, ticks y filas exactas, además
  de los callbacks previos. Son ejemplos de estado nativo, no etiquetas visuales.

El [paquete de cortes](../../../outputs/stratai-radar-hud-binding-20260922/Cortes_para_contrastar_el_radar.md)
está preparado. La comparación visual, el receptor y el alineamiento con vídeo
permanecen nulos en su JSON. Los ticks del SDK no se convierten automáticamente
en tiempo del vídeo. No se ha generado ni inferido un interrogante.

## Material pendiente

Se preguntó al usuario si dispone de una grabación de pantalla de esta misma
partida hecha por un jugador y dónde está guardada. La respuesta sigue pendiente
al cerrar esta preparación. No es una petición de permiso: falta el material
independiente que se pretende comparar. La otra vía sigue siendo una fuente de
cliente con procedencia y compatibilidad suficientes, según la
[dependencia entregada](../../ai-coach/AI_COACH_ICT_I1B_RADAR_STATE_DEPENDENCY.md).

La documentación de [CS Demo Manager](https://cs-demo-manager.com/docs/guides/video)
describe generación de imágenes/vídeo mediante reproducción de la demo, pero
no certifica equivalencia con el radar original del jugador. No se instala ni
ejecuta esa herramienta para sustituir la prueba que falta.

## Verificación y alcance

Cero demos procesadas, cero binarios o paquetes del juego descargados y cero
modificaciones de runtime o contratos. No procede otro smoke funcional. El
archivo de observaciones conserva su hash y cada corte remite a filas existentes
en orden causal. El Excel v7 conserva el criterio experto y no se modifica.
No se vuelven a solicitar decisiones de juego ya resueltas.

El contraste no queda marcado como completado. Sin material nuevo no procede
repetir la sonda, ampliar a otras demos o hacer otra búsqueda equivalente.
