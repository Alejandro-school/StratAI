process.env.SESSION_SECRET_KEY = 'test-session-secret-32-characters-long';
process.env.INTERNAL_SERVICE_SECRET = 'test-internal-secret-32-characters-long';
process.env.CREDENTIAL_ENCRYPTION_KEY = 'test-credential-secret-32-characters-long';
process.env.PIPELINE_NAMESPACE = `stratai:test:pipeline:${process.pid}`;
process.env.BOT_USERNAME = 'test-bot';
process.env.BOT_PASSWORD = 'test-password';
process.env.BOT_SHARED_SECRET = 'AAAAAAAAAAAAAAAAAAAAAA==';
process.env.CRON_ENABLED = 'false';

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('crypto');
const axios = require('axios');

const config = require('../services/config');
let ensureRedis;
let redisClient;
let stopPipelineV2;
let testHooks;

function encryptCredential(value) {
  const nonce = crypto.randomBytes(12);
  const key = crypto.createHash('sha256').update(config.credentialEncryptionKey).digest();
  const cipher = crypto.createCipheriv('aes-256-gcm', key, nonce);
  cipher.setAAD(Buffer.from('v1'));
  const ciphertext = Buffer.concat([cipher.update(value), cipher.final()]);
  return `v1.${Buffer.concat([nonce, ciphertext, cipher.getAuthTag()]).toString('base64url')}`;
}

function credentialsKey(steamId) {
  return `${config.pipelineNamespace}:user:${steamId}:credentials`;
}

function fakeJob(steamId) {
  const job = {
    name: 'discover-user',
    data: { steamId, priority: 1, credentialVersion: 'version-1' },
    updateProgress: async () => {},
    updateData: async (data) => {
      job.data = data;
    },
  };
  return job;
}

