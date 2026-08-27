const crypto = require('crypto');
const path = require('path');
const axios = require('axios');
const IORedis = require('ioredis');
const {
  Queue,
  Worker,
  UnrecoverableError,
} = require('bullmq');
const {
  Counter,
  Gauge,
  Histogram,
  Registry,
  collectDefaultMetrics,
} = require('prom-client');
const config = require('./config');
const { redisClient } = require('./redisClient');
const { resolveSharecode } = require('./steamGateway');
const {
  cleanupPartialDownloads,
  downloadDemo,
  normalizeGcDemoUrl,
} = require('./demoDownload');
const { buildServiceHeaders } = require('./serviceAuth');
const { log } = require('./logger');

const STEAM_API_URL = 'https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/';
const SHARECODE_PATTERN = /^CSGO-[A-Za-z0-9]{5}(?:-[A-Za-z0-9]{5}){4}$/;
const EMPTY_SHARECODES = new Set(['n/a', 'null', 'none']);
const DEMOS_DIR = path.resolve(__dirname, '../../data/demos');
const GC_RESOLVER_VERSION = 2;
const DEMO_DOWNLOADER_VERSION = 2;
const ANALYZER_AUTH_VERSION = 2;
const PARSER_SCHEMA_VERSION = 'v11';
const prefix = config.pipelineNamespace;
const connection = new IORedis(config.redis.url, {
  maxRetriesPerRequest: null,
  enableReadyCheck: true,
});
const queueOptions = { connection, prefix };
const defaultJobOptions = {
  removeOnComplete: { age: 7 * 24 * 3600, count: 10000 },
  removeOnFail: { age: 30 * 24 * 3600, count: 10000 },
};

const discoveryQueue = new Queue('discovery', { ...queueOptions, defaultJobOptions });
const resolveQueue = new Queue('gc-resolution', { ...queueOptions, defaultJobOptions });
const downloadQueue = new Queue('download', { ...queueOptions, defaultJobOptions });
const analysisQueue = new Queue('analysis', { ...queueOptions, defaultJobOptions });

const metricsRegistry = new Registry();
collectDefaultMetrics({ register: metricsRegistry, prefix: 'stratai_node_' });
const stageDuration = new Histogram({
  name: 'stratai_pipeline_stage_duration_seconds',
  help: 'Pipeline stage duration in seconds',
  labelNames: ['stage', 'result'],
  registers: [metricsRegistry],
  buckets: [0.1, 0.5, 1, 5, 15, 30, 60, 180, 600],
});
const jobsTotal = new Counter({
  name: 'stratai_pipeline_jobs_total',
  help: 'Pipeline job transitions',
  labelNames: ['stage', 'result'],
  registers: [metricsRegistry],
});
const activeJobs = new Gauge({
  name: 'stratai_pipeline_active_jobs',
  help: 'Currently active jobs',
  labelNames: ['stage'],
  registers: [metricsRegistry],
});
const queueWaitDuration = new Histogram({
  name: 'stratai_pipeline_queue_wait_seconds',
  help: 'Time from durable enqueue to processing start',
  labelNames: ['stage'],
  registers: [metricsRegistry],
  buckets: [0.1, 0.5, 1, 5, 15, 30, 60, 300, 900, 3600],
});
const demoSizeBytes = new Histogram({
  name: 'stratai_pipeline_demo_size_bytes',
  help: 'Validated decompressed demo size',
  registers: [metricsRegistry],
  buckets: [
    10 * 1024 * 1024,
    50 * 1024 * 1024,
    100 * 1024 * 1024,
    250 * 1024 * 1024,
    500 * 1024 * 1024,
    1024 * 1024 * 1024,
  ],
});
const retriesTotal = new Counter({
  name: 'stratai_pipeline_retries_total',
  help: 'Pipeline attempts after the initial attempt',
  labelNames: ['stage'],
  registers: [metricsRegistry],
});

let workers = [];
let isStarted = false;

function credentialKey(steamId) {
  return `${prefix}:user:${steamId}:credentials`;
}

function stateKey(steamId) {
  return `${prefix}:user:${steamId}:pipeline`;
}

