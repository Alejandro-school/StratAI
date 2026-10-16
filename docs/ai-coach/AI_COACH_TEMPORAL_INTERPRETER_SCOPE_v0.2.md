# Alcance del intérprete temporal ICT-I1

21 de septiembre de 2026. Versión 0.2, con el encuadre experto recibido. Concreta la [propuesta v0.1](../archive/ai-coach/AI_COACH_TEMPORAL_INTERPRETER_PROPOSAL.md), que se conserva como versión presentada. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) fija el estado y la siguiente acción. **Encuadre recibido. El componente descriptivo ICT-I1.A se ha implementado; el plan conserva el estado vigente de cada parte.**

## Criterios recibidos y alcance

El [recibo literal](../evidence/AI_COACH_ICT_I1_EXPERT_RESPONSE_2026-09-21.json) conserva los cuatro puntos del usuario, sus erratas y las dos entidades de espacio del mensaje. La siguiente tabla es síntesis del agente. Describe criterios generales de diseño, no cuatro sucesos detectados en una partida ni una nueva respuesta a K2.

| ID | Aportación del experto | Aplicación y excepción |
|---|---|---|
| ICT-I1-E01 | Un aliado puede ver más defensores en B de los esperados, o percibir indicios por la utilidad utilizada, haciendo más complicada la entrada. | Reconsiderar la lectura y las alternativas con información atribuida. «3, lo normal son 2» es el ejemplo del experto, no una regla global de composición. Mantener separados contacto directo, indicio y estimación. |
| ICT-I1-E02 | Si hay un compañero en medio que consigue una baja, podría convenir aprovechar el espacio y cambiar de ruta. El experto exceptúa el caso en que todo el equipo está en B. | Conservar la distribución y la oportunidad concreta del compañero. No convertir una baja en prueba de ruta libre, control completo, llegada a tiempo o alternativa superior. La excepción no prohíbe toda rotación desde B en otros contextos. |
| ICT-I1-E03 | La presión y utilidad de dos jugadores en B pueden ser un fake para inducir rotaciones CT y facilitar A. | Conservar el engaño como posibilidad compatible con la secuencia. Distinguir presión observada, intención inferida, reacción CT desconocida y accesibilidad física de A. |
| ICT-I1-E04 | La muerte de quien buscaba información puede hacer reconsiderar el plan. | Separar baja observada y función de información inferida. Esa función necesita contexto anterior; no se asigna sólo porque el jugador murió o estaba separado. |

La respuesta concreta **motivos posibles para revisar hipótesis**. Sus expresiones «puede», «es posible» y «se podría» no establecen una orden automática de cambio. Tampoco definen una política exhaustiva para elegir el plan óptimo o retirar definitivamente cualquier lectura anterior. Los factores pueden coexistir y mantener ambigüedad.

## Traducción técnica del agente

Para cada motivo se registrarán señal, actor/receptor, lugar y tiempo disponibles, procedencia, vínculo con la lectura previa y desconocidos. Una observación nueva puede abrir una alternativa sin confirmar que el equipo la ejecuta ni que sea mejor. La ausencia de señal no confirma que no hubiera defensores, comunicación o intención de fake.

El número de lanzamientos sólo podrá usarse con su alcance observado. Convertirlo en número de defensores exigiría evidencia que vincule actores distintos, ventana temporal y ubicación, junto con la disponibilidad de esa información para la perspectiva. Esta condición de trazabilidad es una restricción técnica del agente, no una afirmación literal añadida al usuario. No se atribuye a todo el equipo lo visto por un compañero sin información suficiente sobre receptores.

K1/K2 siguen siendo el ejemplo conocido de continuidad ya revisado. Estos cuatro factores no se insertan retrospectivamente en sus hechos ni obligan a buscar un K3 que confirme cambio. Q1, Q18 y Q26 mantienen sus respuestas. Las 18 valoraciones de hipótesis anteriores siguen pendientes.

## Correspondencia con las fuentes actuales

Esta comprobación acotada usa contratos y fuentes existentes. El grafo de código no respondió (`Transport closed`), por lo que se leyeron directamente los archivos citados. No se ejecutó otra búsqueda de los originales ausentes de AUD-E01/E02 ni un nuevo parsing.

