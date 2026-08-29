#!/usr/bin/env node
'use strict';
const assert = require('node:assert/strict');
const P = require('./planning-membrane');
let checks=0; const eq=(a,b,m)=>{assert.equal(a,b,m);checks++}; const ok=(v,m)=>{assert.ok(v,m);checks++}; const throws=(f,r,m)=>{assert.throws(f,r,m);checks++};
function task(extra={}) { return Object.assign({id:'build',goal:'Create a small tested module.',kind:'creation',consequence:'MEDIUM',uncertainty:'MEDIUM',estimatedSteps:6,artifactCount:2,externalEffects:false,irreversible:false,capabilityState:'READY',requestedMode:'AUTO'},extra); }
function proposal(a, opts={}) { const ms=opts.milestones||[
  {id:'m1',title:'Inspect',objective:'Inspect bounded inputs.',dependsOn:[],acceptanceClaims:[{id:'c1',claim:'Inputs are bounded.',kind:'structure',risk:'medium',passCondition:'schema validates'}],requiredCapabilities:['context.read'],outputs:['input map']},
  {id:'m2',title:'Create',objective:'Create candidate.',dependsOn:['m1'],acceptanceClaims:[{id:'c2',claim:'Candidate behaves as required.',kind:'behavior',risk:'medium',passCondition:'focused tests pass'}],requiredCapabilities:['code.create'],outputs:['candidate']}
]; return JSON.stringify({schema:P.PLAN_SCHEMA,taskId:a.task.id,mode:a.mode,summary:'Visible candidate plan.',assumptions:['Inputs exist.'],milestones:ms,risks:['Capability may be absent.'],questions:[],stopConditions:['Stop on missing evidence.']}); }
(async()=>{
 const d=P.assess(task({id:'chat',goal:'Say hello.',kind:'conversation',consequence:'LOW',uncertainty:'LOW',estimatedSteps:1,artifactCount:0})); eq(d.mode,'DIRECT'); eq(d.plannerCalls,0); eq(P.plannerPrompt(d),null);
 const r=P.assess(task()); eq(r.mode,'ROADMAP'); ok(/Do not provide chain-of-thought/.test(P.plannerPrompt(r)));
 const deep=P.assess(task({consequence:'HIGH',uncertainty:'HIGH',estimatedSteps:12,artifactCount:6,externalEffects:true})); eq(deep.mode,'DEEP'); eq(deep.plannerCalls,2); ok(P.revisionPrompt(deep,P.parseProposal(proposal(deep),deep)).includes('second PUBLIC'));
 eq(P.assess(task({requestedMode:'BRIEF',consequence:'HIGH',uncertainty:'HIGH'})).mode,'BRIEF');
 const hidden=JSON.parse(proposal(r)); hidden.reasoning='secret'; throws(()=>P.parseProposal(hidden,r),/FORBIDDEN_HIDDEN_FIELD/);
 const cyc=JSON.parse(proposal(r)); cyc.milestones[0].dependsOn=['m2']; throws(()=>P.parseProposal(cyc,r),/DEPENDENCY_CYCLE/);
 const parsed=P.parseProposal(proposal(r),r);
 const held=P.compileRoadmap({assessment:r,proposal:parsed,capabilityInventory:[{id:'context.read',status:'available',constraints:[]}]}); eq(held.state,'HELD_CAPABILITY_GAP'); ok(held.capabilityGapReport.missingCapabilities.includes('code.create')); eq(held.milestones[0].acceptance[0].route.verdict,'UNTESTED'); eq(held.creationHandoff.target,'bounded-creation-program-planner-v1'); eq(held.creationHandoff.state,'HELD');
 const ready=P.compileRoadmap({assessment:r,proposal:parsed,capabilityInventory:[{id:'context.read',status:'available',constraints:[]},{id:'code.create',status:'available',constraints:['candidate-only']}]}); eq(ready.state,'READY_FOR_REVIEW'); eq(ready.creationHandoff.state,'CANDIDATE_FOR_CREATION_PLANNER'); eq(ready.truth.noAutomaticExecution,true); eq(ready.truth.silentPlanRewrite,false);
 const hermes=P.hermesQueuePayloads(ready); eq(hermes.length,2); ok(hermes.every(x=>x.axm_rule==='proposal only; no direct edits; review before apply')); ok(hermes.every(x=>x.source==='waldo-visible-roadmap'));
 throws(()=>P.progressView(ready,[{roadmapSha256:ready.roadmapSha256,milestoneId:'m2',state:'DONE',evidenceDigests:['a'.repeat(64)]}]),/BEFORE_DEPENDENCY/);
 throws(()=>P.progressView(ready,[{roadmapSha256:ready.roadmapSha256,milestoneId:'m1',state:'DONE',evidenceDigests:[]}]),/REQUIRES_EVIDENCE/);
 const progress=P.progressView(ready,[{roadmapSha256:ready.roadmapSha256,milestoneId:'m1',state:'DONE',evidenceDigests:['a'.repeat(64)]},{roadmapSha256:ready.roadmapSha256,milestoneId:'m2',state:'DONE',evidenceDigests:['b'.repeat(64)]}]); eq(progress.state,'COMPLETE'); eq(progress.authority,'NONE');
 let directCalls=0; const directRun=await P.runVisiblePlanning({task:task({id:'direct',goal:'Say hello.',kind:'conversation',consequence:'LOW',uncertainty:'LOW',estimatedSteps:1,artifactCount:0}),generate:async()=>{directCalls++;return ''}}); eq(directRun.neuralPlannerCalls,0); eq(directCalls,0);
 let roadmapCalls=0; const roadRun=await P.runVisiblePlanning({task:task({id:'road'}),capabilityInventory:[{id:'context.read',status:'available',constraints:[]},{id:'code.create',status:'available',constraints:[]}],generate:async(prompt,budget)=>{roadmapCalls++;eq(budget.purpose,'PUBLIC_PLAN');const a=P.assess(task({id:'road'}));return proposal(a)}}); eq(roadRun.neuralPlannerCalls,1); eq(roadmapCalls,1);
 let deepCalls=0; const deepTask=task({id:'deep-run',consequence:'HIGH',uncertainty:'HIGH',estimatedSteps:12,artifactCount:6,externalEffects:true}); const deepRun=await P.runVisiblePlanning({task:deepTask,capabilityInventory:[{id:'context.read',status:'available',constraints:[]},{id:'code.create',status:'available',constraints:[]}],generate:async(prompt,budget)=>{deepCalls++;if(deepCalls===2)eq(budget.purpose,'PUBLIC_PLAN_REVISION');return proposal(P.assess(deepTask))}}); eq(deepRun.neuralPlannerCalls,2); eq(deepCalls,2); eq(deepRun.proposalHistory.length,2); eq(deepRun.roadmap.proposalLineage.length,2); eq(deepRun.roadmap.truth.silentPlanRewrite,false);
 console.log('WALDO visible planning membrane selftest PASS ('+checks+' checks)');
})().catch(e=>{console.error(e.stack||e);process.exit(1)});
