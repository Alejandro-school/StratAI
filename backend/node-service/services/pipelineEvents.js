/** Each SSE reader owns its blocking Redis connection and a bounded stream slot. */
function createPipelineEventHandler({ redisClient, getPipelineStatus, eventKey,
  maxStreamsPerUser = 2, maxStreams = 100 }) {
  const users = new Map();
  let active = 0;
  return async function pipelineEvents(req, res) {
    const { steamId } = req.params;
    if (!/^\d{17}$/.test(steamId || '')) return res.status(400).json({ error: 'Invalid steam_id' });
    const count = users.get(steamId) || 0;
    if (count >= maxStreamsPerUser || active >= maxStreams) {
      return res.status(429).json({ error: 'Too many event streams' });
    }
    users.set(steamId, count + 1);
    active += 1;
    let reader;
    let closed = false;
    let clientClosed = false;
    let released = false;
    const release = () => {
      closed = true;
      if (released) return;
      released = true;
      active -= 1;
      const remaining = (users.get(steamId) || 1) - 1;
      if (remaining) users.set(steamId, remaining);
      else users.delete(steamId);
      if (reader) {
        try { reader.destroy(); } catch { /* Already closed, including failed connects. */ }
      }
    };
    const disconnect = () => { clientClosed = true; release(); };
    const readerFailed = () => {
      if (released) return;
      if (!clientClosed && !res.headersSent && !res.writableEnded) res.status(503).json({ error: 'Event stream unavailable' });
      release();
    };
    res.once('close', disconnect);
    res.once('error', disconnect);
    try {
      reader = redisClient.duplicate();
      reader.on('error', readerFailed);
      await reader.connect();
      if (closed) return;
      const lastEventId = req.headers['last-event-id'];
      let cursor = typeof lastEventId === 'string' && /^\d{1,20}-\d{1,20}$/.test(lastEventId)
        ? lastEventId : null;
      if (!cursor) {
        const info = await reader.xInfoStream(eventKey(steamId)).catch(() => null);
        cursor = info?.lastGeneratedId || '0-0';
      }
      const snapshot = await getPipelineStatus(steamId);
      if (closed) return;
      res.writeHead(200, {
        'Content-Type': 'text/event-stream', 'Cache-Control': 'no-store, no-transform',
        Connection: 'keep-alive', 'X-Accel-Buffering': 'no',
      });
      res.flushHeaders();
      if (!res.write(`event: snapshot\ndata: ${JSON.stringify(snapshot)}\n\n`)) return;
      while (!closed) {
        const streams = await reader.xRead([{ key: eventKey(steamId), id: cursor }],
          { BLOCK: 25000, COUNT: 100 });
        if (closed) break;
        if (!streams) {
          if (!res.write(': heartbeat\n\n')) break;
          continue;
        }
        for (const stream of streams) {
          for (const message of stream.messages) {
            cursor = message.id;
            if (!res.write(`id: ${message.id}\nevent: pipeline\ndata: ${message.message.data}\n\n`)) return;
          }
        }
      }
    } catch {
      if (!clientClosed && !res.headersSent && !res.writableEnded) res.status(503).json({ error: 'Event stream unavailable' });
      else if (!clientClosed && !res.writableEnded) res.write('event: pipeline\ndata: {"stage":"failed","error_code":"event_stream_unavailable"}\n\n');
    } finally {
      release();
      res.off('close', disconnect);
      res.off('error', disconnect);
      if (!res.writableEnded) res.end();
    }
  };
}

module.exports = { createPipelineEventHandler };
