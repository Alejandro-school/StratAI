// HeroParticles - Background particle effect for Hero
import React, { useEffect, useRef, useCallback } from 'react';

/**
 * HeroParticles
 * 
 * Canvas-based particle system for the Hero background.
 * Creates a subtle, floating particle effect with electric colors.
 */
const HeroParticles = ({ particleCount = 80 }) => {
  const canvasRef = useRef(null);
  const animationRef = useRef(null);
  const particlesRef = useRef([]);
  const mouseRef = useRef({ x: 0, y: 0 });

  // Initialize particles
  const initParticles = useCallback((width, height) => {
    const particles = [];
    
    for (let i = 0; i < particleCount; i++) {
      particles.push({
        x: Math.random() * width,
        y: Math.random() * height,
        size: Math.random() * 2 + 0.5,
        speedX: (Math.random() - 0.5) * 0.3,
        speedY: (Math.random() - 0.5) * 0.3,
        opacity: Math.random() * 0.5 + 0.1,
        // Color: mix of cyan and magenta
        hue: Math.random() > 0.5 ? 185 : 320, // cyan or magenta
        pulseSpeed: Math.random() * 0.02 + 0.01,
        pulseOffset: Math.random() * Math.PI * 2,
      });
    }
    
    return particles;
  }, [particleCount]);

  // Animation loop
  const animate = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    const { width, height } = canvas;
    
    // Clear canvas
    ctx.clearRect(0, 0, width, height);
    
    const time = Date.now() * 0.001;
    
    // Update and draw particles
    particlesRef.current.forEach((particle, index) => {
      // Update position
      particle.x += particle.speedX;
      particle.y += particle.speedY;
      
      // Wrap around edges
      if (particle.x < 0) particle.x = width;
      if (particle.x > width) particle.x = 0;
      if (particle.y < 0) particle.y = height;
      if (particle.y > height) particle.y = 0;
      
      // Calculate pulsing opacity
      const pulse = Math.sin(time * particle.pulseSpeed * 10 + particle.pulseOffset);
      const currentOpacity = particle.opacity * (0.7 + pulse * 0.3);
      
      // Mouse interaction - subtle attraction
      const dx = mouseRef.current.x - particle.x;
      const dy = mouseRef.current.y - particle.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      
      if (dist < 150) {
        const force = (150 - dist) / 150 * 0.02;
        particle.speedX += dx * force * 0.01;
        particle.speedY += dy * force * 0.01;
      }
      
      // Damping
      particle.speedX *= 0.99;
      particle.speedY *= 0.99;
      
      // Draw particle
      ctx.beginPath();
      ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
      ctx.fillStyle = `hsla(${particle.hue}, 100%, 60%, ${currentOpacity})`;
      ctx.fill();
      
      // Draw connections to nearby particles
      for (let j = index + 1; j < particlesRef.current.length; j++) {
        const other = particlesRef.current[j];
        const dx2 = particle.x - other.x;
        const dy2 = particle.y - other.y;
        const dist2 = Math.sqrt(dx2 * dx2 + dy2 * dy2);
        
        if (dist2 < 100) {
          ctx.beginPath();
          ctx.moveTo(particle.x, particle.y);
          ctx.lineTo(other.x, other.y);
          ctx.strokeStyle = `hsla(${particle.hue}, 100%, 60%, ${(1 - dist2 / 100) * 0.15})`;
          ctx.lineWidth = 0.5;
          ctx.stroke();
        }
      }
    });
    
    animationRef.current = requestAnimationFrame(animate);
  }, []);

  // Handle resize
  const handleResize = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    
    const { innerWidth: width, innerHeight: height } = window;
    canvas.width = width;
    canvas.height = height;
    
    particlesRef.current = initParticles(width, height);
  }, [initParticles]);

  // Handle mouse move
  const handleMouseMove = useCallback((e) => {
    mouseRef.current = { x: e.clientX, y: e.clientY };
  }, []);

  // Setup
  useEffect(() => {
    handleResize();
    window.addEventListener('resize', handleResize);
    window.addEventListener('mousemove', handleMouseMove);
    
    animate();
    
    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('mousemove', handleMouseMove);
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [handleResize, handleMouseMove, animate]);

  return (
    <canvas
      ref={canvasRef}
      className="hero__particles"
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
      }}
    />
  );
};

export default HeroParticles;