function eventKey(steamId) {
  return `${prefix}:events:${steamId}`;
}

function jobIdentity(steamId, sharecode) {
  return crypto.createHash('sha256').update(`${steamId}:${sharecode}`).digest('hex');
}

function demoFilePath(sharecode) {
  const cleaned = sharecode.replace(/CSGO-|-/g, '');
  return path.join(DEMOS_DIR, `match_${cleaned}.dem`);
}

async function updateCredentialCursor(steamId, version, knownCode) {
  const key = credentialKey(steamId);
  const result = await redisClient.eval(
    `
      if ARGV[1] ~= '' and redis.call('HGET', KEYS[1], 'version') ~= ARGV[1] then
        return 0
      end
      if ARGV[2] ~= '' then
        redis.call('HSET', KEYS[1], 'known_code', ARGV[2])
      end
      redis.call(
        'HSET',
        KEYS[1],
        'status',
        'configured',
        'validated_at',
        ARGV[3],
        'discovery_error_code',
        ''
      )
      return 1
    `,
    {
      keys: [key],
      arguments: [version || '', knownCode || '', new Date().toISOString()],
    },
  );
  return result === 1;
}

async function updateCredentialStatus(steamId, version, status, errorCode = '') {
  const result = await redisClient.eval(
    `
      if ARGV[1] ~= '' and redis.call('HGET', KEYS[1], 'version') ~= ARGV[1] then
        return 0
      end
      redis.call(
        'HSET',
        KEYS[1],
        'status',
        ARGV[2],
        'discovery_error_code',
        ARGV[3],
        'updated_at',
        ARGV[4]
      )
      return 1
    `,
    {
      keys: [credentialKey(steamId)],
      arguments: [version || '', status, errorCode, new Date().toISOString()],
    },
  );
  return result === 1;
}

function decryptCredential(envelope) {
  const [version, payload] = String(envelope || '').split('.', 2);
  if (version !== 'v1' || !payload) throw new Error('Invalid credential envelope');
  const padded = payload + '='.repeat((4 - (payload.length % 4)) % 4);
  const raw = Buffer.from(padded, 'base64url');
  const nonce = raw.subarray(0, 12);
  const ciphertext = raw.subarray(12, -16);
  const tag = raw.subarray(-16);
  const key = crypto.createHash('sha256').update(config.credentialEncryptionKey).digest();
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, nonce);
  decipher.setAAD(Buffer.from('v1'));
  decipher.setAuthTag(tag);
  return Buffer.concat([decipher.update(ciphertext), decipher.final()]).toString('utf8');
}

async function publishEvent(steamId, event) {
  const payload = JSON.stringify({
    ...event,
    timestamp: new Date().toISOString(),
  });
  const stream = eventKey(steamId);
  await redisClient.xAdd(stream, '*', { data: payload });
  await redisClient.xTrim(stream, 'MAXLEN', 1000, { strategyModifier: '~' }).catch(() => {});
}

async function setPipelineState(steamId, sharecode, state) {
  const progressByStage = {
    queued: 0,
    resolving: 10,
    downloading: 30,
    analyzing: 70,
    retry_wait: 0,
    completed: 100,
    failed: 100,
  };
  const payload = {
    sharecode,
    ...state,
    progress: state.progress ?? progressByStage[state.stage] ?? 0,
    timestamp: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
  await redisClient.hSet(stateKey(steamId), sharecode, JSON.stringify(payload));
  await publishEvent(steamId, payload);
}

async function waitForSteamRateSlot() {
  const key = `${prefix}:rate-limit:steam`;
  while (true) {
    const acquired = await redisClient.set(key, '1', {
      NX: true,
      PX: config.cron.steamRequestSpacing,
    });
    if (acquired) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
}

async function enqueueSharecode(steamId, sharecode, priority = 100) {
  const state = await redisClient.hGet(stateKey(steamId), sharecode);
  if (state) {
    try {
      if (JSON.parse(state).stage === 'completed') return null;
    } catch {
      await redisClient.hDel(stateKey(steamId), sharecode);
    }
  }

  const identity = jobIdentity(steamId, sharecode);
  const job = await resolveQueue.add(
    'resolve',
    { steamId, sharecode, resolverVersion: GC_RESOLVER_VERSION },
    {
      priority,
      attempts: 4,
      backoff: { type: 'exponential', delay: 2000, jitter: 0.5 },
      deduplication: { id: identity },
    },
  );
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    stage: 'queued',
    attempt: 0,
  });
  return job;
}

