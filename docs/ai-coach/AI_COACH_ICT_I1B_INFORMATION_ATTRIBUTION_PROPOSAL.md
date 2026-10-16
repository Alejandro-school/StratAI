# ICT-I1.B — propuesta de atribución de información

22 de septiembre de 2026. Propuesta corregida tras la respuesta experta; no es un contrato
runtime ni una capacidad táctica habilitada. El [plan](../AI_COACH_IMPLEMENTATION_PLAN.md)
conserva el estado y la siguiente acción.

## Resultado que se propone conseguir

Separar una señal de información disponible para un jugador del estado global
que conoce el analizador. Primera vía candidata: **indicaciones de radar**,
relacionadas con el contacto visual que las origina y con su vigencia.
La corrección experta exige distinguir contacto actual y contacto perdido.
No se presenta el radar como una fuente independiente de avistamientos.
Su contribución a E01 sigue condicionada a la atribución y los datos.

La primera salida debe describir, cuando se acredite, el origen y la antigüedad
de la información de un rival, sin duplicarlo por figurar en varios canales.
Se retira la formulación anterior de «indicio adicional» sin explicar su origen.
La redacción táctica definitiva depende de los datos. No se propone concluir
«vio a tres», «todos lo sabían», «B está cerrado» ni recomendar una rotación.
Se conservan los criterios E01–E04, Q1/Q18/Q26 y K1/K2 ya recibidos.

## Fuente concreta localizada

Se ha leído la dependencia instalada `demoinfocs-golang/v5@v5.2.0`, fijada en
[go.mod](../../backend/go-service/go.mod). Las rutas, símbolos, líneas y hashes
están en la [evidencia](../evidence/AI_COACH_ICT_I1B_PREPARATION_2026-09-22.json).
El grafo MCP no respondió (`Transport closed`); se usó lectura directa acotada.

| Pieza | Hecho comprobado en el código | Alcance y comprobación pendiente |
|---|---|---|
| `common/player.go`, `Player.IsSpottedBy` | Consulta dos palabras `m_bSpottedByMask.0000/.0001` y relaciona un bit con `other.EntityID - 1`. | La propia biblioteca distingue spotted de LOS/FOV y advierte sobre múltiples observadores. La fórmula es una implementación que contrastar, no una prueba de receptor de HUD. |
| `datatables.go`, actualización de máscaras | Registra cambios y emite `PlayerSpottersChanged`. | Un cambio sólo señala el sujeto afectado; no acredita quién recibió una actualización de radar ni qué apareció en pantalla. |
| `s2_commands.go`, tabla Source 2 | Tiene una entrada para `CS_UM_ProcessSpottedEntityUpdate`. | Existe una vía concreta de decodificación en la versión instalada. No se ha demostrado que un mensaje de ese tipo esté presente en nuestras demos. |
| `msg/cstrike15_usermessages.pb.go` | Define `NewUpdate` y actualizaciones con índice de entidad, clase y componentes enteros `OriginX/Y/Z`, entre otros campos. | Presencia, escala, dominio, reloj, reinicio/delta y significado espacial deben comprobarse. No usar getters que conviertan ausencia en cero. |
| `msg/netmessages.pb.go`, `CSVCMsg_UserMessage` | El tipo inspeccionado contiene `MsgType`, `MsgData` y `Passthrough`. | Estos campos no constituyen una lista explícita de receptores. No adjudicar el mensaje a diez jugadores por estar en una demo de espectador. |

**Actualización tras la sonda B1:** se ha validado presencia real de estado nativo
de spotting en una demo, con recálculo de máscaras. No se ha acreditado el
vínculo con HUD ni receptores. La [evidencia nueva](../evidence/AI_COACH_ICT_I1B_SPOTTING_PROBE_2026-09-22.json)
y la [dependencia concreta](AI_COACH_ICT_I1B_RADAR_STATE_DEPENDENCY.md) delimitan
ese resultado. E01 sigue sin habilitar.

Como contraste, los eventos `Kill` del registro de combate acreditan sucesos;
su callback no demuestra distribución de información. Las áreas actuales de
tracking mezclan `LastPlaceName` con un callout calculado por `MapManager`.
Ese campo no se acepta como vínculo semántico de origen inequívoco. No se abre
otra campaña sobre esos dos caminos durante esta preparación.

## Afirmaciones que deben permanecer separadas

1. **Dato nativo:** propiedad/mensaje presente, tipo y valor crudos, entidad y
   momento de entrega. Ausencia y cero son estados distintos.
2. **Atribución:** el dato puede relacionarse con un jugador receptor, con slot,
   controlador, pawn/vida, lado y reloj compatibles. Un bit de spotting puede
   identificar un spotter sin identificar a todos los receptores del radar.
3. **Información disponible:** existe evidencia suficiente del canal y de su
   contenido para ese receptor. No acredita atención, mirada ni comunicación.
4. **Indicio espacial:** la ubicación comunicada, con su precisión y antigüedad,
   se relaciona con una zona relevante mediante una fuente compatible.
5. **Hipótesis:** el indicio puede modificar una lectura previa. No demuestra
   número total de defensores, intención, viabilidad ni superioridad de una acción.

La posición rival global del pawn no puede rellenar la ubicación de una señal.
Si sólo se acredita una máscara, se conserva como diagnóstico de máscara;
no se transforma automáticamente en coordenadas de radar. Si no se acredita
un receptor, no se crea una feature causal del jugador.

## Alcance de la comprobación B1 ejecutada

