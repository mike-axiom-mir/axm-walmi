'use strict';

const crypto = require('node:crypto');

const VERSION = '0.52.0';
const SOCKETS = Object.freeze(['NEURAL_A', 'NEURAL_B']);
const PURPOSES = Object.freeze([
  'CHALLENGE', 'CREATIVE_VARIANT', 'DOMAIN_SECOND_OPINION',
  'PLAN_REVIEW', 'DISCOVERY', 'SPECIALIST_SPOT_ASSIST'
]);
const REQUESTERS = Object.freeze(['MIRROR', 'WALDO', 'HERMES']);
const HIDDEN_KEYS = new Set([
  'analysis','reasoning','chain_of_thought','chainOfThought','scratchpad',
  'hidden_reasoning','hiddenReasoning','logits','internalActivations','hiddenState'
]);
const CONNECTION_SCHEMA = 'axm.neural-socket-connection/v0.52';
const REQUEST_SCHEMA = 'axm.neural-socket-request/v0.52';
const RESULT_SCHEMA = 'axm.neural-socket-result/v0.52';
const DISCONNECT_SCHEMA = 'axm.neural-socket-disconnect/v0.52';

function canon(v) {
  if (v === null || typeof v !== 'object') return JSON.stringify(v);
  if (Array.isArray(v)) return '[' + v.map(canon).join(',') + ']';
  return '{' + Object.keys(v).sort().map(k => JSON.stringify(k) + ':' + canon(v[k])).join(',') + '}';
}
function sha(v) { return crypto.createHash('sha256').update(typeof v === 'string' ? v : canon(v)).digest('hex'); }
function clone(v) { return JSON.parse(JSON.stringify(v)); }
function text(v, label, max = 4000) {
  if (typeof v !== 'string' || !v.trim() || v.length > max) throw new Error(label + '_INVALID');
  const out = v.replace(/\r\n?/g, '\n').trim();
  if (/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(out)) throw new Error(label + '_CONTROL_CHAR');
  return out;
}
function integer(v, label, min, max, fallback) {
  const n = v == null ? fallback : Number(v);
  if (!Number.isSafeInteger(n) || n < min || n > max) throw new Error(label + '_INVALID');
  return n;
}
function socketId(v) { const x = String(v || '').toUpperCase(); if (!SOCKETS.includes(x)) throw new Error('SOCKET_INVALID'); return x; }
function rejectHidden(v, at = '$') {
  if (!v || typeof v !== 'object') return;
  if (Array.isArray(v)) return v.forEach((x,i) => rejectHidden(x, at + '[' + i + ']'));
  for (const [k,x] of Object.entries(v)) {
    if (HIDDEN_KEYS.has(k)) throw new Error('PRIVATE_REASONING_FIELD_REFUSED:' + at + '.' + k);
    rejectHidden(x, at + '.' + k);
  }
}
function emptyState() {
  const core = { version: VERSION, sockets: { NEURAL_A: null, NEURAL_B: null }, authority: 'NONE' };
  return Object.freeze({ ...core, stateDigest: sha(core) });
}
function connect(state, input = {}) {
  state = state || emptyState();
  const id = socketId(input.socketId);
  const capabilities = Array.isArray(input.capabilities) ? [...new Set(input.capabilities.map((x,i)=>text(String(x),'CAP_'+i,120)))].sort() : [];
  const core = {
    schema: CONNECTION_SCHEMA,
    socketId: id,
    adapter: { id:text(input.adapterId,'ADAPTER_ID',160), version:text(input.adapterVersion,'ADAPTER_VERSION',80), descriptorSha256:text(input.adapterDescriptorSha256,'ADAPTER_DIGEST',96) },
    model: { id:text(input.modelId,'MODEL_ID',160), version:text(input.modelVersion,'MODEL_VERSION',80) },
    capabilities,
    contextWindowTokens: integer(input.contextWindowTokens,'CONTEXT_WINDOW',1,10000000,32768),
    maxOutputTokens: integer(input.maxOutputTokens,'MAX_OUTPUT',1,1000000,4096),
    identityMerge:false, memoryPromotion:'NONE', authority:'NONE'
  };
  const connection = Object.freeze({ ...core, connectionDigest: sha(core) });
  const next = clone(state); next.sockets[id] = connection; delete next.stateDigest;
  return Object.freeze({ ...next, stateDigest: sha(next) });
}
function disconnect(state, id, reason = 'EXPLICIT_DISCONNECT') {
  id = socketId(id); state = state || emptyState();
  const prior = state.sockets[id];
  const receiptCore = { schema:DISCONNECT_SCHEMA, socketId:id, priorConnectionDigest:prior ? prior.connectionDigest : null, reason:text(String(reason),'DISCONNECT_REASON',500), authority:'NONE' };
  const receipt = Object.freeze({ ...receiptCore, disconnectDigest:sha(receiptCore) });
  const next = clone(state); next.sockets[id] = null; delete next.stateDigest;
  return { state:Object.freeze({ ...next, stateDigest:sha(next) }), receipt };
}
function request(state, input = {}) {
  const id = socketId(input.socketId); const conn = state && state.sockets && state.sockets[id];
  if (!conn) throw new Error('SOCKET_DISCONNECTED');
  const requester = String(input.requester || '').toUpperCase(); if (!REQUESTERS.includes(requester)) throw new Error('REQUESTER_INVALID');
  const purpose = String(input.purpose || '').toUpperCase(); if (!PURPOSES.includes(purpose)) throw new Error('PURPOSE_INVALID');
  rejectHidden(input.publicPacket);
  const packet = clone(input.publicPacket || {}), packetDigest = sha(packet);
  const requestedCapabilities = Array.isArray(input.requestedCapabilities) ? [...new Set(input.requestedCapabilities.map((x,i)=>text(String(x),'REQ_CAP_'+i,120)))].sort() : [];
  const missing = requestedCapabilities.filter(x => !conn.capabilities.includes(x));
  if (missing.length) throw new Error('SOCKET_CAPABILITY_UNDECLARED:' + missing.join(','));
  const core = {
    schema:REQUEST_SCHEMA, socketId:id, connectionDigest:conn.connectionDigest,
    requester, purpose, taskRef:text(input.taskRef,'TASK_REF',240), publicPacket:packet, publicPacketDigest:packetDigest,
    requestedCapabilities, maxOutputTokens:integer(input.maxOutputTokens,'REQ_MAX_OUTPUT',1,conn.maxOutputTokens,Math.min(conn.maxOutputTokens,2048)),
    maxCalls:1, resultPolicy:'UNTRUSTED_CANDIDATE_DATA', automaticAction:false, automaticAdoption:false, authority:'NONE'
  };
  return Object.freeze({ ...core, requestDigest:sha(core) });
}
function dualRequest(state, inputA, inputB) {
  return Object.freeze({ NEURAL_A: request(state,{...inputA,socketId:'NEURAL_A'}), NEURAL_B: request(state,{...inputB,socketId:'NEURAL_B'}) });
}
function acceptResult(state, req, raw = {}) {
  if (!req || req.schema !== REQUEST_SCHEMA) throw new Error('REQUEST_REQUIRED');
  const conn = state && state.sockets && state.sockets[req.socketId];
  if (!conn || conn.connectionDigest !== req.connectionDigest) throw new Error('RESULT_STALE_CONNECTION');
  rejectHidden(raw);
  const output = text(raw.output,'RESULT_OUTPUT',200000);
  const core = {
    schema:RESULT_SCHEMA, requestDigest:req.requestDigest, socketId:req.socketId, connectionDigest:req.connectionDigest,
    providerResultId:raw.providerResultId == null ? null : text(String(raw.providerResultId),'PROVIDER_RESULT_ID',240),
    output, outputDigest:sha(output), usage:clone(raw.usage || {}),
    state:'UNTRUSTED_NEURAL_RESULT', candidateOnly:true, identityMerge:false, memoryPromotion:'NONE', automaticAction:false, automaticAdoption:false, authority:'NONE'
  };
  return Object.freeze({ ...core, resultDigest:sha(core) });
}
function snapshot(state) {
  const active = SOCKETS.map(id => ({ socketId:id, connected:!!(state && state.sockets && state.sockets[id]), connectionDigest:state && state.sockets && state.sockets[id] ? state.sockets[id].connectionDigest : null }));
  const core = { schema:'axm.neural-socket-snapshot/v0.52', active, socketCount:2, disconnectedByDefault:true, authority:'NONE' };
  return Object.freeze({ ...core, snapshotDigest:sha(core) });
}

module.exports = Object.freeze({ VERSION, SOCKETS, PURPOSES, REQUESTERS, CONNECTION_SCHEMA, REQUEST_SCHEMA, RESULT_SCHEMA, DISCONNECT_SCHEMA, canon, sha, emptyState, connect, disconnect, request, dualRequest, acceptResult, snapshot });
