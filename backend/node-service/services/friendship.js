const SteamUser = require('steam-user');

const Relationship = SteamUser.EFriendRelationship;

function getFriendStatus(relationship) {
  if (relationship === Relationship.Friend) return 'friend';
  if (
    relationship === Relationship.RequestRecipient
    || relationship === Relationship.RequestInitiator
  ) {
    return 'pending';
  }
  if (relationship === Relationship.Blocked) return 'blocked';
  return 'not_friend';
}

async function addFriend(client, steamId, maxAttempts = 3) {
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      await new Promise((resolve, reject) => {
        client.addFriend(steamId, (error) => (error ? reject(error) : resolve()));
      });
      return { duplicate: false };
    } catch (error) {
      if (error.eresult === 14) return { duplicate: true };
      if (error.eresult === 40 || attempt === maxAttempts) throw error;
      const delay = Math.min(1000 * (2 ** (attempt - 1)), 16000);
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }
  return { duplicate: false };
}

module.exports = { addFriend, getFriendStatus };
