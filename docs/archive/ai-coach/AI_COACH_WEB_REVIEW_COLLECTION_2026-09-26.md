# Revisión web sin cuenta y con respuestas persistentes

Cierre del 26/09/2026. Se ha actualizado el sitio existente a petición del usuario: invitación sin cuenta, formulario en cada uno de los 34 casos y tiempos `mm:ss`. La vista del organizador permite consultar las lecturas por caso, con autoría declarada y versiones. Las respuestas se guardan en D1; los borradores locales no se presentan como enviados.

El acceso de plataforma permite visitas anónimas. La aplicación exige el enlace de invitación y reserva la consulta global a otro enlace del organizador. Los enlaces llevan secretos no incluidos en este informe. No se enviaron emails ni invitaciones a terceros. La posesión de la invitación concede acceso y permite reenviarla; no identifica a una persona. La sesión queda en este navegador durante 60 días; otro dispositivo crea otro revisor. La sesión del organizador dura 24 horas y se renueva con su enlace.

La primera lectura se conserva; las correcciones añaden registros. Cada envío vincula caso, revisor, exposición previa, literal, fecha del servidor, episodio/contexto y SHA de la ficha mostrada. Los reintentos son idempotentes. El invitado no puede consultar respuestas ajenas; el organizador filtra por caso y conserva el historial. Los registros técnicos quedan marcados y ocultos en la vista habitual.

Los relojes muestran minutos y segundos enteros truncados, manteniendo el valor original y los estados sin dato/no aplicable. Los 102 bloques JSON de las 34 fichas son idénticos al lote anterior. Permanecen excluidos E01 y ya revisado E02; no hay nuevos criterios de juego. Excel v20 y predicciones congeladas intactos.

Verificación: build Vite correcto, seis pruebas con SQLite real y 34 fichas verificadas; 34 respuestas HTML de producción cotejadas por SHA de su contenido antes del script añadido por Cloudflare. Pruebas sin sesión de plataforma: entrada por invitación, bloqueo sin enlace, aislamiento, guardado, corrección, reintento, consulta del organizador y persistencia tras redespliegue. Las pruebas dejaron seis registros técnicos, cero lecturas expertas. No se hizo QA visual/interactiva de navegador en este turno; preview local respondió 200.

La primera comprobación anónima recibió todavía el acceso anterior de plataforma. Se conservó el fallo, se publicó la misma versión bajo el modo de acceso solicitado y se comprobó el resultado final. La versión 3 oculta pruebas técnicas en la vista habitual. No se reprocesó ninguna demo: no cambió un productor, hecho derivado o contrato de replay. Todos los outputs y bases temporales se retienen.

Fuente: `outputs/stratai-review-shared-20260925/site`. [Evidencia](../../evidence/AI_COACH_WEB_REVIEW_COLLECTION_2026-09-26.json). La siguiente acción es recibir las lecturas en la web y contrastar sólo los casos respondidos, sin entrenamiento automático.
