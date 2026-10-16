# Backend de StratAI

El backend contiene tres aplicaciones con responsabilidades distintas: la API pública, la integración con Steam y el analizador de demos.

## Servicios

| Ruta | Tecnología | Responsabilidad | Puerto local |
| --- | --- | --- | --- |
| `app/` | FastAPI | API pública, sesiones, seguridad y agregación. | `8000` |
| `node-service/` | Node.js | Steam, descargas, colas y coordinación. | `4000` |
| `go-service/` | Go | Parsing y análisis de demos. | `8080` |

Redis conecta los tres servicios. Node.js y Go son internos; el navegador debe comunicarse únicamente con FastAPI.

## Acceso al historial y descarga de demos

El usuario inicia sesión con Steam y, si aún no ha configurado su historial, guarda
su código de autenticación de CS2 y un código compartido de partida. Después accede
al dashboard sin agregar cuentas ni aceptar invitaciones. Los enlaces antiguos a
`/bot-instructions` redirigen a la comprobación de sesión y configuración.

El flujo es:

```text
Sesión Steam → códigos de historial → Web API: siguientes sharecodes
    → cuenta técnica Steam/GC: URL de la demo → descarga → análisis Go
```

La Web API consulta el historial con el Steam ID de la sesión, la clave del servidor,
el código de autenticación del usuario y el último sharecode conocido. Tras una
respuesta válida de Steam se encola también esa primera partida; el cursor sólo
avanza después de encolar cada partida posterior. Un marcador persistente permite
reintentar la primera importación sin perderla si falla la cola. La cuenta técnica mantiene la sesión con el Game Coordinator
para resolver los sharecodes; su lista de amigos no autoriza estas operaciones.
No se piden al usuario su contraseña ni sus códigos de Steam Guard.

Las mutaciones requieren sesión y conservan la protección CSRF. El código de
autenticación se cifra en Redis y no se envía en el trabajo de la cola. La API
rechaza campos adicionales, incluido cualquier Steam ID suministrado en el
onboarding. Configuración y sincronización comparten un límite atómico de una
solicitud cada diez segundos por usuario (`429`, `Retry-After: 10`). La
sincronización sin credenciales devuelve `409`; un fallo de Redis devuelve `503`.
Los antiguos endpoints de amistad de FastAPI y Node se han retirado.

Guardar los códigos no basta para marcar el historial como validado: Node consulta
Steam primero. Una cuenta validada conserva su acceso al dashboard durante las
sincronizaciones y fallos transitorios. Un rechazo explícito de Steam exige renovar
los códigos. Cambiarlos elimina la validación anterior. El dashboard y el historial
se actualizan al terminar las demos, incluso tras reconectar el stream de progreso.

FastAPI, Node y Go deben compartir `REDIS_URL`, `PIPELINE_NAMESPACE`,
`INTERNAL_SERVICE_SECRET` y `CREDENTIAL_ENCRYPTION_KEY`. Go acepta URLs Redis con
usuario, contraseña, base de datos y TLS; `REDIS_ADDR` sólo es una alternativa
heredada cuando no existe `REDIS_URL`. `npm run check:services` verifica la
configuración y Redis antes del arranque. La disponibilidad de Node requiere
Redis, workers activos y al menos una sesión de GC lista.

## Redis y WSL en Windows

Redis debe estar activo antes de ejecutar `npm run start:all`. En Ubuntu, si ya
está instalado:

```sh
sudo service redis-server start
redis-cli ping
```

La respuesta esperada es `PONG`. Desde la raíz del proyecto en Windows, confirma
la conexión con `npm run check:services`.

Si Ubuntu se cierra con `HCS_E_HYPERV_NOT_INSTALLED`, comprueba la virtualización
en el Administrador de tareas. Debe estar habilitada en BIOS/UEFI (Intel VT-x o
AMD SVM). En PowerShell **como administrador**, habilita las características
necesarias y reinicia Windows cuando puedas cerrar tu trabajo:

```powershell
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
```

Después abre Ubuntu y arranca Redis. Estos comandos no habilitan por sí mismos
la virtualización del firmware. Consulta la [guía de Microsoft para este error](https://learn.microsoft.com/en-us/windows/wsl/troubleshooting#error-0x80370102-the-virtual-machine-could-not-be-started-because-a-required-feature-is-not-installed).
En una placa ASUS con procesador AMD, la opción es `SVM Mode`; consulta la
[guía oficial de ASUS](https://www.asus.com/global/support/faq/1045141/).
No reinstales ni elimines la distribución para corregir este problema.

## API FastAPI

```text
app/
  auth/          Dependencias de autenticación
  middleware/    Comportamiento transversal de las peticiones
  matches/       Acceso y transformaciones del dominio de partidas
  routes/        Endpoints agrupados por recurso
  security/      Credenciales y autenticación entre servicios
  tactical/      Lógica del dominio táctico
  utils/         Código existente pendiente de ubicar por dominio
  config.py      Configuración derivada del entorno
  main.py        Creación y montaje de la aplicación
```

Para lógica nueva, evita ampliar `utils/`: crea un módulo con el nombre del dominio propietario.

## Preparación y ejecución

Desde la raíz del repositorio:

```powershell
python -m venv backend/venv
backend\venv\Scripts\python.exe -m pip install -r backend/requirements-dev.txt
Copy-Item backend/.env.example backend/.env
npm run backend
```

En desarrollo, OpenAPI está disponible en `http://localhost:8000/docs`.

## Pruebas

```powershell
npm run lint:python
npm run test:python
npm run test:node
npm run test:go
```

Consulta también:

- [Servicio Steam](node-service/README.md)
- [Analizador de demos](go-service/README.md)
- [Estructura transversal](../docs/repository-structure.md)
- [Motor Python del coach](../ai_coach/README.md), cuyo código y pruebas viven fuera de `backend/`.
- [Operación y validación](../docs/operations/README.md)

Última revisión de estructura: 5 de octubre de 2026.
