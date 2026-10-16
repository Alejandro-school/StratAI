# Entradas visuales en el corte support — fase 3E.10

Fecha: 12 de septiembre de 2026. Estado: extensión optativa implementada y validada; **contacto visual legal pendiente de fuentes suficientes**. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado vigente y la siguiente acción. [Contrato](../../canonical-support-visual-contact-v1.md) y [evidencia reproducible](../../evidence/AI_COACH_SUPPORT_VISUAL_CONTACT_SINGLE_DEMO_SMOKE_2026-09-12.json).

## Resultado y alcance

El nuevo selector `--ai-coach-support-visual-contact-candidate` captura entradas nativas en el mismo corte support y conserva todas las capas 3E.1–3E.9. Productor, consumidor, dos schemas estrictos y materializador/lector forman `1.0.0-candidate.1`. El registro pasa a **82 schemas/19 razones**; parser `v16`, exportación canónica `3.8.0`, calidad `12` y catálogo `1.7.0` conservan su significado.

Se acreditan las componentes consultadas por el accessor nativo de offset de ojos en **32.340 anclajes por rol** y un FOV de referencia estable en **370 por rol**. Esto no acredita la cámara renderizada ni la percepción del jugador. `flash_clear`, `smoke_clear` y `visual_contact` conservan **32.340 abstenciones cada uno y por rol**, con causas separadas. No se publican contactos positivos o negativos como información inicial legal; una fuente insuficiente no prueba ausencia física de contacto.

El rayo se limita a la malla estática seleccionada por el cargador existente, que incluye grupos `clip`, `grenadeclip`, `passbullets` y `window`. Sus materiales, transformaciones y cobertura no acreditan oclusión visual completa ni obstáculos dinámicos. La lista histórica de humo usa cilindros X/Y; las declaraciones nativas consultadas no aportan una evolución volumétrica completa. Duración y máximo de alpha de flash tampoco prueban deterioro visual actual. No se reinterpreta el ledger anterior ni `enemy_los` de 3.8.0.

## Fuente, causalidad y acceso

La captura conserva creación, reemisión inicial, actualización y destrucción; presencia, tipo declarado/nativo, valor previo y frontera exacta. Revalida pawn actual, handle completo, serie, vida y contexto propio/aliado utilizable. Los estados compartidos no cruzan instancia o época. Las seis componentes de celda se releen en cada corte como snapshots actuales, sin presentarlas como una traza independiente de todas sus entregas.

El origen de los rayos se reconstruye desde celda/offset y el accessor de ojos. El verificador de sólo lectura vuelve a calcular todos los segmentos desde los assets identificados por hash, con otra expresión aritmética para los endpoints y el mismo motor de intersección del proyecto. No valida de forma independiente ese motor ni acredita la suficiencia visual de la malla. El FOV rectangular de referencia 16:9 usa origen y destino nativos iguales y positivos, rango válido y scoped explícitamente falso; cero/default y transición conservan dependencias separadas. No se imputa FOV 90 ni una curva de transición.

`initial_features` valida únicamente el prefijo disponible y entrega copias de cinco hechos propios/aliados. El censo, geometría de enemigos, rayos y sus conteos permanecen en procedencia/diagnóstico interno. No se atribuyen a conocimiento subjetivo, voz, atención o intención. Se mantienen **32.340 anclajes/998 filas**, **30.678** con intervalo competitivo y **1.662 preliminares**; cero decisiones elegibles, comparaciones, recomendaciones, entrenamiento o soporte de producto.

## Validación medida

