# Referencias económicas históricas y comprobación — 3D.2

Fecha: 6 de septiembre de 2026. **Paquete y comprobador implementados y validados; compatibilidad efectiva aún pendiente.** Limpieza de esta salida: pendiente por rechazo automático; la limpieza anterior 3D.1 sigue separada. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) selecciona la continuación; la [evidencia](../../evidence/AI_COACH_ECONOMY_RULE_REFERENCES_SINGLE_DEMO_SMOKE_2026-09-06.json) conserva comandos, builds, conteos, hashes e inventario.

## Entrega

Se añadieron lector de los formatos de referencia, registro de fuentes fijadas, productor de paquete y comprobador contra observaciones nativas. Tres schemas nuevos llevan el registro runtime a **46**, con 19 motivos; catálogo `1.7.0`, export `3.8.0`, parser `v16`, quality `12` y manifest base `@3` conservados. No cambió el parser Go ni se promovieron salidas del smoke. [Contrato y comandos](../../contracts/economy/canonical-economy-rule-references-v1.md).

Se fijaron seis archivos de [GameTracking-CS2](https://github.com/SteamTracking/GameTracking-CS2): `steam.inf`, `weapons.vdata` y `gamemode_competitive.cfg` para dos revisiones. El [registro](../../../ai_coach/economy/rule_source_registry.json) conserva URLs inmutables y SHA-256. Las fuentes suman 1.057.392 bytes; son texto de referencia, no nuevas demos. Por compilación: 72 objetos, 71 precios explícitos y 36 defaults. La validación releyó 358 valores sin llamar al constructor del paquete.

Los [metadatos del 24 de agosto](https://raw.githubusercontent.com/SteamTracking/GameTracking-CS2/fd04856b8668f11cb31ed2b22574607dd9c365e0/game/csgo/steam.inf) identifican cliente/servidor 2000888 y revisión fuente 10936007. Los [del 25 de agosto](https://raw.githubusercontent.com/SteamTracking/GameTracking-CS2/acfe24d588d2df0a26da0f964e44d780bd3070eb/game/csgo/steam.inf) identifican 2000891 y 10937988. Comparten patch 1.41.7.7; son compilaciones distintas. Sus archivos económicos consultados tienen hashes idénticos entre ambas revisiones, lo que no acredita todos los cambios intermedios ni el vínculo exacto con la demo. La búsqueda por patch permanece como diagnóstico, sin aprobación.

## Smoke y resultado

Exactamente una partida aprobada de desarrollo: alias `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Una pasada del parser después de los tests, en destino TEMP nuevo y vacío. Los productores cambiados del paquete y comprobador se ejecutaron una vez cada uno sobre esa salida.

| Comprobación | Resultado |
|---|---|
| Entradas nativas contra referencias | 3.700: 3.599 iguales y 101 distintas |
| Diferencia de coste observada | Objeto 51: 350 frente al precio de referencia 1.000 |
| Configuración por instancia | 925 comprobaciones |
| Valores nativos presentes | 125 |
| Variables no capturadas por el contrato v2 | 775 casos: 31 variables por 25 instancias |
| Variable capturada pero ausente | 25 casos de `mp_maxmoney` |
| Defaults distintos | Freeze 20 frente a 15 en 25 instancias; dinero inicial 0 frente a 800 en una |
| Listas nativas completas / desconexiones | 1.465 / 2; cobertura contabilizada |
| Paquetes de reglas aprobados / errores del jugador clasificados | 0 / 0 |

Las 3.700 comparaciones no son compras únicas: una entrada puede persistir en varias capturas. Las 101 diferencias no prueban errores ni acreditan una fórmula de precio condicional. Los defaults no sustituyen configuración del servidor. Las 775 carencias pertenecen al contrato de extracción consultado; no demuestran ausencia de esas variables en la demo. La cobertura inicial se conserva: 77/250 actores, 173 ambiguos y tres equipos competitivos completos. v2 y 3C mantienen 50 fichas, 60 diferencias numéricas y 15 transiciones de inventario.

`npm run check:all` pasó: 88 pruebas frontend, 14 Node con tres omitidas, 43 Python backend, paquetes Go y build frontend. Las **416 pruebas del AI Coach** incluyen 40 nuevas; Ruff y contrato global pasaron. Parser: 88,152 s; paquete: 0,601 s; comprobador: 6,044 s. La validación independiente recalculó precios y configuración desde sus fuentes. La revisión semántica canónica completa terminó con cero errores en 300.284 s.

Los 44 bundles y 50 fuentes quedan protegidos: 4.356 metadatos de archivo y 181 hashes de control iguales. Las 19 evidencias previas y el catálogo conservan SHA-256. No se borraron ni modificaron los dos temporales 3D.1.

## Límites y cierre operativo

El cierre documental verificó 273 enlaces locales de 47 documentos, un único prompt funcional, las 19 evidencias previas y el catálogo intactos, y las fuentes funcionales iguales a la build validada. De 939 archivos previos inventariados, 930 permanecen sin cambios y nueve modificaciones pertenecen al alcance registrado. `git diff --check` pasó.

La identidad exacta del build, configuración efectiva, condiciones de precios, recompensas/bonus previo, loadout/límites y devoluciones continúan sin acreditación suficiente. Todas las puertas de recomendación, entrenamiento y producto permanecen cerradas. La continuación 3D.2B debe obtener las fuentes ausentes; volver a comparar defaults no resolvería el problema.

La nueva salida es `C:/Users/kerch/AppData/Local/Temp/stratai-3d2-smoke-_ko8hcsb`: limpieza pendiente por rechazo automático. Su inventario y hashes se guardaron antes del intento de retirada. Los dos destinos 3D.1 y su rechazo histórico siguen registrados en su propia evidencia, que no se modificó. La limpieza se trata como pendiente operativo separado del avance funcional solicitado por el usuario.
