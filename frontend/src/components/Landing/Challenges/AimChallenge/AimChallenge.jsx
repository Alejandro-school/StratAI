/**
 * AimChallenge - Professional 3D Aim Training Experience
 * 
 * Replaces the original 2D target clicking game with an immersive
 * 3D aim training tool featuring:
 * - Holographic CS2 character targets
 * - Professional analytics (TTK, precision, mouse pathing, heatmap)
 * - GSAP/Anime.js animations
 * - Glassmorphic HUD
 */
import React from 'react';
import AimTraining3D from './AimTraining3D';
import ChallengeWrapper from '../ChallengeWrapper';

const AimChallenge = () => {
  return (
    <ChallengeWrapper
      type="MECÁNICA"
      title="Entrenamiento de Aim"
      subtitle="Elimina los hologramas enemigos lo más rápido posible. Tu precisión, tiempo de reacción y patrón de movimiento serán analizados por IA."
      showTimer={false}
      fullScreen={true}
    >
      <AimTraining3D />
    </ChallengeWrapper>
  );
};

export default AimChallenge;
