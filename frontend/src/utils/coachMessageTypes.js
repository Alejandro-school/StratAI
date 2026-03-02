export const MESSAGE_CARD_TYPES = {
  ERROR: 'error',
  ECONOMY: 'economy',
  UTILITY: 'utility',
  DUEL: 'duel',
  TIMELINE: 'timeline'
};

export const createCoachMessage = ({
  id,
  sender,
  text,
  context = null,
  cards = [],
  interaction = null
}) => ({
  id,
  sender,
  text,
  context,
  cards,
  interaction
});

export const createReplayAction = ({
  matchId,
  round,
  tick = null,
  timestamp,
  title,
  description
}) => ({
  matchId,
  round,
  tick,
  timestamp,
  title,
  description
});
