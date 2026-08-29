'use strict';
const crypto = require('node:crypto');

const VERSION='0.53.0';
const EPISODE_SCHEMA='axm.visible-frontier-episode/v0.53';
const HYPOTHESIS_SCHEMA='axm.determinization-hypothesis/v0.53';
const EVALUATION_SCHEMA='axm.determinization-evaluation/v0.53';
const FAST_PATH_SCHEMA='axm.deterministic-fast-path-decision/v0.53';
const FACTORY_HANDOFF_SCHEMA='axm.determinization-factory-handoff/v0.53';
const DECISIONS=Object.freeze(['KEEP_NEURAL','NEEDS_MORE_EXPERIENCE','NEW_CAPABILITY_HYPOTHESIS','READY_FOR_REVIEW']);
const MECHANISMS=Object.freeze(['DETERMINISTIC_RULE','NEURAL_DECISION_SUMMARY','HYBRID_HANDOFF','EXTERNAL_OBSERVATION']);
const FACTORY_KINDS=Object.freeze(['HAND','SKILL','ORGAN','CAPABILITY']);
const HIDDEN=new Set(['analysis','reasoning','chain_of_thought','chainOfThought','scratchpad','hidden_reasoning','hiddenReasoning','logits','internalActivations','hiddenState','rawPrompt','rawResponse']);

function canon(v){if(v===null||typeof v!=='object')return JSON.stringify(v);if(Array.isArray(v))return'['+v.map(canon).join(',')+']';return'{'+Object.keys(v).sort().map(k=>JSON.stringify(k)+':'+canon(v[k])).join(',')+'}'}
function sha(v){return crypto.createHash('sha256').update(typeof v==='string'?v:canon(v)).digest('hex')}
function clone(v){return JSON.parse(JSON.stringify(v))}
function text(v,label,max=4000){if(typeof v!=='string'||!v.trim()||v.length>max)throw Error(label+'_INVALID');const x=v.replace(/\r\n?/g,'\n').trim();if(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(x))throw Error(label+'_CONTROL_CHAR');return x}
function rejectHidden(v,at='$'){if(!v||typeof v!=='object')return;if(Array.isArray(v))return v.forEach((x,i)=>rejectHidden(x,at+'['+i+']'));for(const[k,x]of Object.entries(v)){if(HIDDEN.has(k))throw Error('PRIVATE_REASONING_FIELD_REFUSED:'+at+'.'+k);rejectHidden(x,at+'.'+k)}}
function arr(v,label,max=32){if(v==null)return[];if(!Array.isArray(v)||v.length>max)throw Error(label+'_INVALID');return [...new Set(v.map((x,i)=>text(String(x),label+'_'+i,800)))].sort()}
function mechanism(v){const x=String(v||'').toUpperCase();if(!MECHANISMS.includes(x))throw Error('MECHANISM_INVALID');return x}
function factoryKind(v){const x=String(v||'CAPABILITY').toUpperCase();if(!FACTORY_KINDS.includes(x))throw Error('FACTORY_KIND_INVALID');return x}

function normalizeEpisode(input={}){
  rejectHidden(input);
  if(!Array.isArray(input.steps)||input.steps.length<1||input.steps.length>128)throw Error('EPISODE_STEPS_INVALID');
  const steps=input.steps.map((s,i)=>({
    stepId:text(String(s.stepId||('step-'+(i+1))),'STEP_ID',120),
    operationKey:text(s.operationKey,'OPERATION_KEY',240),
    mechanismClass:mechanism(s.mechanismClass),
    inputContractDigest:text(s.inputContractDigest,'INPUT_CONTRACT_DIGEST',120),
    outputContractDigest:text(s.outputContractDigest,'OUTPUT_CONTRACT_DIGEST',120),
    visibleDecisionSummary:s.visibleDecisionSummary==null?null:text(s.visibleDecisionSummary,'VISIBLE_SUMMARY',1600),
    repeatKey:s.repeatKey==null?null:text(s.repeatKey,'REPEAT_KEY',240),
    externalEffects:s.externalEffects===true,
    uncertainties:arr(s.uncertainties,'UNCERTAINTIES',16),
    verifierStatus:['PASS','FAIL','UNKNOWN','NOT_RUN'].includes(String(s.verifierStatus||'UNKNOWN').toUpperCase())?String(s.verifierStatus||'UNKNOWN').toUpperCase():'UNKNOWN',
    cost:{neuralUnits:Number.isFinite(s.cost&&s.cost.neuralUnits)?Math.max(0,s.cost.neuralUnits):null,latencyMs:Number.isFinite(s.cost&&s.cost.latencyMs)?Math.max(0,s.cost.latencyMs):null}
  }));
  const core={schema:EPISODE_SCHEMA,version:VERSION,episodeId:text(input.episodeId,'EPISODE_ID',160),taskRef:text(input.taskRef,'TASK_REF',240),sourceController:text(String(input.sourceController||'WALDO').toUpperCase(),'SOURCE_CONTROLLER',40),steps,authority:'NONE'};
  return Object.freeze({...core,episodeDigest:sha(core)});
}

