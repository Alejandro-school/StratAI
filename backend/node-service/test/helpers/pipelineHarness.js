const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Execute the real pipeline module with only external side effects replaced.
// No Redis connection, Steam client, HTTP request or download can escape this harness.
function loadPipeline({ download, post, get, resolve, redis = {} } = {}) {
  const queues = new Map();
  const requests = [];
  const states = [];
  class Queue {
    constructor(name) { this.name = name; this.jobs = []; queues.set(name, this); }
    async add(name, data, options) {
      const job = { id: String(this.jobs.length + 1), name, data, options };
      this.jobs.push(job);
      return job;
    }
    async getJobs() { return this.jobs; }
  }
  class Metric { inc() {} dec() {} observe() {} }
  const transaction = {
    hSet() { return this; }, zAdd() { return this; }, del() { return this; },
    async exec() { return []; },
  };
  const dependencies = {
    crypto: require('node:crypto'), path,
    axios: { get, post: async (url, body, options) => {
      requests.push({ url, body, options });
      return post ? post(url, body, options) : { data: { status: 'success' } };
    } },
    ioredis: class {},
    bullmq: { Queue, Worker: class {}, UnrecoverableError: Error },
    'prom-client': {
      Counter: Metric, Gauge: Metric, Histogram: Metric, Registry: class {},
      collectDefaultMetrics() {},
    },
    './config': {
      pipelineNamespace: 'stratai:contract-test', redis: { url: 'unused' },
      services: { goService: 'http://unused.invalid' }, http: { goTimeout: 1000 },
      credentialEncryptionKey: 'isolated-history-encryption-key',
      steam: { apiKey: 'isolated-api-key' }, cron: { steamRequestSpacing: 1 },
      downloadQueue: { maxRetries: 4, retryBaseDelay: 1 },
    },
    './redisClient': { redisClient: {
      async hSet(key, field, value) { states.push(JSON.parse(value)); },
      async xAdd() {}, async xTrim() {}, multi: () => transaction,
      ...redis,
    } },
    './steamGateway': { resolveSharecode: resolve || (() => { throw new Error('Steam is forbidden'); }) },
    './demoDownload': {
      downloadDemo: async () => {
        if (!download) throw new Error('Downloads are forbidden');
        return typeof download === 'function' ? download() : download;
      },
      normalizeGcDemoUrl: (url) => url, cleanupPartialDownloads() {},
    },
    './serviceAuth': { buildServiceHeaders: () => ({ 'X-Test-Transport': 'isolated' }) },
    './logger': { log() {} },
  };
  const filename = path.resolve(__dirname, '../../services/pipelineV2.js');
  const module = { exports: {} };
  const execute = vm.runInThisContext(
    `(function(require, module, exports, __filename, __dirname) {\n${fs.readFileSync(filename, 'utf8')}\n})`,
    { filename },
  );
  execute((name) => {
    if (!Object.hasOwn(dependencies, name)) throw new Error(`Unstubbed dependency: ${name}`);
    return dependencies[name];
  }, module, module.exports, filename, path.dirname(filename));
  return { ...module.exports, ...module.exports.testHooks, queues, requests, states };
}

function analysisJob(data = {}) {
  const job = {
    attemptsMade: 0,
    data: {
      steamId: 'contract-actor', sharecode: 'contract-sharecode', matchId: 'contract-match',
      matchDate: '', matchDuration: 0, mapName: 'de_mirage', filePath: '/unused/demo.dem',
      checksum: 'a'.repeat(64), bytes: 200000, ...data,
    },
    async updateProgress() {},
    async updateData(next) { job.data = next; },
    async retry(state, options) { job.retried = { state, options }; },
  };
  return job;
}

module.exports = { loadPipeline, analysisJob };
