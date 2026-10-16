# Conciliación de flujos monetarios y entradas v1

Contratos optativos `stratai.economy_cashflow_observations@1` y `stratai.economy_cashflow_reconciliation@1`, versión `1.0.0-candidate.1`. El selector **11** conserva los selectores 1–10 y añade capturas e informe de ejecución. Registro runtime: **59 schemas y 19 razones**. Catálogo `1.7.0`, export base `3.8.0`, parser `v16`, quality `12` y manifest `@3` conservan su significado.

## Frontera y procedencia

La [traza v1](canonical-economy-transaction-trace-v1.md) entrega cuenta y gasto cuando cambia cada propiedad, pero no contiene una lectura fresca de ambos en todos los cortes de [tenencia](canonical-economy-ownership-v1.md) y [entradas](canonical-economy-entry-identity-v1.md). Los actores de entregas tempranas también pueden faltar. El selector 11 añade exclusivamente esa lectura común y la referencia nativa de pawn en cada entrega monetaria; las fuentes anteriores siguen intactas.

Los callbacks del proyecto se registran en orden explícito. Al recibir `FrameDone`, la traza incrementa su contador antes del callback de identidad que captura tenencia, entradas y dinero. El corte usa todavía el fotograma terminado: **`trace_next_frame = cut.frame + 1`**. El prefijo de traza del marcador debe contener todas y sólo las filas con `frame <= cut.frame`. Una prueba del despachador comprueba esta relación en varios cierres consecutivos; el lector la exige en todos los cortes reales. No se equiparan contadores sólo por tener el mismo valor.

Las lecturas frescas de `m_pInGameMoneyServices.m_iAccount`, `m_pInGameMoneyServices.m_iCashSpentThisRound` y `m_hPlayerPawn` se ejecutan dentro del mismo callback que captura entradas, sin avanzar el parser entre ellas. Cada corte conserva marcador, secuencia del estado de entradas y prefijo de traza. Sólo se comparte almacenamiento si las lecturas y el estado de entradas son idénticos. Un par omitido del informe por payloads idénticos y falta de entregas se contabiliza como omisión de almacenamiento; si sus datos eran desconocidos, sigue sin acreditarse ausencia de actividad económica.

El catálogo de controladores conserva alias local por vida del objeto SDK, índice, serie y fotogramas observados de creación/destrucción. Una sustitución del índice invalida la vida anterior, incluso sin aviso de destrucción. La captura exige controlador vivo y pawn asociado por handle completo, incluido todo el número de serie. El catálogo de controladores es una observación del productor, no un identificador de transacción ni una prueba externa de identidad.

`economy_cashflow_observations.json` fija nombre, bytes y SHA-256 tanto de `economy_entry_observations.json` como de `economy_transaction_trace.json`. Cada entrega monetaria de la traza debe tener exactamente una fila auxiliar con su secuencia y handle de pawn leído entonces. No se completa el alias de esa entrega usando capturas posteriores. El informe fija el hash de las nuevas observaciones y valida toda la cadena previa.

El SDK fijado cierra el comando tras sus mensajes y eventos diferidos. Estas fronteras acreditan orden de entrega y prefijos del parser; no acreditan subtick físico, orden relativo de todos los consumidores internos del SDK ni conocimiento del jugador. Los 14 archivos de SDK/despachador consultados se identifican por hash en la evidencia de fase.

## Cambios entre cortes

Se comparan únicamente cortes consecutivos de la misma instancia, fotogramas consecutivos, ticks de demo no decrecientes y muestras del reloj del mismo fotograma/época. Un salto de instancia, época, vida, controlador, actor o lado mantiene abstención. Cortes distintos con el mismo tick no se fusionan.

Las entradas completas se comparan como multiconjuntos de sus cinco campos crudos, incluyendo presencia y tipo. El índice permite consultar la fuente de cada extremo; no identifica una entrada persistente. Las multiplicidades, reordenaciones y cambios de metadatos quedan explícitos. Con duplicados, los índices conservados son representantes de ocurrencias equivalentes, sin afirmar cuál sobrevivió. La resolución y el estado de vínculo del objeto proceden del lector 3D.7 en ese mismo corte; un sentinel no adquiere una entidad.

