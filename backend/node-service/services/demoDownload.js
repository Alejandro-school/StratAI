const crypto = require('crypto');
const fs = require('fs');
const fsPromises = require('fs/promises');
const http = require('http');
const https = require('https');
const path = require('path');
const { Transform, pipeline } = require('stream');
const { promisify } = require('util');
const unbzip2Stream = require('unbzip2-stream');
const config = require('./config');

const pipelineAsync = promisify(pipeline);
const MAX_REDIRECTS = 3;
const PROGRESS_INTERVAL_BYTES = 16 * 1024 * 1024;
const LEGACY_VALVE_REPLAY_HOST = /^replay\d+\.valve\.net$/;

function isAllowedHost(hostname) {
  const normalized = hostname.toLowerCase();
  return config.downloadQueue.allowedHosts.some((allowed) => (
    allowed.startsWith('.')
      ? normalized.endsWith(allowed)
      : normalized === allowed
  ));
}

function validateDemoUrl(value) {
  const url = new URL(value);
  const isSecureCdn = url.protocol === 'https:' && isAllowedHost(url.hostname);
  const isLegacyValveReplay = (
    url.protocol === 'http:'
    && LEGACY_VALVE_REPLAY_HOST.test(url.hostname.toLowerCase())
  );
  if (!isSecureCdn && !isLegacyValveReplay) {
    throw new Error('Demo URL host or protocol is not allowed');
  }
  return url;
}

function normalizeGcDemoUrl(value) {
  const url = new URL(value);
  if (
    ['http:', 'https:'].includes(url.protocol)
    && LEGACY_VALVE_REPLAY_HOST.test(url.hostname.toLowerCase())
  ) {
    url.protocol = 'http:';
  }
  return validateDemoUrl(url.href);
}

async function hasEnoughDiskSpace(directory) {
  if (typeof fsPromises.statfs !== 'function') return true;
  const stats = await fsPromises.statfs(directory);
  return stats.bavail * stats.bsize >= config.downloadQueue.minFreeBytes;
}

async function validateExistingDemo(filePath) {
  try {
    const [stats, expectedChecksum] = await Promise.all([
      fsPromises.stat(filePath),
      fsPromises.readFile(`${filePath}.sha256`, 'utf8'),
    ]);
    if (
      stats.size < config.downloadQueue.minBytes
      || stats.size > config.downloadQueue.maxBytes
    ) {
      return null;
    }

    const digest = crypto.createHash('sha256');
    const stream = fs.createReadStream(filePath);
    for await (const chunk of stream) digest.update(chunk);
    const checksum = digest.digest('hex');
    return checksum === expectedChecksum.trim()
      ? { filePath, checksum, bytes: stats.size, reused: true }
      : null;
  } catch {
    return null;
  }
}

function openResponse(url, redirectsLeft = MAX_REDIRECTS) {
  return new Promise((resolve, reject) => {
    const transport = url.protocol === 'https:' ? https : http;
    const request = transport.get(url, { timeout: 30000 }, (response) => {
      if (
        response.statusCode >= 300
        && response.statusCode < 400
        && response.headers.location
      ) {
        response.resume();
        if (redirectsLeft === 0) {
          reject(new Error('Too many demo download redirects'));
          return;
        }
        try {
          const redirected = validateDemoUrl(new URL(response.headers.location, url).href);
          resolve(openResponse(redirected, redirectsLeft - 1));
        } catch (error) {
          reject(error);
        }
        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        const error = new Error(`Demo download returned HTTP ${response.statusCode}`);
        error.statusCode = response.statusCode;
        reject(error);
        return;
      }
      resolve({ response, finalUrl: url });
    });
    request.once('timeout', () => request.destroy(new Error('Demo download timed out')));
    request.once('error', reject);
  });
}

async function writeChecksum(filePath, checksum) {
  const temporary = `${filePath}.sha256.part`;
  await fsPromises.writeFile(temporary, checksum, { encoding: 'utf8', mode: 0o600 });
  await fsPromises.rename(temporary, `${filePath}.sha256`);
}

async function downloadDemo(urlValue, filePath, { onProgress } = {}) {
  const existing = await validateExistingDemo(filePath);
  if (existing) return existing;

  await fsPromises.mkdir(path.dirname(filePath), { recursive: true });
  await Promise.all([
    fsPromises.unlink(filePath).catch((error) => {
      if (error.code !== 'ENOENT') throw error;
    }),
    fsPromises.unlink(`${filePath}.sha256`).catch((error) => {
      if (error.code !== 'ENOENT') throw error;
    }),
  ]);
  if (!(await hasEnoughDiskSpace(path.dirname(filePath)))) {
    const error = new Error('Insufficient disk space for demo download');
    error.code = 'INSUFFICIENT_DISK';
    throw error;
  }

  const url = validateDemoUrl(urlValue);
  const partPath = `${filePath}.part.${crypto.randomUUID()}`;
  const digest = crypto.createHash('sha256');
  let bytes = 0;
  let nextProgressBytes = PROGRESS_INTERVAL_BYTES;
  const meter = new Transform({
    transform(chunk, _encoding, callback) {
      bytes += chunk.length;
      if (bytes > config.downloadQueue.maxBytes) {
        callback(new Error('Demo exceeds configured size limit'));
        return;
      }
      digest.update(chunk);
      if (onProgress && bytes >= nextProgressBytes) {
        nextProgressBytes = bytes + PROGRESS_INTERVAL_BYTES;
        try {
          onProgress(bytes);
        } catch {
          // Progress reporting must not interrupt a valid download.
        }
      }
      callback(null, chunk);
    },
  });

  try {
    const { response, finalUrl } = await openResponse(url);
    const streams = [response];
    if (finalUrl.pathname.toLowerCase().endsWith('.bz2')) {
      streams.push(unbzip2Stream());
    }
    streams.push(
      meter,
      fs.createWriteStream(partPath, { flags: 'wx', mode: 0o600 }),
    );
    await pipelineAsync(...streams);
    if (bytes < config.downloadQueue.minBytes) {
      throw new Error(`Demo is too small (${bytes} bytes)`);
    }

    const checksum = digest.digest('hex');
    await fsPromises.rename(partPath, filePath);
    await writeChecksum(filePath, checksum);
    return { filePath, checksum, bytes, reused: false };
  } catch (error) {
    await fsPromises.unlink(partPath).catch(() => {});
    throw error;
  }
}

async function cleanupPartialDownloads(directory) {
  const entries = await fsPromises.readdir(directory, { withFileTypes: true }).catch(() => []);
  await Promise.all(entries
    .filter((entry) => (
      entry.isFile()
      && (entry.name.includes('.part.') || entry.name.endsWith('.sha256.part'))
    ))
    .map((entry) => fsPromises.unlink(path.join(directory, entry.name)).catch(() => {})));
}

module.exports = {
  cleanupPartialDownloads,
  downloadDemo,
  normalizeGcDemoUrl,
  validateDemoUrl,
  validateExistingDemo,
};
