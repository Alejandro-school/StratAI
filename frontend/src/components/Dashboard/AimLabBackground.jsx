// frontend/src/components/Dashboard/AimLabBackground.jsx
// Fondo interactivo tipo Aim Lab con mirilla y targets
import React, { useState, useEffect, useRef, useCallback } from 'react';
import '../../styles/Dashboard/aimLabBackground.css';

const AimLabBackground = () => {
  const [cursorPos, setCursorPos] = useState({ x: 0, y: 0 });
  const [targets, setTargets] = useState([]);
  const [kills, setKills] = useState(0);
  const [shots, setShots] = useState(0);
  const [particles, setParticles] = useState([]);
  const containerRef = useRef(null);
  const targetIdRef = useRef(0);
  const isProcessingClick = useRef(false); // Para evitar duplicados
  const handleShootRef = useRef(null); // Referencia estable al manejador de disparo

  // Generar target aleatorio SOLO EN LATERALES
  const spawnTarget = useCallback(() => {
    if (!containerRef.current) return;
    
    const container = containerRef.current;
    const rect = container.getBoundingClientRect();
    
    // Tamaño REALMENTE aleatorio entre 0.6 y 1.4
    const targetScale = 0.6 + Math.random() * 0.8; // Rango más amplio
    const targetSize = 60 * targetScale;
    
    const viewportHeight = window.innerHeight;
    const viewportWidth = window.innerWidth;
    
    // ZONAS DE SPAWN SOLO EN LATERALES
    const marginTop = 650; // Evitar header + KPIs
    const marginBottom = 100; // Margen inferior
    
    // ZONA IZQUIERDA: desde 0 hasta 300px del borde izquierdo
    const leftZoneWidth = 300;
    const leftZoneStart = 0;
    const leftZoneEnd = leftZoneWidth;
    
    // ZONA DERECHA: desde viewportWidth-300 hasta viewportWidth
    const rightZoneWidth = 300;
    const rightZoneStart = viewportWidth - rightZoneWidth;
    const rightZoneEnd = viewportWidth;
    
    const availableHeight = viewportHeight - marginTop - marginBottom - targetSize;
    
    if (availableHeight < 100) return; // No spawns si no hay espacio vertical
    
    // Elegir zona aleatoria (izquierda o derecha)
    const useLeftZone = Math.random() < 0.5;
    let spawnX, spawnY;
    
    if (useLeftZone) {
      // SPAWN EN ZONA IZQUIERDA
      spawnX = leftZoneStart + Math.random() * (leftZoneEnd - leftZoneStart - targetSize);
      spawnX = Math.max(leftZoneStart + 20, spawnX); // Margen mínimo
    } else {
      // SPAWN EN ZONA DERECHA
      spawnX = rightZoneStart + Math.random() * (rightZoneEnd - rightZoneStart - targetSize);
      spawnX = Math.min(rightZoneEnd - targetSize - 20, spawnX); // Margen mínimo
    }
    
    spawnY = marginTop + Math.random() * availableHeight;
    
    const newTarget = {
      id: targetIdRef.current++,
      x: spawnX,
      y: spawnY,
      scale: targetScale,
      hit: false
    };
    
    console.log(`🎯 Spawning target ${newTarget.id} at (${spawnX.toFixed(0)}, ${spawnY.toFixed(0)}) in ${useLeftZone ? 'LEFT' : 'RIGHT'} zone`);
    
    setTargets(prev => [...prev, newTarget]);
    
    // Eliminar target después de 3 segundos si no fue golpeado
    setTimeout(() => {
      setTargets(prev => prev.filter(t => t.id !== newTarget.id));
    }, 3000);
  }, []);

  // Spawn targets periódicamente - MÁS FRECUENTE
  useEffect(() => {
    const interval = setInterval(() => {
      if (targets.length < 6) { // Máximo 6 targets a la vez
        spawnTarget();
      }
    }, 800); // Cada 800ms (más frecuente)
    
    return () => clearInterval(interval);
  }, [targets.length, spawnTarget]);

  // Crear partículas
  const createParticles = useCallback((x, y) => {
    const timestamp = Date.now();
    const newParticles = Array.from({ length: 12 }, (_, i) => ({
      id: `${timestamp}-${i}`, // ID único
      x,
      y,
      angle: (Math.PI * 2 * i) / 12,
      speed: Math.random() * 3 + 2,
      life: 1
    }));
    
    setParticles(prev => [...prev, ...newParticles]);
    
    // Eliminar partículas después de la animación
    setTimeout(() => {
      setParticles(prev => prev.filter(p => !newParticles.some(np => np.id === p.id)));
    }, 600);
  }, []);

  // Disparar - MEJORADO CON DETECCIÓN MÁS PRECISA
  const handleShoot = useCallback((e) => {
    // Prevenir duplicados
    if (isProcessingClick.current) return;
    isProcessingClick.current = true;
    
    // Solo disparar si NO estamos sobre el dashboard
    const clickedOnDashboard = e.target.closest('.dashboard-improved, .kpi-card, .grid-item, .action-btn, button, a, input, select, textarea');
    
    if (clickedOnDashboard) {
      isProcessingClick.current = false;
      return;
    }
    
    // Incrementar disparos
    setShots(prev => prev + 1);
    
    const clickX = e.clientX;
    const clickY = e.clientY;
    
    // Verificar si golpeó un target - DETECCIÓN MÁS PRECISA
    let targetHit = false;
    
    setTargets(prev => {
      const newTargets = prev.map(target => {
        if (target.hit) return target; // Ya fue golpeado
        
        // Calcular distancia desde el centro del target
        const targetCenterX = target.x;
        const targetCenterY = target.y;
        
        const distance = Math.sqrt(
          Math.pow(clickX - targetCenterX, 2) + Math.pow(clickY - targetCenterY, 2)
        );
        
        // Radio más generoso basado en el tamaño visual del target
        const hitRadius = 40 * target.scale; // Aumentado de 30 a 40
        
        console.log(`Target ${target.id}: distance=${distance.toFixed(1)}, hitRadius=${hitRadius.toFixed(1)}, scale=${target.scale}`);
        
        if (distance < hitRadius) {
          if (!targetHit) { // Solo contar el primero
            targetHit = true;
            setKills(k => k + 1);
            createParticles(targetCenterX, targetCenterY);
            console.log(`✅ HIT! Target ${target.id} destroyed`);
          }
          return { ...target, hit: true, scale: 0 };
        }
        return target;
      });
      
      // Filtrar targets golpeados
      return newTargets.filter(t => !t.hit);
    });

    // Efecto de disparo en la mirilla
    const crosshair = document.querySelector('.aim-crosshair');
    if (crosshair) {
      crosshair.classList.add('shooting');
      setTimeout(() => {
        crosshair.classList.remove('shooting');
      }, 100);
    }
    
    // Permitir siguiente click después de un breve delay
    setTimeout(() => {
      isProcessingClick.current = false;
    }, 50);
  }, [createParticles]);

  // Mantener una referencia estable al manejador para usarlo en listeners globales
  useEffect(() => {
    handleShootRef.current = handleShoot;
  }, [handleShoot]);

  // Seguir cursor y escuchar clicks globales SIN referenciar directamente a handleShoot
  useEffect(() => {
    const handleMouseMove = (e) => {
      setCursorPos({
        x: e.clientX,
        y: e.clientY
      });
    };

    const handleClick = (e) => {
      if (handleShootRef.current) {
        handleShootRef.current(e);
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('click', handleClick);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('click', handleClick);
    };
  }, []);

  const accuracy = shots > 0 ? Math.round((kills / shots) * 100) : 0;

  return (
    <div 
      ref={containerRef}
      className="aimlab-background"
    >
      {/* Grid de fondo */}
      <div className="aimlab-grid"></div>
      
      {/* Targets */}
      {targets.map(target => (
        <div
          key={target.id}
          className="aim-target"
          style={{
            left: `${target.x}px`,
            top: `${target.y}px`,
            transform: `translate(-50%, -50%) scale(${target.scale})`
          }}
        >
          <div className="target-outer"></div>
          <div className="target-middle"></div>
          <div className="target-center"></div>
        </div>
      ))}

      {/* Partículas */}
      {particles.map(particle => (
        <div
          key={particle.id}
          className="particle"
          style={{
            left: `${particle.x}px`,
            top: `${particle.y}px`,
            transform: `translate(-50%, -50%) translate(${Math.cos(particle.angle) * particle.speed * 20}px, ${Math.sin(particle.angle) * particle.speed * 20}px)`
          }}
        />
      ))}

      {/* Mirilla */}
      <div 
        className="aim-crosshair"
        style={{
          left: `${cursorPos.x}px`,
          top: `${cursorPos.y}px`
        }}
      >
        <div className="crosshair-line crosshair-top"></div>
        <div className="crosshair-line crosshair-right"></div>
        <div className="crosshair-line crosshair-bottom"></div>
        <div className="crosshair-line crosshair-left"></div>
        <div className="crosshair-dot"></div>
      </div>

      {/* Stats del minijuego */}
      <div className="aimlab-stats">
        <div className="stat-item">
          <span className="stat-label">KILLS</span>
          <span className="stat-value">{kills}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">ACCURACY</span>
          <span className="stat-value">{accuracy}%</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">SHOTS</span>
          <span className="stat-value">{shots}</span>
        </div>
      </div>
    </div>
  );
};

export default AimLabBackground;