async function runDiscovery(job) {
  if (job.name === 'scan-users') {
    const users = await redisClient.sMembers(`${prefix}:users`);
    for (let offset = 0; offset < users.length; offset += 100) {
      const batch = users.slice(offset, offset + 100);
      await discoveryQueue.addBulk(batch.map((steamId) => ({
        name: 'discover-user',
        data: { steamId, priority: 100 },
        opts: {
          priority: 100,
          attempts: 4,
          backoff: { type: 'exponential', delay: 5000, jitter: 0.5 },
          deduplication: { id: steamId },
        },
      })));
    }
    return { users: users.length };
  }

  const { steamId, priority = 0, credentialVersion } = job.data;
  const credentials = await redisClient.hGetAll(credentialKey(steamId));
  if (!credentials.auth_code || !credentials.known_code) {
    throw new UnrecoverableError('User credentials are not configured');
  }
  if (credentialVersion && credentials.version !== credentialVersion) {
    const staleJob = new UnrecoverableError('Credential version has been replaced');
    staleJob.pipelineCode = 'stale_credential_job';
    throw staleJob;
  }
  const effectiveCredentialVersion = credentialVersion || credentials.version;
  await job.updateData({
    ...job.data,
    credentialVersion: effectiveCredentialVersion,
  });
  await updateCredentialStatus(
    steamId,
    effectiveCredentialVersion,
    'discovering',
  );
  await publishEvent(steamId, {
    job_id: String(job.id),
    stage: 'discovery',
    attempt: job.attemptsMade + 1,
    progress: 0,
  });

  const authCode = decryptCredential(credentials.auth_code);
  let currentCode = credentials.known_code;
  let discovered = 0;

  while (discovered < 50) {
    await waitForSteamRateSlot();
    let response;
    try {
      response = await axios.get(STEAM_API_URL, {
        params: {
          key: config.steam.apiKey,
          steamid: steamId,
          steamidkey: authCode,
          knowncode: currentCode,
        },
        timeout: 10000,
      });
    } catch (error) {
      if (error.response?.status === 412) {
        await updateCredentialStatus(
          steamId,
          effectiveCredentialVersion,
          'needs_credentials',
          'steam_credentials_invalid',
        );
        const invalidCredentials = new UnrecoverableError('Steam credentials or known code are invalid');
        invalidCredentials.pipelineCode = 'steam_credentials_invalid';
        throw invalidCredentials;
      }
      if (error.response?.status === 429) {
        error.pipelineCode = 'steam_rate_limited';
      } else if (error.response?.status === 503) {
        error.pipelineCode = 'steam_unavailable';
      }
      throw error;
    }

    const nextCode = response.data?.result?.nextcode?.trim();
    if (!nextCode || EMPTY_SHARECODES.has(nextCode.toLowerCase())) break;
    if (!SHARECODE_PATTERN.test(nextCode)) {
      throw new UnrecoverableError('Steam returned an invalid sharecode');
    }
    if (
      effectiveCredentialVersion
      && await redisClient.hGet(credentialKey(steamId), 'version') !== effectiveCredentialVersion
    ) {
      const staleJob = new UnrecoverableError('Credential version has been replaced');
      staleJob.pipelineCode = 'stale_credential_job';
      throw staleJob;
    }

    await enqueueSharecode(steamId, nextCode, priority);
    if (!(await updateCredentialCursor(steamId, effectiveCredentialVersion, nextCode))) {
      const staleJob = new UnrecoverableError('Credential version has been replaced');
      staleJob.pipelineCode = 'stale_credential_job';
      throw staleJob;
    }
    currentCode = nextCode;
    discovered += 1;
    await job.updateProgress({ discovered, cursor: nextCode.slice(-5) });
  }

  if (discovered === 0) {
    await updateCredentialCursor(steamId, effectiveCredentialVersion, '');
  }
  return { discovered };
}

