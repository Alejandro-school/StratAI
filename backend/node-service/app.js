const express = require('express');
const { downloadDemos } = require('./controllers/downloadController');

const app = express();
app.use(express.json());

const PORT = 3001;

// Endpoint para iniciar la descarga
app.post('/download-demo', downloadDemos);

app.listen(PORT, () => {
    console.log(`🚀 Servidor Node.js corriendo en el puerto ${PORT}`);
});
