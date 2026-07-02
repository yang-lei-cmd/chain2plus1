// 使用 Web Audio API 生成通知提示音（无需外部音频文件）
let audioCtx: AudioContext | null = null;

function getAudioCtx(): AudioContext {
  if (!audioCtx) {
    audioCtx = new AudioContext();
  }
  return audioCtx;
}

export function playNotificationSound(type: 'info' | 'success' | 'error' = 'info') {
  try {
    const ctx = getAudioCtx();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);

    switch (type) {
      case 'success':
        osc.frequency.value = 880;
        osc.type = 'sine';
        gain.gain.value = 0.3;
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.15);
        // Second note
        const osc2 = ctx.createOscillator();
        const gain2 = ctx.createGain();
        osc2.connect(gain2);
        gain2.connect(ctx.destination);
        osc2.frequency.value = 1100;
        osc2.type = 'sine';
        gain2.gain.value = 0.3;
        osc2.start(ctx.currentTime + 0.15);
        osc2.stop(ctx.currentTime + 0.3);
        break;
      case 'error':
        osc.frequency.value = 440;
        osc.type = 'sawtooth';
        gain.gain.value = 0.2;
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.3);
        break;
      default: // info
        osc.frequency.value = 660;
        osc.type = 'sine';
        gain.gain.value = 0.2;
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.12);
        break;
    }
  } catch {
    // Audio not available (e.g., no user interaction yet)
  }
}
