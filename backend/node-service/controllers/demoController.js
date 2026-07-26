const express = require('express');
const { redisClient } = require('../services/redisClient');
const { internalOnly } = require('../middleware/security');
const {
  eventKey,
  getPipelineStatus,
  metricsRegistry,
  triggerDiscovery,
} = require('../services/pipelineV2');
const { log } = require('../services/logger');

const router = express.Router();
const STEAM_ID_PATTERN = /^\d{17}$/;

router.post('/internal/v2/discovery', internalOnly, async (req, res) => {
  const {
    steam_id: steamId,
    priority = 100,
    credential_version: credentialVersion,
  } = req.body || {};
  if (!STEAM_ID_PATTERN.test(steamId || '')) {
    return res.status(400).json({ error: 'Invalid or missing steam_id' });
  }
  try {
    const result = await triggerDiscovery(
      steamId,
      Number(priority) || 0,
      credentialVersion,
    );
    return res.status(202).json(result);
  } catch (error) {
    log('error', 'discovery_enqueue_failed', { error_code: error.name });
    return res.status(503).json({ error: 'Pipeline queue unavailable' });
  }
});

router.get('/internal/v2/pipeline-status/:steamId', internalOnly, async (req, res) => {
  const { steamId } = req.params;
  if (!STEAM_ID_PATTERN.test(steamId)) {
    return res.status(400).json({ error: 'Invalid steam_id' });
  }
  try {
    return res.json(await getPipelineStatus(steamId));
  } catch {
    return res.status(503).json({ error: 'Pipeline status unavailable' });
  }
});

router.get('/internal/v2/events/:steamId', internalOnly, async (req, res) => {
  const { steamId } = req.params;
  if (!STEAM_ID_PATTERN.test(steamId)) {
    return res.status(400).json({ error: 'Invalid steam_id' });
  }

  const lastEventId = req.headers['last-event-id'];
  let cursor = typeof lastEventId === 'string' && /^\d+-\d+$/.test(lastEventId)
    ? lastEventId
    : null;
  if (!cursor) {
    const streamInfo = await redisClient.xInfoStream(eventKey(steamId)).catch(() => null);
    cursor = streamInfo?.lastGeneratedId || '0-0';
  }

  let snapshot;
  try {
    snapshot = await getPipelineStatus(steamId);
  } catch {
    return res.status(503).json({ error: 'Pipeline status unavailable' });
  }

  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache, no-transform',
    Connection: 'keep-alive',
    'X-Accel-Buffering': 'no',
  });
  res.flushHeaders();
  res.write(`event: snapshot\ndata: ${JSON.stringify(snapshot)}\n\n`);

  let closed = false;
  req.once('close', () => {
    closed = true;
  });

  while (!closed) {
    try {
      const streams = await redisClient.xRead(
        [{ key: eventKey(steamId), id: cursor }],
        { BLOCK: 25000, COUNT: 100 },
      );
      if (!streams) {
        res.write(': heartbeat\n\n');
        continue;
      }
      for (const stream of streams) {
        for (const message of stream.messages) {
          cursor = message.id;
          res.write(`id: ${message.id}\nevent: pipeline\ndata: ${message.message.data}\n\n`);
        }
      }
    } catch {
      if (!closed) {
        res.write(`event: pipeline\ndata: ${JSON.stringify({
          stage: 'failed',
          error_code: 'event_stream_unavailable',
          timestamp: new Date().toISOString(),
        })}\n\n`);
      }
      break;
    }
  }
  res.end();
});

router.get('/internal/v2/metrics', internalOnly, async (_req, res) => {
  res.setHeader('Content-Type', metricsRegistry.contentType);
  res.send(await metricsRegistry.metrics());
});

module.exports = router;
