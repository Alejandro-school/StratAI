# Contexto de ronda y fichas de revisión v2

Contrato candidato 1.0.0, 2026-09-25. Aplica IC27 y la petición posterior «Hazlo»: excluir rondas de cuchillos del análisis e incorporar el reloj a las fichas. No cambia el detector táctico temporal, sus criterios ni las predicciones congeladas.

## Captura

`--ai-coach-support-objective-clock-candidate` captura los relojes nativos existentes y añade `support_round_context_observations.json`, contrato `stratai.support_round_context_observations@1`. El flujo de exportación normal permanece igual. El nuevo sidecar enlaza el SHA del soporte, la demo y la build; cada corte conserva frame, tick, instancia, epoch, sesión y frontera de mensajes. Sólo se retienen nueve convars de equipamiento inicial/C4 y tres propiedades públicas de las reglas. Valores vacíos, ausentes, nulos y asignaciones repetidas son distintos. Una nueva sesión invalida la configuración anterior; no se rellenan defaults.

## Elegibilidad causal

La regla `native_knife_configuration_and_inventory` requiere conjuntamente:

- `mp_give_player_c4=0` o `false` observado; armas primarias, secundarias y granadas iniciales vacías en ambos bandos; melee inicial `weapon_knife` en ambos.
- Prohibición nativa de compra de ambos bandos, calentamiento falso y bomba sin plantar.
- Inventarios completos, con slots y propietarios exactos, de todos los aliados vivos de un equipo de cinco identidades presentes; sólo contienen entidades `CKnife`.
- Ninguna señal de C4 en el prefijo de esa ronda.

Esto permite excluir una ronda configurada y corroborada como cuchillos. No garantiza reconocer todas las implementaciones de servidores/plugins. La ausencia de cualquiera de estas pruebas mantiene `unknown`; ni número de ronda, ni falta de C4, ni arma equipada bastan. Una contradicción entre configuración e inventario bloquea la clasificación.

`bomb_objective` exige calentamiento falso y evidencia propia/aliada de C4 o plantado público en el prefijo de la misma instancia/epoch/sesión. Se conserva durante una posterior ausencia del portador; no se arrastra a otra ronda. Este estado identifica una ronda con objetivo C4, no acredita su carácter competitivo/ranked. Calentamiento nativo positivo produce `warmup`.

Un tramo sólo es elegible si todas sus muestras son `bomb_objective`. Si todas son cuchillos/calentamiento, se excluye. Tramos mixtos o con incertidumbre quedan pendientes de contexto. No se consulta la respuesta humana para inferir el tipo de ronda. La exclusión experta de E01 se conserva separadamente como decisión del protocolo.

## Relojes y vistas

`stratai.contextual_review_packet@2` mantiene las posiciones, recursos, jugadores y C4 de v1, añade contexto por muestra y conserva los tres estados de alcance. El reloj de tramo se mide desde la primera muestra. El transcurrido de ronda se mide desde el evento público de fin de congelación, con ticks del parser; no es una cuenta atrás que compense pausas.

La cuenta atrás reutiliza `ai_coach.support.objective_clock`: origen de ronda nativo, intervalo y ticks de servidor alineados o simulación propia/aliada corroborada; requiere estado de pausa disponible. Después de plantar, el reloj de ronda deja de aplicar y se muestra el plazo nativo de bomba cuando está acreditado. No se sustituye un dato ausente por duración configurada menos transcurrido. El redondeo del HUD no está acreditado.

Las fichas v1 y sus plantillas siguen siendo reproducibles. Las v2 no muestran predicciones ni desenlaces. Las excluidas muestran una explicación y ocultan el mapa/análisis. Las pendientes permiten inspeccionar hechos pero bloquean la interpretación táctica.

## Uso y validación

Desde la raíz, con directorios vacíos ya creados bajo TEMP:

```powershell
.\backend\venv\Scripts\python.exe -B -m ai_coach.interpretation.review_packet_v2 materialize --source <captura> --output <TEMP-vacio> --side T --intervals 1:65 1118:2222 --radar frontend/public/maps/de_mirage_radar_psd.webp
.\backend\venv\Scripts\python.exe -B -m ai_coach.interpretation.review_packet_v2 validate --source <captura> --output <fichas> --radar frontend/public/maps/de_mirage_radar_psd.webp
.\backend\venv\Scripts\python.exe -B -m ai_coach.interpretation.review_packet_v2 analyze --source <captura> --output <otro-TEMP-vacio> --side T --intervals <inicio:fin> --map-vmap backend/data/maps/de_mirage/de_mirage_texto.vmap
```

`analyze` pasa exclusivamente los tramos elegibles al detector temporal. Conserva registros con interpretación nula para exclusiones/pendientes. La vía histórica `contextual_io` permanece para reproducir resultados congelados; los análisis nuevos deben usar la vía con elegibilidad. No se recalculan las predicciones primarias del piloto.

El lector v2 vuelve a calcular cada hecho, HTML, frontera y referencia desde las fuentes; rechaza mutaciones y sobreescrituras. Los schemas estrictos se añaden al registro runtime. La verificación incluye casos de falsa exclusión, presencia/nulidad, cambios de sesión, prefijos futuros, pausas, reloj ausente/desalineado y bloqueo antes del detector. El smoke real obligatorio utiliza exactamente la demo de E01, sin modificar el corpus ni promover salidas. Resultados y limitaciones reales se registran en el cierre de fase.
