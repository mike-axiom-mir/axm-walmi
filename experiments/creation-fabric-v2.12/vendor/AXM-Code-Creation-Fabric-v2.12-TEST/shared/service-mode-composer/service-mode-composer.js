'use strict';

const crypto = require('node:crypto');
const Catalog = require('./service-catalog.json');
const Teams = require('../ephemeral-specialist-team/team-fabric.js');
const SpecialistLibrary = require('../specialists/axm-specialist-library.js');

const VERSION = '0.1.0';
const REQUEST_SCHEMA = 'axm.service-mode-request/v0.54-sidecar';
const PLAN_SCHEMA = 'axm.service-mode-composition/v0.54-sidecar';
const SNAPSHOT_SCHEMA = 'axm.service-mode-snapshot/v0.54-sidecar';
const CONNECTION_STATES = Object.freeze(['CONNECTED','AVAILABLE','DISABLED','UNKNOWN']);
const SOCKET_STATES = Object.freeze(['CONNECTED','DISCONNECTED','UNKNOWN']);
const ID = /^[a-z0-9][a-z0-9._-]{0,127}$/;
const CAP = /^[a-z][a-z0-9._:-]{1,127}$/;

function canon(value) {
  if (value === null || typeof value !== 'object') return JSON.stringify(value);
  if (Array.isArray(value)) return '[' + value.map(canon).join(',') + ']';
  return '{' + Object.keys(value).sort().map(key => JSON.stringify(key) + ':' + canon(value[key])).join(',') + '}';
}
function sha(value) {
  return crypto.createHash('sha256').update(typeof value === 'string' ? value : canon(value)).digest('hex');
}
function clone(value) { return JSON.parse(JSON.stringify(value)); }
function exact(value, fields, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(label + '_OBJECT_REQUIRED');
  const actual = Object.keys(value).sort();
  const expected = fields.slice().sort();
  if (canon(actual) !== canon(expected)) throw new Error(label + '_FIELDS_INVALID');
  return value;
}
function text(value, label, maximum = 4000) {
  if (typeof value !== 'string') throw new Error(label + '_STRING_REQUIRED');
  const out = value.replace(/\r\n?/g, '\n').trim();
  if (!out || out.length > maximum || /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(out)) throw new Error(label + '_INVALID');
  return out;
}
function identifier(value, label) {
  const out = text(String(value), label, 128).toLowerCase();
  if (!ID.test(out)) throw new Error(label + '_INVALID');
  return out;
}
function capability(value, label) {
  const out = text(String(value), label, 128).toLowerCase();
  if (!CAP.test(out)) throw new Error(label + '_INVALID');
  return out;
}
function uniqueStrings(values, label, normalizer, maximum = 128) {
  if (!Array.isArray(values) || values.length > maximum) throw new Error(label + '_ARRAY_INVALID');
  const out = values.map((value, index) => normalizer(value, label + '_' + index));
  if (new Set(out).size !== out.length) throw new Error(label + '_DUPLICATE');
  return out.sort();
}
function validateCatalog() {
  if (!Catalog || Catalog.schema !== 'axm.service-mode-catalog/v0.54-sidecar' || Catalog.authority !== 'NONE') throw new Error('SERVICE_CATALOG_IDENTITY_INVALID');
  if (!Array.isArray(Catalog.modes) || Catalog.modes.length < 1 || Catalog.modes.length > 64) throw new Error('SERVICE_CATALOG_MODES_INVALID');
  const seen = new Set();
  for (const mode of Catalog.modes) {
    exact(mode, ['id','title','taskKind','signals','requiredCapabilityGroups','optionalCapabilities','mirrorProfiles','waldoProfiles','hermesPolicy','neuralPurposes'], 'SERVICE_MODE');
    const modeId = identifier(mode.id, 'SERVICE_MODE_ID');
    if (seen.has(modeId)) throw new Error('SERVICE_MODE_ID_DUPLICATE');
    seen.add(modeId);
    if (!['conversation','analysis','creation','operation','research'].includes(mode.taskKind)) throw new Error('SERVICE_MODE_TASK_KIND_INVALID');
    uniqueStrings(mode.signals, 'SERVICE_MODE_SIGNALS', (v,l) => text(String(v), l, 120).toLowerCase(), 64);
    if (!Array.isArray(mode.requiredCapabilityGroups) || mode.requiredCapabilityGroups.length > 16) throw new Error('SERVICE_MODE_REQUIRED_GROUPS_INVALID');
    mode.requiredCapabilityGroups.forEach((group, index) => {
      const normalized = uniqueStrings(group, 'SERVICE_MODE_REQUIRED_GROUP_' + index, capability, 16);
      if (!normalized.length) throw new Error('SERVICE_MODE_REQUIRED_GROUP_EMPTY');
    });
    uniqueStrings(mode.optionalCapabilities, 'SERVICE_MODE_OPTIONAL_CAPS', capability, 64);
    uniqueStrings(mode.mirrorProfiles, 'SERVICE_MODE_MIRROR_PROFILES', identifier, 5);
    uniqueStrings(mode.waldoProfiles, 'SERVICE_MODE_WALDO_PROFILES', identifier, 5);
    exact(mode.hermesPolicy, ['mode','reasons','discoveryEligible'], 'SERVICE_MODE_HERMES_POLICY');
    if (mode.hermesPolicy.mode !== 'TRIGGER_ONLY' || typeof mode.hermesPolicy.discoveryEligible !== 'boolean') throw new Error('SERVICE_MODE_HERMES_POLICY_INVALID');
    uniqueStrings(mode.hermesPolicy.reasons, 'SERVICE_MODE_HERMES_REASONS', (v,l) => text(String(v), l, 120).toUpperCase(), 16);
    uniqueStrings(mode.neuralPurposes, 'SERVICE_MODE_NEURAL_PURPOSES', (v,l) => text(String(v), l, 120).toUpperCase(), 16);
  }
  if (!seen.has('general-service')) throw new Error('GENERAL_SERVICE_MODE_REQUIRED');
  return true;
}

