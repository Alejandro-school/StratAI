const { loadPipeline, analysisJob } = require('./pipelineHarness');

// Used by Go/Python contract tests and the retained one-demo smoke.
// Optional JSON input supplies only local fixture metadata, never a real download.
async function emitRequests(data = {}) {
  const job = analysisJob(data);
  const pipeline = loadPipeline({ download: {
    filePath: job.data.filePath, checksum: job.data.checksum, bytes: job.data.bytes,
  } });
  await pipeline.runDownload(job);
  const queued = pipeline.analysisQueue.jobs[0];
  await pipeline.runAnalysis(analysisJob(queued.data));
  await pipeline.runAnalysis(analysisJob(data));
  const legacy = analysisJob({ ...data, parserSchemaVersion: 'v11' });
  legacy.failedReason = 'Request failed with status code 422';
  pipeline.analysisQueue.jobs = [legacy];
  const recovered = await pipeline.recoverParserUpgradeFailures();
  if (recovered !== 1 || !legacy.retried) throw new Error('Legacy job was not recovered');
  await pipeline.runAnalysis(legacy);
  return pipeline.requests.map(({ body }) => JSON.parse(body));
}

if (require.main === module) {
  const fs = require('node:fs');
  const data = process.argv[2] ? JSON.parse(fs.readFileSync(process.argv[2], 'utf8')) : {};
  emitRequests(data).then((requests) => process.stdout.write(JSON.stringify(requests)))
    .catch((error) => { console.error(error); process.exitCode = 1; });
}
module.exports = { emitRequests };
