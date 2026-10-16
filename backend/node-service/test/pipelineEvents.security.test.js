const test = require('node:test');
const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');

const USER_A = '76561198000000001';
const USER_B = '76561198000000002';
const USER_C = '76561198000000003';
const settle = () => new Promise((resolve) => setImmediate(resolve));

class StreamResponse extends EventEmitter {
  constructor(writeResults = []) {
    super();
    this.statusCode = 200;
    this.headers = {};
    this.writes = [];
    this.writeResults = [...writeResults];
    this.headersSent = false;
    this.writableEnded = false;
    this.destroyed = false;
  }

  status(code) { this.statusCode = code; return this; }
  setHeader(name, value) { this.headers[name] = value; return this; }
  writeHead(code, headers) { this.statusCode = code; Object.assign(this.headers, headers); this.headersSent = true; return this; }
  flushHeaders() { this.headersSent = true; }
  json(body) { this.body = body; this.end(); return this; }
  write(body) {
    this.headersSent = true;
    this.writes.push(body);
    return this.writeResults.length ? this.writeResults.shift() : true;
  }
  end() { this.writableEnded = true; this.emit('finish'); return this; }
  destroy() { this.destroyed = true; this.emit('close'); return this; }
}

class StreamReader extends EventEmitter {
  constructor({ failConnect = false, emitConnectError = false } = {}) {
    super();
    this.failConnect = failConnect;
    this.emitConnectError = emitConnectError;
    this.isOpen = false;
    this.connectCalls = 0;
    this.destroyCalls = 0;
    this.reads = [];
    this.pending = null;
  }

  async connect() {
    this.connectCalls += 1;
    if (this.failConnect) {
      const error = new Error('Synthetic connection failure');
      if (this.emitConnectError) this.emit('error', error);
      throw error;
    }
    this.isOpen = true;
    return this;
  }
  async xInfoStream() { return { lastGeneratedId: '0-0' }; }
  xRead(streams, options) {
    assert.equal(this.isOpen, true, 'Blocking reads require the dedicated connected reader');
    this.reads.push({ streams, options });
    return new Promise((resolve, reject) => { this.pending = { resolve, reject }; });
  }
  resolveRead(streams) {
    const pending = this.pending;
    this.pending = null;
    assert.ok(pending, 'A read must already be blocked');
    pending.resolve(streams);
  }
  destroy() {
    this.destroyCalls += 1;
    this.isOpen = false;
    const pending = this.pending;
    this.pending = null;
    pending?.reject(new Error('Synthetic connection closed'));
  }
}

function harness(t, options = {}) {
  // Deferred import lets unrelated Node tests collect during implementation.
  const { createPipelineEventHandler } = require('../services/pipelineEvents');
  const readers = [];
  const requests = [];
  let connectFailuresRemaining = options.connectFailures || 0;
  let sharedReadCalls = 0;
  const redisClient = {
    duplicate() {
      const reader = new StreamReader({
        failConnect: connectFailuresRemaining > 0,
        emitConnectError: options.emitConnectError,
      });
      connectFailuresRemaining = Math.max(0, connectFailuresRemaining - 1);
      readers.push(reader);
      return reader;
    },
    async xInfoStream() { return { lastGeneratedId: '0-0' }; },
    async xRead() {
      sharedReadCalls += 1;
      throw new Error('The shared Redis connection must never execute a blocking read');
    },
  };
  const handler = createPipelineEventHandler({
    redisClient,
    getPipelineStatus: async (steamId) => ({ steam_id: steamId, stage: 'idle' }),
    eventKey: (steamId) => `synthetic:events:${steamId}`,
    maxStreamsPerUser: options.maxStreamsPerUser ?? 2,
    maxStreams: options.maxStreams ?? 100,
  });

  function start(steamId = USER_A, writeResults = []) {
    const req = new EventEmitter();
    req.params = { steamId };
    req.headers = {};
    const res = new StreamResponse(writeResults);
    const completed = handler(req, res);
    const request = { req, res, completed };
    requests.push(request);
    return request;
  }

  t.after(async () => {
    for (const { res } of requests) res.emit('close');
    for (const reader of readers) if (reader.isOpen || reader.pending) reader.destroy();
    await Promise.allSettled(requests.map(({ completed }) => completed));
  });
  return { start, readers, get sharedReadCalls() { return sharedReadCalls; } };
}

