import { describe,expect,it } from 'vitest';
import { Vector3 } from 'three';
import { writeEntryPose,writeToolPose } from './sceneTransitions';

describe('authored camera paths',()=>{
  it('joins the current pose and finishes above the open site, looking at CT',()=>{
    const from=new Vector3(75,118,90),look=new Vector3(-17,0,0),position=new Vector3(),target=new Vector3();
    writeEntryPose(0,from,look,position,target);expect(position.toArray()).toEqual(from.toArray());expect(target.toArray()).toEqual(look.toArray());
    for(let i=0;i<=100;i++){writeEntryPose(i/100,from,look,position,target);expect(position.y).toBeGreaterThanOrEqual(17);expect(position.toArray().every(Number.isFinite)).toBe(true);}
    expect(position.distanceTo(new Vector3(-1,17,31))).toBeLessThan(1e-8);expect(target.distanceTo(new Vector3(4.5,.35,25))).toBeLessThan(1e-8);
  });
  it('keeps tool travel continuous and restores the exact approved endpoint',()=>{
    const from=new Vector3(-6,17,34),look=new Vector3(3,0,26),to=new Vector3(-6,17,34),focus=new Vector3(8,2,25),right=new Vector3(1,0,0),position=new Vector3(),target=new Vector3();
    writeToolPose(0,from,look,to,focus,position,target,right,1);expect(position.toArray()).toEqual(from.toArray());expect(target.toArray()).toEqual(look.toArray());
    let previous=position.clone();
    for(let i=1;i<=100;i++){writeToolPose(i/100,from,look,to,focus,position,target,right,1);expect(position.y).toBeGreaterThanOrEqual(17);expect(position.distanceTo(previous)).toBeLessThan(.1);previous.copy(position);}
    expect(position.distanceTo(to)).toBeLessThan(1e-8);expect(target.distanceTo(focus)).toBeLessThan(1e-8);
  });
});
