'use strict';

const Determinization = require('../determinization-steward/determinization-steward.js');
const Lessons = require('../code-capability-fabric/capability-lesson-steward-v1.js');
const Builders = require('../capability-fabric/builder-registry.js');
const GrammarGlass = require('../code-capability-fabric/language-organs/code-grammar-glass.js');
const Teams = require('../ephemeral-specialist-team/team-fabric.js');

const VERSION = '0.54.0';
const SCHEMA = 'axm.waldo.workshop-integration/v0.54';
const CREATION_BUILDERS = Object.freeze([
  'bounded-css-token-stylesheet-v1',
  'svg-status-badge-v1',
  'pure-json-transform-v1',
  'closed-object-contract-adapter-v2',
  'bounded-record-query-v1'
]);

function clone(value) { return JSON.parse(JSON.stringify(value)); }
function text(value, label, max = 4000) {
  if (typeof value !== 'string' || !value.trim() || value.length > max) throw new Error(label + '_INVALID');
  return value.trim();
}
function digest64(value, label) {
  const raw = String(value || '').replace(/^sha256:/, '');
  if (!/^[a-f0-9]{64}$/.test(raw)) throw new Error(label + '_INVALID');
  return raw;
}
function byteRef(record, id, schema, digestValue) {
  const digest = digest64(digestValue, 'BYTE_REF_DIGEST');
  return {
    id: text(id, 'BYTE_REF_ID', 127),
    schema: text(schema, 'BYTE_REF_SCHEMA', 220),
    sha256: 'sha256:' + digest,
    byteLength: Buffer.byteLength(Determinization.canon(record), 'utf8')
  };
}

function creationInventory() {
  const active = new Set(Builders.activeIds());
  const entries = CREATION_BUILDERS.map(id => {
    const descriptor = Builders.describe(id);
    return {
      id,
      active: active.has(id),
      descriptor,
      purpose: id === 'bounded-css-token-stylesheet-v1' ? 'CSS_STYLE_CREATION' :
        id === 'svg-status-badge-v1' ? 'SVG_MARKUP_CREATION' :
        id === 'pure-json-transform-v1' ? 'JAVASCRIPT_STRING_RECORD_TRANSFORM' :
        id === 'closed-object-contract-adapter-v2' ? 'JAVASCRIPT_CLOSED_OBJECT_CONTRACT_ADAPTER' :
        'JAVASCRIPT_BOUNDED_RECORD_QUERY'
    };
  });
  return Object.freeze({
    schema: 'axm.waldo.creation-capability-inventory/v0.54',
    entries,
    allRequiredActive: entries.every(entry => entry.active && entry.descriptor),
    authority: 'NONE'
  });
}

function compileCreationCapability(id, parameters) {
  if (!CREATION_BUILDERS.includes(id)) throw new Error('CREATION_BUILDER_NOT_IN_WALDO_V054_SET');
  const descriptor = Builders.describe(id);
  if (!descriptor || descriptor.status !== Builders.ACTIVE) throw new Error('CREATION_BUILDER_NOT_ACTIVE');
  const build = Builders.compileActive(id, descriptor.implementationDigest, clone(parameters || {}));
  const core = {
    schema: 'axm.waldo.compiled-creation-capability/v0.54',
    builder: descriptor,
    build,
    deterministicCandidate: true,
    executed: false,
    installed: false,
    authority: 'NONE'
  };
  return Object.freeze({ ...core, candidateDigest: Determinization.sha(core) });
}

function determinizationToLessonCandidate(input = {}) {
  const hypothesis = input.hypothesis;
  const evaluation = input.evaluation;
  if (!hypothesis || hypothesis.schema !== Determinization.HYPOTHESIS_SCHEMA) throw new Error('DETERMINIZATION_HYPOTHESIS_REQUIRED');
  if (!evaluation || evaluation.schema !== Determinization.EVALUATION_SCHEMA || evaluation.hypothesisDigest !== hypothesis.hypothesisDigest) throw new Error('DETERMINIZATION_EVALUATION_REQUIRED');
  const factory = Determinization.factoryHandoff(hypothesis, evaluation);
  if (factory.state !== 'READY_FOR_FACTORY_REVIEW') throw new Error('DETERMINIZATION_NOT_READY_FOR_LESSON_REVIEW');

  const hypothesisRef = byteRef(hypothesis, 'determinization-hypothesis', hypothesis.schema, hypothesis.hypothesisDigest);
  const evaluationRef = byteRef(evaluation, 'determinization-evaluation', evaluation.schema, evaluation.evaluationDigest);
  const sourceRefs = [hypothesisRef, evaluationRef, ...(Array.isArray(input.sourceRefs) ? clone(input.sourceRefs) : [])];
  const request = Lessons.sealRequest({
    schema: Lessons.REQUEST_SCHEMA,
    id: text(input.id, 'LESSON_REQUEST_ID', 127),
    laneId: text(input.laneId, 'LESSON_LANE_ID', 127),
    lessonKind: text(input.lessonKind, 'LESSON_KIND', 64),
    applicability: clone(input.applicability),
    rule: clone(input.rule),
    provenance: {
      trialObservationRef: input.trialObservationRef == null ? null : clone(input.trialObservationRef),
      acceptedArtifactRef: input.acceptedArtifactRef == null ? null : clone(input.acceptedArtifactRef),
      sourceRefs
    },
    evidence: clone(input.evidence),
    rights: clone(input.rights),
    review: clone(input.review),
    rootsGate: clone(input.rootsGate),
    authorization: clone(input.authorization),
    resources: clone(input.resources),
    privacy: clone(Lessons.PRIVACY),
    authority: 'NONE'
  });
  const lesson = Lessons.createLesson(request);
  return Object.freeze({
    schema: 'axm.waldo.determinization-lesson-handoff/v0.54',
    factoryHandoff: factory,
    request,
    lesson,
    persistentLearningAdmitted: false,
    authority: 'NONE'
  });
}

