const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const { loadPipeline, analysisJob } = require('./helpers/pipelineHarness');

const seed = 'CSGO-AAAAA-BBBBB-CCCCC-DDDDD-EEEEE';
const next = 'CSGO-FFFFF-GGGGG-HHHHH-IIIII-JJJJJ';

function credentials(overrides = {}) {
  const nonce = crypto.randomBytes(12);
  const key = crypto.createHash('sha256').update('isolated-history-encryption-key').digest();
  const cipher = crypto.createCipheriv('aes-256-gcm', key, nonce);
  cipher.setAAD(Buffer.from('v1'));
  const encrypted = Buffer.concat([cipher.update('test-history-code'), cipher.final()]);
  return {
    auth_code: `v1.${Buffer.concat([nonce, encrypted, cipher.getAuthTag()]).toString('base64url')}`,
    known_code: seed, initial_code: seed, version: 'version-1', status: 'pending_validation',
    ...overrides,
  };
}

function setup(get, overrides = {}) {
  const stored = credentials(overrides);
  const calls = [];
  const events = [];
  const pipeline = loadPipeline({ get, redis: {
    async hGetAll(key) { return key.endsWith(':credentials') ? stored : {}; },
    async hGet(_key, field) { return stored[field]; },
    async exists() { return 1; },
    async sMembers() { return ['76561198000000000']; },
    async xAdd(_key, _id, { data }) { events.push(JSON.parse(data)); },
    async set() { return true; },
    async eval(_script, options) { calls.push(options); return 1; },
  } });
  const job = analysisJob({ steamId: '76561198000000000', credentialVersion: 'version-1' });
  job.name = 'discover-user';
  return { pipeline, job, calls, events };
}

test('first connection queues the supplied match even when Steam has no newer match', async () => {
  const { pipeline, job } = setup(async () => ({ status: 200, data: { result: { nextcode: 'n/a' } } }));
  await pipeline.runDiscovery(job);
  assert.deepEqual(pipeline.resolveQueue.jobs.map(item => item.data.sharecode), [seed]);
});

test('first connection imports the seed and subsequent matches in order', async () => {
  let calls = 0;
  const { pipeline, job } = setup(async () => ({ data: { result: { nextcode: calls++ === 0 ? next : 'n/a' } } }));
  await pipeline.runDiscovery(job);
  assert.deepEqual(pipeline.resolveQueue.jobs.map(item => item.data.sharecode), [seed, next]);
});

test('rejected credentials cannot queue the supplied match', async () => {
  const { pipeline, job, events } = setup(async () => { const error = new Error('rejected'); error.response = { status: 412 }; throw error; });
  await assert.rejects(pipeline.runDiscovery(job));
  assert.equal(pipeline.resolveQueue.jobs.length, 0);
  assert.equal(events.at(-1).stage, 'failed');
  assert.equal(events.at(-1).error_code, 'steam_credentials_invalid');
});

test('malformed successful Steam responses do not validate history or queue a match', async () => {
  const { pipeline, job } = setup(async () => ({ status: 200, data: {} }));
  await assert.rejects(pipeline.runDiscovery(job), /invalid history response/i);
  assert.equal(pipeline.resolveQueue.jobs.length, 0);
});

test('an already validated account remains configured during a background sync or outage', async () => {
  for (const status of ['discovering', 'retry_wait', 'discovery_failed']) {
    const { pipeline } = setup(undefined, { status, validated_at: '2026-10-15T00:00:00Z' });
    assert.equal((await pipeline.getPipelineStatus('76561198000000000')).configured, true, status);
  }
});

test('missing validation and revoked history access require credentials', async () => {
  for (const stored of [{ status: 'pending_validation' }, { status: 'needs_credentials', validated_at: '2026-10-15T00:00:00Z' }]) {
    const { pipeline } = setup(undefined, stored);
    assert.equal((await pipeline.getPipelineStatus('76561198000000000')).configured, false);
  }
});

test('an explicit empty history response still imports the supplied match', async () => {
  const { pipeline, job } = setup(async () => ({ status: 204 }));
  await pipeline.runDiscovery(job);
  assert.deepEqual(pipeline.resolveQueue.jobs.map(item => item.data.sharecode), [seed]);
});

test('invalid sharecodes and blank response fields never queue the initial match', async () => {
  for (const nextcode of ['https://untrusted.example/demo', '']) {
    const { pipeline, job } = setup(async () => ({ data: { result: { nextcode } } }));
    await assert.rejects(pipeline.runDiscovery(job));
    assert.equal(pipeline.resolveQueue.jobs.length, 0);
  }
});

test('a repeated cursor stops discovery instead of enqueueing the same match fifty times', async () => {
  const { pipeline, job } = setup(async () => ({ data: { result: { nextcode: seed } } }), { initial_code: '' });
  await assert.rejects(pipeline.runDiscovery(job), /repeated sharecode/);
  assert.equal(pipeline.resolveQueue.jobs.length, 0);
});

test('scheduled and manual discovery share a deduplication identity for the current credentials', async () => {
  const { pipeline } = setup();
  await pipeline.triggerDiscovery('76561198000000000', 1);
  await pipeline.runDiscovery({ name: 'scan-users' });
  const [manual, scheduled] = pipeline.discoveryQueue.jobs;
  assert.equal(manual.options.deduplication.id, scheduled.options.deduplication.id);
  assert.equal(manual.data.credentialVersion, 'version-1');
  assert.equal(scheduled.data.credentialVersion, 'version-1');
});
