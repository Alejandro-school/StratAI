const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { EventEmitter } = require('node:events');
const { createRequire } = require('node:module');

const gatewayPath = path.resolve(__dirname, '../services/steamGateway.js');
const requireFromGateway = createRequire(gatewayPath);
const source = fs.readFileSync(gatewayPath, 'utf8');
const sharecode = 'CSGO-AAAAA-BBBBB-CCCCC-DDDDD-EEEEE';
const demoUrl = 'http://replay123.valve.net/730/test.dem.bz2';

function gatewayHarness() {
  const cache = new Map();
  const requests = [];
  const csgo = new EventEmitter();
  const bot = { client: {}, csgo, activeRequests: 0, available: true };
  // A social relationship must never be consulted to resolve a sharecode.
  Object.defineProperty(bot.client, 'myFriends', { get() { throw new Error('Unexpected friendship check'); } });
  Object.defineProperty(bot, 'friendsListReady', { get() { throw new Error('Unexpected friends-list dependency'); } });
  csgo._send = (_type, _protobuf, payload) => {
    requests.push(payload);
    setImmediate(() => csgo.emit('matchList', [{
      matchid: payload.matchid,
      matchtime: 1,
      roundstatsall: [{ map: demoUrl, match_duration: 100 }],
    }]));
  };
  const pool = {
    bots: [bot], initialize() {}, getPrimaryBot: () => bot,
    getAvailableBot: () => bot.available ? bot : null,
  };
  const replacements = {
    './botPool': { botPool: pool },
    './config': { pipelineNamespace: 'isolated:test', ttl: { demoUrl: 60 } },
    './redisClient': { redisClient: { get: async key => cache.get(key), set: async (key, value) => cache.set(key, value), del: async key => cache.delete(key) } },
    './demoDownload': { normalizeGcDemoUrl: value => new URL(value) },
  };
  const context = {
    module: { exports: {} },
    require: name => name in replacements ? replacements[name] : requireFromGateway(name),
    setTimeout, clearTimeout,
  };
  vm.runInNewContext(source, context, { filename: gatewayPath });
  return { gateway: context.module.exports, bot, requests };
}

test('resolves a sharecode with no friendship state and only the match token', async () => {
  const { gateway, requests, bot } = gatewayHarness();
  const result = await gateway.resolveSharecode(sharecode);
  assert.equal(result.demoUrl, demoUrl);
  assert.deepEqual(Object.keys(requests[0]).sort(), ['matchid', 'outcomeid', 'token']);
  assert.equal(bot.activeRequests, 0);
});

test('requires a GC session for uncached demos and can retry after reconnection', async () => {
  const { gateway, bot, requests } = gatewayHarness();
  bot.available = false;
  await assert.rejects(gateway.resolveSharecode(sharecode), /No Steam GC session/);
  assert.equal(requests.length, 0);
  bot.available = true;
  assert.equal((await gateway.resolveSharecode(sharecode)).demoUrl, demoUrl);
});

test('reuses a resolved URL without another GC request', async () => {
  const { gateway, bot, requests } = gatewayHarness();
  await gateway.resolveSharecode(sharecode);
  bot.available = false;
  assert.equal((await gateway.resolveSharecode(sharecode)).demoUrl, demoUrl);
  assert.equal(requests.length, 1);
});

test('a missing demo URL releases the bot before another request', async () => {
  const { gateway, bot } = gatewayHarness();
  bot.csgo._send = (_type, _protobuf, payload) => {
    setImmediate(() => bot.csgo.emit('matchList', [{ matchid: payload.matchid, roundstatsall: [] }]));
  };
  await assert.rejects(gateway.resolveSharecode(sharecode, 0), /did not return a demo URL/);
  assert.equal(bot.activeRequests, 0);
});

test('refreshing an expired CDN URL bypasses the cached resolver result', async () => {
  const { gateway, requests } = gatewayHarness();
  await gateway.resolveSharecode(sharecode);
  await gateway.resolveSharecode(sharecode, 2, true);
  assert.equal(requests.length, 2);
});

test('a synchronous GC transport failure can be retried without an orphaned request', async () => {
  const { gateway, bot } = gatewayHarness();
  const send = bot.csgo._send;
  bot.csgo._send = () => { throw new Error('GC transport unavailable'); };
  await assert.rejects(gateway.resolveSharecode(sharecode), /GC transport unavailable/);
  assert.equal(bot.activeRequests, 0);
  bot.csgo._send = send;
  assert.equal((await gateway.resolveSharecode(sharecode)).demoUrl, demoUrl);
});
