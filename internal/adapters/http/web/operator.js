const form = document.querySelector('#session-form');
const sessionInput = document.querySelector('#session-id');
const tokenInput = document.querySelector('#operator-token');
const controls = document.querySelector('#controls');
const statusLine = document.querySelector('#status');
const createButton = document.querySelector('#create');
const startButton = document.querySelector('#start');
const stopButton = document.querySelector('#stop');
const viewerLink = document.querySelector('#viewer');

let sessionID = '';
let operatorToken = '';
let audioContext = null;
let mediaStream = null;
let source = null;
let processor = null;
let silentGain = null;
let capturing = false;
let acceptChunks = false;
let nextSequence = 1;
let queuedChunks = 0;
let sentChunks = 0;
let sendChain = Promise.resolve();
let sendFailure = null;
let flushResolver = null;

sessionInput.value = `mic-${Date.now().toString(36)}`;

function setStatus(message) {
  statusLine.textContent = message;
}

async function responseMessage(response) {
  const text = await response.text();
  if (!text) return '';
  try {
    const value = JSON.parse(text);
    return value.message || value.error || text;
  } catch {
    return text;
  }
}

async function createSession(id, title, token) {
  const response = await fetch('/api/sessions', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ session_id: id, title, language: 'en' }),
  });
  if (!response.ok) throw new Error(`No se pudo crear la sesión: ${await responseMessage(response)}`);
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  createButton.disabled = true;
  const id = sessionInput.value.trim();
  const title = document.querySelector('#title').value.trim();
  const token = tokenInput.value;
  try {
    await createSession(id, title, token);
    sessionID = id;
    operatorToken = token;
    tokenInput.value = '';
    form.hidden = true;
    controls.hidden = false;
    document.querySelector('#session-heading').textContent = `${title} · ${id}`;
    viewerLink.href = `/talks/${encodeURIComponent(id)}`;
    setStatus('Sesión creada. Abrí el visor y después iniciá el micrófono.');
  } catch (error) {
    setStatus(error.message);
    createButton.disabled = false;
  }
});

function releaseAudio() {
  for (const node of [source, processor, silentGain]) {
    try { node?.disconnect(); } catch {}
  }
  mediaStream?.getTracks().forEach((track) => track.stop());
  mediaStream = null;
  if (audioContext && audioContext.state !== 'closed') void audioContext.close();
  audioContext = null;
  source = null;
  processor = null;
  silentGain = null;
  capturing = false;
}

async function postChunk(buffer, sequence) {
  for (let attempt = 0; attempt < 10; attempt++) {
    const response = await fetch(`/api/sessions/${encodeURIComponent(sessionID)}/audio`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${operatorToken}`,
        'Content-Type': 'application/octet-stream',
        'X-Audio-Sequence': String(sequence),
      },
      body: buffer,
    });
    if (response.status === 202) {
      sentChunks++;
      setStatus(`Micrófono en vivo · ${sentChunks} chunks PCM enviados`);
      return;
    }
    if (response.status === 429) {
      await new Promise((resolve) => setTimeout(resolve, 200));
      continue;
    }
    throw new Error(`El gateway respondió HTTP ${response.status}: ${await responseMessage(response)}`);
  }
  throw new Error('La cola de audio del gateway permaneció llena.');
}

function failCapture(error) {
  sendFailure = error;
  acceptChunks = false;
  releaseAudio();
  startButton.disabled = true;
  stopButton.disabled = false;
  setStatus(`Se detuvo la captura: ${error.message} Usá “Detener micrófono y cerrar charla”.`);
}

function enqueueChunk(buffer) {
  if (!acceptChunks || sendFailure) return;
  if (queuedChunks >= 50) {
    failCapture(new Error('el envío no alcanza el ritmo del micrófono'));
    return;
  }
  const sequence = nextSequence++;
  queuedChunks++;
  sendChain = sendChain
    .then(() => postChunk(buffer, sequence))
    .catch((error) => failCapture(error))
    .finally(() => { queuedChunks--; });
}

async function startCapture() {
  startButton.disabled = true;
  setStatus('Solicitando permiso para usar el micrófono…');
  try {
    if (!navigator.mediaDevices?.getUserMedia) {
      throw new Error('Este navegador no permite capturar audio aquí. Abrí localhost o usá HTTPS.');
    }
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (!AudioContextClass) throw new Error('Este navegador no soporta AudioWorklet.');
    mediaStream = await navigator.mediaDevices.getUserMedia({
      audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true, autoGainControl: true },
    });
    audioContext = new AudioContextClass();
    await audioContext.audioWorklet.addModule('/assets/mic-worklet.js');
    source = audioContext.createMediaStreamSource(mediaStream);
    processor = new AudioWorkletNode(audioContext, 'sintonizados-mic-capture', {
      numberOfInputs: 1,
      numberOfOutputs: 1,
      outputChannelCount: [1],
    });
    silentGain = audioContext.createGain();
    silentGain.gain.value = 0;
    processor.port.onmessage = (event) => {
      if (event.data?.type === 'chunk') enqueueChunk(event.data.buffer);
      if (event.data?.type === 'flushed' && flushResolver) {
        const resolve = flushResolver;
        flushResolver = null;
        resolve();
      }
    };
    acceptChunks = true;
    capturing = true;
    source.connect(processor);
    processor.connect(silentGain);
    silentGain.connect(audioContext.destination);
    await audioContext.resume();
    stopButton.disabled = false;
    setStatus('Micrófono en vivo · hablá para enviar audio y ver subtítulos.');
  } catch (error) {
    acceptChunks = false;
    releaseAudio();
    startButton.disabled = false;
    setStatus(`No se pudo iniciar el micrófono: ${error.message}`);
  }
}

async function stopCapture() {
  stopButton.disabled = true;
  startButton.disabled = true;
  if (capturing && processor && !sendFailure) {
    await new Promise((resolve) => {
      const timeout = setTimeout(() => {
        if (!flushResolver) return;
        flushResolver = null;
        resolve();
      }, 1500);
      flushResolver = () => {
        clearTimeout(timeout);
        resolve();
      };
      processor.port.postMessage({ type: 'flush' });
    });
  }
  acceptChunks = false;
  releaseAudio();
  await sendChain;
  try {
    const response = await fetch(`/api/sessions/${encodeURIComponent(sessionID)}/end`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${operatorToken}` },
    });
    if (!response.ok) throw new Error(await responseMessage(response));
    setStatus(sendFailure
      ? `Charla cerrada. La captura se interrumpió: ${sendFailure.message}`
      : `Charla cerrada. Se enviaron ${sentChunks} chunks de audio.`);
  } catch (error) {
    setStatus(`No se pudo cerrar la sesión: ${error.message}`);
  }
  operatorToken = '';
}

startButton.addEventListener('click', startCapture);
stopButton.addEventListener('click', stopCapture);
