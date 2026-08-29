'use strict';
const assert=require('node:assert/strict');
const D=require('./determinization-steward');
const L=require('./specialist-lifecycle');
let pass=0;const test=(n,f)=>{f();pass++;console.log('PASS',n)};
const episode={episodeId:'e1',taskRef:'task:website',sourceController:'WALDO',steps:[
 {stepId:'s1',operationKey:'choose-creative-direction',mechanismClass:'NEURAL_DECISION_SUMMARY',inputContractDigest:'sha256:a',outputContractDigest:'sha256:b',visibleDecisionSummary:'Choose one visual direction',verifierStatus:'UNKNOWN',uncertainties:['taste']},
 {stepId:'s2',operationKey:'validate-route-packet',mechanismClass:'HYBRID_HANDOFF',inputContractDigest:'sha256:c',outputContractDigest:'sha256:d',visibleDecisionSummary:'Validate exact packet fields',repeatKey:'route-validation-v1',verifierStatus:'PASS'}
]};
test('private reasoning is refused',()=>assert.throws(()=>D.normalizeEpisode({...episode,analysis:'secret'}),/PRIVATE_REASONING/));
const h=D.harvest(episode);
test('creative frontier step stays neural',()=>assert.equal(h.hypotheses[0].decision,'KEEP_NEURAL'));
test('mechanical repeated step becomes capability hypothesis',()=>assert.equal(h.hypotheses[1].decision,'NEW_CAPABILITY_HYPOTHESIS'));
test('hypothesis cannot self admit',()=>assert.equal(h.hypotheses[1].truth.automaticAdmission,false));
const hyp=h.hypotheses[1];
test('too little replay stays needs-more-experience',()=>assert.equal(D.evaluate(hyp,[{trialId:'t1',fixtureDigest:'f1',outputDigest:'o1',verifierStatus:'PASS',deterministicRuntimeUsed:true}]).decision,'NEEDS_MORE_EXPERIENCE'));
const trials=[
 {trialId:'t1',fixtureDigest:'f1',outputDigest:'o1',verifierStatus:'PASS',deterministicRuntimeUsed:true,latencyMs:2},
 {trialId:'t2',fixtureDigest:'f1',outputDigest:'o1',verifierStatus:'PASS',deterministicRuntimeUsed:true,latencyMs:2},
 {trialId:'t3',fixtureDigest:'f2',outputDigest:'o2',verifierStatus:'PASS',deterministicRuntimeUsed:true,latencyMs:3}
];
const ev=D.evaluate(hyp,trials);
test('three passes across two fixtures become ready for review',()=>assert.equal(ev.decision,'READY_FOR_REVIEW'));
test('ready for review is not admission',()=>assert.equal(ev.truth.readyForReviewIsNotAdmission,true));
test('divergent replay blocks review',()=>assert.equal(D.evaluate(hyp,[...trials,{trialId:'t4',fixtureDigest:'f1',outputDigest:'DIFFERENT',verifierStatus:'PASS',deterministicRuntimeUsed:true}]).decision,'NEEDS_MORE_EXPERIENCE'));
test('undeclared side effect blocks review',()=>assert.equal(D.evaluate(hyp,[...trials.slice(0,2),{trialId:'t3',fixtureDigest:'f2',outputDigest:'o2',verifierStatus:'PASS',deterministicRuntimeUsed:true,undeclaredSideEffects:true}]).decision,'NEEDS_MORE_EXPERIENCE'));
const handoff=D.factoryHandoff(hyp,ev);
test('factory handoff is review-only',()=>{assert.equal(handoff.state,'READY_FOR_FACTORY_REVIEW');assert.equal(handoff.automaticBuild,false);assert.equal(handoff.automaticAdmission,false)});
test('unadmitted catalog keeps neural route',()=>assert.equal(D.fastPathDecision({operationKey:hyp.operationKey,inputContractDigest:hyp.inputContractDigest,catalog:[{status:'REVIEW',operationKey:hyp.operationKey,inputContractDigest:hyp.inputContractDigest}]}).state,'NEURAL_OR_EXISTING_ROUTE_REQUIRED'));
const fp={status:'ADMITTED_FAST_PATH',operationKey:hyp.operationKey,inputContractDigest:hyp.inputContractDigest,executorRef:'hand:route-validator',replayEvidenceDigest:'sha256:evidence'};
test('one exact admitted fast path can avoid neural call',()=>{const d=D.fastPathDecision({operationKey:hyp.operationKey,inputContractDigest:hyp.inputContractDigest,catalog:[fp]});assert.equal(d.state,'USE_DETERMINISTIC_FAST_PATH');assert.equal(d.neuralCallAvoidable,true);assert.equal(d.automaticExecution,false)});
test('ambiguous fast paths hold',()=>assert.equal(D.fastPathDecision({operationKey:hyp.operationKey,inputContractDigest:hyp.inputContractDigest,catalog:[fp,{...fp,executorRef:'hand:other'}]}).state,'HOLD_AMBIGUOUS_FAST_PATH'));
function seat(owner='WALDO',profile='validation'){return{id:'seat1',owner,state:'ACTIVE',specialistId:'mask:'+profile,profileId:profile,packageFingerprint:'pkg1',compiledPackageBytes:123,package:{x:1},innerSettings:{},authority:'NONE',seatDigest:'old'}}
test('unpinned specialist disappears when task completes',()=>{const x=L.complete(seat(),'done');assert.equal(x.state,'REVOKED');assert.equal(x.seat,null);assert.equal(x.receipt.compiledPackageBytesReleased,123)});
const pinned=L.pin(seat(),'WALDO','keep warm');
test('owner can pin own specialist',()=>assert.equal(pinned.pin.locked,true));
test('other controller cannot pin it',()=>assert.throws(()=>L.pin(seat(),'MIRROR'),/OWN_SEAT/));
const idle=L.complete(pinned,'done');
test('pinned specialist becomes idle locked and keeps package',()=>{assert.equal(idle.state,'IDLE_LOCKED');assert.equal(idle.seat.state,'IDLE_LOCKED');assert.equal(idle.receipt.rawPackageRetained,true)});
test('idle locked specialist can resume without recompilation',()=>{const r=L.resume(idle.seat,'WALDO','next');assert.equal(r.state,'ACTIVE');assert.equal(r.packageFingerprint,'pkg1')});
test('all five slots may be resident/pinned',()=>{const seats=Array.from({length:5},(_,i)=>({...idle.seat,id:'s'+i}));assert.equal(L.residentCount(seats),5);assert.equal(L.canAllocate(seats,5),false)});
test('idle locked matching specialist is reusable',()=>assert.equal(L.reusable([idle.seat],'validation').state,'IDLE_LOCKED'));
test('controller can unpin and release bytes',()=>{const x=L.unpin(idle.seat,'WALDO');assert.equal(x.state,'REVOKED');assert.equal(x.receipt.compiledPackageBytesReleased,123)});
test('Hermes cannot pin normal specialist seat',()=>assert.throws(()=>L.pin(seat(),'HERMES'),/ONLY_MIRROR_OR_WALDO/));
console.log('RESULT',pass+'/22 PASS');
