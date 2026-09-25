const TARGET_RATE = 16000;
const CHUNK_SAMPLES = 1600;

class SintonizadosMicCapture extends AudioWorkletProcessor {
  constructor() {
    super();
    this.step = sampleRate / TARGET_RATE;
    this.position = 0;
    this.previous = 0;
    this.hasPrevious = false;
    this.samples = new Int16Array(CHUNK_SAMPLES);
    this.offset = 0;
    this.stopped = false;
    this.port.onmessage = (event) => {
      if (event.data?.type !== 'flush') return;
      this.publishPartial();
      this.stopped = true;
      this.port.postMessage({ type: 'flushed' });
    };
  }

  process(inputs) {
    if (this.stopped) return false;
    const input = inputs[0]?.[0];
    if (!input?.length) return true;

    let data = input;
    if (this.hasPrevious) {
      data = new Float32Array(input.length + 1);
      data[0] = this.previous;
      data.set(input, 1);
    }

    while (this.position + 1 < data.length) {
      const index = Math.floor(this.position);
      const fraction = this.position - index;
      const sample = data[index] + (data[index + 1] - data[index]) * fraction;
      const bounded = Math.max(-1, Math.min(1, sample));
      this.samples[this.offset++] = bounded < 0
        ? Math.round(bounded * 32768)
        : Math.round(bounded * 32767);
      if (this.offset === CHUNK_SAMPLES) this.publishFull();
      this.position += this.step;
    }

    this.position -= input.length - 1;
    this.previous = input[input.length - 1];
    this.hasPrevious = true;
    return true;
  }

  publishFull() {
    const buffer = this.samples.buffer;
    this.port.postMessage({ type: 'chunk', buffer }, [buffer]);
    this.samples = new Int16Array(CHUNK_SAMPLES);
    this.offset = 0;
  }

  publishPartial() {
    if (this.offset === 0) return;
    const buffer = this.samples.slice(0, this.offset).buffer;
    this.port.postMessage({ type: 'chunk', buffer }, [buffer]);
    this.samples = new Int16Array(CHUNK_SAMPLES);
    this.offset = 0;
  }
}

registerProcessor('sintonizados-mic-capture', SintonizadosMicCapture);
