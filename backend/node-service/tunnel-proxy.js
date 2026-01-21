const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');
const cors = require('cors');

const app = express();
const PORT = 9000;

// Configuración de puertos de tus servicios
const SERVICES = {
  FRONTEND: 'http://127.0.0.1:3000',
  PYTHON_BACKEND: 'http://127.0.0.1:8000',
  GO_SERVICE: 'http://127.0.0.1:8080'
};

app.use(cors());

// Logging detallado para diagnóstico
app.use((req, res, next) => {
  console.log(`[Proxy] ${req.method} ${req.url}`);
  console.log(`        - Host: ${req.headers.host}`);
  console.log(`        - X-Forwarded-Host: ${req.headers['x-forwarded-host']}`);
  console.log(`        - X-Forwarded-Proto: ${req.headers['x-forwarded-proto']}`);
  next();
});

// Helper para configurar el proxy
const proxyOptions = (target) => ({
  target,
  changeOrigin: true,
  xfwd: true, // Lo activamos para que gestione cabeceras estándar
  onProxyReq: (proxyReq, req, res) => {
    // FORZAMOS que el host original llegue al backend
    // Esto es vital para que Steam sepa a dónde volver
    proxyReq.setHeader('X-Forwarded-Host', req.headers.host);
    proxyReq.setHeader('X-Forwarded-Proto', req.headers['x-forwarded-proto'] || 'https');
  },
  logLevel: 'debug'
});

// Creamos las instancias de proxy una sola vez para eficiencia y evitar fugas
const pythonProxy = createProxyMiddleware(proxyOptions(SERVICES.PYTHON_BACKEND));
const goProxy = createProxyMiddleware(proxyOptions(SERVICES.GO_SERVICE));
const frontendProxy = createProxyMiddleware({ ...proxyOptions(SERVICES.FRONTEND), ws: true });

// Lógica de enrutamiento MANUAL para evitar que Express "recorte" las rutas
// (mount path stripping) y para manejar colisiones como /steam vs /steam-login-success
app.use((req, res, next) => {
  const url = req.url;

  // 1. Python Backend: Rutas que SON de API
  // Chequeamos específicamente por prefijos claros o rutas exactas de API
  const isPython = 
    url.startsWith('/auth/') || 
    url === '/auth' ||
    url.startsWith('/steam/') || // Esto evita /steam-login-success porque no tiene la barra final
    url.startsWith('/docs') || 
    url.startsWith('/openapi.json') || 
    url.startsWith('/ping');

  if (isPython) {
    console.log(`[Proxy] Routing to Python Backend: ${url}`);
    return pythonProxy(req, res, next);
  }

  // 2. Go Service: Procesamiento de demos
  const isGo = 
    url.startsWith('/process-demo') || 
    url.startsWith('/match-details') || 
    url.startsWith('/health');

  if (isGo) {
    console.log(`[Proxy] Routing to Go Service: ${url}`);
    return goProxy(req, res, next);
  }

  // 3. Frontend: Todo lo demás (incluyendo /steam-login-success)
  // console.log(`[Proxy] Routing to Frontend: ${url}`);
  return frontendProxy(req, res, next);
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`===============================================`);
  console.log(`🚀 TUNNEL PROXY (V3) corriendo en http://0.0.0.0:${PORT}`);
  console.log(`   - Auth/Steam API -> Port 8000 (Preserving path)`);
  console.log(`   - Go/Demos API   -> Port 8080 (Preserving path)`);
  console.log(`   - Frontend/Web   -> Port 3000`);
  console.log(`===============================================`);
});