test('SSE reads use a connected duplicate and never block the shared Redis client', { timeout: 2000 }, async (t) => {
  const state = harness(t);
  const request = state.start();
  await settle();

  assert.equal(request.res.statusCode, 200);
  assert.equal(state.readers.length, 1);
  assert.equal(state.readers[0].connectCalls, 1);
  assert.equal(state.readers[0].reads.length, 1);
  assert.ok(state.readers[0].reads[0].options.BLOCK > 0);
  assert.equal(state.sharedReadCalls, 0);
  assert.match(request.res.writes.join(''), /event: snapshot/);
});

for (const signal of ['close', 'error']) {
  test(`response ${signal} interrupts the pending Redis read and destroys its connection`, { timeout: 2000 }, async (t) => {
    const state = harness(t);
    const request = state.start();
    await settle();
    assert.ok(state.readers[0].pending);

    request.res.emit(signal, ...(signal === 'error' ? [new Error('Synthetic client error')] : []));
    await request.completed;

    assert.equal(state.readers[0].isOpen, false);
    assert.ok(state.readers[0].destroyCalls > 0);
    assert.equal(state.readers[0].pending, null);
  });
}

test('per-user stream quota rejects a third connection and disconnect releases its slot', { timeout: 2000 }, async (t) => {
  const state = harness(t, { maxStreamsPerUser: 2 });
  const first = state.start();
  const second = state.start();
  await settle();
  const rejected = state.start();
  await settle();

  assert.equal(rejected.res.statusCode, 429);
  assert.equal(state.readers.length, 2);
  assert.equal(second.res.statusCode, 200);
  first.res.emit('close');
  await first.completed;
  const replacement = state.start();
  await settle();
  assert.equal(replacement.res.statusCode, 200);
  assert.equal(state.readers.length, 3);
  assert.equal(state.readers.filter((reader) => reader.isOpen).length, 2);
});

test('global stream quota also applies across distinct users', { timeout: 2000 }, async (t) => {
  const state = harness(t, { maxStreams: 2 });
  state.start(USER_A);
  state.start(USER_B);
  await settle();
  const rejected = state.start(USER_C);
  await settle();

  assert.equal(rejected.res.statusCode, 429);
  assert.equal(state.readers.length, 2);
});

test('reader connection failure returns 503 and releases the reserved quota slot', { timeout: 2000 }, async (t) => {
  const state = harness(t, { connectFailures: 1, maxStreams: 1, maxStreamsPerUser: 1 });
  const failed = state.start();
  await failed.completed;
  assert.equal(failed.res.statusCode, 503);
  assert.equal(state.readers.filter((reader) => reader.isOpen).length, 0);

  const recovered = state.start();
  await settle();

  assert.equal(recovered.res.statusCode, 200);
  assert.equal(state.readers.filter((reader) => reader.isOpen).length, 1);
});

test('a Redis error event followed by connect rejection returns 503 and releases its stream slot', { timeout: 2000 }, async (t) => {
  const state = harness(t, {
    connectFailures: 1,
    emitConnectError: true,
    maxStreams: 1,
    maxStreamsPerUser: 1,
  });
  const failed = state.start();
  await failed.completed;
  assert.equal(state.readers.filter((reader) => reader.isOpen).length, 0);

  const recovered = state.start();
  await settle();

  assert.equal(recovered.res.statusCode, 200, 'Failed readers must not retain the quota slot');
  assert.equal(state.readers.filter((reader) => reader.isOpen).length, 1);
  assert.equal(failed.res.statusCode, 503, 'Redis failure is not a successful empty HTTP response');
});

for (const streamResult of [null, [{ name: 'synthetic', messages: [
  { id: '1-0', message: { data: '{"stage":"synthetic"}' } },
]}]]) {
  test(`backpressure on ${streamResult ? 'an event' : 'a heartbeat'} closes its dedicated reader`, { timeout: 2000 }, async (t) => {
    const state = harness(t);
    const request = state.start(USER_A, [true, false]);
    await settle();
    state.readers[0].resolveRead(streamResult);
    await request.completed;

    assert.equal(request.res.writes.length, 2);
    assert.equal(state.readers[0].isOpen, false);
    assert.ok(state.readers[0].destroyCalls > 0);
    assert.ok(request.res.writableEnded || request.res.destroyed);
  });
}