function grammarGlassStarToCreation(input = {}) {
  const packet = GrammarGlass.createProductionDraftCandidatePacket({
    star: input.star,
    projectId: input.projectId,
    directionSha256: input.directionSha256 == null ? null : input.directionSha256,
    workContextRef: input.workContextRef == null ? null : input.workContextRef
  });
  if (!packet || packet.result !== 'INERT_PRODUCTION_DRAFT_CANDIDATE_PACKET_READY_NOT_ADMITTED') throw new Error('GRAMMAR_GLASS_CREATION_PACKET_NOT_READY');
  return packet;
}

function teamFromGrammarGlassCandidate(packet, input = {}) {
  if (!packet || packet.schema !== 'axm.code.grammar-glass-production-draft-candidate.v1' || packet.result !== 'INERT_PRODUCTION_DRAFT_CANDIDATE_PACKET_READY_NOT_ADMITTED') throw new Error('GRAMMAR_GLASS_CANDIDATE_REQUIRED');
  if (!GrammarGlass.digestCurrent(packet, 'candidatePacketSha256')) throw new Error('GRAMMAR_GLASS_CANDIDATE_DIGEST_INVALID');
  const task = {
    id: 'grammar-glass-' + String(packet.projectId || 'candidate').replace(/[^A-Za-z0-9._:-]+/g, '-').slice(0, 80),
    goal: 'Review and develop the Grammar Glass structural candidate for ' + (packet.languageIds || []).join(' + ') + ' while preserving unresolved boundaries and lineage.',
    kind: 'creation',
    consequence: String(input.consequence || 'LOW').toUpperCase(),
    uncertainty: String(input.uncertainty || 'HIGH').toUpperCase(),
    estimatedSteps: Number.isSafeInteger(input.estimatedSteps) ? input.estimatedSteps : 6,
    artifactCount: 1,
    domains: [...new Set(['grammar-glass', ...(packet.languageIds || []), ...(packet.connectionClasses || []).map(x => String(x).toLowerCase())])],
    requestedSpecialists: input.requestedSpecialists == null ? null : input.requestedSpecialists
  };
  const team = Teams.createTeam({
    task,
    mirror: input.mirror || {},
    waldo: input.waldo || {},
    cycle: input.cycle || 0,
    consent: input.consent || {},
    runtime: input.runtime || {},
    resourceObservation: input.resourceObservation || {},
    deps: input.deps || {}
  });
  return Object.freeze({
    schema: 'axm.waldo.grammar-glass-specialist-handoff/v0.54',
    candidatePacketDigest: packet.candidatePacketSha256,
    task,
    team,
    candidateStillUnadmitted: true,
    authority: 'NONE'
  });
}

function snapshot() {
  const core = {
    schema: SCHEMA,
    version: VERSION,
    creation: creationInventory(),
    lessons: Lessons.snapshot(),
    grammarGlass: GrammarGlass.snapshot(),
    seams: [
      'DETERMINIZATION_TO_EVIDENCE_BOUND_LESSON',
      'GRAMMAR_GLASS_TO_PRODUCTION_DRAFT_CANDIDATE',
      'GRAMMAR_GLASS_TO_MIRROR_WALDO_SPECIALIST_TEAM',
      'WALDO_TO_DETERMINISTIC_CSS_SVG_JAVASCRIPT_BUILDERS',
      'WALDO_TO_DETERMINISTIC_OBJECT_ADAPTER_AND_RECORD_QUERY'
    ],
    runtimeDependencyOnAXMRepository: false,
    automaticAction: false,
    automaticAdmission: false,
    authority: 'NONE'
  };
  return Object.freeze({ ...core, snapshotDigest: Determinization.sha(core) });
}

module.exports = Object.freeze({
  VERSION,
  SCHEMA,
  CREATION_BUILDERS,
  creationInventory,
  compileCreationCapability,
  determinizationToLessonCandidate,
  grammarGlassStarToCreation,
  teamFromGrammarGlassCandidate,
  snapshot
});
