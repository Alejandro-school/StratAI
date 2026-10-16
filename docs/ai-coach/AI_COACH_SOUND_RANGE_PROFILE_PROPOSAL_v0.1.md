# Propuesta de perfil de alcances sonoros v0.1

12/10/2026. **Pendiente de criterio experto; no implementada ni aceptada**.
Continúa el [encuadre espacial recibido](AI_COACH_SOUND_INFORMATION_SCOPE_v0.2.md)
sin reabrir 3D, direcciones, alturas, ausencia de FOV/bloqueo visual o comunicación
posible. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado.

La [auditoría real](../../outputs/stratai-sound-20261012/RESULTADO.md) obtiene
emisiones positivas y posiciones: 1.062 pasos, incluidos 502 de procedencia CT.
Los pasos no publican radio y `player_sound`, la fuente con ese campo, tiene cero
emisiones en esta partida. Los mensajes de arma incluyen recargas y utilidad,
sin radio explícito; los inicios SOS conservan parámetros empaquetados sin una
decodificación de alcance acreditada. No se concluye que esos bytes carezcan de
alcance o que otra demo/fuente tampoco pueda aportarlo.

Para avanzar se propone usar un **perfil de radios aproximados por familia**,
separado de los datos nativos. Este cambio de fuente/certeza necesita criterio
de juego antes de implementarse. Los valores iniciales siguen **sin fijar**.
No se toma de una página externa un supuesto radio universal ni se convierte
una cifra propuesta en radio nativo verificado.

Primer alcance concreto para revisar: **pasos y recargas**. Ambos tienen
emisiones positivas recuperables; el vínculo de los mensajes de recarga con el
emisor aún debe contrastarse en el productor, sin resolverlo por cercanía
temporal. Las coordenadas de fuente/receptor serían nativas al callback y el
observador directo tendría que estar vivo. Las demás familias permanecerían
desconocidas mientras falte una emisión clasificada y un perfil admitido.

| Parámetro nuevo | Valor pendiente | Procedencia que se mostraría |
|---|---|---|
| Alcance de pasos | Criterio experto inicial o calibración reproducible | Estimación de perfil, separada de la emisión nativa |
| Alcance de recarga | Criterio experto inicial o calibración reproducible | Estimación de perfil, separada de la emisión nativa |

Ejemplo ilustrativo, no medición de esta partida: un CT produce un paso debajo
de B Apps. La distancia 3D puede estar dentro del radio que se adopte, aunque el
T mire en otro sentido. Sólo permitiría «recepción posible según el perfil».
Un CT fuera de ese radio no se convierte en silencio demostrado: el perfil es
aproximado. Sin radio acordado, la recepción queda desconocida.

El perfil no acredita escucha real, identidad reconocida, coordenadas percibidas
exactas, atención, enmascaramiento, número de rivales o seguridad al salir.
Tres pasos del mismo CT siguen siendo tres emisiones, sin deducir tres CT.
Los parámetros se versionarían con fuente, límites y revisión pendiente de
calibración; las pruebas recomputarían distancias y causalidad de forma
independiente. La revisión experta no sustituiría esa verificación factual ni
la evaluación de validez del perfil. Repetir únicamente la misma partida
aprobada en TEMP nuevo cuando exista productor, tras sus pruebas.

Interpretación general y Qwen permanecen pendientes/intactos respectivamente.
No hay recepción activa, nuevos inputs del modelo o nuevas lecturas por este
documento.