async function runResolve(job) {
  const { steamId, sharecode, refreshToken } = job.data;
  const identity = jobIdentity(steamId, sharecode);
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    stage: 'resolving',
    attempt: job.attemptsMade + 1,
  });
  const result = await resolveSharecode(sharecode);
  const matchId = result.matchID.toString();
  const downloadIdentity = refreshToken ? `${identity}-${refreshToken}` : identity;
  await downloadQueue.add(
    'download',
    {
      steamId,
      sharecode,
      matchId,
      demoUrl: result.demoUrl,
      downloaderVersion: DEMO_DOWNLOADER_VERSION,
      matchDate: result.matchDate,
      matchDuration: result.matchDuration,
      mapName: result.mapName || 'unknown',
    },
    {
      attempts: config.downloadQueue.maxRetries,
      backoff: { type: 'exponential', delay: config.downloadQueue.retryBaseDelay, jitter: 0.5 },
      deduplication: { id: downloadIdentity },
    },
  );
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    match_id: matchId,
    stage: 'downloading',
    attempt: job.attemptsMade + 1,
  });
  return { matchId };
}

async function runDownload(job) {
  const {
    steamId,
    sharecode,
    matchId,
    demoUrl,
  } = job.data;
  const identity = jobIdentity(steamId, sharecode);
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    match_id: matchId,
    stage: 'downloading',
    attempt: job.attemptsMade + 1,
  });

  try {
    const download = await downloadDemo(demoUrl, demoFilePath(sharecode), {
      onProgress: (downloadedBytes) => {
        job.updateProgress({ stage: 'downloading', bytes: downloadedBytes }).catch(() => {});
        log('info', 'demo_download_progress', {
          job_id: identity,
          stage: 'downloading',
          bytes: downloadedBytes,
        });
      },
    });
    demoSizeBytes.observe(download.bytes);
    await analysisQueue.add(
      'analyze',
      {
        ...job.data,
        ...download,
        analyzerAuthVersion: ANALYZER_AUTH_VERSION,
        parserSchemaVersion: PARSER_SCHEMA_VERSION,
      },
      {
        attempts: 3,
        backoff: { type: 'exponential', delay: 5000, jitter: 0.5 },
        deduplication: { id: identity },
      },
    );
    await setPipelineState(steamId, sharecode, {
      job_id: identity,
      match_id: matchId,
      stage: 'analyzing',
      attempt: job.attemptsMade + 1,
      bytes: download.bytes,
    });
    return download;
  } catch (error) {
    if ([403, 410].includes(error.statusCode)) {
      await resolveQueue.add(
        'resolve',
        {
          steamId,
          sharecode,
          refreshToken: Date.now().toString(),
          resolverVersion: GC_RESOLVER_VERSION,
        },
        {
          priority: 1,
          attempts: 4,
          backoff: { type: 'exponential', delay: 2000, jitter: 0.5 },
          deduplication: { id: `${jobIdentity(steamId, sharecode)}-refresh-${Date.now()}` },
        },
      );
      await setPipelineState(steamId, sharecode, {
        job_id: identity,
        match_id: matchId,
        stage: 'retry_wait',
        error_code: 'cdn_url_expired',
      });
      return { requeued: true };
    }
    throw error;
  }
}