- `npm run check:all`: correcto, **16,963 s**. Suite AI Coach completa: **1.133 pruebas**, **41 nuevas**, **235,775 s**. Seis funciones Go nuevas; Ruff, vet, módulos y validador contractual correctos. Un primer pase AI Coach encontró una expectativa histórica de 80 schemas en una segunda prueba; se actualizó a 82 y se repitió toda la suite antes del smoke. No fue un defecto del productor.
- Una sola pasada de la misma demo aprobada `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, después de las suites: **132,087 s**. Build `working-tree-support-visual-contact-3e10-20260912-r1`; comando y hash del binario en la evidencia.
- Validador semántico canónico completo: **403,714 s**, cero errores. Materializador completo: **5738.573 s**; lector completo con recomposición de toda la cadena: **3826.178 s**. No se transfirió la validación de un consumidor anterior para sustituir estos recorridos.
- Fuente nueva: **275.567 entregas, 89.402 estados, 30.716 cortes y 213.897 referencias actuales de pawn**. Doce vidas creadas, dos destrucciones, once vidas capturadas y 25 dominios. Recálculo independiente de **2.138.970 campos entregados** y **1.283.382 componentes de coordenadas**; hechos iniciales y matriz completa idénticos al informe.
- Verificador de segmentos: **827.122 rayos**, **58.924 libres/768.198 bloqueados** en la malla del proxy, cero discrepancias. En los anclajes, **2.144 pares** tienen FOV de referencia evaluable, contrastado mediante otro cálculo angular. Por rol: 615 dentro/457 fuera y 138.796 sin FOV evaluable; estos conteos son internos y no acreditan contacto.
- Seis controles reales del acceso inicial cubren FOV observado, default y transición para ambos roles. Cambiar un valor del enemigo, invertir todos los flags internos de rayos e insertar datos futuros malformados no cambia las features iniciales. Los schemas rechazan contacto inventado tanto positivo como negativo.

Los datos reales incluyen valores cero, estados scoped/no scoped, varios FOV de destino/origen, tasas y tiempos de transición, duración/alpha de flash, creación, reemisión, actualización, destrucción e instancias separadas. Coordenadas nativas X/Y cero se conservan 541 veces cada una. FOV no observado: **29.397 dependencias de default/rango y 2.573 de transición por rol**. No se afirma que todos los casos de borde estén presentes en esta demo: geometría/campos ausentes, tipos inválidos/no finitos, límites y vueltas angulares, reutilización de handle/vida y corrupción del orden causal se comprueban con fixtures. El pitch equivalente por vuelta positivo es cobertura de fixture, no una variante real evaluada en este smoke.

## Conservación y retención

Los **72 artefactos canónicos** conservan SHA-256 y conteos. Las **nueve fuentes y once contextos previos** conservan hechos, selección, IDs/T0, matrices y clasificación. Se comparan las fuentes con 3E.9 R2 y los contextos materializados con 3E.9 R1, manteniendo intactas ambas procedencias originales. Los metadatos de build y referencias nuevas se registran como tales.

Inventario protegido de **44 bundles/50 demos**, **4.356 archivos/181 hashes**, sin cambios; **318 archivos** de salida/trabajo 3E.9 y las evidencias anteriores permanecen intactos. Los **487 archivos funcionales** coinciden con los bytes de la build probada. Catálogo, `AGENTS.md`, interfaz temporal y dependencias preservados; los cambios previos ajenos siguen en el worktree.

El cierre documental comprueba **451 enlaces locales en 46 documentos**, sin destinos ausentes, un único prompt y `git diff --check` correcto. De 671 archivos inventariados al inicio, 656 mantienen su hash y los 15 restantes pertenecen al alcance de código/documentación de esta fase; sólo se retira el prompt completado. Las 40 evidencias JSON anteriores conservan sus hashes.

Salida retenida en `C:/Users/kerch/AppData/Local/Temp/stratai-3e10-smoke-02o2gm9w`: **95 archivos/3111724212 bytes**, sellados antes y después del lector. Trabajo, binarios, helpers, logs y recálculos retenidos en `C:/Users/kerch/AppData/Local/Temp/stratai-3e10-work-q_q0h0h7`, con inventario SHA-256. Ningún borrado, solicitud de limpieza, descarga, adquisición, publicación, promoción, migración 44/44 ni cambio de `CURRENT`.

## Dependencia y siguiente intervención

Para acreditar contacto legal hace falta vincular a la demo geometría/materiales/transformaciones de oclusión visual y obstáculos dinámicos, evolución del volumen de humo, deterioro visual por flash en su reloj y FOV/óptica efectivos cuando intervengan defaults o transiciones. Cada vía requiere cobertura causal y comprobación independiente. Más muestras de los mismos proxies no sustituyen ese material positivo.

La siguiente intervención seleccionada es **3E.11: congelar el protocolo de oportunidades anterior a entrada/compromiso S2–S3**. El T0 de 3E.1 ya contiene un aumento observado de separación; no representa una decisión previa a entrar. La fase siguiente debe definir primero una oportunidad causal defendible, incluyendo espera/no entrada, sin selección por muerte, futuro o geometría enemiga oculta. Es una fase documental/de selección; cualquier cambio posterior del productor requiere una acción concreta propia. Contacto legal, NAV físico, velocidad instantánea, attachment transportado, observación legal de C4 caída, temporización completa, cooperación e intención y economía conservan sus dependencias.