| Componente | Base comprobada | Límite para este intérprete |
|---|---|---|
| Corte, identidad y población | `ai_coach/support/preparation.py`, funciones `clock_ok`, `roster` y `select_pairs`, junto con los schemas C0. | Época, instancia, secuencia, lado y vida deben concordar. C0 incluye exclusiones; un actor ausente no equivale automáticamente a una baja. |
| Hechos propios/aliados | Prefijos support y uniones C0 exactas de los nueve contextos acreditados. `current_features` sólo proyecta posiciones, vida y disponibilidad del par seleccionado. | Puede prepararse un registro descriptivo con hechos observables y huecos. El par C0 no representa por sí solo a los cinco jugadores ni prueba conocimiento compartido. |
| Continuidad temporal | Cortes exactos de support con procedencia y reloj. | No alimentar el registro de T0 con los descriptores posteriores de ejecución del par C0. Reconstruir cada prefijo antes de resumir cambios. |
| Ver a varios defensores | [Contrato visual](../canonical-support-visual-contact-v1.md) y `support_preparation_visual_contact_cut.schema.json`: contacto legal `unavailable/null`. | No hay base en ese contrato para afirmar «vio tres defensores» ni para negar el contacto. Mantener la afirmación dependiente hasta disponer de una fuente atribuible y causal. |
| Utilidad o una baja como información | Un evento global de demo y su autor no acreditan automáticamente percepción/receptores ni zona semántica disponible para el juicio. | Antes de conectar E01/E02/E04, verificar el canal concreto, el prefijo, las identidades y su alcance. No hay en esta recepción una nueva fuente positiva que elimine esa comprobación. |
| Ruta, rotación CT y acceso a A | [Matriz de dependencias](AI_COACH_SUPPORT_C0_DEPENDENCIES.md); NAV parcial y límites de información rival. | E02/E03 pueden definir preguntas e hipótesis de desarrollo. No autorizan afirmar tránsito, espacio libre, rotación observada o superioridad. |
| Función e intención | Contexto e hipótesis revisables del [diseño colectivo](AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md). | «Busca información» y «hace fake» son inferencias. Una muerte o utilidad abundante aisladas no se convierten en etiquetas ciertas. |

## Primer alcance técnico delimitado: ICT-I1.A

Preparar el adaptador de prefijos propios/aliados y el registro descriptivo versionado, en el módulo de dominio propuesto. Este componente puede avanzar usando las fuentes acreditadas sin esperar a resolver toda la percepción o geometría. Debe preservar identidad, causalidad, precisión, exclusiones y desconocidos y presentar cambios de organización sin adjudicar intenciones.

Las cuatro aportaciones se conservarán como criterios de las hipótesis condicionadas y sus requisitos de evidencia. No se activará una regla positiva sólo con una etiqueta humana introducida como dato de entrada, con información rival omnisciente o con el nombre del caso. La parte dependiente de contactos, conocimiento, rutas o funciones se habilitará únicamente cuando su entrada esté justificada. El registro podrá explicar qué falta, sin inventar motivos o elegir una acción.

Este alcance no cambia el catálogo, S2–S3, elegibilidad, entrenamiento ni evaluación independiente. No requiere otra aprobación genérica del mismo enfoque. Si durante su implementación aparece un criterio táctico nuevo o una ampliación material, se volverá al experto sólo por ese punto.

## Verificación que deberá acompañar la implementación

- Igual prefijo y configuración deben producir el mismo registro al modificar futuro, resultados, respuestas humanas o datos rivales ocultos.
- Identidad incompatible, datos ausentes y discontinuidades deben conservarse como límites, sin convertirse en muerte, retirada o ausencia de adversarios.
- Los cuatro factores tendrán contrastes de presencia, ausencia de prueba y precondición incumplida. Para E02 se conservará expresamente el contraste con todo el equipo en B. Los casos sintéticos comprobarán la lógica; no acreditarán fuentes reales ni calidad táctica.
- E01 distinguirá contactos e indicios; E03 no inventará rotaciones; E04 no deducirá el rol de la muerte. Ninguno recomendará una ruta superior sin viabilidad y evaluación.
- K1/K2 servirán como desarrollo conocido, manteniendo la respuesta humana fuera de la entrada. Las lecturas anteriores no se reescribirán con hechos posteriores.
- Tras los checks públicos y pruebas afectadas, todo cambio funcional de hechos derivados, schemas o contratos ejecutará el smoke de exactamente una demo aprobada en TEMP nuevo, con conteos, causalidad, semántica y contraste independiente. Se retendrá la salida. Ese smoke no sustituye evaluación táctica independiente.

## Registro de esta recepción

El [Excel v6](../../outputs/stratai-ict-i1-feedback-20260921/delivery/StratAI_Plan_del_coach_ICT_I1_v6.xlsx) incorpora IC13–IC16 en «Contexto colectivo»: literal en D18:D21, síntesis en C18:C21 y límites del agente en E18:E21. IC08 enlaza esos criterios. Guía y Fuentes identifican la recepción. No se altera ninguna respuesta Q1, ficha K1/K2 ni valoración anterior.

El [informe de recepción](../archive/ai-coach/AI_COACH_ICT_I1_EXPERT_REVIEW_2026-09-21.md) registra preservación y verificación documental. Esta entrega incorpora criterios y delimita implementación; no crea un motor ni declara resueltas sus dependencias.

## Aplicación del alcance — 21 de septiembre de 2026

El [cierre ICT-I1.A](../archive/ai-coach/AI_COACH_ICT_I1A_PHASE_2026-09-21.md) verifica el adaptador y el registro descriptivo. No habilita reglas tácticas ni todos los contextos posibles; el [contrato ejecutable](../contracts/context/canonical-temporal-description-v1.md) delimita los campos proyectados y los límites. El encuadre y Excel v6 permanecen sin nuevos criterios.
