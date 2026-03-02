import React from 'react';
import { MESSAGE_CARD_TYPES } from '../../utils/coachMessageTypes';
import ErrorCard from './ErrorCard';
import EconomyCard from './EconomyCard';
import UtilityCard from './UtilityCard';
import DuelCard from './DuelCard';
import TimelineCard from './TimelineCard';

const CardRenderer = ({ card, onOpenReview }) => {
  switch (card.type) {
    case MESSAGE_CARD_TYPES.ERROR:
      return <ErrorCard card={card} onOpenReview={onOpenReview} />;
    case MESSAGE_CARD_TYPES.ECONOMY:
      return <EconomyCard card={card} onOpenReview={onOpenReview} />;
    case MESSAGE_CARD_TYPES.UTILITY:
      return <UtilityCard card={card} onOpenReview={onOpenReview} />;
    case MESSAGE_CARD_TYPES.DUEL:
      return <DuelCard card={card} onOpenReview={onOpenReview} />;
    case MESSAGE_CARD_TYPES.TIMELINE:
      return <TimelineCard card={card} onOpenReview={onOpenReview} />;
    default:
      return null;
  }
};

export default CardRenderer;