Alcance de ICT-I1.B1 ejecutado como diagnóstico nativo: una sonda acotada sobre **la
misma demo de desarrollo** `demo-a11eb0d600ee702c9edb`, sin activar una regla E01.
Los requisitos de HUD/receptor/codec/zona de esta propuesta siguen pendientes.
La sonda se limitó a capturar las propiedades/mensajes identificados y el contexto mínimo para
comprobarlos, en un TEMP nuevo, conservando las salidas.

- Registrar presencia y tipo antes de leer valores; comprobar el nombre real de
  las propiedades. La ruta de `IsSpottedBy` y la ruta anidada
  `m_entitySpottedState` observada en otro tipo de objeto no se dan por equivalentes.
- Retener frame, tick, época, orden de callback/entrega e identidad en ese punto.
  Un muestreo posterior no se atribuye retrospectivamente al tick del evento.
- Contrastar bits con una decodificación independiente y probar extremos de
  palabra, varios bits, cero, desconexión, sustitución/reutilización de slot,
  cambios de pawn, muerte y cambio de ronda/lado. No usar sólo el helper que
  originó el valor como verificador.
- Determinar qué aporta la perspectiva de grabación al receptor. Contrastar con
  una fuente independiente de entrega/presentación si existe. Sin esa prueba,
  mantener la atribución pendiente, aunque aparezcan mensajes no vacíos.
- Conservar las coordenadas del mensaje como crudas hasta verificar codec,
  dominio, unidades y momento de entrega. No utilizar la posición global rival
  para obtener una coincidencia circular.
- No inventar una duración de validez. Sin soporte para persistencia/revocación,
  limitar el hecho a la entrega acreditada; no arrastrar señales antiguas.
- Distinguir el contacto que origina la señal de su representación persistente.
  La explicación experta de punto rojo e interrogante no prueba que el decoder
  exponga esos estados. Si no pueden distinguirse, conservar estado desconocido.
  No interpretar «unos segundos» como un TTL numérico ni sumar ambos estados
  como rivales/contactos independientes.
- Antes de hablar de B, exigir un vínculo espacial de procedencia separada,
  compatible y con fronteras/solapamientos tratados. Un punto cerca de B o el
  lugar del observador no sitúa por sí solo al rival en B.

Esta comprobación puede terminar con diagnóstico nativo útil y receptor o zona
pendientes. No será un éxito táctico ni habilitará recomendaciones por acumular
registros. Si faltan las propiedades/mensajes, registrar el resultado y detener
esa vía; no probar otras demos hasta obtener un positivo.

## Ejemplos para revisar el criterio de juego

Ejemplos **hipotéticos**, no sucesos detectados en las cinco partidas:

| Caso | Información acreditada | Tratamiento propuesto |
|---|---|---|
| Contacto actual representado en radar | Se acredita el origen visual, el receptor, la identidad y la vigencia del dato. | Describir una misma información de contacto; no sumar visión y radar como dos rivales. |
| Actualizaciones repetidas | Varias entregas se refieren a la misma identidad/vida rival. | Una identidad, no varios defensores. No sumar mensajes como jugadores. |
| Múltiples observadores | Hay varios bits de spotting y no se resuelve su relación con receptores. | Diagnóstico nativo; no ampliar automáticamente el conocimiento a todo el equipo. |
| Contacto perdido | Según el experto, el punto pasa a interrogante durante unos segundos. Esa transición aún debe acreditarse en los datos. | Conservar el antecedente si está probado. No confirmar posición actual ni inventar duración exacta. |
| Máscara o señal ausente | El canal no está capturado, tiene cero bits o deja de informar. | No deducir zona libre ni ausencia de rivales. |
| Sin vínculo espacial | Se conoce receptor/entidad, pero la zona no está acreditada. | No activar la variante «más oposición en B». |
| Información de espectador | Hay posición global rival, sin entrega atribuible. | Excluirla de la interpretación del jugador. |

No se convierte «tres, normalmente dos» en un umbral universal. Los recuentos
describen identidades respaldadas por el canal, no el total real de defensores.
E02, E03 y E04 mantienen sus dependencias y excepciones previas.

## Corrección experta recibida

> Una persona solamente aparece en el radar cuando hay contacto visual. En forma de punto rojo, y cuando se pierde la visibilidad se pone un interrogante que dura unos segundos. Por tanto, eso que indicas es imposible

Literal conservado en el [recibo](../evidence/AI_COACH_ICT_I1B_EXPERT_CORRECTION_2026-09-22.json).
La pregunta anterior confundía el contacto del jugador analizado con el origen
visual de la información del radar. Se retira esa premisa y la alternativa
«radar o contacto visual»; no se vuelve a solicitar esa elección.

Síntesis del agente: separar observador, receptor y vigencia, además de distinguir
contacto actual y perdido. El usuario no ha fijado segundos exactos, confirmado
campos del parser ni aprobado una regla de mayor oposición. El aviso del SDK
de que spotting no equivale a LOS/FOV no prueba una mecánica sin origen visual.

## Verificación y entrega posteriores

La futura sonda/capa funcional requerirá pruebas de parsing/ausencia/tipos,
identidad y causalidad; invariancia al modificar futuro, rival global y etiquetas;
los checks públicos y el smoke de exactamente una demo posterior a tests.
Se conservarán comandos, hashes, tiempos, conteos, exclusiones y recálculo
independiente. No repetir el lote técnico de cinco partidas para esta preparación.

Sólo después de acreditar el canal y recibir el criterio correspondiente se
delimitará una regla condicionada y su evaluación. El entrenamiento sigue
pendiente de etiquetas, separación por partida y revisión independiente.
El Excel v7 incorpora IC17 con literal, síntesis y límites separados; v6 y todas
las versiones anteriores se conservan. La respuesta corrige el encuadre y no
certifica las dependencias factuales.
