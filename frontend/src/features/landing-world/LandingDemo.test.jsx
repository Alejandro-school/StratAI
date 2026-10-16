import React from 'react';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LandingDemo } from './LandingDemo';
import { CHAT_DEMO_SCRIPT as scripts } from './replayDemoData';

const script = scripts.es;
let play, pause;
beforeEach(() => {
  vi.useFakeTimers();
  play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockImplementation(function () {
    Object.defineProperty(this, 'paused', { configurable:true, value:false });
    fireEvent.play(this); return Promise.resolve();
  });
  pause = vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(function () {
    Object.defineProperty(this, 'paused', { configurable:true, value:true }); fireEvent.pause(this);
  });
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.useRealTimers(); });
const advance = async ms => { await act(async () => { vi.advanceTimersByTime(ms); }); };
const mount = async () => { const result=render(<LandingDemo reducedMotion />); await act(async()=>{}); return {...result,video:screen.getByLabelText(/Vídeo de la jugada/)}; };
const at = (video,time) => { video.currentTime=time; fireEvent.timeUpdate(video); };

describe('original scripted tactical replay', () => {
  it('starts muted on opening, rather than requiring native video controls', async () => {
    const {video}=await mount(); expect(play).toHaveBeenCalled(); expect(video.muted).toBe(true); expect(video.paused).toBe(false);
    fireEvent.click(screen.getByRole('button',{name:'Pausar jugada'})); expect(video.paused).toBe(true);
    fireEvent.click(screen.getByRole('button',{name:'Reproducir jugada'})); expect(video.paused).toBe(false);
  });
  it('pauses twice, reveals each analysis progressively and resumes automatically', async () => {
    const {video}=await mount(); at(video,1.4); expect(video.paused).toBe(true); expect(video.currentTime).toBe(1.3);
    expect(screen.queryByRole('article',{name:script.firstInsights[0].text})).toBeNull();
    await advance(500); expect(screen.getByRole('article',{name:script.firstInsights[0].text})).toBeTruthy();
    expect(screen.queryByRole('article',{name:script.firstInsights[1].text})).toBeNull();
    await advance(6000); await advance(1500); expect(screen.getByRole('article',{name:script.firstInsights[1].text})).toBeTruthy();
    await advance(6000); expect(video.paused).toBe(true); await advance(3000); expect(video.paused).toBe(false);
    at(video,6.1); expect(video.paused).toBe(true); expect(video.currentTime).toBe(6);
    await advance(500); expect(screen.getByRole('article',{name:script.criticalInsights[0].text})).toBeTruthy();
    await advance(8000); await advance(1500); expect(screen.getByRole('article',{name:script.criticalInsights[1].text})).toBeTruthy();
    await advance(8000); await advance(3000); expect(video.paused).toBe(false);
    fireEvent.ended(video); fireEvent.ended(video);
    expect(screen.getAllByRole('article',{name:script.finalSummary.text})).toHaveLength(1);
  });
  it('allows continuing early without repeating a checkpoint or duplicating messages', async () => {
    const {video}=await mount(); at(video,1.4); await advance(500);
    fireEvent.click(screen.getByRole('button',{name:'Continuar jugada'})); at(video,1.5);
    await advance(20000); expect(video.paused).toBe(false);
    expect(screen.getAllByRole('article',{name:script.firstInsights[0].text})).toHaveLength(1);
  });
  it('chapter jumps preserve analysis and restart cancels its pending messages', async () => {
    const {video}=await mount(); fireEvent.click(screen.getByRole('button',{name:'00:06 · Decisión'}));
    expect(video.currentTime).toBe(6); expect(video.paused).toBe(true);
    await advance(500); expect(screen.getByRole('article',{name:script.criticalInsights[0].text})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:'00:06 · Decisión'}));
    expect(screen.queryByRole('article',{name:script.criticalInsights[0].text})).toBeNull();
    await advance(500); expect(screen.getByRole('article',{name:script.criticalInsights[0].text})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:'Reiniciar demo'})); await advance(60000);
    expect(video.currentTime).toBe(0); expect(video.paused).toBe(false); expect(screen.queryAllByRole('article')).toHaveLength(0);
  });
  it('keeps the original conversation after the round and cancels answers on close', async () => {
    const {video,unmount}=await mount(); fireEvent.ended(video);
    fireEvent.click(screen.getByRole('button',{name:script.userQuestions[0].question}));
    expect(screen.getByRole('article',{name:script.userQuestions[0].question})).toBeTruthy();
    await advance(1000); expect(screen.getByRole('article',{name:script.userQuestions[0].response})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:script.userQuestions[1].question})); unmount();
    expect(vi.getTimerCount()).toBe(0); expect(pause).toHaveBeenCalled();
  });
  it('offers manual play when autoplay is denied', async () => {
    play.mockRejectedValueOnce(new DOMException('Autoplay denied','NotAllowedError'));
    const {video}=await mount();
    fireEvent.click(screen.getByRole('button',{name:'Reproducir jugada'})); await act(async()=>{});
    expect(video.paused).toBe(false);
  });
  it('cancels an in-progress analysis without resuming a closed replay', async () => {
    const {video,unmount}=await mount(); at(video,1.4); await advance(500);
    const calls=play.mock.calls.length; unmount(); expect(vi.getTimerCount()).toBe(0);
    await advance(30000); expect(play).toHaveBeenCalledTimes(calls);
  });
  it('types the original comment visibly while the video is paused', async () => {
    render(<LandingDemo />); await act(async()=>{});
    const video=screen.getByLabelText(/Vídeo de la jugada/); at(video,1.4); await advance(500); await advance(50);
    const message=screen.getByRole('article',{name:script.firstInsights[0].text});
    expect(message.querySelector('p').textContent).toBe(script.firstInsights[0].text.slice(0,4));
    await advance(5000); expect(message.querySelector('p').textContent).toBe(script.firstInsights[0].text); expect(video.paused).toBe(true);
  });
});