function inferFactory(step){
  const op=step.operationKey.toLowerCase();
  if(op.includes('perspective')||op.includes('summarize')||op.includes('review-prompt'))return'SKILL';
  if(op.includes('route')||op.includes('transform')||op.includes('validate')||op.includes('write')||op.includes('materialize'))return'HAND';
  if(op.includes('organ')||op.includes('grammar')||op.includes('domain'))return'ORGAN';
  return'CAPABILITY';
}

function hypothesisFromStep(episode,step){
  const mechanicalSignal=!!step.repeatKey||['DETERMINISTIC_RULE','HYBRID_HANDOFF'].includes(step.mechanismClass)||/route|validate|transform|format|map|filter|materialize|serialize|parse|select|compare|check/i.test(step.operationKey);
  const decision=mechanicalSignal?'NEW_CAPABILITY_HYPOTHESIS':'KEEP_NEURAL';
  const core={schema:HYPOTHESIS_SCHEMA,version:VERSION,decision,episodeRef:{episodeId:episode.episodeId,episodeDigest:episode.episodeDigest},sourceStepId:step.stepId,operationKey:step.operationKey,inputContractDigest:step.inputContractDigest,outputContractDigest:step.outputContractDigest,factoryKind:inferFactory(step),repeatKey:step.repeatKey,externalEffectsObserved:step.externalEffects,uncertainties:step.uncertainties,requiredReplay:{minimumTrials:3,minimumDistinctFixtures:2,verifierPassRequired:true,noDivergenceRequired:true,noUndeclaredSideEffectsRequired:true},truth:{hypothesisIsNotCapability:true,privateReasoningCaptured:false,automaticBuild:false,automaticAdmission:false},authority:'NONE'};
  return Object.freeze({...core,hypothesisDigest:sha(core)});
}

function harvest(input={}){
  const episode=normalizeEpisode(input);
  const hypotheses=episode.steps.map(s=>hypothesisFromStep(episode,s));
  return Object.freeze({schema:'axm.determinization-harvest/v0.53',episodeRef:{episodeId:episode.episodeId,episodeDigest:episode.episodeDigest},hypotheses,counts:Object.fromEntries(DECISIONS.map(d=>[d,hypotheses.filter(h=>h.decision===d).length])),truth:{visibleEpisodeOnly:true,hiddenReasoningNotRequested:true,oneThoughtMaySeedHypothesisButCannotSelfAdmit:true},authority:'NONE',harvestDigest:sha(hypotheses)});
}

