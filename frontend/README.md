# Frontend de StratAI

Aplicación React que presenta el historial de partidas, el análisis de rendimiento, el mapa táctico, las repeticiones 2D y el asistente de coaching.

## Desarrollo local

Requiere Node.js 20.19 o posterior.

```powershell
npm ci
Copy-Item .env.example .env
npm start
```

La aplicación se abre en `http://localhost:3000`. Para ejecutar todo StratAI, usa `npm run start:all` desde la raíz del repositorio.

## Estructura

```text
src/
  auth/          Autenticación y protección de rutas
  components/    Componentes existentes agrupados por área
  features/      Módulos autocontenidos organizados por dominio
  hooks/         Hooks transversales todavía no migrados a una feature
  pages/         Puntos de entrada de las rutas
  stores/        Estado global compartido
  styles/        Estilos existentes todavía no colocados junto a su feature
  test/          Configuración común de Vitest
  workers/       Procesamiento en Web Workers
```

`features/replay2d/` es la referencia para código nuevo. Una feature puede contener:

```text
features/<feature>/
  api/           Acceso a endpoints de la feature
  components/    Interfaz específica del dominio
  domain/        Modelos y transformaciones sin dependencias de React
  hooks/         Estado y coordinación de React
  index.js       API pública del módulo
```

Evita añadir nueva lógica de dominio a `utils/`. Colócala en la feature propietaria; reserva los módulos compartidos para código realmente transversal.

## Archivos principales

| Archivo | Función |
| --- | --- |
| `index.html` | Documento de entrada de Vite. |
| `src/index.jsx` | Montaje de React. |
| `src/App.jsx` | Rutas y proveedores globales. |
| `vite.config.js` | Desarrollo, pruebas y compilación. |
| `.env.example` | Variables públicas admitidas por el frontend. |

## Variables de entorno

| Variable | Valor local predeterminado |
| --- | --- |
| `VITE_API_URL` | `http://localhost:8000` |

Node.js y Go son servicios internos y no deben exponerse directamente al navegador.

## Landing oficial y acceso con Steam

La ruta pública `/` utiliza `src/features/landing-world/`: el recorrido de Mirage,
sus paneles y el replay original. Sus estilos permanecen dentro de `.landing-world`
para conservar los controles y el scroll de las pantallas de cuenta. Los textos
y el guion del replay pertenecen a la misma feature; la landing anterior se ha retirado.

Todos los accesos a Steam utilizan `VITE_API_URL` y `/auth/steam/login`. En desarrollo,
la API predeterminada es `http://localhost:8000`; en una compilación de producción
sin esa variable se usa el mismo origen. Si sirves el build desde otro origen,
compila con la URL pública de la API. El servidor debe configurar `FRONTEND_URL`
para el retorno a `/steam-login-success`, `PUBLIC_BACKEND_URL` para el callback de
Steam y `ALLOWED_ORIGINS` para el frontend. La sesión y el onboarding existentes
deciden después entre `/history-code` y `/dashboard`.

El acceso y los paneles también están disponibles mientras carga el mapa 3D.
La entrada independiente `landing-world.html` se conserva como herramienta
opcional de revisión; la aplicación oficial se inicia con los comandos normales.

## Comandos

```powershell
npm start
npm test
npm run test:watch
npm run build
npm run preview
```

## Convenciones

- Componentes y páginas: `PascalCase.jsx`.
- Hooks: `useNombre.js`.
- Carpetas de features y secciones: `kebab-case`.
- Pruebas: junto al archivo probado con sufijo `.test.jsx` o `.test.js`.
- Mantén estable la URL pública aunque cambie el nombre interno de una página.

Última revisión: 26 de julio de 2026.
