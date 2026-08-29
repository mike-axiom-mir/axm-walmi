'use strict';
const assert = require('node:assert/strict');
const Team = require('./team-fabric');

let pass = 0;
function test(name, fn) { fn(); pass += 1; console.log('PASS', name); }
function pkg(id) {
  const profile = id.split(':').pop();
  return {
    fingerprint: Team.sha(['pkg', id]),
    mask: { id, runtimeProfile: { id: profile } },
    runtime: { outputSchema: { type: 'object' } },
    files: [{ path: 'PERSPECTIVE.md', content: id }]
  };
}
const ids = ['requirements','domain','experiment','validation','uncertainty','failure','implementation','provenance','evidence','stance'];
const deps = {
  compileMask: id => pkg(id),
  recommend: ({task,maxSpecialists}) => {
    const offset = task.includes('implementation') ? 5 : 0;
    return { recommendations: ids.slice(offset, offset + maxSpecialists).map((x, i) => ({ specialist: { id: 'mask:' + x }, score: 10 - i })) };
  }
};
const task = { id:'build-x', goal:'Build and verify a bounded creation system', kind:'creation', consequence:'MEDIUM', uncertainty:'MEDIUM', estimatedSteps:8, artifactCount:4, domains:['code','verification'] };

test('controller maxima are Mirror 5 Waldo 5 Hermes 2 plus one discovery burst', () => {
  assert.equal(Team.CONTROLLERS.MIRROR.normalMax, 5);
  assert.equal(Team.CONTROLLERS.WALDO.normalMax, 5);
  assert.equal(Team.CONTROLLERS.HERMES.normalMax, 2);
  assert.equal(Team.CONTROLLERS.HERMES.discoveryBurstMax, 1);
});

test('max is not quota and requested zero produces no specialists', () => {
  const t = Team.createTeam({ task:{...task,requestedSpecialists:0}, mirror:{requestedCount:0}, waldo:{requestedCount:0}, deps });
  assert.equal(t.pools.MIRROR.seats.length,0); assert.equal(t.pools.WALDO.seats.length,0);
});

test('mirror and waldo never exceed five', () => {
  const many = ids.map(x=>'mask:'+x);
  const t = Team.createTeam({ task, mirror:{requestedCount:5,specialistIds:many}, waldo:{requestedCount:5,specialistIds:many.slice().reverse()}, deps });
  assert.equal(t.pools.MIRROR.seats.length,5); assert.equal(t.pools.WALDO.seats.length,5);
});

test('each specialist has isolated inner settings and no authority', () => {
  const t = Team.createTeam({ task, mirror:{requestedCount:1,specialistIds:['mask:requirements'],settingsBySeat:[{maxActions:3,maxToolCalls:2}]}, waldo:{requestedCount:0}, deps });
  const s=t.pools.MIRROR.seats[0]; assert.equal(s.innerSettings.maxActions,3); assert.equal(s.innerSettings.maxToolCalls,2); assert.equal(s.innerSettings.memoryPolicy,'EPHEMERAL'); assert.equal(s.authority,'NONE');
});

test('compiled package bytes are measured for later hardware policy work', () => {
  const t = Team.createTeam({ task, mirror:{requestedCount:1,specialistIds:['mask:requirements']}, waldo:{requestedCount:0}, resourceObservation:{freeRamMB:1234}, deps });
  assert.ok(t.pools.MIRROR.totalCompiledBytes>0); assert.equal(t.resourceObservation.freeRamMB,1234); assert.equal(t.truth.hardwareObservationDoesNotYetResizePools,true);
});

test('spot use must cross controllers', () => {
  const t = Team.createTeam({ task, mirror:{requestedCount:1,specialistIds:['mask:requirements']}, waldo:{requestedCount:0}, deps });
  assert.throws(()=>Team.requestSpotUse(t,{fromController:'MIRROR',fromSeatId:t.pools.MIRROR.seats[0].id,targetController:'MIRROR',need:'validate'}),/MUST_CROSS/);
});

test('spot use can compile one temporary specialist into a free target slot', () => {
  const t = Team.createTeam({ task, mirror:{requestedCount:1,specialistIds:['mask:requirements']}, waldo:{requestedCount:0}, deps });
  const r=Team.requestSpotUse(t,{fromController:'MIRROR',fromSeatId:t.pools.MIRROR.seats[0].id,targetController:'WALDO',need:'implementation validation',leaseCycles:1});
  const x=Team.resolveSpotUse(t,r,deps); assert.equal(x.state,'SPOT_SPECIALIST_COMPILED'); assert.equal(x.assignedSeat.spotUse,true); assert.equal(x.assignedSeat.innerSettings.leaseCycles,1);
});