async function runAnalysis(job) {
  const {
    steamId,
    sharecode,
    matchId,
    matchDate,
    matchDuration,
    mapName,
    filePath,
    checksum,
    bytes,
    parserSchemaVersion,
  } = job.data;
  const identity = jobIdentity(steamId, sharecode);
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    match_id: matchId,
    stage: 'analyzing',
    attempt: job.attemptsMade + 1,
    bytes,
  });

  const pathAndQuery = '/process-demo';
  const payload = JSON.stringify({
    demo_path: filePath,
    steam_id: steamId,
    match_id: matchId,
    match_date: matchDate,
    match_duration: matchDuration,
    checksum,
    parser_schema_version: parserSchemaVersion || PARSER_SCHEMA_VERSION,
    job_id: identity,
  });
  const body = Buffer.from(payload);
  const response = await axios.post(
    `${config.services.goService}${pathAndQuery}`,
    payload,
    {
      timeout: config.http.goTimeout,
      headers: {
        ...buildServiceHeaders('POST', pathAndQuery, body),
        'Content-Type': 'application/json',
      },
      transformRequest: [(data) => data],
    },
  );
  if (response.data?.status !== 'success') {
    throw new Error('Go analyzer returned an incomplete result');
  }

  const processed = {
    match_id: matchId,
    steam_id: steamId,
    map_name: mapName,
    date: matchDate,
    duration: matchDuration,
    checksum,
    processed_at: new Date().toISOString(),
  };
  await redisClient.multi()
    .hSet(`${prefix}:matches`, matchId, JSON.stringify(processed))
    .zAdd(`${prefix}:user:${steamId}:processed`, {
      score: Date.parse(matchDate) || Date.now(),
      value: matchId,
    })
    .del(`${prefix}:dashboard-stats:${steamId}`)
    .exec();
  await setPipelineState(steamId, sharecode, {
    job_id: identity,
    match_id: matchId,
    stage: 'completed',
    attempt: job.attemptsMade + 1,
    bytes,
  });
  return processed;
}

function attachWorkerEvents(worker, stage) {
  worker.on('active', () => activeJobs.inc({ stage }));
  worker.on('completed', () => {
    activeJobs.dec({ stage });
    jobsTotal.inc({ stage, result: 'completed' });
  });
  worker.on('failed', (job, error) => {
    activeJobs.dec({ stage });
    jobsTotal.inc({ stage, result: 'failed' });
    const isTerminal = error instanceof UnrecoverableError
      || job.attemptsMade >= (job.opts.attempts || 1);
    if (stage === 'discovery' && job?.data?.steamId) {
      if (
        error.pipelineCode !== 'steam_credentials_invalid'
        && error.pipelineCode !== 'stale_credential_job'
      ) {
        const status = isTerminal ? 'discovery_failed' : 'retry_wait';
        const errorCode = error.pipelineCode || error.name || 'pipeline_error';
        updateCredentialStatus(
          job.data.steamId,
          job.data.credentialVersion,
          status,
          errorCode,
        ).then((updated) => {
          if (!updated) return;
          return publishEvent(job.data.steamId, {
            job_id: String(job.id),
            stage: status,
            attempt: job.attemptsMade,
            progress: isTerminal ? 100 : 0,
            error_code: errorCode,
          });
        }).catch(() => {});
      }
      return;
    }
    if (!job?.data?.steamId || !job?.data?.sharecode) return;
    setPipelineState(job.data.steamId, job.data.sharecode, {
      job_id: jobIdentity(job.data.steamId, job.data.sharecode),
      match_id: job.data.matchId,
      stage: isTerminal ? 'failed' : 'retry_wait',
      attempt: job.attemptsMade,
      error_code: error.pipelineCode || error.name || 'pipeline_error',
    }).catch(() => {});
  });
}

function timedProcessor(stage, processor) {
  return async (job) => {
    queueWaitDuration.observe(
      { stage },
      Math.max(0, (Date.now() - job.timestamp) / 1000),
    );
    if (job.attemptsMade > 0) {
      retriesTotal.inc({ stage });
    }
    const end = stageDuration.startTimer({ stage });
    const logicalJobId = job.data?.steamId && job.data?.sharecode
      ? jobIdentity(job.data.steamId, job.data.sharecode)
      : job.id;
    log('info', 'pipeline_stage_started', {
      job_id: logicalJobId,
      stage,
      attempt: job.attemptsMade + 1,
    });
    try {
      const result = await processor(job);
      end({ result: 'success' });
      log('info', 'pipeline_stage_completed', { job_id: logicalJobId, stage });
      return result;
    } catch (error) {
      end({ result: 'error' });
      log('error', 'pipeline_stage_failed', {
        job_id: logicalJobId,
        stage,
        error_code: error.pipelineCode || error.name || 'pipeline_error',
      });
      throw error;
    }
  };
}

