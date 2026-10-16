import React, { useEffect, useRef } from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createNavigation } from './cameraNavigation';
import { useExplorationInput } from './useExplorationInput';

afterEach(cleanup);
function Fixture({ enabled, navigation, interaction, wake }) {
  const stage=useRef(), renderRequest=useRef(wake);
  useExplorationInput({stage,enabled,navigation,interaction,renderRequest});
  useEffect(() => {
    const ids=new Set();
    stage.current.setPointerCapture=id=>ids.add(id);
    stage.current.hasPointerCapture=id=>ids.has(id);
    stage.current.releasePointerCapture=id=>ids.delete(id);
  },[]);
  return <div ref={stage} data-testid="stage" tabIndex={0}><button>Fallback action</button></div>;
}
const pointer=(element,type,values) => {
  const event=new Event(type,{bubbles:true,cancelable:true});
  Object.assign(event,{pointerId:1,button:0,clientX:100,clientY:100,...values});fireEvent(element,event);
};
describe('exploration gestures', () => {
  it('pans, pinches and cancels cleanly; panel mode releases captures and wheel ownership', () => {
    const navigation={current:createNavigation()}, interaction={current:false}, wake=vi.fn();
    const props={navigation,interaction,wake};
    const view=render(<Fixture {...props} enabled />), element=view.getByTestId('stage');
    pointer(element,'pointerdown',{});pointer(element,'pointermove',{clientX:130,clientY:115});
    expect(navigation.current.x).not.toBe(3);expect(interaction.current).toBe(true);
    pointer(element,'pointerdown',{pointerId:2,clientX:230,clientY:115});
    const zoom=navigation.current.zoom;
    pointer(element,'pointermove',{pointerId:2,clientX:280,clientY:115});
    expect(navigation.current.zoom).toBeGreaterThan(zoom);
    pointer(element,'pointercancel',{});pointer(element,'pointercancel',{pointerId:2});
    expect(interaction.current).toBe(false);
    pointer(element,'pointerdown',{});
    view.rerender(<Fixture {...props} enabled={false} />);
    expect(interaction.current).toBe(false);expect(element.hasPointerCapture(1)).toBe(false);
    const previous={...navigation.current},wheel=new Event('wheel',{bubbles:true,cancelable:true});
    Object.assign(wheel,{deltaY:300,deltaMode:0});fireEvent(element,wheel);
    expect(wheel.defaultPrevented).toBe(false);expect(navigation.current).toEqual(previous);expect(wake).toHaveBeenCalled();
  });
  it('keeps fallback buttons usable and right-drag changes orientation instead of panning', () => {
    const navigation={current:createNavigation()}, interaction={current:false},wake=vi.fn();
    const view=render(<Fixture enabled navigation={navigation} interaction={interaction} wake={wake} />),element=view.getByTestId('stage');
    pointer(view.getByRole('button'),'pointerdown',{});expect(interaction.current).toBe(false);
    const before={...navigation.current};pointer(element,'pointerdown',{button:2});pointer(element,'pointermove',{clientX:160,clientY:130});
    expect(navigation.current.yaw).not.toBe(before.yaw);expect(navigation.current.x).toBe(before.x);expect(navigation.current.z).toBe(before.z);
  });
});
