const PLAYER_NAMES = [
  'Nexus', 'Raze', 'Vektor', 'Horizon', 'Pulse', 'Ghostline', 'Rift', 'Anchor',
  'ZeroCool', 'Kova', 'Nightshift', 'Orbit', 'Mako', 'Frostbyte', 'Echo', 'Drift',
  'Cypher', 'Nova', 'Vertex', 'Spectre', 'Kinetic', 'Astra', 'Flux', 'Rook', 'Sable',
  'Recoil', 'Quartz', 'Ember', 'CipherNine', 'Raven', 'Torque', 'Glitch', 'Vanta',
  'Strafe', 'Crux', 'NeonFox', 'Mirage', 'Static', 'Kepler', 'Warden', 'Helix',
  'Talon', 'VectorTwo', 'Havoc', 'Lynx', 'Monolith', 'Jolt', 'Iris', 'Onyx', 'Delta',
];

const LEADING_POINTS = [812, 744, 683, 635, 614, 597, 579, 561, 544, 527];

export const createLeaderboard = (userName, userPoints, userRank = 6) => (
  PLAYER_NAMES.map((name, index) => {
    const rank = index + 1;
    const points = LEADING_POINTS[index] ?? Math.max(82, 525 - ((rank - 10) * 11));
    return {
      id: `player-${rank}`,
      rank,
      name: rank === userRank ? userName : name,
      points: rank === userRank ? userPoints : points,
      completedMissions: Math.min(3, Math.floor(points / 270)),
      matches: Math.min(10, 3 + (rank % 8)),
      trend: rank % 4 === 0 ? -1 : rank % 3 === 0 ? 2 : 1,
      isCurrentUser: rank === userRank,
    };
  })
);
