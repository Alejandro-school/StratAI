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
const { createPipelineEventHandler } = require('../services/pipelineEvents');

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

router.get('/internal/v2/events/:steamId', internalOnly,
  createPipelineEventHandler({ redisClient, eventKey, getPipelineStatus }));

router.get('/internal/v2/metrics', internalOnly, async (_req, res) => {
  res.setHeader('Content-Type', metricsRegistry.contentType);
  res.send(await metricsRegistry.metrics());
});

module.exports = router;
