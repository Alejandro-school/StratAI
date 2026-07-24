# StratAI Frontend

Interfaz React de StratAI, desarrollada y compilada con Vite.

## Desarrollo local

Requiere Node.js 20.19 o posterior.

```powershell
npm install
npm start
```

La aplicación se abre en `http://localhost:3000`. Para ejecutar todos los servicios del proyecto, usa `npm run start:all` desde la raíz.

## Variables de entorno

Copia `.env.example` a `.env` si necesitas sobrescribir las URLs locales:

| Variable | Valor local predeterminado |
|---|---|
| `VITE_API_URL` | `http://localhost:8000` |
| `VITE_NODE_URL` | `http://localhost:4000` |

## Comandos

```powershell
npm start       # servidor de desarrollo
npm test        # pruebas con Vitest
npm run build   # build de producción en build/
npm run preview # previsualizar el build
```

## Archivos principales

- `index.html`: documento de entrada de Vite.
- `vite.config.js`: configuración de desarrollo, pruebas y build.
- `src/index.jsx`: montaje de React.
- `src/App.jsx`: rutas principales.

Última revisión: 24 de julio de 2026.
