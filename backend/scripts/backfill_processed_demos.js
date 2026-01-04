/**
 * backfill_processed_demos.js
 * ---------------------------
 * Script para registrar en Redis las demos ya procesadas que existen en data/exports/
 * pero no están en la lista processed_demos:{steamID}.
 * 
 * USO: cd node-service && node ../scripts/backfill_processed_demos.js <STEAM_ID>
 * 
 * Este script es necesario una sola vez para demos procesadas antes del fix en steamDownloader.js
 */

require('dotenv').config({ path: require('path').join(__dirname, '../node-service/.env') });
const fs = require('fs');
const path = require('path');
const { redisClient, ensureRedis } = require('../node-service/services/redisClient');

const EXPORTS_DIR = path.join(__dirname, '../data/exports');

async function main() {
  const steamID = process.argv[2];
  
  if (!steamID) {
    console.error('❌ Uso: cd backend && node scripts/backfill_processed_demos.js <STEAM_ID>');
    console.log('   Ejemplo: node scripts/backfill_processed_demos.js 76561198123456789');
    process.exit(1);
  }
  
  console.log(`\n📦 Backfill de demos procesadas para Steam ID: ${steamID}\n`);
  
  // Conectar a Redis usando el cliente compartido
  await ensureRedis();
  console.log('✅ Conectado a Redis\n');
  
  // Verificar demos ya registradas
  const existingRaw = await redisClient.lRange(`processed_demos:${steamID}`, 0, -1);
  const existingMatchIds = new Set(
    existingRaw.map(raw => {
      try {
        return JSON.parse(raw).match_id;
      } catch {
        return null;
      }
    }).filter(Boolean)
  );
  
  console.log(`📊 Demos ya registradas en Redis: ${existingMatchIds.size}`);
  
  // Leer carpetas en exports
  const folders = fs.readdirSync(EXPORTS_DIR, { withFileTypes: true })
    .filter(d => d.isDirectory() && d.name.startsWith('match_'))
    .map(d => d.name);
  
  console.log(`📁 Carpetas de demos en exports: ${folders.length}\n`);
  
  let added = 0;
  let skipped = 0;
  let errors = 0;
  
  for (const folder of folders) {
    const matchId = folder.replace('match_', '');
    
    // Skip test folder
    if (matchId === 'test_verification') {
      skipped++;
      continue;
    }
    
    // Skip if already registered
    if (existingMatchIds.has(matchId)) {
      console.log(`⏭️  ${matchId} - ya registrada`);
      skipped++;
      continue;
    }
    
    // Leer metadata.json
    const metadataPath = path.join(EXPORTS_DIR, folder, 'metadata.json');
    if (!fs.existsSync(metadataPath)) {
      console.log(`⚠️  ${matchId} - sin metadata.json`);
      errors++;
      continue;
    }
    
    try {
      const metadata = JSON.parse(fs.readFileSync(metadataPath, 'utf-8'));
      
      const demoData = {
        match_id: metadata.match_id || matchId,
        steam_id: steamID,
        map_name: metadata.map_name || 'unknown',
        date: metadata.date || '',
        duration: metadata.duration_seconds || 0,
        processed_at: new Date().toISOString()
      };
      
      await redisClient.rPush(`processed_demos:${steamID}`, JSON.stringify(demoData));
      console.log(`✅ ${matchId} - ${metadata.map_name} - ${metadata.date?.split('T')[0] || 'sin fecha'}`);
      added++;
    } catch (err) {
      console.error(`❌ ${matchId} - Error: ${err.message}`);
      errors++;
    }
  }
  
  // Invalidar caché del dashboard
  await redisClient.del(`dashboard_stats:${steamID}`);
  console.log(`\n🗑️  Cache de dashboard invalidada`);
  
  console.log(`\n📊 Resumen:`);
  console.log(`   ✅ Añadidas: ${added}`);
  console.log(`   ⏭️  Saltadas: ${skipped}`);
  console.log(`   ❌ Errores: ${errors}`);
  console.log(`   📦 Total en Redis: ${existingMatchIds.size + added}`);
  
  await redisClient.quit();
  console.log('\n👋 Hecho!\n');
}

main().catch(err => {
  console.error('❌ Error fatal:', err);
  process.exit(1);
});
