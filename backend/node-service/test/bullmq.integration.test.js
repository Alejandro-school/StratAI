const test = require('node:test');
const assert = require('node:assert/strict');

test(
  'BullMQ deduplicates jobs and schedulers across replicas',
  { skip: process.env.REDIS_INTEGRATION !== 'true', timeout: 30000 },
  async () => {
    const IORedis = require('ioredis');
    const { Queue } = require('bullmq');
    const redisUrl = process.env.REDIS_URL || 'redis://127.0.0.1:6379';
    const queueName = `pipeline-integration-${process.pid}`;
    const prefix = `stratai:test:${process.pid}`;
    const firstConnection = new IORedis(redisUrl, { maxRetriesPerRequest: null });
    const secondConnection = new IORedis(redisUrl, { maxRetriesPerRequest: null });
    const firstReplica = new Queue(queueName, { connection: firstConnection, prefix });
    const secondReplica = new Queue(queueName, { connection: secondConnection, prefix });

    try {
      const first = await firstReplica.add(
        'discover',
        { steamId: '76561198000000000' },
        { deduplication: { id: '76561198000000000' } },
      );
      const duplicate = await secondReplica.add(
        'discover',
        { steamId: '76561198000000000' },
        { deduplication: { id: '76561198000000000' } },
      );
      assert.equal(duplicate.id, first.id);
      assert.equal(await firstReplica.getWaitingCount(), 1);

      await Promise.all([
        firstReplica.upsertJobScheduler(
          'global-discovery',
          { every: 60000 },
          { name: 'scan-users', data: {} },
        ),
        secondReplica.upsertJobScheduler(
          'global-discovery',
          { every: 60000 },
          { name: 'scan-users', data: {} },
        ),
      ]);
      const schedulers = await firstReplica.getJobSchedulers();
      assert.equal(
        schedulers.filter((scheduler) => scheduler.key === 'global-discovery').length,
        1,
      );
    } finally {
      await firstReplica.obliterate({ force: true });
      await Promise.all([firstReplica.close(), secondReplica.close()]);
      await Promise.all([firstConnection.quit(), secondConnection.quit()]);
    }
  },
);
