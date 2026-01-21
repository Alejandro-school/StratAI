import { useEffect, useRef, useState } from 'react';

/**
 * useScrambleText - Hacker/Decrypting Text Effect
 * 
 * Creates a Matrix-style scrambling text animation that reveals the final text
 * character by character with random glitch characters.
 * 
 * @param {string} finalText - The final text to reveal
 * @param {object} options - Animation options
 * @param {number} options.duration - Total animation duration in ms (default: 2000)
 * @param {number} options.scrambleSpeed - Speed of character changes in ms (default: 50)
 * @param {string} options.characters - Character set for scrambling (default: alphanumeric + symbols)
 * @param {boolean} options.autoStart - Start animation on mount (default: true)
 * @returns {[string, function]} - [currentText, startAnimation]
 */
const useScrambleText = (
  finalText, 
  options = {}
) => {
  const {
    duration = 2000,
    scrambleSpeed = 50,
    characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%^&*()_+-=[]{}|;:,.<>?',
    autoStart = true
  } = options;

  const [displayText, setDisplayText] = useState('');
  const intervalRef = useRef(null);
  const resolvedIndexRef = useRef(0);

  const getRandomChar = () => {
    return characters[Math.floor(Math.random() * characters.length)];
  };

  const startAnimation = () => {
    resolvedIndexRef.current = 0;
    const totalFrames = Math.floor(duration / scrambleSpeed);
    const charsPerFrame = finalText.length / totalFrames;
    let frameCount = 0;

    // Clear any existing interval
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }

    intervalRef.current = setInterval(() => {
      frameCount++;
      
      // Calculate how many characters should be resolved by now
      const targetResolvedIndex = Math.min(
        Math.floor(frameCount * charsPerFrame),
        finalText.length
      );

      let newText = '';
      
      for (let i = 0; i < finalText.length; i++) {
        if (i < targetResolvedIndex) {
          // Character is resolved
          newText += finalText[i];
          resolvedIndexRef.current = Math.max(resolvedIndexRef.current, i + 1);
        } else {
          // Character is still scrambling
          newText += finalText[i] === ' ' ? ' ' : getRandomChar();
        }
      }

      setDisplayText(newText);

      // Animation complete
      if (frameCount >= totalFrames || targetResolvedIndex >= finalText.length) {
        clearInterval(intervalRef.current);
        setDisplayText(finalText);
      }
    }, scrambleSpeed);
  };

  useEffect(() => {
    if (autoStart) {
      // Small delay before starting for dramatic effect
      const timeout = setTimeout(() => {
        startAnimation();
      }, 100);

      return () => {
        clearTimeout(timeout);
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
        }
      };
    }
  }, [finalText, autoStart]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  return [displayText, startAnimation];
};

export default useScrambleText;
