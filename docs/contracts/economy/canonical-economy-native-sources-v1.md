# Fuentes nativas de compras y reconstrucción por objeto

Contrato `1.0.0-candidate.1`, 6 de septiembre de 2026. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene el estado y siguiente acción. Esta extensión optativa conserva el [candidato económico v2](canonical-economy-candidate-v2.md) y el [contrato de oportunidades 3C](canonical-economy-opportunities-v1.md). Su contenido es evidencia de ejecución; no sustituye el estado inicial ni acredita decisiones elegibles.

## Fuentes observadas

| Fuente | Qué conserva | Qué no demuestra |
|---|---|---|
| `CDemoFileHeader` | Presencia y valor de `build_num` y `patch_version`, copiados con tick y procedencia. | Compatibilidad de precios, recompensas o reglas de otra versión. |
| `CSVCMsg_ServerInfo` | Protocolo de red observado. | Identidad equivalente a build del juego o a versión del extractor. |
| `m_weaponPurchasesThisRound.m_weaponPurchases` | Lista nativa con ID de objeto y contador. | Instante exacto de cada transacción ni una cesta completa. |
| `m_weaponPurchasesThisMatch.m_weaponPurchases` | Contador nativo por objeto en la partida, como referencia separada. | Historial legal previo de otras partidas ni motivo de una compra. |
| `m_vecSellbackPurchaseEntries` | ID de objeto, coste registrado, armadura/casco anteriores y handle nativo del objeto. | Precio general de tienda, disponibilidad del loadout o importe efectivamente reembolsado. |
| Dinero y gasto nativo | Valores en la misma captura; los límites se contrastan con v2. | Correspondencia automática entre saldo y cada compra/recogida. |
| Eventos genéricos económicos | Conteos de nombres permitidos para comprobar si existe `item_purchase` en la fuente. | Compras inferidas a partir de `item_pickup`. No se retienen payloads con identidades. |