async function triggerDiscovery(steamId, priority = 100, credentialVersion) {
  const job = await discoveryQueue.add(
    'discover-user',
    { steamId, priority, credentialVersion },
    {
      priority,
      attempts: 4,
      backoff: { type: 'exponential', delay: 5000, jitter: 0.5 },
      deduplication: { id: credentialVersion ? `${steamId}-${credentialVersion}` : steamId },
    },
  );
  return { status: 'queued', discovery_job_id: job.id };
}

async function getPipelineStatus(steamId) {
  const [credentialsExist, credentialStatus, rawStates] = await Promise.all([
    redisClient.exists(credentialKey(steamId)),
    redisClient.hGet(credentialKey(steamId), 'status'),
    redisClient.hGetAll(stateKey(steamId)),
  ]);
  const discoveryErrorCode = credentialsExist > 0
    ? await redisClient.hGet(credentialKey(steamId), 'discovery_error_code')
    : null;
  const jobs = Object.values(rawStates)
    .flatMap((value) => {
      try {
        return [JSON.parse(value)];
      } catch {
        return [];
      }
    })
    .sort((a, b) => b.updated_at.localeCompare(a.updated_at));
  const counts = jobs.reduce((result, item) => {
    result[item.stage] = (result[item.stage] || 0) + 1;
    return result;
  }, {});
  return {
    configured: credentialsExist > 0 && credentialStatus === 'configured',
    credential_status: credentialStatus || 'missing',
    discovery_error_code: discoveryErrorCode || null,
    counts,
    jobs: jobs.slice(0, 25),
  };
}

async function recoverLegacyGcFailures() {
  const failedJobs = await resolveQueue.getJobs(['failed'], 0, 9999);
  let recovered = 0;
  for (const job of failedJobs) {
    if (
      job.data.resolverVersion
      || job.failedReason !== 'Steam GC did not return a demo URL'
    ) {
      continue;
    }

    await setPipelineState(job.data.steamId, job.data.sharecode, {
      job_id: jobIdentity(job.data.steamId, job.data.sharecode),
      stage: 'retry_wait',
      attempt: job.attemptsMade,
      error_code: 'resolver_upgrade_retry',
    });
    await job.updateData({
      ...job.data,
      resolverVersion: GC_RESOLVER_VERSION,
    });
    await job.retry('failed', {
      resetAttemptsMade: true,
      resetAttemptsStarted: true,
    });
    recovered += 1;
  }
  return recovered;
}

async function recoverTlsUpgradeFailures() {
  const failedJobs = await downloadQueue.getJobs(['failed'], 0, 9999);
  let recovered = 0;
  for (const job of failedJobs) {
    if (
      job.data.downloaderVersion
      || !job.failedReason?.includes('SSL routines:ssl3_read_bytes')
    ) {
      continue;
    }

    await job.updateData({
      ...job.data,
      demoUrl: normalizeGcDemoUrl(job.data.demoUrl).href,
      downloaderVersion: DEMO_DOWNLOADER_VERSION,
    });
    await job.retry('failed', {
      resetAttemptsMade: true,
      resetAttemptsStarted: true,
    });
    await setPipelineState(job.data.steamId, job.data.sharecode, {
      job_id: jobIdentity(job.data.steamId, job.data.sharecode),
      match_id: job.data.matchId,
      stage: 'retry_wait',
      attempt: job.attemptsMade,
      error_code: 'downloader_transport_upgrade',
    });
    recovered += 1;
  }
  return recovered;
}

async function recoverServiceAuthFailures() {
  const failedJobs = await analysisQueue.getJobs(['failed'], 0, 9999);
  let recovered = 0;
  for (const job of failedJobs) {
    if (
      job.data.analyzerAuthVersion
      || job.failedReason !== 'Request failed with status code 503'
    ) {
      continue;
    }

    await job.updateData({
      ...job.data,
      analyzerAuthVersion: ANALYZER_AUTH_VERSION,
    });
    await job.retry('failed', {
      resetAttemptsMade: true,
      resetAttemptsStarted: true,
    });
    await setPipelineState(job.data.steamId, job.data.sharecode, {
      job_id: jobIdentity(job.data.steamId, job.data.sharecode),
      match_id: job.data.matchId,
      stage: 'retry_wait',
      attempt: job.attemptsMade,
      error_code: 'service_auth_upgrade',
    });
    recovered += 1;
  }
  return recovered;
}

