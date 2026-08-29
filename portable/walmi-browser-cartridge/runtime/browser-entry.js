'use strict';

async function loadClassicScript(host, path) {
  const source = await host.getText(path);
  const url = URL.createObjectURL(new Blob([source], { type: 'text/javascript' }));
  try {
    await new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = url;
      script.onload = () => { script.remove(); resolve(); };
      script.onerror = () => { script.remove(); reject(new Error('CLASSIC_SCRIPT_LOAD_FAILED:' + path)); };
      document.head.appendChild(script);
    });
  } finally {
    URL.revokeObjectURL(url);
  }
}

async function waitFor(predicate, timeoutMs, label) {
  const started = performance.now();
  while (!predicate()) {
    if (performance.now() - started > timeoutMs) throw new Error(label + '_TIMEOUT');
    await new Promise(resolve => setTimeout(resolve, 20));
  }
}

async function startAxmMirrorGo(host) {
  const goRuntime = host.manifest.runtime && host.manifest.runtime.go;
  if (!goRuntime) throw new Error('GO_RUNTIME_DESCRIPTOR_MISSING');
  if (typeof globalThis.Go !== 'function') {
    await loadClassicScript(host, goRuntime.exec);
  }
  const go = new globalThis.Go();
  const bytes = await host.getBytes(goRuntime.wasm);
  const instantiated = await WebAssembly.instantiate(bytes, go.importObject);
  const runPromise = Promise.resolve(go.run(instantiated.instance));
  await waitFor(() => globalThis.WALMIAxmMirrorReady === true, 10000, 'AXM_MIRROR_GO_READY');
  return { go, runPromise };
}

function invokeGo(operation, ...objects) {
  if (!globalThis.WALMIAxmMirrorReady || !globalThis.WALMIAxmMirror) {
    throw new Error('AXM_MIRROR_GO_NOT_READY');
  }
  const args = [operation, ...objects.map(value => typeof value === 'string' ? value : JSON.stringify(value))];
  return JSON.parse(globalThis.WALMIAxmMirror.invoke(...args));
}

function neuralDescriptor(host) {
  return (host.manifest.runtime && host.manifest.runtime.neural) || { state: 'MISSING' };
}

export async function createWalmiRuntime(host) {
  let active = false;
  let goRuntime = null;
  let turns = 0;

  return {
    async activate() {
      goRuntime = await startAxmMirrorGo(host);
      active = true;
      const goInfo = JSON.parse(globalThis.WALMIAxmMirror.info());
      const neural = neuralDescriptor(host);
      host.emit('mirror', {
        state: 'ACTIVE_BROWSER_WASM',
        sourceCommit: host.manifest.source && host.manifest.source.commit,
        go: goInfo.result || goInfo
      });
      host.emit('waldo', {
        state: neural.state === 'BUNDLED' ? 'NEURAL_ENGINE_DECLARED' : 'HELD_BROWSER_NEURAL_ENGINE_NOT_BUNDLED',
        neural,
        note: 'AXM Mirror control/memory logic is live locally. Neural generation is not substituted.'
      });
      host.emit('hermes', { sleeping: true, state: 'TRIGGER_ONLY' });
      host.emit('status', 'AXM MIRROR GO/WASM ACTIVE');
      await host.appendExperience('AXM_MIRROR_BROWSER_WASM_ACTIVATED', {
        go: goInfo.result || goInfo,
        neural
      });
    },

    async chat(message) {
      if (!active) throw new Error('WALMI_RUNTIME_INACTIVE');
      turns += 1;
      const neural = neuralDescriptor(host);
      const text = String(message || '');

      if (text.startsWith('/axm ')) {
        const firstSpace = text.indexOf(' ', 5);
        const operation = firstSpace < 0 ? text.slice(5).trim() : text.slice(5, firstSpace).trim();
        const payloadText = firstSpace < 0 ? '{}' : text.slice(firstSpace + 1).trim();
        let payload;
        try { payload = JSON.parse(payloadText || '{}'); }
        catch (_) { return { message: 'AXM command JSON is invalid.', waldo: { state: 'COMMAND_REFUSED_INVALID_JSON' } }; }
        const result = invokeGo(operation, payload);
        await host.appendExperience('AXM_BROWSER_OPERATION', { operation, ok: result.ok === true });
        return {
          message: JSON.stringify(result, null, 2),
          mirror: { operation, result },
          waldo: { state: 'DETERMINISTIC_AXM_OPERATION_COMPLETE', neuralGenerationUsed: false },
          hermes: { sleeping: true, state: 'TRIGGER_ONLY' }
        };
      }

      await host.appendExperience('CHAT_INPUT_OBSERVED_WITHOUT_BROWSER_NEURAL_BACKEND', { turn: turns, text });
      return {
        message: 'AXM Mirror Go/WASM is running locally, but this cartridge does not contain a browser-native neural engine + model weights yet. I will not fake a WALDO neural reply. Use /axm <operation> <json> to exercise the real deterministic Go surfaces, or add the declared neural engine/weights to this cartridge.',
        mirror: { state: 'EXPERIENCE_OBSERVED', turn: turns, text },
        waldo: { state: 'HELD_BROWSER_NEURAL_ENGINE_NOT_BUNDLED', neural },
        hermes: { sleeping: true, state: 'TRIGGER_ONLY' }
      };
    },

    async exportState() {
      return {
        files: {
          'state/browser-runtime-state.json': JSON.stringify({
            schema: 'walmi.browser-runtime-state/v1',
            exportedAt: new Date().toISOString(),
            turns,
            axmMirrorGoReady: globalThis.WALMIAxmMirrorReady === true,
            neural: neuralDescriptor(host),
            authority: 'NONE'
          }, null, 2)
        }
      };
    },

    async deactivate() {
      active = false;
      if (globalThis.WALMIAxmMirror && typeof globalThis.WALMIAxmMirror.shutdown === 'function') {
        globalThis.WALMIAxmMirror.shutdown();
      }
      if (goRuntime && goRuntime.runPromise) {
        await Promise.race([goRuntime.runPromise, new Promise(resolve => setTimeout(resolve, 1000))]);
      }
      goRuntime = null;
    }
  };
}
