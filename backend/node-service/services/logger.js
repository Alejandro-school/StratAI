function log(level, event, fields = {}) {
  const entry = JSON.stringify({
    timestamp: new Date().toISOString(),
    level,
    service: 'steam-pipeline',
    event,
    ...fields,
  });
  const writer = level === 'error' ? console.error : console.log;
  writer(entry);
}

module.exports = { log };
