const test = require('node:test');
const assert = require('node:assert/strict');
const { loadPipeline, analysisJob } = require('./helpers/pipelineHarness');
const { emitRequests } = require('./helpers/emitAnalysisRequest');

test('new, default and recovered analysis jobs emit the supported canonical request', async () => {
  const requests = await emitRequests();
  assert.equal(requests.length, 3);
  for (const request of requests) {
    assert.equal(request.parser_schema_version, 'v16');
    assert.equal(request.match_id, 'contract-match');
    assert.equal(request.checksum, 'a'.repeat(64));
    assert.equal(request.demo_path, '/unused/demo.dem');
    assert.match(request.job_id, /^[a-f0-9]{64}$/);
  }
});

test('parser recovery retries only stale contract failures and preserves job identity', async () => {
  const pipeline = loadPipeline();
  const stale = analysisJob({ parserSchemaVersion: 'v11' });
  stale.failedReason = 'Request failed with status code 422';
  const current = analysisJob({ parserSchemaVersion: 'v16' });
  current.failedReason = stale.failedReason;
  const unrelated = analysisJob({ parserSchemaVersion: 'v11' });
  unrelated.failedReason = 'Request failed with status code 503';
  pipeline.analysisQueue.jobs = [stale, current, unrelated];
  const old = { ...stale.data };
  assert.equal(await pipeline.recoverParserUpgradeFailures(), 1);
  assert.deepEqual(stale.data, { ...old, parserSchemaVersion: 'v16' });
  assert.deepEqual(stale.retried, {
    state: 'failed', options: { resetAttemptsMade: true, resetAttemptsStarted: true },
  });
  assert.equal(current.retried, undefined);
  assert.equal(unrelated.retried, undefined);
  assert.equal(pipeline.states[0].error_code, 'parser_upgrade');
  assert.equal(await pipeline.recoverParserUpgradeFailures(), 0);
});

test('a rejected analyzer response never completes the job', async () => {
  const pipeline = loadPipeline({ post: async () => { throw new Error('contract rejected'); } });
  await assert.rejects(pipeline.runAnalysis(analysisJob()), /contract rejected/);
  assert.equal(pipeline.requests.length, 1);
  assert.equal(pipeline.states.some((state) => state.stage === 'completed'), false);
});

test('a queued stale version is recorded as the current attempt before sending HTTP', async () => {
  const job = analysisJob({ parserSchemaVersion: 'v11' });
  const pipeline = loadPipeline({ post: async (url, body) => {
    assert.equal(job.data.parserSchemaVersion, 'v16');
    assert.equal(JSON.parse(body).parser_schema_version, job.data.parserSchemaVersion);
    return { data: { status: 'success' } };
  } });
  await pipeline.runAnalysis(job);
  assert.equal(pipeline.requests.length, 1);
  assert.equal(job.data.parserSchemaVersion, 'v16');
});

test('expired CDN links request a fresh resolver result and retain a bounded refresh count', async () => {
  const pipeline = loadPipeline({ download: async () => {
    const error = new Error('URL expired'); error.statusCode = 410; throw error;
  } });
  await pipeline.runDownload(analysisJob());
  assert.equal(pipeline.resolveQueue.jobs[0].data.urlRefreshAttempts, 1);
  assert.equal(pipeline.states.at(-1).stage, 'retry_wait');
  await assert.rejects(pipeline.runDownload(analysisJob({ urlRefreshAttempts: 2 })), /remains unavailable/);
  assert.equal(pipeline.resolveQueue.jobs.length, 1);
});

test('a CDN refresh explicitly bypasses cached URLs in the Steam gateway', async () => {
  let resolverArguments;
  const pipeline = loadPipeline({ resolve: async (...args) => {
    resolverArguments = args;
    return { matchID: 'contract-match', demoUrl: 'https://replay.valve.net/test.dem.bz2' };
  } });
  await pipeline.runResolve(analysisJob({ refreshToken: 'retry-1', urlRefreshAttempts: 1 }));
  assert.deepEqual(resolverArguments, ['contract-sharecode', 2, true]);
  assert.equal(pipeline.downloadQueue.jobs[0].data.urlRefreshAttempts, 1);
});