function modeById(modeId) {
  return Catalog.modes.find(mode => mode.id === modeId) || null;
}
function subjectWords(subject) {
  return text(subject, 'SERVICE_SUBJECT', 6000).toLowerCase().replace(/[^a-z0-9+#.-]+/g, ' ').split(/\s+/).filter(Boolean);
}
function signalScore(subjectLower, words, signal) {
  const normalized = signal.toLowerCase();
  if (!normalized) return 0;
  if (normalized.includes(' ')) return subjectLower.includes(normalized) ? 4 + normalized.split(/\s+/).length : 0;
  return words.includes(normalized) ? 4 : 0;
}
function classifyService(subject, explicitModeId = null) {
  validateCatalog();
  const normalizedSubject = text(subject, 'SERVICE_SUBJECT', 6000);
  if (explicitModeId != null) {
    const selected = modeById(identifier(explicitModeId, 'EXPLICIT_SERVICE_MODE'));
    if (!selected) throw new Error('EXPLICIT_SERVICE_MODE_UNKNOWN');
    return Object.freeze({
      state: 'SERVICE_MODE_SELECTED_EXPLICITLY',
      subject: normalizedSubject,
      selectedMode: clone(selected),
      alternatives: [],
      heuristic: false,
      authority: 'NONE'
    });
  }
  const lower = normalizedSubject.toLowerCase();
  const words = subjectWords(normalizedSubject);
  const scored = Catalog.modes.filter(mode => mode.id !== 'general-service').map(mode => ({
    mode,
    score: mode.signals.reduce((sum, signal) => sum + signalScore(lower, words, signal), 0)
  })).filter(row => row.score > 0).sort((a,b) => b.score - a.score || a.mode.id.localeCompare(b.mode.id));

  if (!scored.length) {
    return Object.freeze({
      state: 'SERVICE_MODE_SELECTED_GENERAL_FALLBACK',
      subject: normalizedSubject,
      selectedMode: clone(modeById('general-service')),
      alternatives: [],
      heuristic: true,
      authority: 'NONE'
    });
  }

  const topScore = scored[0].score;
  const top = scored.filter(row => row.score === topScore);
  if (top.length > 1) {
    return Object.freeze({
      state: 'SERVICE_MODE_SELECTION_REQUIRED',
      subject: normalizedSubject,
      selectedMode: null,
      alternatives: top.map(row => ({ id: row.mode.id, title: row.mode.title, score: row.score })),
      heuristic: true,
      authority: 'NONE'
    });
  }
  return Object.freeze({
    state: 'SERVICE_MODE_SELECTED_HEURISTIC',
    subject: normalizedSubject,
    selectedMode: clone(scored[0].mode),
    alternatives: scored.slice(1, 4).map(row => ({ id: row.mode.id, title: row.mode.title, score: row.score })),
    heuristic: true,
    score: scored[0].score,
    authority: 'NONE'
  });
}

function normalizeHostApp(value, index) {
  exact(value, ['id','label','connectionState','capabilities'], 'HOST_APP_' + index);
  const state = text(String(value.connectionState), 'HOST_APP_STATE_' + index, 32).toUpperCase();
  if (!CONNECTION_STATES.includes(state)) throw new Error('HOST_APP_STATE_INVALID');
  return Object.freeze({
    id: identifier(value.id, 'HOST_APP_ID_' + index),
    label: text(value.label, 'HOST_APP_LABEL_' + index, 160),
    connectionState: state,
    capabilities: uniqueStrings(value.capabilities, 'HOST_APP_CAPS_' + index, capability, 128)
  });
}
function normalizeHostApps(values) {
  if (values == null) return [];
  if (!Array.isArray(values) || values.length > 128) throw new Error('HOST_APPS_INVALID');
  const out = values.map(normalizeHostApp).sort((a,b) => a.id.localeCompare(b.id));
  if (new Set(out.map(app => app.id)).size !== out.length) throw new Error('HOST_APP_ID_DUPLICATE');
  return out;
}
function flattenRequired(mode) {
  return [...new Set(mode.requiredCapabilityGroups.flat().map(String))].sort();
}
function planApps(mode, hostAppsInput) {
  const apps = normalizeHostApps(hostAppsInput);
  const connected = apps.filter(app => app.connectionState === 'CONNECTED');
  const requiredCapabilities = flattenRequired(mode);
  const wanted = new Set([...requiredCapabilities, ...mode.optionalCapabilities]);
  const relevant = connected.filter(app => app.capabilities.some(cap => wanted.has(cap)));
  const groups = mode.requiredCapabilityGroups.map((alternatives, index) => {
    const matches = relevant.filter(app => app.capabilities.some(cap => alternatives.includes(cap))).map(app => app.id).sort();
    return { index, alternatives: alternatives.slice().sort(), matchedAppIds: matches, satisfied: matches.length > 0 };
  });
  const missing = groups.filter(group => !group.satisfied).map(group => ({ index: group.index, alternatives: group.alternatives }));

  const remaining = new Set(groups.filter(group => group.satisfied).map(group => group.index));
  const primary = [];
  while (remaining.size) {
    const ranked = relevant.map(app => {
      const covers = groups.filter(group => remaining.has(group.index) && group.matchedAppIds.includes(app.id)).map(group => group.index);
      return { app, covers };
    }).filter(row => row.covers.length).sort((a,b) => b.covers.length - a.covers.length || a.app.id.localeCompare(b.app.id));
    if (!ranked.length) break;
    primary.push(ranked[0].app.id);
    ranked[0].covers.forEach(index => remaining.delete(index));
  }

  const optionalMatches = mode.optionalCapabilities.map(cap => ({
    capability: cap,
    matchedAppIds: relevant.filter(app => app.capabilities.includes(cap)).map(app => app.id).sort()
  })).filter(row => row.matchedAppIds.length);

  return Object.freeze({
    schema: 'axm.service-mode-app-plan/v0.54-sidecar',
    primaryAppIds: primary,
    relevantConnectedApps: relevant.map(app => clone(app)),
    requiredGroups: groups,
    missingRequiredCapabilityGroups: missing,
    optionalMatches,
    truth: {
      connectedDoesNotMeanAuthorized: true,
      appSelectionDoesNotActivateConnection: true,
      appSelectionDoesNotGrantPermission: true,
      disabledOrAvailableAppsAreNotSelectedAsConnected: true
    },
    activationAuthority: false,
    permissionGrant: false,
    authority: 'NONE'
  });
}

function profileMasks(profileIds) {
  const catalog = SpecialistLibrary.catalog().slice().sort((a,b) => a.id.localeCompare(b.id));
  const maskIds = [];
  const gaps = [];
  for (const profileId of profileIds) {
    const matches = catalog.filter(mask => mask.runtimeProfile && mask.runtimeProfile.id === profileId);
    if (!matches.length) gaps.push(profileId);
    else maskIds.push(matches[0].id);
  }
  return { maskIds, gaps };
}
function composeSpecialists(mode, subject, input = {}) {
  const mirror = profileMasks(mode.mirrorProfiles);
  const waldo = profileMasks(mode.waldoProfiles);
  const task = {
    id: 'service-' + mode.id + '-' + sha(subject).slice(0, 12),
    goal: subject,
    kind: mode.taskKind,
    consequence: String(input.consequence || 'LOW').toUpperCase(),
    uncertainty: String(input.uncertainty || 'MEDIUM').toUpperCase(),
    estimatedSteps: Number.isSafeInteger(input.estimatedSteps) ? input.estimatedSteps : 5,
    artifactCount: Number.isSafeInteger(input.artifactCount) ? input.artifactCount : 1,
    domains: ['service-mode', mode.id],
    requestedSpecialists: null
  };
  const team = Teams.createTeam({
    task,
    mirror: { requestedCount: mirror.maskIds.length, specialistIds: mirror.maskIds },
    waldo: { requestedCount: waldo.maskIds.length, specialistIds: waldo.maskIds },
    cycle: Number.isSafeInteger(input.cycle) ? input.cycle : 0,
    consent: clone(input.consent || {}),
    runtime: clone(input.runtime || {}),
    resourceObservation: clone(input.resourceObservation || {}),
    deps: input.deps || {}
  });
  return Object.freeze({
    team,
    requestedProfiles: { MIRROR: mode.mirrorProfiles.slice(), WALDO: mode.waldoProfiles.slice() },
    resolvedMaskIds: { MIRROR: mirror.maskIds, WALDO: waldo.maskIds },
    specialistProfileGaps: [...mirror.gaps.map(id => ({ owner:'MIRROR', profileId:id })), ...waldo.gaps.map(id => ({ owner:'WALDO', profileId:id }))],
    authority: 'NONE'
  });
}

function normalizeNeuralSockets(values) {
  if (values == null) return [];
  if (!Array.isArray(values) || values.length > 2) throw new Error('NEURAL_SOCKETS_INVALID');
  const out = values.map((value,index) => {
    exact(value, ['id','state'], 'NEURAL_SOCKET_' + index);
    const state = text(String(value.state), 'NEURAL_SOCKET_STATE_' + index, 32).toUpperCase();
    if (!SOCKET_STATES.includes(state)) throw new Error('NEURAL_SOCKET_STATE_INVALID');
    return { id: identifier(value.id, 'NEURAL_SOCKET_ID_' + index), state };
  }).sort((a,b) => a.id.localeCompare(b.id));
  if (new Set(out.map(row => row.id)).size !== out.length) throw new Error('NEURAL_SOCKET_ID_DUPLICATE');
  return out;
}
function neuralPlan(mode, input = {}) {
  const sockets = normalizeNeuralSockets(input.neuralSockets);
  const connected = sockets.filter(socket => socket.state === 'CONNECTED');
  const allow = input.allowNeural === true;
  return Object.freeze({
    allowedPurposes: mode.neuralPurposes.slice(),
    connectedSocketIds: connected.map(socket => socket.id),
    suggestedSocketIds: allow ? connected.slice(0, 2).map(socket => socket.id) : [],
    enabledByCaller: allow,
    automaticCall: false,
    candidateOnly: true,
    identityMerge: false,
    memoryPromotion: false,
    authority: 'NONE'
  });
}

function readiness(appPlan, specialistPlan) {
  const appGap = appPlan.missingRequiredCapabilityGroups.length > 0;
  const specialistGap = specialistPlan.specialistProfileGaps.length > 0;
  if (appGap && specialistGap) return 'DEGRADED_APP_AND_SPECIALIST_GAPS';
  if (appGap) return 'DEGRADED_MISSING_APP_CAPABILITIES';
  if (specialistGap) return 'DEGRADED_SPECIALIST_PROFILE_GAPS';
  return 'READY';
}

function planService(input = {}) {
  exact(input, ['schema','subject','modeId','hostApps','consequence','uncertainty','estimatedSteps','artifactCount','cycle','consent','runtime','resourceObservation','allowNeural','neuralSockets'], 'SERVICE_REQUEST');
  if (input.schema !== REQUEST_SCHEMA) throw new Error('SERVICE_REQUEST_SCHEMA_INVALID');
  const classification = classifyService(input.subject, input.modeId);
  if (!classification.selectedMode) {
    const core = {
      schema: PLAN_SCHEMA,
      version: VERSION,
      state: 'HELD_SERVICE_SELECTION_REQUIRED',
      subject: classification.subject,
      classification,
      mode: null,
      appPlan: null,
      specialistPlan: null,
      hermesPolicy: null,
      neuralPlan: null,
      truth: {
        serviceModeIsCompositionNotAuthority: true,
        connectedDoesNotMeanAuthorized: true,
        noAppWasActivated: true,
        noToolWasExecuted: true,
        noNeuralCallWasMade: true
      },
      authority: 'NONE'
    };
    return Object.freeze({ ...core, planDigest: sha(core) });
  }

  const mode = classification.selectedMode;
  const appPlan = planApps(mode, input.hostApps);
  const specialistPlan = composeSpecialists(mode, classification.subject, input);
  const neural = neuralPlan(mode, input);
  const core = {
    schema: PLAN_SCHEMA,
    version: VERSION,
    state: readiness(appPlan, specialistPlan),
    subject: classification.subject,
    classification,
    mode,
    appPlan,
    specialistPlan,
    hermesPolicy: clone(mode.hermesPolicy),
    neuralPlan: neural,
    truth: {
      serviceModeIsCompositionNotAuthority: true,
      connectedDoesNotMeanAuthorized: true,
      missingAppCapabilityDoesNotInventConnection: true,
      specialistViewDoesNotGrantTools: true,
      hermesRemainsTriggerOnly: true,
      neuralSocketSuggestionDoesNotMakeCall: true,
      noAppWasActivated: true,
      noToolWasExecuted: true,
      noNeuralCallWasMade: true
    },
    authority: 'NONE'
  };
  return Object.freeze({ ...core, planDigest: sha(core) });
}

function snapshot() {
  validateCatalog();
  const core = {
    schema: SNAPSHOT_SCHEMA,
    version: VERSION,
    modeCount: Catalog.modes.length,
    modeIds: Catalog.modes.map(mode => mode.id).sort(),
    catalogDigest: sha(Catalog),
    appConnectionModel: 'HOST_DECLARED_CAPABILITIES_CONNECTED_IS_NOT_AUTHORIZED',
    specialistModel: 'EXPLICIT_SERVICE_PROFILES_COMPILED_INTO_EXISTING_MIRROR_WALDO_POOLS',
    hermesModel: 'TRIGGER_ONLY',
    neuralModel: 'OPTIONAL_CANDIDATE_ONLY_NO_AUTOMATIC_CALL',
    automaticAppActivation: false,
    automaticPermissionGrant: false,
    automaticToolExecution: false,
    automaticNeuralCall: false,
    authority: 'NONE'
  };
  return Object.freeze({ ...core, snapshotDigest: sha(core) });
}

module.exports = Object.freeze({
  VERSION,
  REQUEST_SCHEMA,
  PLAN_SCHEMA,
  SNAPSHOT_SCHEMA,
  CONNECTION_STATES,
  classifyService,
  normalizeHostApps,
  planApps,
  profileMasks,
  composeSpecialists,
  neuralPlan,
  planService,
  snapshot,
  canon,
  sha
});