Los nombres de propiedades se contrastaron con las [definiciones extraídas del juego por cs2-dumper](https://github.com/a2x/cs2-dumper/blob/main/output/client_dll.hpp). Sirven para localizar campos; sus offsets y la versión actual del repositorio no acreditan presencia ni compatibilidad en una demo histórica. La extracción usa propiedades con nombre de la demo, no offsets de memoria de una instalación actual.

Además, `ItemRefund` de [demoinfocs v5.2.0](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/datatables.go) se deduce de destrucción de entidad, zona de compra y aumento de dinero en el mismo tick. Es una señal de la librería, no un evento nativo con importe de devolución. Se conserva el límite ya vigente de v1/v2: un callback no prueba transacción ni regla económica.

## Captura y disponibilidad

Se leen las fuentes en cada fotograma de freeze y en su fotograma final. Se guarda una nueva fila cuando cambia algún valor, presencia, longitud de lista, lado o conexión, y en los límites de ronda/freeze. No se guarda una fila nueva sólo porque cambie el reloj. Cada fila retenida conserva su tick de observación y disponibilidad; no se mueve a un tick posterior ni se usa para rellenar un T0 ausente.

La lista vacía sólo está observada si existe la lista nativa. Una raíz ausente produce `unavailable`; una longitud no interpretable o superior a 256 produce abstención; miembros ausentes producen `partial`. Cero y `false` requieren una propiedad nativa presente. Las capturas copian los valores y mantienen los alias locales de participantes de v2. Un handle de objeto es un identificador técnico de entidad, no un identificador de persona.

Se conserva la información del lado de cada actor. La reconstrucción agrupa exclusivamente por instancia y actor; no compara el contador de un jugador con otro ni cruza reinicios de ronda. Los datos siguen separados de las features iniciales de 3C. Los cambios que sucedan y se anulen dentro de un mismo fotograma pueden no quedar representados: no se garantiza una lista completa de transacciones intermedias.

## Reconstrucción permitida

`m_nPrevArmor` se conserva como metadato nativo, incluidos valores especiales con signo o su codificación de 32 bits sin signo. No representa necesariamente una cantidad física de armadura; no se normaliza ni se usa para calcular una devolución.

El informe conserva una fila por captura, incluidas las abstenciones. Entre dos capturas del mismo actor, conectado y en el mismo lado, compara listas nativas completas:

- **Contadores:** valor anterior, posterior y diferencia por ID de objeto. La ausencia del ID en una lista completa representa ausencia de contador listado; no se aplica a una lista ausente. IDs duplicados o inválidos bloquean la diferencia. Una disminución se conserva como disminución; no se etiqueta automáticamente como refund.
- **Entradas de devolución:** multiconjunto de entradas añadidas y retiradas, con su coste nativo. Se respetan duplicados y cambios de orden. Una retirada no prueba devolución; puede haber pérdida de elegibilidad o cambios de la lista. El coste registrado no se asigna a otras armas ni a alternativas hipotéticas.
- **Tiempo:** intervalo entre observaciones, sin orden de subtick. Capturas en el mismo tick, desconexión, cambio de lado o miembro ausente producen abstención. El fin de freeze y las rondas sin enlace competitivo quedan separados del interior competitivo.

`refund_amount=null`, `basket=null` y `decision_eligible=false` son obligatorios. Se cuentan diferencias por objeto, entradas y capturas; no se presentan como el número de consejos o de compras completas correctamente evaluadas.

## Compatibilidad de reglas

La evaluación distingue versiones observadas, ausentes y conflictivas. Una versión observada permite identificar una fuente; no hace compatible un paquete de reglas. Se mantienen bloqueos explícitos para vínculo con assets de precios versionados, recompensas/bonus previo, loadout/límites de compra, refunds y cooperación. El coste de una entrada observada no sustituye un precio general ni prueba que otra cesta fuese legal.

Este contrato no importa reglas por fecha aproximada, por protocolo de red ni por un hash legado sin vínculo demostrado. `comparison_allowed=false`, `training_allowed=false` y `product_supported=false` permanecen obligatorios. El informe no tiene un canal de aprobación implícita para modelos o recomendaciones.

## Artefactos y comandos

| Artefacto | Schema |
|---|---|
| `economy_native_sources.json` | [stratai.economy_native_sources@1](../../../ai_coach/contracts/economy_native_sources.schema.json) |
| `economy_native_sources_manifest.json` | [stratai.economy_native_sources_manifest@1](../../../ai_coach/contracts/economy_native_sources_manifest.schema.json) |
| `economy_native_reconstruction.json` | [stratai.economy_native_reconstruction@1](../../../ai_coach/contracts/economy_native_reconstruction.schema.json) |

El manifest de fuentes vincula por hash y tamaño los datos y el manifest v2 generado en la misma ejecución. La reconstrucción vincula por SHA-256 ambos artefactos de origen y v2. El registro runtime tiene 43 schemas; catálogo `1.7.0`, IDs, elegibilidad, export `3.8.0`, parser `v16`, quality `12` y manifest base `@3` mantienen su significado.

El selector económico `3` emite v2 más las fuentes nativas. El valor por defecto sigue siendo `1`; los modos `1` y `2` conservan su salida. Con identidad aprobada, `$economyRunRoot` nuevo y vacío bajo TEMP y las variables del contrato v2:

```powershell
# Desde backend/go-service; salida validada antes de leer la demo.
go run ./cmd/canonical-export --demo $approvedDemo --output $economyRunRoot --alias $sourceAlias --checksum $sourceSHA256 --maps-root ../data/maps --build-id $buildIdentifier --ai-coach-economy-candidate --economy-candidate-version 3

# Desde la raíz; $nativeReconstructionRoot es otra carpeta TEMP nueva y vacía.
backend/venv/Scripts/python.exe -B -m ai_coach.economy.native_reconstruction --source $sourceNativeManifest --output $nativeReconstructionRoot --require-observed
```

El lector de fuentes valida primero v2, schema, hashes, identidad, orden, límites, disponibilidad y coincidencia de dinero/gasto en las capturas compartidas. El validador de reconstrucción recalcula las diferencias y membresías desde las fuentes sin llamar al constructor. `--require-observed` exige capturas, verificaciones de fotograma y observaciones de versión no vacías; no garantiza por sí solo contadores positivos o reglas compatibles. Esas coberturas deben constar por separado en la evidencia del smoke.
