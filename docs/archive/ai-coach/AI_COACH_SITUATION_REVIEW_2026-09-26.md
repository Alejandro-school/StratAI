# Revisión centrada en una situación — 26 de septiembre de 2026

El usuario cuestiona por qué debe recorrer toda la ronda para interpretar una
situación y solicita: «Por tanto, hay que modificar el html no?». Se corrige la
presentación del instrumento existente; no se abre una fase táctica ni se cambia
la unidad de revisión acordada en TP10. El corte final ya era la situación objetivo.

## Entrega

- 34 fichas abren pausadas en el instante objetivo, con ronda y tiempo transcurrido
  mm:ss destacados tanto encima del mapa como junto al formulario.
- «Ver contexto anterior» despliega reproducción y trayectorias. Saltar a una baja
  abre ese contexto; cerrar o volver restaura el corte y detiene la reproducción.
- Un envío desde el pasado conserva el texto y pide volver al corte. Los tres
  campos y el vínculo servidor caso/episodio/contexto/hash siguen intactos.
- Instrumento `stratai.review_web@5`, publicado en el mismo sitio e invitaciones.
  Pestañas @4 solicitan recarga; las respuestas históricas conservan su instrumento.

## Comprobaciones

`npm run build` y `npm test` en el proyecto web: 18 pruebas correctas, incluidas
entrada/salida del contexto, saltos desde bajas, pausa y protección del envío.
Los 34 HTML cifrados se descifran sólo con invitación y preservan los tres bloques
JSON originales. El suplemento Kill conserva sus 34 hashes. Se verificaron
34/34 páginas alojadas por HTTP contra los hashes del build y sus etiquetas de
instante, acceso, guardado, revisión e idempotencia. Los 12 registros anteriores
conservan su hash; se añadieron sólo pruebas técnicas 13 y 14 (`is_test=1`).

No se hizo inspección visual en navegador. La primera comprobación alojada del
texto falló por leer el JSON local con la codificación Windows predeterminada;
se repitió con UTF-8 explícito, sin cambios al producto ni otra publicación.

No se modifica parsing, exportación, hechos derivados, schemas ni contratos de
datos de replay. No corresponde repetir la demo para este ajuste de presentación.
No cambia ningún criterio de juego de IC31 ni se añade una fila al Excel v22;
el cambio de instrumento queda registrado en TP10 v0.13 y plan v2.96.

El detector sigue congelado y no consume el suplemento por bajas; los envíos no
entrenan un modelo. La siguiente acción es recoger las lecturas del instante
marcado, conservando carencias y diferencias de información al contrastarlas.

[Evidencia](../../evidence/AI_COACH_SITUATION_REVIEW_2026-09-26.json).
