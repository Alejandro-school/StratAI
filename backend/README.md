# Backend de StratAI

El backend contiene tres aplicaciones con responsabilidades distintas: la API pública, la integración con Steam y el analizador de demos.

## Servicios

| Ruta | Tecnología | Responsabilidad | Puerto local |
| --- | --- | --- | --- |
| `app/` | FastAPI | API pública, sesiones, seguridad y agregación. | `8000` |
| `node-service/` | Node.js | Steam, descargas, colas y coordinación. | `4000` |
| `go-service/` | Go | Parsing y análisis de demos. | `8080` |

Redis conecta los tres servicios. Node.js y Go son internos; el navegador debe comunicarse únicamente con FastAPI.

## API FastAPI

```text
app/
  auth/          Dependencias de autenticación
  middleware/    Comportamiento transversal de las peticiones
  routes/        Endpoints agrupados por recurso
  security/      Credenciales y autenticación entre servicios
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

Última revisión: 26 de julio de 2026.