function evaluate(hypothesis,trials=[]){
  if(!hypothesis||hypothesis.schema!==HYPOTHESIS_SCHEMA)throw Error('HYPOTHESIS_REQUIRED');
  if(!Array.isArray(trials)||trials.length>256)throw Error('TRIALS_INVALID');
  const normalized=trials.map((t,i)=>({trialId:text(String(t.trialId||('trial-'+(i+1))),'TRIAL_ID',120),fixtureDigest:text(t.fixtureDigest,'FIXTURE_DIGEST',120),outputDigest:text(t.outputDigest,'OUTPUT_DIGEST',120),verifierStatus:String(t.verifierStatus||'UNKNOWN').toUpperCase(),undeclaredSideEffects:t.undeclaredSideEffects===true,deterministicRuntimeUsed:t.deterministicRuntimeUsed===true,latencyMs:Number.isFinite(t.latencyMs)?Math.max(0,t.latencyMs):null}));
  const byFixture=new Map(); for(const t of normalized){if(!byFixture.has(t.fixtureDigest))byFixture.set(t.fixtureDigest,new Set());byFixture.get(t.fixtureDigest).add(t.outputDigest)}
  const divergence=[...byFixture.entries()].filter(([,outs])=>outs.size>1).map(([fixture])=>fixture);
  const passCount=normalized.filter(t=>t.verifierStatus==='PASS'&&t.deterministicRuntimeUsed&&!t.undeclaredSideEffects).length;
  const distinctFixtures=byFixture.size;
  const failed=normalized.some(t=>t.verifierStatus==='FAIL'||t.undeclaredSideEffects);
  let decision='NEEDS_MORE_EXPERIENCE';
  if(hypothesis.decision==='KEEP_NEURAL')decision='KEEP_NEURAL';
  else if(failed||divergence.length)decision='NEEDS_MORE_EXPERIENCE';
  else if(normalized.length>=hypothesis.requiredReplay.minimumTrials&&distinctFixtures>=hypothesis.requiredReplay.minimumDistinctFixtures&&passCount===normalized.length)decision='READY_FOR_REVIEW';
  const core={schema:EVALUATION_SCHEMA,version:VERSION,hypothesisDigest:hypothesis.hypothesisDigest,decision,trialCount:normalized.length,passCount,distinctFixtures,divergentFixtureDigests:divergence,failedOrSideEffectObserved:failed,trials:normalized,truth:{readyForReviewIsNotAdmission:true,readyForReviewIsNotRuntimeAuthority:true,neuralFallbackRemainsAvailable:true},authority:'NONE'};
  return Object.freeze({...core,evaluationDigest:sha(core)});
}

function factoryHandoff(hypothesis,evaluation){
  if(!evaluation||evaluation.schema!==EVALUATION_SCHEMA||evaluation.hypothesisDigest!==hypothesis.hypothesisDigest)throw Error('EVALUATION_BINDING_INVALID');
  const target={HAND:'hand-specification-foundry',SKILL:'agent-tool-forge/skills',ORGAN:'deterministic-organ-fabric',CAPABILITY:'capability-fabric'}[factoryKind(hypothesis.factoryKind)];
  const core={schema:FACTORY_HANDOFF_SCHEMA,version:VERSION,state:evaluation.decision==='READY_FOR_REVIEW'?'READY_FOR_FACTORY_REVIEW':'HELD_EVIDENCE_NOT_READY',targetFactory:target,hypothesisDigest:hypothesis.hypothesisDigest,evaluationDigest:evaluation.evaluationDigest,operationKey:hypothesis.operationKey,inputContractDigest:hypothesis.inputContractDigest,outputContractDigest:hypothesis.outputContractDigest,automaticBuild:false,automaticInstall:false,automaticAdmission:false,automaticPromotion:false,authority:'NONE'};
  return Object.freeze({...core,handoffDigest:sha(core)});
}

function fastPathDecision(input={}){
  const operationKey=text(input.operationKey,'FAST_OPERATION',240),inputContractDigest=text(input.inputContractDigest,'FAST_INPUT_CONTRACT',120); const catalog=Array.isArray(input.catalog)?input.catalog:[];
  const matches=catalog.filter(x=>x&&x.status==='ADMITTED_FAST_PATH'&&x.operationKey===operationKey&&x.inputContractDigest===inputContractDigest&&typeof x.executorRef==='string'&&x.executorRef&&typeof x.replayEvidenceDigest==='string'&&x.replayEvidenceDigest);
  const selected=matches.length===1?clone(matches[0]):null;
  const state=selected?'USE_DETERMINISTIC_FAST_PATH':matches.length>1?'HOLD_AMBIGUOUS_FAST_PATH':'NEURAL_OR_EXISTING_ROUTE_REQUIRED';
  const core={schema:FAST_PATH_SCHEMA,version:VERSION,state,operationKey,inputContractDigest,selected,matchCount:matches.length,neuralCallAvoidable:selected!==null,automaticExecution:false,automaticAdmission:false,authority:'NONE'};
  return Object.freeze({...core,decisionDigest:sha(core)});
}

module.exports=Object.freeze({VERSION,EPISODE_SCHEMA,HYPOTHESIS_SCHEMA,EVALUATION_SCHEMA,FAST_PATH_SCHEMA,FACTORY_HANDOFF_SCHEMA,DECISIONS,MECHANISMS,FACTORY_KINDS,canon,sha,normalizeEpisode,harvest,evaluate,factoryHandoff,fastPathDecision});
