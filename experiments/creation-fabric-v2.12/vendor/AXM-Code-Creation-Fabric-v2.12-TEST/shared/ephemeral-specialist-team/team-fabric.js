'use strict';

const crypto = require('node:crypto');

const VERSION = '0.51.0';
const TEAM_SCHEMA = 'axm.ephemeral-specialist-team/v0.51';
const SPOT_SCHEMA = 'axm.specialist-spot-use-request/v0.51';
const HERMES_TRIGGER_SCHEMA = 'axm.hermes-outer-eye-trigger/v0.51';
const HERMES_LOG_SCHEMA = 'axm.hermes-outer-eye-log/v0.51';
const REVOCATION_SCHEMA = 'axm.ephemeral-specialist-revocation/v0.51';

const CONTROLLERS = Object.freeze({
  MIRROR: Object.freeze({ normalMax: 5, triggeredOnly: false, discoveryBurstMax: 0 }),
  WALDO: Object.freeze({ normalMax: 5, triggeredOnly: false, discoveryBurstMax: 0 }),
  HERMES: Object.freeze({ normalMax: 2, triggeredOnly: true, discoveryBurstMax: 1 })
});

const HERMES_ROLES = Object.freeze(['OUTER_ANALYST', 'GAP_ANALYST', 'DISCOVERY_SCOUT']);
const INNER_DEFAULTS = Object.freeze({
  maxActions: 8,
  maxToolCalls: 6,
  leaseCycles: 1,
  evidencePolicy: 'MATERIAL_CLAIMS_REQUIRE_EVIDENCE',
  actionPolicy: 'PROPOSE_THEN_AUTHORIZE',
  memoryPolicy: 'EPHEMERAL',
  retainAfterRevoke: 'FINGERPRINT_AND_REVIEWED_LESSONS_ONLY',
  spotUseRequestsAllowed: true,
  authority: 'NONE'
});