test('spot use reuses matching active target specialist first', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:1,specialistIds:['mask:requirements']},waldo:{requestedCount:1,specialistIds:['mask:validation']},deps});
  const r=Team.requestSpotUse(t,{fromController:'MIRROR',fromSeatId:t.pools.MIRROR.seats[0].id,targetController:'WALDO',need:'validation',preferredProfileId:'validation'});
  const x=Team.resolveSpotUse(t,r,deps); assert.equal(x.state,'REUSE_ACTIVE_SPECIALIST'); assert.equal(x.assignedSeat.specialistId,'mask:validation');
});

test('spot use holds when target pool has no free slot', () => {
  const many=['mask:requirements','mask:domain','mask:experiment','mask:validation','mask:uncertainty'];
  const t=Team.createTeam({task,mirror:{requestedCount:1,specialistIds:['mask:failure']},waldo:{requestedCount:5,specialistIds:many},deps});
  const r=Team.requestSpotUse(t,{fromController:'MIRROR',fromSeatId:t.pools.MIRROR.seats[0].id,targetController:'WALDO',need:'provenance'});
  assert.equal(Team.resolveSpotUse(t,r,deps).state,'HELD_NO_FREE_TARGET_SLOT');
});

test('hermes remains asleep when unavailable or unconsented', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:10,consent:{hermesReasoning:true,discovery:true},runtime:{hermesReasoningAvailable:false},deps});
  const a=Team.hermesTriggerAssessment(t,{crossPoolDisagreements:4,missingCapabilities:4}); assert.deepEqual(a.roles,[]);
});

test('outer analyst triggers on disagreement', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:1,consent:{hermesReasoning:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{crossPoolDisagreements:1}); assert.ok(a.roles.includes('OUTER_ANALYST'));
});

test('gap analyst triggers on missing capability', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:1,consent:{hermesReasoning:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{missingCapabilities:1}); assert.ok(a.roles.includes('GAP_ANALYST'));
});

test('at most two normal hermes reasoning seats trigger', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:1,consent:{hermesReasoning:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{crossPoolDisagreements:2,missingCapabilities:2,repairLoops:4}); assert.equal(a.normalTriggeredCount,2);
});

test('discovery scout does not trigger before cycle ten', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:9,consent:{hermesReasoning:true,discovery:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{}); assert.ok(!a.roles.includes('DISCOVERY_SCOUT'));
});

test('one discovery burst triggers every ten completed cycles with consent', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:10,consent:{hermesReasoning:true,discovery:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{}); assert.ok(a.roles.includes('DISCOVERY_SCOUT')); assert.equal(a.discoveryBurstCount,1);
});

test('discovery burst respects already-run marker', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:20,consent:{hermesReasoning:true,discovery:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{discoveryAlreadyRun:true}); assert.ok(!a.roles.includes('DISCOVERY_SCOUT'));
});

test('hermes request gets one reasoning call and no control transfer', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:1,consent:{hermesReasoning:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{crossPoolDisagreements:1}); const req=Team.hermesRequests(t,a,{summary:'public team state'})[0]; assert.equal(req.reasoningCalls,1); assert.equal(req.controlTransfer,false); assert.equal(req.automaticAction,false);
});

test('hermes log is outer perspective only', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:0},waldo:{requestedCount:0},cycle:1,consent:{hermesReasoning:true},runtime:{hermesReasoningAvailable:true},deps});
  const a=Team.hermesTriggerAssessment(t,{missingCapabilities:1}); const req=Team.hermesRequests(t,a,{summary:'public'})[0];
  const log=Team.validateHermesLog(req,{schema:Team.HERMES_LOG_SCHEMA,requestId:req.id,role:req.role,summary:'Gap found',observations:['one'],gaps:['missing verifier'],recommendedSpotUse:[],evidenceRefs:['receipt:a'],limitations:['single pass']}); assert.equal(log.truth.controlTransfer,false); assert.equal(log.authority,'NONE');
});

test('revocation drops raw package and records released bytes', () => {
  const t=Team.createTeam({task,mirror:{requestedCount:1,specialistIds:['mask:requirements']},waldo:{requestedCount:0},deps}); const s=t.pools.MIRROR.seats[0]; const r=Team.revokeSeat(s); assert.equal(r.rawSpecialistPackageRetained,false); assert.equal(r.compiledPackageBytesReleased,s.compiledPackageBytes); assert.equal(r.memoryPromotion,'NONE');
});

test('team has no authority and max remains a ceiling not a requirement', () => {
  const t=Team.createTeam({task:{...task,requestedSpecialists:2},mirror:{requestedCount:2},waldo:{requestedCount:2},deps}); assert.equal(t.authority,'NONE'); assert.equal(t.pools.MIRROR.truth.maxIsNotQuota,true); assert.equal(t.pools.WALDO.truth.maxIsNotQuota,true);
});

console.log('RESULT', pass + '/20 PASS');