async function recoverParserUpgradeFailures() {
  const failedJobs = await analysisQueue.getJobs(['failed'], 0, 9999);
  let recovered = 0;
  for (const job of failedJobs) {
    if (
      job.data.parserSchemaVersion === PARSER_SCHEMA_VERSION
      || job.failedReason !== 'Request failed with status code 422'
    ) {
      continue;
    }

    await job.updateData({
      ...job.data,
      parserSchemaVersion: PARSER_SCHEMA_VERSION,
    });
    await job.retry('failed', {
      resetAttemptsMade: true,
      resetAttemptsStarted: true,
    });
    await setPipelineState(job.data.steamId, job.data.sharecode, {
      job_id: jobIdentity(job.data.steamId, job.data.sharecode),
      match_id: job.data.matchId,
      stage: 'retry_wait',
      attempt: job.attemptsMade,
      error_code: 'parser_upgrade',
    });
    recovered += 1;
  }
  return recovered;
}

async function startPipelineV2() {
  if (isStarted || !config.pipelineV2Enabled) return;
  isStarted = true;
  try {
    await cleanupPartialDownloads(DEMOS_DIR);
    const recovered = await recoverLegacyGcFailures();
    if (recovered > 0) {
      log('info', 'legacy_gc_jobs_recovered', { count: recovered });
    }
    const recoveredDownloads = await recoverTlsUpgradeFailures();
    if (recoveredDownloads > 0) {
      log('info', 'legacy_download_jobs_recovered', { count: recoveredDownloads });
    }
    const recoveredAnalyses = await recoverServiceAuthFailures();
    if (recoveredAnalyses > 0) {
      log('info', 'legacy_analysis_jobs_recovered', { count: recoveredAnalyses });
    }
    const recoveredParses = await recoverParserUpgradeFailures();
    if (recoveredParses > 0) {
      log('info', 'legacy_parser_jobs_recovered', { count: recoveredParses });
    }

    const definitions = [
      [discoveryQueue.name, 'discovery', runDiscovery, { concurrency: 2 }],
      [resolveQueue.name, 'resolving', runResolve, { concurrency: config.gcQueue.concurrency }],
      [downloadQueue.name, 'downloading', runDownload, { concurrency: config.downloadQueue.concurrency }],
      [analysisQueue.name, 'analyzing', runAnalysis, { concurrency: config.goQueue.concurrency }],
    ];
    workers = definitions.map(([name, stage, processor, options]) => {
      const worker = new Worker(name, timedProcessor(stage, processor), {
        ...queueOptions,
        ...options,
      });
      attachWorkerEvents(worker, stage);
      return worker;
    });

    if (config.cron.enabled) {
      await discoveryQueue.upsertJobScheduler(
        'match-discovery',
        { pattern: config.cron.interval },
        { name: 'scan-users', data: {} },
      );
    }
  } catch (error) {
    isStarted = false;
    throw error;
  }
}

async function stopPipelineV2() {
  const gracefulClose = Promise.all(workers.map((worker) => worker.close())).then(() => true);
  const closedGracefully = await Promise.race([
    gracefulClose,
    new Promise((resolve) => setTimeout(() => resolve(false), 60000)),
  ]);
  if (!closedGracefully) {
    await Promise.all(workers.map((worker) => worker.close(true)));
  }
  await Promise.all([
    discoveryQueue.close(),
    resolveQueue.close(),
    downloadQueue.close(),
    analysisQueue.close(),
  ]);
  await connection.quit();
  workers = [];
  isStarted = false;
}

function isPipelineReady() {
  return isStarted && workers.every((worker) => worker.isRunning());
}

module.exports = {
  eventKey,
  getPipelineStatus,
  isPipelineReady,
  metricsRegistry,
  startPipelineV2,
  stopPipelineV2,
  triggerDiscovery,
  testHooks: {
    discoveryQueue,
    downloadQueue,
    analysisQueue,
    recoverLegacyGcFailures,
    recoverParserUpgradeFailures,
    recoverServiceAuthFailures,
    recoverTlsUpgradeFailures,
    resolveQueue,
    runDiscovery,
  },
};