test(
  'discovery commits cursor after enqueue and handles zero matches and 412',
  { skip: process.env.REDIS_INTEGRATION !== 'true', timeout: 30000 },
  async () => {
    ({ ensureRedis, redisClient } = require('../services/redisClient'));
    ({ stopPipelineV2, testHooks } = require('../services/pipelineV2'));
    await ensureRedis();
    const originalAxiosGet = axios.get;
    const originalQueueAdd = testHooks.resolveQueue.add.bind(testHooks.resolveQueue);
    const originalGetJobs = testHooks.resolveQueue.getJobs.bind(testHooks.resolveQueue);
    const originalDownloadGetJobs = testHooks.downloadQueue.getJobs.bind(
      testHooks.downloadQueue,
    );
    const originalAnalysisGetJobs = testHooks.analysisQueue.getJobs.bind(
      testHooks.analysisQueue,
    );
    const initialCode = 'CSGO-11111-22222-33333-44444-55555';
    const nextCode = 'CSGO-AAAAA-BBBBB-CCCCC-DDDDD-EEEEE';

    try {
      const steamId = '76561198000000000';
      await redisClient.hSet(credentialsKey(steamId), {
        auth_code: encryptCredential('history-auth-code'),
        known_code: initialCode,
        version: 'version-1',
        status: 'pending_validation',
      });
      let calls = 0;
      axios.get = async () => {
        calls += 1;
        return { data: { result: { nextcode: calls === 1 ? nextCode : 'n/a' } } };
      };
      const result = await testHooks.runDiscovery(fakeJob(steamId));
      assert.equal(result.discovered, 1);
      assert.equal(
        await redisClient.hGet(credentialsKey(steamId), 'known_code'),
        nextCode,
      );
      const queuedCount = (
        await testHooks.resolveQueue.getWaitingCount()
        + await testHooks.resolveQueue.getPrioritizedCount()
      );
      assert.equal(queuedCount, 1);

      const zeroMatchUser = '76561198000000001';
      await redisClient.hSet(credentialsKey(zeroMatchUser), {
        auth_code: encryptCredential('history-auth-code'),
        known_code: initialCode,
        version: 'version-1',
        status: 'pending_validation',
      });
      axios.get = async () => ({ data: { result: { nextcode: 'n/a' } } });
      assert.deepEqual(
        await testHooks.runDiscovery(fakeJob(zeroMatchUser)),
        { discovered: 0 },
      );
      assert.equal(
        await redisClient.hGet(credentialsKey(zeroMatchUser), 'status'),
        'configured',
      );

      const enqueueFailureUser = '76561198000000002';
      await redisClient.hSet(credentialsKey(enqueueFailureUser), {
        auth_code: encryptCredential('history-auth-code'),
        known_code: initialCode,
        version: 'version-1',
        status: 'pending_validation',
      });
      axios.get = async () => ({ data: { result: { nextcode: nextCode } } });
      testHooks.resolveQueue.add = async () => {
        throw new Error('simulated queue failure');
      };
      await assert.rejects(testHooks.runDiscovery(fakeJob(enqueueFailureUser)));
      assert.equal(
        await redisClient.hGet(credentialsKey(enqueueFailureUser), 'known_code'),
        initialCode,
      );
      testHooks.resolveQueue.add = originalQueueAdd;

      const invalidUser = '76561198000000003';
      await redisClient.hSet(credentialsKey(invalidUser), {
        auth_code: encryptCredential('invalid-history-auth-code'),
        known_code: initialCode,
        version: 'version-1',
        status: 'pending_validation',
      });
      axios.get = async () => {
        const error = new Error('precondition failed');
        error.response = { status: 412 };
        throw error;
      };
      await assert.rejects(testHooks.runDiscovery(fakeJob(invalidUser)));
      assert.equal(
        await redisClient.hGet(credentialsKey(invalidUser), 'status'),
        'needs_credentials',
      );

      const legacyUser = '76561198000000004';
      const legacySharecode = 'CSGO-FFFFF-GGGGG-HHHHH-IIIII-JJJJJ';
      let recoveredData;
      let retried = false;
      testHooks.resolveQueue.getJobs = async () => [{
        data: { steamId: legacyUser, sharecode: legacySharecode },
        failedReason: 'Steam GC did not return a demo URL',
        attemptsMade: 4,
        updateData: async (data) => {
          recoveredData = data;
        },
        retry: async () => {
          retried = true;
        },
      }];
      assert.equal(await testHooks.recoverLegacyGcFailures(), 1);
      assert.equal(recoveredData.resolverVersion, 2);
      assert.equal(retried, true);
      assert.equal(
        JSON.parse(
          await redisClient.hGet(
            `${config.pipelineNamespace}:user:${legacyUser}:pipeline`,
            legacySharecode,
          ),
        ).error_code,
        'resolver_upgrade_retry',
      );

      let recoveredDownloadData;
      let downloadRetried = false;
      testHooks.downloadQueue.getJobs = async () => [{
        data: {
          steamId: legacyUser,
          sharecode: legacySharecode,
          matchId: '123456789',
          demoUrl: 'https://replay196.valve.net/demo.dem.bz2',
        },
        failedReason: 'write EPROTO: SSL routines:ssl3_read_bytes: handshake failure',
        attemptsMade: 4,
        updateData: async (data) => {
          recoveredDownloadData = data;
        },
        retry: async () => {
          downloadRetried = true;
        },
      }];
      assert.equal(await testHooks.recoverTlsUpgradeFailures(), 1);
      assert.equal(recoveredDownloadData.downloaderVersion, 2);
      assert.equal(
        recoveredDownloadData.demoUrl,
        'http://replay196.valve.net/demo.dem.bz2',
      );
      assert.equal(downloadRetried, true);

      let recoveredAnalysisData;
      let analysisRetried = false;
      testHooks.analysisQueue.getJobs = async () => [{
        data: {
          steamId: legacyUser,
          sharecode: legacySharecode,
          matchId: '123456789',
        },
        failedReason: 'Request failed with status code 503',
        attemptsMade: 3,
        updateData: async (data) => {
          recoveredAnalysisData = data;
        },
        retry: async () => {
          analysisRetried = true;
        },
      }];
      assert.equal(await testHooks.recoverServiceAuthFailures(), 1);
      assert.equal(recoveredAnalysisData.analyzerAuthVersion, 2);
      assert.equal(analysisRetried, true);

      let recoveredParserData;
      let parserRetried = false;
      testHooks.analysisQueue.getJobs = async () => [{
        data: {
          steamId: legacyUser,
          sharecode: legacySharecode,
          matchId: '123456789',
          parserSchemaVersion: 'v2',
        },
        failedReason: 'Request failed with status code 422',
        attemptsMade: 3,
        updateData: async (data) => {
          recoveredParserData = data;
        },
        retry: async () => {
          parserRetried = true;
        },
      }];
      assert.equal(await testHooks.recoverParserUpgradeFailures(), 1);
      assert.equal(recoveredParserData.parserSchemaVersion, 'v3');
      assert.equal(parserRetried, true);
    } finally {
      axios.get = originalAxiosGet;
      testHooks.resolveQueue.add = originalQueueAdd;
      testHooks.resolveQueue.getJobs = originalGetJobs;
      testHooks.downloadQueue.getJobs = originalDownloadGetJobs;
      testHooks.analysisQueue.getJobs = originalAnalysisGetJobs;
      await testHooks.resolveQueue.obliterate({ force: true });
      await testHooks.downloadQueue.obliterate({ force: true });
      await testHooks.analysisQueue.obliterate({ force: true });
      await testHooks.discoveryQueue.obliterate({ force: true });
      const keys = await redisClient.keys(`${config.pipelineNamespace}:*`);
      if (keys.length) await redisClient.del(keys);
      await stopPipelineV2();
      if (redisClient.isOpen) await redisClient.quit();
    }
  },
);
