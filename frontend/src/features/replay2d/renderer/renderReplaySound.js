import {worldToScreen} from './replayViewport';

const kinds={footstep:'pasos',reload:'recarga',scope:'zoom',gunshot:'disparo',grenade:'granada',
  landing:'aterrizaje',jump:'salto',c4:'C4',knife:'cuchillo',handling:'arma',item:'objeto',world:'entorno'};

export function drawSoundReviewMarkers(context,markers,config,viewport,{hasLevels,activeLevel,zThreshold}={}) {
  for(const marker of markers||[]) {
    const p=worldToScreen(marker,config,viewport);
    const level=marker.z>zThreshold?'upper':'lower';
    const otherLevel=hasLevels&&level!==activeLevel;
    const height=otherLevel?(level==='upper'?' ↑':' ↓'):'';
    context.save();
    context.globalAlpha=otherLevel?0.65:1;
    context.strokeStyle=marker.side==='CT'?'#69c7ff':'#ffbe55';
    context.lineWidth=2;
    context.setLineDash([3,3]);
    context.beginPath();
    context.arc(p.x,p.y,9,0,2*Math.PI);
    context.stroke();
    context.fillStyle=context.strokeStyle;
    context.font='600 11px system-ui';
    context.textAlign='center';
    context.fillText(`${marker.side} · ${kinds[marker.kind]||'sonido'}${height} · hace ${marker.age_seconds.toFixed(1)} s`,p.x,p.y-19);
    context.restore();
  }
}
