process.env.SESSION_SECRET_KEY = 'test-secret';

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('crypto');
const fs = require('fs/promises');
const os = require('os');
const path = require('path');

const {
  cleanupPartialDownloads,
  normalizeGcDemoUrl,
  validateDemoUrl,
  validateExistingDemo,
} = require('../services/demoDownload');

test('accepts HTTPS CDNs and only the exact legacy Valve replay HTTP hosts', () => {
  assert.equal(
    validateDemoUrl('https://cdn.steamcontent.com/demo.dem.bz2').protocol,
    'https:',
  );
  assert.equal(
    validateDemoUrl('http://replay196.valve.net/demo.dem.bz2').protocol,
    'http:',
  );
  assert.throws(
    () => validateDemoUrl('http://cdn.steamcontent.com/demo.dem.bz2'),
    /not allowed/,
  );
  assert.throws(
    () => validateDemoUrl('https://steamcontent.com.attacker.example/demo.dem'),
    /not allowed/,
  );
});

test('preserves legacy Valve HTTP URLs without relaxing the allowlist', () => {
  assert.equal(
    normalizeGcDemoUrl('http://replay123.valve.net/demo.dem.bz2').href,
    'http://replay123.valve.net/demo.dem.bz2',
  );
  assert.equal(
    normalizeGcDemoUrl('https://replay123.valve.net/demo.dem.bz2').href,
    'http://replay123.valve.net/demo.dem.bz2',
  );
  assert.throws(
    () => normalizeGcDemoUrl('http://valve.net.attacker.example/demo.dem.bz2'),
    /not allowed/,
  );
  assert.throws(
    () => normalizeGcDemoUrl('ftp://replay123.valve.net/demo.dem.bz2'),
    /not allowed/,
  );
});

test('reuses a demo only when its checksum sidecar is valid', async () => {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'stratai-demo-'));
  const demoPath = path.join(directory, 'match.dem');
  const content = Buffer.alloc(102400, 7);
  const checksum = crypto.createHash('sha256').update(content).digest('hex');
  await fs.writeFile(demoPath, content);
  await fs.writeFile(`${demoPath}.sha256`, checksum);

  const valid = await validateExistingDemo(demoPath);
  assert.equal(valid.checksum, checksum);
  await fs.writeFile(`${demoPath}.sha256`, 'invalid');
  assert.equal(await validateExistingDemo(demoPath), null);
});

test('cleans abandoned partial downloads without touching completed demos', async () => {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'stratai-parts-'));
  const partial = path.join(directory, 'match.dem.part.deadbeef');
  const checksumPartial = path.join(directory, 'match.dem.sha256.part');
  const complete = path.join(directory, 'match.dem');
  await fs.writeFile(partial, 'partial');
  await fs.writeFile(checksumPartial, 'partial');
  await fs.writeFile(complete, 'complete');

  await cleanupPartialDownloads(directory);
  await assert.rejects(fs.stat(partial));
  await assert.rejects(fs.stat(checksumPartial));
  assert.equal((await fs.stat(complete)).isFile(), true);
});