function canon(v) {
  if (v === null || typeof v !== 'object') return JSON.stringify(v);
  if (Array.isArray(v)) return '[' + v.map(canon).join(',') + ']';
  return '{' + Object.keys(v).sort().map(k => JSON.stringify(k) + ':' + canon(v[k])).join(',') + '}';
}
function sha(v) { return crypto.createHash('sha256').update(typeof v === 'string' ? v : canon(v)).digest('hex'); }
function clone(v) { return JSON.parse(JSON.stringify(v)); }
function boundedText(v, label, max = 4000) {
  if (typeof v !== 'string' || !v.trim() || v.length > max) throw new Error(label + '_INVALID');
  const out = v.replace(/\r\n?/g, '\n').trim();
  if (/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(out)) throw new Error(label + '_CONTROL_CHAR');
  return out;
}
function boundedInt(v, label, min, max, fallback) {
  const n = v == null ? fallback : Number(v);
  if (!Number.isSafeInteger(n) || n < min || n > max) throw new Error(label + '_INVALID');
  return n;
}
function controller(v) {
  const x = String(v || '').toUpperCase();
  if (!CONTROLLERS[x]) throw new Error('CONTROLLER_INVALID');
  return x;
}
function normalizeTask(raw = {}) {
  const kind = String(raw.kind || 'analysis').toLowerCase();
  if (!['conversation', 'analysis', 'creation', 'operation', 'research'].includes(kind)) throw new Error('TASK_KIND_INVALID');
  const consequence = String(raw.consequence || 'LOW').toUpperCase();
  const uncertainty = String(raw.uncertainty || 'LOW').toUpperCase();
  if (!['LOW', 'MEDIUM', 'HIGH'].includes(consequence) || !['LOW', 'MEDIUM', 'HIGH'].includes(uncertainty)) throw new Error('TASK_LEVEL_INVALID');
  const domains = Array.isArray(raw.domains) ? [...new Set(raw.domains.map((x, i) => boundedText(String(x), 'TASK_DOMAIN_' + i, 120).toLowerCase()))] : [];
  return Object.freeze({
    id: boundedText(String(raw.id || 'task'), 'TASK_ID', 120),
    goal: boundedText(raw.goal, 'TASK_GOAL', 6000),
    kind,
    consequence,
    uncertainty,
    estimatedSteps: boundedInt(raw.estimatedSteps, 'TASK_STEPS', 0, 128, 1),
    artifactCount: boundedInt(raw.artifactCount, 'TASK_ARTIFACTS', 0, 128, 0),
    domains,
    requestedSpecialists: raw.requestedSpecialists == null ? null : boundedInt(raw.requestedSpecialists, 'TASK_REQUESTED_SPECIALISTS', 0, 5, 0)
  });
}
function specialistNeed(task) {
  if (task.requestedSpecialists != null) return task.requestedSpecialists;
  if (task.kind === 'conversation' && task.estimatedSteps <= 1 && task.uncertainty === 'LOW' && task.consequence === 'LOW') return 0;
  let score = 1;
  if (task.kind === 'creation' || task.kind === 'operation') score += 1;
  if (task.estimatedSteps >= 4) score += 1;
  if (task.estimatedSteps >= 9) score += 1;
  if (task.artifactCount >= 3) score += 1;
  if (task.uncertainty === 'MEDIUM') score += 1;
  if (task.uncertainty === 'HIGH') score += 2;
  if (task.consequence === 'HIGH') score += 1;
  if (task.domains.length >= 2) score += 1;
  return Math.max(0, Math.min(5, score));
}
function defaultDeps(deps = {}) {
  let library = deps.library || null;
  let router = deps.router || null;
  if (!library) {
    try { library = require('../specialists/axm-specialist-library.js'); } catch (_) {}
  }
  if (!router) {
    try { router = require('../specialists/specialist-router.js'); } catch (_) {}
  }
  return { library, router, compileMask: deps.compileMask || (library && library.compileMask), recommend: deps.recommend || (router && router.recommend) };
}
function candidateIds(task, owner, deps = {}) {
  const d = defaultDeps(deps);
  if (!d.recommend) return [];
  const lenses = owner === 'MIRROR'
    ? ['', ' requirements evidence failure provenance stance']
    : ['', ' implementation domain experiment validation uncertainty'];
  const out = [];
  for (const suffix of lenses) {
    const rec = d.recommend({ task: task.goal + suffix, maxSpecialists: 3, modelCapabilities: deps.modelCapabilities || {} });
    for (const item of (rec && rec.recommendations) || []) {
      const id = item && item.specialist && item.specialist.id;
      if (id && !out.includes(id)) out.push(id);
    }
  }
  return out;
}
function compilePackage(specialistId, deps = {}) {
  const d = defaultDeps(deps);
  if (!d.compileMask) throw new Error('SPECIALIST_COMPILER_UNAVAILABLE');
  return d.compileMask(specialistId, deps.modelCapabilities || {});
}
function innerSettings(owner, ordinal, overrides = {}) {
  const raw = { ...INNER_DEFAULTS, ...(overrides || {}) };
  return Object.freeze({
    seatOrdinal: ordinal,
    owner,
    maxActions: boundedInt(raw.maxActions, 'INNER_MAX_ACTIONS', 1, 128, INNER_DEFAULTS.maxActions),
    maxToolCalls: boundedInt(raw.maxToolCalls, 'INNER_MAX_TOOL_CALLS', 0, 128, INNER_DEFAULTS.maxToolCalls),
    leaseCycles: boundedInt(raw.leaseCycles, 'INNER_LEASE_CYCLES', 1, 100, INNER_DEFAULTS.leaseCycles),
    evidencePolicy: boundedText(String(raw.evidencePolicy), 'INNER_EVIDENCE_POLICY', 120),
    actionPolicy: boundedText(String(raw.actionPolicy), 'INNER_ACTION_POLICY', 120),
    memoryPolicy: 'EPHEMERAL',
    retainAfterRevoke: 'FINGERPRINT_AND_REVIEWED_LESSONS_ONLY',
    spotUseRequestsAllowed: raw.spotUseRequestsAllowed !== false,
    authority: 'NONE'
  });
}
function makeSeat(owner, task, specialistId, ordinal, deps, settingsOverride, reason = 'GOAL_COMPOSITION') {
  const pkg = compilePackage(specialistId, deps);
  const packageBytes = Buffer.byteLength(JSON.stringify(pkg), 'utf8');
  const packageFingerprint = pkg.fingerprint || sha(pkg);
  const profileId = pkg && pkg.mask && pkg.mask.runtimeProfile && pkg.mask.runtimeProfile.id || null;
  const core = {
    id: 'seat-' + sha([owner, task.id, specialistId, ordinal, reason]).slice(0, 16),
    owner,
    state: 'ACTIVE',
    reason,
    specialistId,
    profileId,
    packageFingerprint,
    compiledPackageBytes: packageBytes,
    innerSettings: innerSettings(owner, ordinal, settingsOverride),
    package: pkg,
    spotUse: reason === 'SPOT_USE',
    authority: 'NONE'
  };
  return Object.freeze({ ...core, seatDigest: sha(core) });
}
function composePool({ owner, task, requestedCount = null, specialistIds = null, settingsBySeat = [], deps = {} } = {}) {
  owner = controller(owner);
  if (owner === 'HERMES') throw new Error('HERMES_POOL_IS_TRIGGER_ONLY');
  task = normalizeTask(task);
  const cap = CONTROLLERS[owner].normalMax;
  const count = Math.min(cap, requestedCount == null ? specialistNeed(task) : boundedInt(requestedCount, 'POOL_REQUESTED_COUNT', 0, cap, 0));
  if (count === 0) return Object.freeze({ owner, cap, desired: 0, seats: [], totalCompiledBytes: 0, authority: 'NONE' });
  let ids = Array.isArray(specialistIds) ? [...new Set(specialistIds.map(String))] : candidateIds(task, owner, deps);
  ids = ids.slice(0, count);
  const seats = ids.map((id, i) => makeSeat(owner, task, id, i + 1, deps, settingsBySeat[i] || {}, 'GOAL_COMPOSITION'));
  return Object.freeze({
    owner,
    cap,
    desired: count,
    selected: seats.length,
    unfilled: Math.max(0, count - seats.length),
    seats,
    totalCompiledBytes: seats.reduce((n, s) => n + s.compiledPackageBytes, 0),
    truth: { maxIsNotQuota: true, unfilledSeatsAreAllowed: true, packagesAreEphemeral: true },
    authority: 'NONE'
  });
}
function createTeam({ task, mirror = {}, waldo = {}, cycle = 0, consent = {}, runtime = {}, resourceObservation = {}, deps = {} } = {}) {
  task = normalizeTask(task);
  const mirrorPool = composePool({ owner: 'MIRROR', task, ...mirror, deps: mirror.deps || deps });
  const waldoPool = composePool({ owner: 'WALDO', task, ...waldo, deps: waldo.deps || deps });
  const core = {
    schema: TEAM_SCHEMA,
    version: VERSION,
    task,
    cycle: boundedInt(cycle, 'TEAM_CYCLE', 0, Number.MAX_SAFE_INTEGER, 0),
    pools: { MIRROR: mirrorPool, WALDO: waldoPool },
    hermes: {
      normalMax: CONTROLLERS.HERMES.normalMax,
      discoveryBurstMax: CONTROLLERS.HERMES.discoveryBurstMax,
      reasoningAvailable: runtime.hermesReasoningAvailable === true,
      consent: consent.hermesReasoning === true,
      discoveryConsent: consent.discovery === true,
      activeRequests: []
    },
    resourceObservation: clone(resourceObservation || {}),
    retainedAfterRevocation: 'RECEIPTS_AND_EXPLICITLY_REVIEWED_LESSONS_ONLY',
    truth: {
      mirrorMaxNotQuota: true,
      waldoMaxNotQuota: true,
      hermesTriggeredNotAlwaysOn: true,
      hardwareObservationDoesNotYetResizePools: true,
      specialistPackageDoesNotGrantAuthority: true
    },
    authority: 'NONE'
  };
  return Object.freeze({ ...core, teamDigest: sha(core) });
}
function findSeat(team, owner, seatId) {
  owner = controller(owner);
  const pool = owner === 'HERMES' ? null : team.pools[owner];
  return pool && pool.seats.find(s => s.id === seatId) || null;
}
function requestSpotUse(team, input = {}) {
  const from = controller(input.fromController);
  const target = controller(input.targetController);
  if (from === target) throw new Error('SPOT_USE_MUST_CROSS_CONTROLLER');
  if (from === 'HERMES') throw new Error('HERMES_OUTER_EYE_RECOMMENDS_SPOT_USE_BUT_DOES_NOT_OWN_WORK_SEATS');
  const seat = findSeat(team, from, boundedText(input.fromSeatId, 'SPOT_FROM_SEAT', 120));
  if (!seat || seat.state !== 'ACTIVE' || !seat.innerSettings.spotUseRequestsAllowed) throw new Error('SPOT_FROM_SEAT_NOT_ACTIVE_OR_ALLOWED');
  const need = boundedText(input.need, 'SPOT_NEED', 1800);
  const core = {
    schema: SPOT_SCHEMA,
    id: 'spot-' + sha([team.teamDigest, seat.id, target, need]).slice(0, 16),
    teamDigest: team.teamDigest,
    fromController: from,
    fromSeatId: seat.id,
    targetController: target,
    need,
    preferredProfileId: input.preferredProfileId == null ? null : boundedText(String(input.preferredProfileId), 'SPOT_PROFILE', 120),
    leaseCycles: boundedInt(input.leaseCycles, 'SPOT_LEASE_CYCLES', 1, 10, 1),
    evidenceNeeded: Array.isArray(input.evidenceNeeded) ? input.evidenceNeeded.slice(0, 16).map((x, i) => boundedText(String(x), 'SPOT_EVIDENCE_' + i, 500)) : [],
    state: 'REQUESTED',
    authorityTransfer: false,
    automaticAcceptance: false,
    authority: 'NONE'
  };
  return Object.freeze({ ...core, requestDigest: sha(core) });
}
function resolveSpotUse(team, request, deps = {}) {
  if (!request || request.schema !== SPOT_SCHEMA || request.teamDigest !== team.teamDigest) throw new Error('SPOT_REQUEST_INVALID_OR_STALE');
  if (request.targetController === 'HERMES') {
    return Object.freeze({ requestDigest: request.requestDigest, state: 'HELD_HERMES_IS_TRIGGER_LANE', assignedSeat: null, authority: 'NONE' });
  }
  const pool = team.pools[request.targetController];
  let existing = null;
  if (request.preferredProfileId) existing = pool.seats.find(s => s.profileId === request.preferredProfileId && s.state === 'ACTIVE') || null;
  if (existing) return Object.freeze({ requestDigest: request.requestDigest, state: 'REUSE_ACTIVE_SPECIALIST', assignedSeat: clone(existing), authority: 'NONE' });
  if (pool.seats.filter(s => s.state === 'ACTIVE').length >= pool.cap) {
    return Object.freeze({ requestDigest: request.requestDigest, state: 'HELD_NO_FREE_TARGET_SLOT', assignedSeat: null, authority: 'NONE' });
  }
  const task = { ...team.task, goal: request.need, requestedSpecialists: 1 };
  const ids = candidateIds(normalizeTask(task), request.targetController, deps);
  if (!ids.length) return Object.freeze({ requestDigest: request.requestDigest, state: 'HELD_NO_SPECIALIST_MATCH', assignedSeat: null, authority: 'NONE' });
  const spotSeat = makeSeat(request.targetController, normalizeTask(task), ids[0], pool.seats.length + 1, deps, { leaseCycles: request.leaseCycles }, 'SPOT_USE');
  return Object.freeze({ requestDigest: request.requestDigest, state: 'SPOT_SPECIALIST_COMPILED', assignedSeat: clone(spotSeat), authority: 'NONE' });
}
function hermesTriggerAssessment(team, signals = {}) {
  if (!team || team.schema !== TEAM_SCHEMA) throw new Error('TEAM_REQUIRED');
  const enabled = team.hermes.reasoningAvailable && team.hermes.consent;
  const outerScore =
    boundedInt(signals.crossPoolDisagreements, 'HERMES_DISAGREEMENTS', 0, 1000, 0) * 2 +
    (boundedInt(signals.repairLoops, 'HERMES_REPAIR_LOOPS', 0, 1000, 0) >= 2 ? 2 : 0) +
    (signals.roadmapDrift === true ? 2 : 0) +
    (boundedInt(signals.unresolvedHighRiskClaims, 'HERMES_HIGH_RISK', 0, 1000, 0) > 0 ? 2 : 0) +
    (team.task.consequence === 'HIGH' ? 1 : 0);
  const gapScore =
    (boundedInt(signals.missingCapabilities, 'HERMES_MISSING_CAPS', 0, 1000, 0) > 0 ? 2 : 0) +
    (boundedInt(signals.blockedSpotRequests, 'HERMES_BLOCKED_SPOTS', 0, 1000, 0) > 0 ? 2 : 0) +
    (boundedInt(signals.specialistAbstentions, 'HERMES_ABSTENTIONS', 0, 1000, 0) >= 2 ? 1 : 0) +
    (['DEGRADED', 'UNKNOWN', 'BLOCKED'].includes(String(signals.capabilityState || '').toUpperCase()) ? 2 : 0);
  const discoveryDue = team.cycle > 0 && team.cycle % 10 === 0 && team.hermes.discoveryConsent && signals.discoveryAlreadyRun !== true;
  const roles = [];
  if (enabled && outerScore >= 2) roles.push('OUTER_ANALYST');
  if (enabled && gapScore >= 2) roles.push('GAP_ANALYST');
  if (enabled && discoveryDue) roles.push('DISCOVERY_SCOUT');
  const core = {
    schema: HERMES_TRIGGER_SCHEMA,
    teamDigest: team.teamDigest,
    cycle: team.cycle,
    enabled,
    outerScore,
    gapScore,
    discoveryDue,
    roles,
    normalTriggeredCount: roles.filter(x => x !== 'DISCOVERY_SCOUT').length,
    discoveryBurstCount: roles.includes('DISCOVERY_SCOUT') ? 1 : 0,
    truth: {
      hermesIsNotAlwaysOn: true,
      maximumTwoNormalReasoningSeats: true,
      discoveryIsOneTemporaryBurstEveryTenCompletedCycles: true,
      triggerDoesNotTransferControl: true
    },
    authority: 'NONE'
  };
  return Object.freeze({ ...core, triggerDigest: sha(core) });
}
function hermesRequests(team, assessment, publicSnapshot = {}) {
  if (!assessment || assessment.schema !== HERMES_TRIGGER_SCHEMA || assessment.teamDigest !== team.teamDigest) throw new Error('HERMES_TRIGGER_INVALID_OR_STALE');
  const snapshot = clone(publicSnapshot || {});
  const snapshotDigest = sha(snapshot);
  const focus = {
    OUTER_ANALYST: 'Inspect the whole collaboration from outside. Identify contradictions, blind spots, duplicated effort, roadmap drift, hidden assumptions, system-level risk, and one highest-value next check.',
    GAP_ANALYST: 'Inspect the collaboration for missing capabilities, missing evidence, missing specialist perspective, blocked spot-use needs, and gaps between claimed readiness and observed support. Recommend bounded holds or one spot-use request.',
    DISCOVERY_SCOUT: 'Perform one bounded discovery pass for a novel but relevant perspective, capability, workflow seam, or reusable specialist need. Treat discoveries as hypotheses until evidenced. Do not install, promote, or act automatically.'
  };
  return Object.freeze(assessment.roles.map((role, i) => ({
    schema: 'axm.hermes-outer-eye-request/v0.51',
    id: 'hermes-' + role.toLowerCase() + '-' + sha([assessment.triggerDigest, role, i]).slice(0, 12),
    role,
    teamDigest: team.teamDigest,
    triggerDigest: assessment.triggerDigest,
    publicSnapshotDigest: snapshotDigest,
    publicSnapshot: snapshot,
    focus: focus[role],
    requestedOutputSchema: HERMES_LOG_SCHEMA,
    reasoningCalls: 1,
    controlTransfer: false,
    automaticAction: false,
    authority: 'NONE'
  })));
}
function validateHermesLog(request, log) {
  if (!request || !HERMES_ROLES.includes(request.role)) throw new Error('HERMES_REQUEST_INVALID');
  if (!log || log.schema !== HERMES_LOG_SCHEMA || log.requestId !== request.id || log.role !== request.role) throw new Error('HERMES_LOG_BINDING_INVALID');
  const result = {
    schema: HERMES_LOG_SCHEMA,
    requestId: request.id,
    role: request.role,
    summary: boundedText(log.summary, 'HERMES_LOG_SUMMARY', 3000),
    observations: Array.isArray(log.observations) ? log.observations.slice(0, 24).map((x, i) => boundedText(String(x), 'HERMES_OBS_' + i, 1200)) : [],
    contradictions: Array.isArray(log.contradictions) ? log.contradictions.slice(0, 24).map((x, i) => boundedText(String(x), 'HERMES_CONTRA_' + i, 1200)) : [],
    gaps: Array.isArray(log.gaps) ? log.gaps.slice(0, 24).map((x, i) => boundedText(String(x), 'HERMES_GAP_' + i, 1200)) : [],
    recommendedSpotUse: Array.isArray(log.recommendedSpotUse) ? clone(log.recommendedSpotUse.slice(0, 8)) : [],
    evidenceRefs: Array.isArray(log.evidenceRefs) ? log.evidenceRefs.slice(0, 32).map((x, i) => boundedText(String(x), 'HERMES_EVIDENCE_' + i, 500)) : [],
    limitations: Array.isArray(log.limitations) ? log.limitations.slice(0, 16).map((x, i) => boundedText(String(x), 'HERMES_LIMIT_' + i, 800)) : [],
    truth: { outerPerspectiveOnly: true, hiddenReasoningNotRequested: true, controlTransfer: false, automaticAction: false },
    authority: 'NONE'
  };
  return Object.freeze({ ...result, logDigest: sha(result) });
}
function revokeSeat(seat, reason = 'TASK_OR_LEASE_COMPLETE') {
  if (!seat || seat.state !== 'ACTIVE') throw new Error('ACTIVE_SEAT_REQUIRED');
  const core = {
    schema: REVOCATION_SCHEMA,
    seatId: seat.id,
    owner: seat.owner,
    specialistId: seat.specialistId,
    profileId: seat.profileId,
    packageFingerprint: seat.packageFingerprint,
    compiledPackageBytesReleased: seat.compiledPackageBytes,
    reason: boundedText(String(reason), 'REVOCATION_REASON', 500),
    retained: ['seatDigest', 'packageFingerprint', 'compiledPackageBytesReleased', 'reviewedLessonsOnlyIfSeparatelyApproved'],
    rawSpecialistPackageRetained: false,
    memoryPromotion: 'NONE',
    authority: 'NONE'
  };
  return Object.freeze({ ...core, revocationDigest: sha(core) });
}

module.exports = Object.freeze({
  VERSION, TEAM_SCHEMA, SPOT_SCHEMA, HERMES_TRIGGER_SCHEMA, HERMES_LOG_SCHEMA, REVOCATION_SCHEMA,
  CONTROLLERS, HERMES_ROLES, INNER_DEFAULTS,
  canon, sha, normalizeTask, specialistNeed, composePool, createTeam,
  requestSpotUse, resolveSpotUse, hermesTriggerAssessment, hermesRequests,
  validateHermesLog, revokeSeat
});
