# Información disponible en replay: contrato optativo v1

Encuadre del 10/10: [alcance y literal](../../ai-coach/AI_COACH_AVAILABLE_INFORMATION_SCOPE_v0.1.md).
Instrumento de desarrollo, sin modificar bundle 3.8, catálogo ni schemas canónicos.

- Proyección `stratai.team_replay_projection@9`, activada mediante el sexto
  argumento `{information: true}` del proyector sobre las fuentes factuales @8.
  La vía sin esa opción conserva sus resultados y versiones anteriores.
- Entrada `stratai.replay_context_input@7`; mantiene las guardas y redacción
  factual protegidas de @6. La lectura contextual continúa encuadrada en T;
  la proyección informativa y la interfaz se verifican en T/CT.
- `information.schema_id = stratai.replay_information@1`.
  `round_start_tick` separa instancias aunque se repita el número de ronda.
- `communication_model = immediate_simulated_possible_reception` y
  `reception_status = possible_not_confirmed`. Se supone comunicación inmediata
  para compañeros presentes, incluidos muertos como posibles receptores. No acredita voz, audición, atención
  ni conocimiento recordado. No se añaden radios, demoras u omisiones inventados.
- `own_team_state_scope = collective_demo_overview_not_individual_knowledge`:
  la vista colectiva de recursos propios/aliados no demuestra que cada jugador
  conozca esos recursos. Las fuentes de granadas y sonido siguen sus fases.
- `enemy_identity_scope = opaque_demo_tracking_not_recognized_person`: los IDs
  enlazan registros del parser y no acreditan reconocimiento de una persona.
  Coincidir un ID entre contactos no prueba una rotación percibida por el jugador.

## Observaciones, memoria y receptores

`observations` conserva posición fechada, observador, rival observado, estado
estimated/observed, fuente y disponibilidad. La fuente calculada exige @2,
productor world_occlusion_v2 y stride 4. La fuente confirmada conserva el gate
independiente del proyector; no hay fuente positiva confirmada en la demo real.
No se incorporan salud, equipo, orientación ni posiciones ocultas de rivales.

Cada observación tiene `recipient_basis`: fecha/referencia de la muestra propia
y los jugadores presentes. El observador exige vida en su fuente y ausencia de
una muerte previa; la muerte de un receptor no descarta comunicación simulada.
`recipients` contiene
el observador con `individual_observation/origin` y los compañeros con
`simulated_team_communication/possible`, cada uno con disponibilidad inmediata.
Un muerto no origina observaciones visuales nuevas; puede ser receptor posible.

Se conserva un informe reciente por observador/rival y los últimos informes
directos y compartidos que cada posible receptor podía tener por rival, por
separado. Un mensaje de recepción incierta no borra su observación directa.
`by_player` los referencia;
no demuestra que realmente los recibiera. Los IDs incorporan ronda, inicio,
fecha, observador y rival. Los registros no cambian después de su disponibilidad.
`known_team_players` conserva identidades propias con su primera muestra causal;
salir de snapshots no borra informes anteriores; la elegibilidad de recepción
posterior queda sin fuente en ese registro, sin afirmar que no pueda recibir
información mediante otra vía.

`contacts` consolida un rival por identidad, mantiene procedencias simultáneas
y separa `current_contact` de `last_observed`. Posición y antigüedad siempre
pertenecen a la observación. Una muestra sin visión no renueva el recuerdo;
ausencia de contacto no acredita ausencia del enemigo ni zona despejada.
No hay caducidad táctica universal ni movimiento supuesto. La baja conocida
retira su contacto/marcador, pero conserva la observación histórica.

`frame.last_seen_markers` es una representación derivada de los recuerdos;
contiene `knowledge=historical_contact` y `current_position_status=unknown`.
No contiene `alive`, recursos ni dirección; el mapa usa círculos huecos con
antigüedad, separados de jugadores vivos y sin objetivos de selección vivos.

## Consumo y texto

El consumidor comprueba fuentes, fronteras, IDs, elegibilidad de receptores,
consistencia de registros en el tiempo y correspondencia mapa/memoria/contactos.
La verdad del origen calculado conserva sus límites; un sello no demuestra
percepción humana. El smoke contrasta las derivaciones contra fuentes reales.

Hechos nuevos: `information_policy`, `enemy_information` y
`possible_player_information`. La zona del recuerdo se calcula sobre su posición
histórica usando la referencia geométrica ya acordada. No se convierte en zona
actual. Los textos se reconstruyen desde esos hechos y se verifican por igualdad,
manteniendo la distinción entre observación, estimación y recepción posible.
No se ejecutan nuevas generaciones de interpretación para este bloque; la
comparación sigue agrupada con granadas, tras su propio encuadre y cierre.

## Reparación técnica detectada durante el smoke

El registro previo de acontecimientos podía perder lanzamientos según el orden
de dos callbacks del mismo evento: creación del proyectil y captura factual.
La creación y el registro se ejecutan ahora en un solo callback, también para
los enlaces de humo/fuego. El fin de un fuego no reobserva su centro histórico.
Esto conserva `stratai.replay_events@1` y las condiciones de perspectiva ya
acordadas; no añade inferencias desde utilidad enemiga. Se verifica en ambos
órdenes de registro y contra el observador nativo de utilidad en la misma demo.
