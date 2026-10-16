# Sitio restringido para varios revisores — 25 de septiembre de 2026

Publicado el [sitio de revisión](https://stratai-revision-tp10.kerch0k.chatgpt.site) con acceso **sólo al propietario**.
El usuario solicitó un enlace web restringido después del fallo del servidor
local. Falta recibir los correos de las personas a invitar; no se ha ampliado
el acceso ni enviado invitaciones. Compartir la URL por sí solo no concede acceso.

El sitio conserva los mismos 34 casos E03–E36 y el Excel vacío por hash. Cambian
sólo las instrucciones del índice y de entrega: cada revisor usa su propia copia,
se identifica con alias y fecha y declara exposición por caso. Entrega su archivo
a quien le invitó. No hay envío automático de respuestas desde la web.

Las primeras lecturas se recogen antes de consultar a otros revisores. Se conservan
literales, discrepancias y datos ausentes por caso/revisor. No se establece una
regla de consenso ni se multiplican los casos independientes por el número de
personas que opinan. No cambia el detector ni se entrena un modelo.

## Verificación y publicación

- Compilación estática y enlaces correctos; sintaxis de scripts y JSON incrustado comprobada.
- 34 fichas y hoja idénticas a la entrega anterior; paquete original y artefactos congelados intactos.
- Fuente del sitio aislada del repositorio principal: `642fef50aaf342026d35efc8580ee2e46eece119`.
- Archivo de despliegue con 39 archivos, cotejado con la fuente y la compilación.
- Publicación privada confirmada por Sites; política `custom`, un propietario, cero visitantes externos y cero grupos.
- Peticiones HTTP verificadas según la [evidencia](../../evidence/AI_COACH_RESTRICTED_REVIEW_SITE_2026-09-25.json); sin exponer credenciales en archivos.
- 38 archivos leídos con la credencial existente del propietario: bytes de origen preservados. El proveedor añade su script de challenge al final de los HTML y se comprueba aparte. Peticiones anónimas con cabeceras de navegador reciben 401 en índice, ficha directa y Excel.
- La apertura automática del navegador no pudo completarse por un fallo de la herramienta local. No se atribuye una revisión visual autenticada nueva.

Se retienen compilación, archivo y staging con [inventario](../../../outputs/stratai-review-shared-20260925/retained-inventory.json).
La preparación oficial de Sites se ejecuta conservando staging; se usa tar nativo
de Windows tras la incompatibilidad de rutas de unidad con el wrapper de shell.
No hay nuevo parsing, smoke funcional, publicación de demos completas, promoción
de corpus ni cambios del flujo temporal de StratAI.

El siguiente paso es añadir únicamente los correos que indique el usuario, tras
comprobar de nuevo la política de acceso y que las invitaciones externas estén
habilitadas. Recibir luego las lecturas agrupadas, conservando identidad, versión,
exposición y literal de cada revisor antes del contraste con predicciones.