Cada canal monetario conserva valores inicial/final, diferencia de capturas y todas las entregas seleccionadas. Un delta individual exige valor anterior disponible, valor actual interpretable y coincidencia de actor, lado, instancia y handle completo en la propia entrega. Baselines y entregas del fotograma de creación se marcan como posible inicialización/reenvío: no reciben delta transaccional. Pueden anclar lecturas posteriores, pero no reparan la comparabilidad del intervalo. Un alias ausente o una vida nueva nunca se rellena desde el futuro.

Con un canal completo se publican suma positiva, suma negativa en valor absoluto, neto y residuo entre última entrega y captura final. Se retienen entregas iguales y movimientos compensados aunque el neto sea cero. Cuando faltan datos, los totales quedan `null`, junto con motivos y deltas individuales que sí puedan verificarse. Cuenta/gasto observados con valor cero son distintos de ausencia. Un descenso de gasto se conserva sin decidir si fue reinicio, devolución u otra causa.

## Comprobación aritmética y límites

Se publica el coste neto registrado de entradas (`costes añadidos − costes retirados`) y, sólo con ambos canales y listas comparables, dos contrastes explícitos:

```text
account_delta_plus_entry_cost_net = diferencia de cuenta + coste neto de entradas
spent_delta_minus_entry_cost_net = diferencia de gasto − coste neto de entradas
```

Estas fórmulas comparan cantidades observadas. No son reglas acreditadas de precios o devoluciones; un residuo no se reparte entre objetos ni se elimina escogiendo otra regla. Una igualdad puede ser consistente con varias operaciones y no prueba pago, intención ni causa.

| Estado | Significado |
|---|---|
| `not_comparable` | Falta compatibilidad temporal, identidad, listas o canal monetario. Motivos separados y fuentes disponibles conservadas. |
| `no_change` | Intervalo retenido y comparable sin diferencias de entradas ni entregas; no se usa para las omisiones con datos desconocidos. |
| `deliveries_without_net_change` | Entregas conservadas y ambos contrastes a cero, sin cambios de entradas. |
| `counter_decreased` | Canal de gasto completo con alguna disminución; conserva residuos y movimientos, sin clasificar su causa. |
| `amounts_match_single_entry` | Ambos contrastes a cero y una ocurrencia cambiada con vínculo exacto; acredita consistencia aritmética. |
| `amounts_match_multiple_entries` | Ambos contrastes a cero y varias ocurrencias con vínculo exacto; no reparte las entregas por objeto. |
| `amounts_match_without_unique_object` | Ambos contrastes a cero, pero hay multiplicidad o algún vínculo de objeto no exacto. |
| `residuals_present` | Datos comparables, sin descenso de gasto, y algún contraste distinto de cero; cantidades sin asignación causal. |

La matriz incluye las 50 oportunidades de la fuente de control, incluidas las que no reciben intervalos. El enlace requiere la frontera competitiva y roster acreditados; contabiliza intervalos retenidos, que pueden ser no comparables. No mide cobertura de consejo ni recupera T0. Una frontera entre instancias puede figurar como intervalo no comparable asociado a la oportunidad posterior, sin atribuirle una operación económica.

Todos los intervalos mantienen `cause=not_accredited`, `refund_amount=null`, `decision_eligible=false`. Las transacciones itemizadas verificadas son cero; `comparison_allowed`, `training_allowed` y `product_supported` permanecen `false`. Saldo compensado, entrada retirada o igualdad exacta no demuestran compra, reembolso efectivo, drop voluntario, regalo ni cooperación. Precios/reglas compatibles, cesta completa y estado inicial no se reconstruyen desde ejecución posterior.

## Ejecución y validación

El productor exige fuente aprobada, alias opaco, checksum, build y TEMP nuevo/vacío, con `--ai-coach-economy-candidate --economy-candidate-version 11`. Desde la raíz:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.cashflow --source $cashflowSource --output $newEmptyTemp --require-observed
```

El materializador rechaza salida no temporal, no vacía o con enlaces antes de leer la fuente. Valida schemas estrictos, hashes, cobertura de cortes/entregas/pawns, prefijos completos, reloj, series y vidas disponibles. El validador recalcula el informe completo desde las fuentes; `--require-observed` exige registros afectados, sin convertir abstenciones reales en un fallo artificial de cobertura.

El cierre exige suites públicas y AI Coach, una única partida aprobada, recálculo independiente y validación canónica completa. Los fixtures cubren también variantes no presentes en esa partida; la evidencia separa ambas coberturas. Los temporales se conservan con inventario y hashes por indicación del usuario; no se intenta borrarlos ni se solicita revisión de limpieza.
