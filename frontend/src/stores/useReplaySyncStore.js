import { create } from 'zustand';

/**
 * Global state store for synchronizing the AI Coach Chat with the 2D Replay Viewer.
 * This acts as the "bridge" between the two isolated components.
 */
const useReplaySyncStore = create((set) => ({
  // --- State ---
  isPlaying: false,         // Is the replay currently playing an AI clip?
  activeClip: null,         // The current clip data from the AI
  currentPlaybackTick: 0,   // The current tick of the 2D viewer (updated by the viewer)
  annotations: [],          // Active annotations to draw on the canvas

  // --- Actions for the Chat Component (The Sender) ---
  
  /**
   * Called when the user clicks "Play Clip" in the AI Chat.
   * Sends the clip data to the global state so the viewer can react.
   */
  playAiClip: (clipData) => set({
    isPlaying: true,
    activeClip: {
      startTick: clipData.startTick,
      criticalTick: clipData.criticalTick,
      endTick: clipData.endTick,
      focusPlayerId: clipData.focusPlayerId
    },
    annotations: clipData.annotations || []
  }),

  /**
   * Called to explicitly stop or clear the current AI clip.
   */
  stopAiClip: () => set({
    isPlaying: false,
    activeClip: null,
    annotations: []
  }),

  // --- Actions for the 2D Viewer Component (The Receiver) ---

  /**
   * Called continuously by the 2D viewer's render loop (e.g. requestAnimationFrame).
   * Used to check if the clip has reached its end.
   */
  updateCurrentTick: (tick) => set((state) => {
    // Auto-pause logic: If we are playing an AI clip and we reach or pass the endTick
    if (state.isPlaying && state.activeClip && tick >= state.activeClip.endTick) {
      return { 
        currentPlaybackTick: tick,
        isPlaying: false // Auto-pause
      };
    }
    
    return { currentPlaybackTick: tick };
  })
}));

export default useReplaySyncStore;
