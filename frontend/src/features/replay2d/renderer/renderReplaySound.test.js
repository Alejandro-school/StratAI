import {expect,it,vi} from 'vitest';
import {drawSoundReviewMarkers} from './renderReplaySound';
import {createViewport} from './replayViewport';

it('draws the dated demo point and distinguishes height without drawing a live rival or a hearing circle',()=>{
  const ctx=Object.fromEntries(['save','restore','beginPath','arc','stroke','setLineDash','fillText'].map(k=>[k,vi.fn()]));
  drawSoundReviewMarkers(ctx,[{x:20,y:-30,z:110,side:'CT',kind:'footstep',age_seconds:0.25}],
    {pos_x:0,pos_y:0,scale:1},createViewport(1024,1024),{hasLevels:true,activeLevel:'lower',zThreshold:0});
  expect(ctx.arc).toHaveBeenCalledWith(20,30,9,0,2*Math.PI);
  expect(ctx.fillText).toHaveBeenCalledWith('CT · pasos ↑ · hace 0.3 s',20,11);
  expect(ctx.stroke).toHaveBeenCalledOnce();
});
