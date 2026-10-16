# Entradas económicas y vida de objeto v1

Contratos optativos `stratai.economy_entry_observations@1` y `stratai.economy_entry_identity_links@1`, versión `1.0.0-candidate.1`. El selector **10** conserva los modos 1–9 y añade evidencia de ejecución. El registro tiene **57 schemas y 19 razones**; catálogo `1.7.0`, export base `3.8.0`, parser `v16`, quality `12` y manifest `@3` conservan su significado.

## Captura común y procedencia

La lectura se ejecuta de forma síncrona dentro de `captureOwnership`, después de consultar dueño e inventarios y antes de avanzar el contador de fotogramas. No se entrega otro callback ni se avanza el parser entre ambas lecturas. Se conserva el corte de `FrameDone` definido en [tenencia v1](canonical-economy-ownership-v1.md). En el [SDK fijado v5.2.0](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/parsing.go), ese cierre sucede después de los mensajes encolados del comando y sus eventos diferidos. No significa subtick del servidor ni cierre de todos los comandos que comparten tick.

Cada corte de tenencia tiene exactamente un marcador de entradas. El marcador referencia la secuencia del corte; cada estado referencia además el estado exacto de tenencia, el primer corte y las listas de todos sus pawns vivos, en el mismo orden. Sólo se comparte almacenamiento cuando otra lectura fresca devuelve el mismo payload y el mismo estado de tenencia. Cambios de actor, vida, tenencia, campos o referencias separan estados; no se rellenan cortes anteriores. La causalidad y las fronteras estrictas se verifican mediante la cadena de tenencia → identidad → traza → candidato v2 → canónico.

`economy_entry_observations.json` fija nombre, bytes y SHA-256 de `economy_ownership_observations.json`. El informe fija a su vez el hash de las observaciones de entradas. No se une la fuente histórica 3D.1 por tick, cercanía o tipo de arma: sigue siendo un control separado.

## Valores y alcance

La propiedad raíz es `m_pBuyServices.m_vecSellbackPurchaseEntries`. Se conservan longitud e índice de cada entrada, incluidos duplicados. Una lista vacía requiere raíz observada. Raíz ausente, tipo no admitido, longitud superior a 256 y miembros parciales mantienen estados explícitos. No se consultan índices inactivos.

Cada miembro —`m_unDefIdx`, `m_nCost`, `m_nPrevArmor`, `m_bPrevHelmet`, `m_hItem`— conserva estado, tipo escalar entregado por el SDK y valor. Se admiten booleanos y enteros dentro de la sonda de 32 bits; otros tipos/valores quedan explícitamente sin interpretación y sin copiar payloads arbitrarios. Cero, `false`, `-1` y `4294967295` no se normalizan. Casco codificado como entero 0/1 conserva ese entero. La validación semántica comprueba por separado los rangos de cada miembro.

`m_nPrevArmor` es metadato nativo; no es armadura física reconstruida. El coste corresponde a esa entrada observada: no es precio general, importe efectivamente pagado ni reembolso. Los handles se contrastan por índice, serie completa, clase y vida disponible en el corte. Un sentinel inválido no identifica una entidad; cero presente se comprueba como referencia y no se convierte en ausencia. Si hay resolución, se lee también la definición nativa del equipo para contrastar ambos canales sin paquete de precios externo.

## Informe y abstenciones

Los registros corresponden a **entrada por estado distinto**, con instancia, primer corte, fotograma, tick, pawn, actor/lado observados, índice de entrada, alias de vida cuando resuelve, estado de referencia, tenencia, multiplicidad y coste registrado. El campo `entity_alias` puede existir sin vínculo exacto: describe la resolución del handle, no la atribución económica completa. El origen permite consultar los cinco valores originales mediante estado, pawn e índice.

| Estado principal | Condición |
|---|---|
| `exact_link` | Listas completas, miembros válidos, referencia única por vida, actor/lado únicos, definición coincidente y tenencia bilateral del mismo pawn y actor en el corte. |
| `partial_list` | Alguna lista está ausente/parcial/fuera de sonda, o un handle no interpretable impide comprobar exclusividad. |
| `invalid_member_value` | El escalar se conserva, pero no cumple el dominio del miembro. |
| `identity_ambiguous` | Entrada duplicada, varias entradas resueltas a la misma vida o actor no único. |
| `special_entry_without_entity` | Handle sentinel inválido; no se acredita entidad. El nombre no clasifica el objeto como armadura ni explica el sentinel. |
| `reference_unresolved` | Handle sin vida disponible, serie distinta o clase incompatible. |
| `actor_unavailable` | Falta actor o lado en ese estado. |
| `definition_unavailable` / `definition_disagreement` | Falta definición nativa interpretable del equipo o discrepa con la entrada. |
| `ownership_disagreement` | Dueño o inventario señalan otro pawn. |
| `ownership_unaccredited` | No existe coincidencia bilateral suficiente con el actor de la lista. |

La matriz conserva oportunidades sin vínculo y oportunidades de instancias sin enlace competitivo. Un vínculo sólo se asigna a una oportunidad si también pasa el enlace competitivo y roster de tenencia. No requiere ni concede T0 precompra. Los motivos de referencia, tenencia, presencia y duplicidad siguen visibles aunque prevalezca otra abstención principal.

No se usa el índice de lista como ID persistente entre cortes. Una entrada añadida o retirada no acredita por sí sola compra completa, devolución efectiva, drop voluntario, regalo o cooperación. El informe exige `cause=not_accredited`, `refund_amount=null`, `decision_eligible=false` y cero transacciones itemizadas verificadas. `comparison_allowed`, `training_allowed` y `product_supported` son siempre `false`.

## Ejecución y validación

El productor usa `--ai-coach-economy-candidate --economy-candidate-version 10`, fuente aprobada, alias opaco, checksum, build y TEMP nuevo/vacío. Desde la raíz:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.entry_identity --source $entrySource --output $newEmptyTemp --require-observed
```

El materializador rechaza destinos no temporales, no vacíos o con enlaces antes de leer la fuente. El lector comprueba schema, cadena de hashes, prefijo causal, cobertura exacta de cortes/pawns, índices, series, tipos y estado común. El validador del informe recalcula filas y matriz desde las fuentes. El gate de cierre exige pruebas, una única partida aprobada, recálculo independiente y validación canónica completa. Los casos sintéticos no acreditan que esa variante aparezca en la demo.

Por indicación del usuario de 8 de septiembre de 2026, las salidas temporales se conservan con inventario. No se intenta borrarlas ni se solicita revisión de limpieza; esa retención no bloquea el cierre funcional. Fuentes, 44 bundles y temporales anteriores permanecen de sólo lectura.
