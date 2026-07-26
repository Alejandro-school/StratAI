const test = require('node:test');
const assert = require('node:assert/strict');
const SteamUser = require('steam-user');

const { addFriend, getFriendStatus } = require('../services/friendship');

test('maps both directions of a Steam invitation to pending', () => {
  const relationship = SteamUser.EFriendRelationship;
  assert.equal(getFriendStatus(relationship.Friend), 'friend');
  assert.equal(getFriendStatus(relationship.RequestRecipient), 'pending');
  assert.equal(getFriendStatus(relationship.RequestInitiator), 'pending');
  assert.equal(getFriendStatus(relationship.Blocked), 'blocked');
  assert.equal(getFriendStatus(undefined), 'not_friend');
});

test('treats Steam DuplicateName as an idempotent friend request', async () => {
  let calls = 0;
  const client = {
    addFriend(_steamId, callback) {
      calls += 1;
      const error = new Error('DuplicateName');
      error.eresult = 14;
      callback(error);
    },
  };

  assert.deepEqual(
    await addFriend(client, '76561198000000000'),
    { duplicate: true },
  );
  assert.equal(calls, 1);
});

test('returns success when Steam accepts the request', async () => {
  const client = {
    addFriend(_steamId, callback) {
      callback(null);
    },
  };

  assert.deepEqual(
    await addFriend(client, '76561198000000000'),
    { duplicate: false },
  );
});
