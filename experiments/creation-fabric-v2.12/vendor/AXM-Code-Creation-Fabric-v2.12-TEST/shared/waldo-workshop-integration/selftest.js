'use strict';

const assert = require('node:assert/strict');
const Integration = require('./waldo-workshop-integration.js');
const Determinization = require('../determinization-steward/determinization-steward.js');
const GrammarGlass = require('../code-capability-fabric/language-organs/code-grammar-glass.js');
const Teams = require('../ephemeral-specialist-team/team-fabric.js');

let pass = 0;
function test(name, fn) { fn(); pass += 1; console.log('PASS', name); }
function ref(n) { return { id:'ref-' + n, schema:'axm.test.ref/v1', sha256:'sha256:' + Number(n).toString(16).padStart(64,'0'), byteLength:100 + n }; }

const inventory = Integration.creationInventory();
test('all selected deterministic creation builders are active', () => {
  assert.equal(inventory.allRequiredActive, true);
  assert.deepEqual(inventory.entries.map(x => x.id).sort(), ['bounded-css-token-stylesheet-v1','bounded-record-query-v1','closed-object-contract-adapter-v2','pure-json-transform-v1','svg-status-badge-v1']);
});

test('CSS builder is callable through WALDO integration', () => {
  const built = Integration.compileCreationCapability('bounded-css-token-stylesheet-v1', {
    resultSchemaId:'axm.test.css/v1', prefix:'demo',
    tokens:[{name:'accent',kind:'COLOR_HEX',defaultValue:'#aabbcc'}],
    maxInputBytes:1024, maxOutputBytes:2048
  });
  assert.equal(built.build.capabilityKind,'HAND');
  assert.match(built.build.source,/"prefix":"demo"/);
  assert.match(built.build.source,/rows\.push\('  --'\+CONFIG\.prefix/);
  assert.equal(built.executed,false);
});

test('SVG builder is callable through WALDO integration', () => {
  const built = Integration.compileCreationCapability('svg-status-badge-v1', {
    resultSchemaId:'axm.test.svg/v1', label:'state', value:'ready', background:'#112233', foreground:'#aabbcc', width:180,
    maxInputBytes:1024, maxOutputBytes:4096
  });
  assert.match(built.build.source,/image\/svg\+xml/);
  assert.equal(built.installed,false);
});

test('bounded JavaScript transform is callable through WALDO integration', () => {
  const built = Integration.compileCreationCapability('pure-json-transform-v1', {
    inputField:'source', outputField:'target', defaultValue:'none', outputSchema:'axm.test.transform/v1',
    maxInputKeys:8, maxValueLength:64, maxInputBytes:1024, maxOutputBytes:256
  });
  assert.match(built.build.source,/INPUT_BYTES_EXCEEDED/);
  assert.equal(built.authority,'NONE');
});

const harvested = Determinization.harvest({
  episodeId:'episode-lesson', taskRef:'task:lesson', sourceController:'WALDO',
  steps:[{
    stepId:'step-1', operationKey:'transform.record', mechanismClass:'HYBRID_HANDOFF',
    inputContractDigest:'input-contract-a', outputContractDigest:'output-contract-b',
    visibleDecisionSummary:'A repeated bounded record transform can be replayed deterministically.',
    repeatKey:'record-transform-v1', externalEffects:false, uncertainties:[], verifierStatus:'PASS',
    cost:{neuralUnits:1,latencyMs:20}
  }]
});
const hypothesis = harvested.hypotheses[0];
const evaluation = Determinization.evaluate(hypothesis,[
  {trialId:'t1',fixtureDigest:'fixture-a',outputDigest:'output-a',verifierStatus:'PASS',undeclaredSideEffects:false,deterministicRuntimeUsed:true},
  {trialId:'t2',fixtureDigest:'fixture-a',outputDigest:'output-a',verifierStatus:'PASS',undeclaredSideEffects:false,deterministicRuntimeUsed:true},
  {trialId:'t3',fixtureDigest:'fixture-b',outputDigest:'output-b',verifierStatus:'PASS',undeclaredSideEffects:false,deterministicRuntimeUsed:true}
]);

test('proven determinization result becomes an evidence-bound lesson candidate', () => {
  const handoff = Integration.determinizationToLessonCandidate({
    hypothesis, evaluation,
    id:'det-lesson-1', laneId:'knowledge.code.application-logic', lessonKind:'COUNTERTEST',
    applicability:{languageIds:['javascript'],familyIds:[],signalCodes:['REPEAT_TRANSFORM']},
    rule:{
      triggerCodes:['REPEAT_TRANSFORM'], actionCodes:['TRY_DETERMINISTIC_FAST_PATH'],
      expectedOutcomeCodes:['REPEAT_NEURAL_COMPUTE_AVOIDED_CANDIDATE'],
      limitationCodes:['EXACT_CONTRACT_MATCH_REQUIRED'], countertestCodes:['REPLAY_HELD_OUT_FIXTURE']
    },
    trialObservationRef:ref(31), acceptedArtifactRef:null, sourceRefs:[ref(32)],
    evidence:{supportRefs:[ref(33)],contradictionRefs:[],unknownRefs:[]},
    rights:{state:'RESEARCH_ONLY_HOLD',directReuseAllowed:false,authorityRef:null},
    review:{authoredAt:'2026-08-24T20:00:00.000Z',reviewAfter:'2026-08-25T20:00:00.000Z',expiresAt:'2026-09-24T20:00:00.000Z'},
    rootsGate:[
      {root:'TRUTH',verdict:'PASS',evidenceRefs:[ref(41)]},
      {root:'AGENCY_NON_DOMINATION',verdict:'PASS',evidenceRefs:[ref(42)]},
      {root:'CONTINUITY',verdict:'PASS',evidenceRefs:[ref(43)]},
      {root:'WISDOM_OVER_SPEED',verdict:'PASS',evidenceRefs:[ref(44)]}
    ],
    authorization:{
      decisionRef:ref(51), scope:'CREATE_DETACHED_LESSON_AND_LIBRARY_RELEASE_CANDIDATES_ONLY',
      lessonCandidate:true, releaseCandidate:true, persistentLearning:false, runningAttemptMutation:false,
      install:false, integrate:false, publish:false, train:false, physicalActuation:false,
      promote:false, canon:false, authenticatedIdentityProven:false, authority:'NONE'
    },
    resources:{maxSourceRefs:8,maxEvidenceRefs:8,maxEvaluationCases:8,maxLibraryLessons:16,maxLessonBytes:32768,maxReleaseBytes:262144,maxIterations:1,maxNetworkRequests:0,maxChildProcesses:0}
  });
  assert.equal(handoff.factoryHandoff.state,'READY_FOR_FACTORY_REVIEW');
  assert.equal(handoff.lesson.status,'PROPOSED');
  assert.equal(handoff.persistentLearningAdmitted,false);
  assert.ok(handoff.request.provenance.sourceRefs.some(x => x.schema === hypothesis.schema));
  assert.ok(handoff.request.provenance.sourceRefs.some(x => x.schema === evaluation.schema));
});

test('not-yet-proven determinization cannot enter lesson path', () => {
  const held = Determinization.evaluate(hypothesis,[{trialId:'t1',fixtureDigest:'fixture-a',outputDigest:'output-a',verifierStatus:'PASS',undeclaredSideEffects:false,deterministicRuntimeUsed:true}]);
  assert.throws(() => Integration.determinizationToLessonCandidate({hypothesis,evaluation:held}),/NOT_READY/);
});

function specialistPackage(id) {
  const profile = id.split(':').pop();
  return {fingerprint:Teams.sha(['pkg',id]),mask:{id,runtimeProfile:{id:profile}},runtime:{outputSchema:{type:'object'}},files:[{path:'PERSPECTIVE.md',content:id}]};
}
const deps = {
  compileMask:id => specialistPackage(id),
  recommend:() => ({recommendations:[{specialist:{id:'mask:requirements'},score:2},{specialist:{id:'mask:validation'},score:1}]})
};

const packetCore = {
  schema:'axm.code.grammar-glass-production-draft-candidate.v1',
  version:'1.0.0', result:'INERT_PRODUCTION_DRAFT_CANDIDATE_PACKET_READY_NOT_ADMITTED',
  projectId:'glass-demo', directionSha256:null, workContextRef:null, draftStarSha256:'star-a', draftRecipeDigest:'recipe-a', previewDigest:'preview-a',
  languageIds:['javascript','rust'], typedAtomAncestry:[], connectionClasses:['CONTRAST'], compositeKind:'CROSS_GRAMMAR_COMPOSITE_FORMATION',
  compositeLineageDigest:'lineage-a', grammarComponentLineage:[], unresolvedVisible:true, sourceCodeIncluded:false,
  requestedNextSurface:'EXPLICIT_PRODUCTION_DRAFT_CREATION_BY_EXTERNAL_AUTHORIZED_CALLER', integrationSurfaceMap:{},
  truth:{packetIsNotProductionBatch:true,packetIsNotAdmission:true,packetIsNotSelection:true,packetIsNotExecution:true,packetIsNotPromotion:true,packetMayBeIgnoredOrEdited:true},
  authority:'NONE'
};
const packet = Object.freeze({...packetCore,candidatePacketSha256:GrammarGlass.hash(packetCore)});

test('Grammar Glass candidate directly spawns bounded Mirror and WALDO review pools', () => {
  const handoff = Integration.teamFromGrammarGlassCandidate(packet,{
    mirror:{requestedCount:1,specialistIds:['mask:requirements']},
    waldo:{requestedCount:1,specialistIds:['mask:validation']}, deps
  });
  assert.equal(handoff.team.pools.MIRROR.seats.length,1);
  assert.equal(handoff.team.pools.WALDO.seats.length,1);
  assert.ok(handoff.task.domains.includes('javascript'));
  assert.ok(handoff.task.domains.includes('rust'));
  assert.equal(handoff.candidateStillUnadmitted,true);
});

test('tampered Grammar Glass packet cannot enter specialist path', () => {
  assert.throws(() => Integration.teamFromGrammarGlassCandidate({...packet,languageIds:['python']},{deps}),/DIGEST_INVALID/);
});

test('integration snapshot exposes all four callable seams', () => {
  const snap = Integration.snapshot();
  assert.equal(snap.creation.allRequiredActive,true);
  assert.ok(snap.seams.includes('DETERMINIZATION_TO_EVIDENCE_BOUND_LESSON'));
  assert.ok(snap.seams.includes('GRAMMAR_GLASS_TO_PRODUCTION_DRAFT_CANDIDATE'));
  assert.ok(snap.seams.includes('GRAMMAR_GLASS_TO_MIRROR_WALDO_SPECIALIST_TEAM'));
  assert.ok(snap.seams.includes('WALDO_TO_DETERMINISTIC_CSS_SVG_JAVASCRIPT_BUILDERS'));
  assert.ok(snap.seams.includes('WALDO_TO_DETERMINISTIC_OBJECT_ADAPTER_AND_RECORD_QUERY'));
  assert.equal(snap.runtimeDependencyOnAXMRepository,false);
});

console.log('RESULT', pass + '/9 PASS');
