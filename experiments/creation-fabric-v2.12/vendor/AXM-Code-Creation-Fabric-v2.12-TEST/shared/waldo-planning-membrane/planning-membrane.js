'use strict';

const crypto = require('node:crypto');
const CapabilityGap = require('../ai-native-hands/capability-gap-hand');
const EvidenceRouter = require('../ai-native-hands/evidence-router-hand');

const VERSION = '0.50.0';
const ASSESSMENT_SCHEMA = 'axm.waldo.planning-assessment/v0.50';
const PLAN_SCHEMA = 'axm.waldo.visible-plan-proposal/v0.50';
const ROADMAP_SCHEMA = 'axm.waldo.visible-roadmap/v0.50';
const PROGRESS_SCHEMA = 'axm.waldo.roadmap-progress/v0.50';
const CREATION_HANDOFF_SCHEMA = 'axm.waldo.creation-roadmap-handoff/v0.50';
const MODES = Object.freeze(['DIRECT', 'BRIEF', 'ROADMAP', 'DEEP']);
const MODE_LIMITS = Object.freeze({
  DIRECT: { plannerCalls: 0, maxMilestones: 0, maxTokens: 0, revisionPass: false },
  BRIEF: { plannerCalls: 1, maxMilestones: 4, maxTokens: 600, revisionPass: false },
  ROADMAP: { plannerCalls: 1, maxMilestones: 10, maxTokens: 1400, revisionPass: false },
  DEEP: { plannerCalls: 2, maxMilestones: 16, maxTokens: 2600, revisionPass: true }
});
const HIDDEN_KEYS = new Set(['analysis','reasoning','chain_of_thought','chainOfThought','thoughts','scratchpad','hidden_reasoning','hiddenReasoning']);
const KINDS = ['conversation','analysis','creation','operation'];
const LEVELS = ['LOW','MEDIUM','HIGH'];
const CAP_STATES = ['READY','DEGRADED','UNKNOWN','BLOCKED'];

function canon(v) {
  if (v === null || typeof v !== 'object') return JSON.stringify(v);
  if (Array.isArray(v)) return '[' + v.map(canon).join(',') + ']';
  return '{' + Object.keys(v).sort().map(k => JSON.stringify(k) + ':' + canon(v[k])).join(',') + '}';
}
function sha(v) { return crypto.createHash('sha256').update(typeof v === 'string' ? v : canon(v)).digest('hex'); }
function text(v, label, max = 4000) {
  if (typeof v !== 'string' || !v.trim() || v.length > max) throw new Error(label + '_INVALID');
  const out = v.replace(/\r\n?/g, '\n').trim();
  if (/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(out)) throw new Error(label + '_CONTROL_CHAR');
  return out;
}
function id(v, label) {
  const out = text(String(v || ''), label, 120);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,119}$/.test(out)) throw new Error(label + '_INVALID');
  return out;
}
function list(v, label, max = 32, maxText = 1200) {
  if (v == null) return [];
  if (!Array.isArray(v) || v.length > max) throw new Error(label + '_INVALID');
  return [...new Set(v.map((x,i) => text(String(x), label + '_' + i, maxText)))];
}
function choice(v, allowed, label) { if (!allowed.includes(v)) throw new Error(label + '_INVALID'); return v; }
function clone(v) { return JSON.parse(JSON.stringify(v)); }
function rejectHidden(v, at = '$') {
  if (!v || typeof v !== 'object') return;
  if (Array.isArray(v)) return v.forEach((x,i) => rejectHidden(x, at + '[' + i + ']'));
  for (const [k,x] of Object.entries(v)) {
    if (HIDDEN_KEYS.has(k)) throw new Error('VISIBLE_PLAN_FORBIDDEN_HIDDEN_FIELD:' + at + '.' + k);
    rejectHidden(x, at + '.' + k);
  }
}

function normalizeTask(raw = {}) {
  const allowed = new Set(['id','goal','kind','consequence','uncertainty','estimatedSteps','artifactCount','externalEffects','irreversible','capabilityState','requestedMode','constraints']);
  const extra = Object.keys(raw).filter(k => !allowed.has(k));
  if (extra.length) throw new Error('TASK_UNKNOWN_FIELD:' + extra.sort()[0]);
  const requestedMode = String(raw.requestedMode || 'AUTO').toUpperCase();
  choice(requestedMode, ['AUTO', ...MODES], 'REQUESTED_MODE');
  const number = (v, fallback, max, label) => {
    const n = v == null ? fallback : Number(v);
    if (!Number.isSafeInteger(n) || n < 0 || n > max) throw new Error(label + '_INVALID');
    return n;
  };
  return Object.freeze({
    id: id(raw.id || 'task', 'TASK_ID'), goal: text(raw.goal, 'TASK_GOAL', 6000),
    kind: choice(raw.kind || 'analysis', KINDS, 'TASK_KIND'),
    consequence: choice(String(raw.consequence || 'LOW').toUpperCase(), LEVELS, 'TASK_CONSEQUENCE'),
    uncertainty: choice(String(raw.uncertainty || 'LOW').toUpperCase(), LEVELS, 'TASK_UNCERTAINTY'),
    estimatedSteps: number(raw.estimatedSteps, 1, 64, 'TASK_STEPS'),
    artifactCount: number(raw.artifactCount, 0, 64, 'TASK_ARTIFACTS'),
    externalEffects: raw.externalEffects === true, irreversible: raw.irreversible === true,
    capabilityState: choice(String(raw.capabilityState || 'READY').toUpperCase(), CAP_STATES, 'TASK_CAPABILITY_STATE'),
    requestedMode, constraints: list(raw.constraints, 'TASK_CONSTRAINTS', 32, 1600)
  });
}

function assess(raw) {
  const task = normalizeTask(raw); let score = 0;
  if (task.kind === 'creation') score += 2; else if (task.kind !== 'conversation') score += 1;
  if (task.consequence === 'MEDIUM') score += 1; if (task.consequence === 'HIGH') score += 3;
  if (task.uncertainty === 'MEDIUM') score += 1; if (task.uncertainty === 'HIGH') score += 2;
  if (task.estimatedSteps >= 4) score += 1; if (task.estimatedSteps >= 8) score += 2;
  if (task.artifactCount >= 2) score += 1; if (task.artifactCount >= 5) score += 1;
  if (task.externalEffects) score += 2; if (task.irreversible) score += 3;
  if (task.capabilityState === 'DEGRADED' || task.capabilityState === 'UNKNOWN') score += 1;
  if (task.capabilityState === 'BLOCKED') score += 2;
  if (task.kind === 'conversation' && task.estimatedSteps <= 1 && !task.externalEffects && !task.irreversible) score = 0;
  const auto = score <= 1 ? 'DIRECT' : score <= 3 ? 'BRIEF' : score <= 6 ? 'ROADMAP' : 'DEEP';
  const mode = task.requestedMode === 'AUTO' ? auto : task.requestedMode, limits = MODE_LIMITS[mode];
  const core = { schema: ASSESSMENT_SCHEMA, version: VERSION, task, score, mode, plannerCalls: limits.plannerCalls,
    maxMilestones: limits.maxMilestones, maxTokensPerPlannerCall: limits.maxTokens, revisionPass: limits.revisionPass,
    visiblePlanRequired: mode !== 'DIRECT', publicReasoningOnly: true, hiddenReasoningRequested: false, authority: 'NONE' };
  return Object.freeze({ ...core, assessmentSha256: sha(core) });
}

function plannerPrompt(a) {
  if (!a || a.schema !== ASSESSMENT_SCHEMA) throw new Error('ASSESSMENT_REQUIRED');
  if (a.mode === 'DIRECT') return null;
  return [
    'You are WALDO producing a PUBLIC planning artifact before consequential work.',
    'Do not provide chain-of-thought, private reasoning, scratchpad, hidden analysis, or internal deliberation.',
    'Return ONLY JSON using schema ' + PLAN_SCHEMA + '.',
    'This is a candidate roadmap, not permission to act.',
    'Task id: ' + a.task.id, 'Goal: ' + a.task.goal, 'Planning mode: ' + a.mode,
    'Maximum milestones: ' + a.maxMilestones, 'Constraints: ' + JSON.stringify(a.task.constraints),
    'Fields: schema, taskId, mode, summary, assumptions[], milestones[], risks[], questions[], stopConditions[].',
    'Each milestone: {id,title,objective,dependsOn[],acceptanceClaims[{id,claim,kind,risk,passCondition}],requiredCapabilities[],outputs[]}.',
    'Claim kind must be one of: ' + Object.keys(EvidenceRouter.MATRIX).join(', ') + '.',
    'Every acceptance claim starts UNTESTED; do not mark milestones complete.'
  ].join('\n');
}
function revisionPrompt(a, proposal) {
  return [
    'You are WALDO producing a second PUBLIC planning artifact.',
    'Do not provide chain-of-thought, private reasoning, scratchpad, hidden analysis, or internal deliberation.',
    'Improve only dependency, risk, capability, evidence, or stop-condition coverage. Return ONLY the same ' + PLAN_SCHEMA + ' JSON.',
    'Prior public plan digest: ' + proposal.proposalSha256,
    'Prior public plan: ' + JSON.stringify(proposal)
  ].join('\n');
}

function parseProposal(value, a) {
  let p; try { p = typeof value === 'string' ? JSON.parse(value) : clone(value); } catch (e) { throw new Error('VISIBLE_PLAN_JSON_INVALID:' + e.message); }
  rejectHidden(p);
  if (!p || p.schema !== PLAN_SCHEMA || p.taskId !== a.task.id || p.mode !== a.mode) throw new Error('VISIBLE_PLAN_BINDING_INVALID');
  if (!Array.isArray(p.milestones) || p.milestones.length < 1 || p.milestones.length > a.maxMilestones) throw new Error('VISIBLE_PLAN_MILESTONES_INVALID');
  const seen = new Set();
  const milestones = p.milestones.map((m,mi) => {
    if (!m || typeof m !== 'object' || Array.isArray(m)) throw new Error('MILESTONE_INVALID');
    const mid = id(m.id, 'MILESTONE_ID'); if (seen.has(mid)) throw new Error('MILESTONE_DUPLICATE:' + mid); seen.add(mid);
    const claims = (m.acceptanceClaims || []).map((c,ci) => ({ id:id(c.id,'CLAIM_ID'), claim:text(c.claim,'CLAIM',2000),
      kind:text(c.kind,'CLAIM_KIND',80).toLowerCase(), risk:choice(String(c.risk || 'medium').toLowerCase(),['low','medium','high'],'CLAIM_RISK'),
      passCondition:c.passCondition == null ? null : text(c.passCondition,'PASS_CONDITION',2000) }));
    claims.forEach(c => { if (!EvidenceRouter.MATRIX[c.kind]) throw new Error('CLAIM_KIND_UNKNOWN:' + c.kind); });
    return { id:mid, title:text(m.title,'MILESTONE_TITLE',300), objective:text(m.objective,'MILESTONE_OBJECTIVE',2400),
      dependsOn:list(m.dependsOn,'DEPENDS_ON',32,120), acceptanceClaims:claims,
      requiredCapabilities:list(m.requiredCapabilities,'REQUIRED_CAPABILITIES',64,160), outputs:list(m.outputs,'OUTPUTS',32,1000), ordinal:mi + 1 };
  });
  const ids = new Set(milestones.map(m => m.id)); milestones.forEach(m => m.dependsOn.forEach(d => { if (!ids.has(d)) throw new Error('MILESTONE_DEPENDENCY_MISSING:' + d); }));
  const visiting = new Set(), done = new Set(), byId = new Map(milestones.map(m => [m.id,m]));
  function visit(x) { if (visiting.has(x)) throw new Error('MILESTONE_DEPENDENCY_CYCLE:' + x); if (done.has(x)) return; visiting.add(x); byId.get(x).dependsOn.forEach(visit); visiting.delete(x); done.add(x); }
  milestones.forEach(m => visit(m.id));
  const core = { schema:PLAN_SCHEMA, taskId:p.taskId, mode:p.mode, summary:text(p.summary,'PLAN_SUMMARY',3000),
    assumptions:list(p.assumptions,'ASSUMPTIONS',32,1200), milestones, risks:list(p.risks,'RISKS',32,1600),
    questions:list(p.questions,'QUESTIONS',32,1600), stopConditions:list(p.stopConditions,'STOP_CONDITIONS',32,1600),
    truth:{ publicArtifact:true, chainOfThought:false, permissionToAct:false }, authority:'NONE' };
  return Object.freeze({ ...core, proposalSha256:sha(core) });
}

function compileRoadmap({ assessment, proposal = null, capabilityInventory = [], proposalHistory = null } = {}) {
  if (!assessment || assessment.schema !== ASSESSMENT_SCHEMA) throw new Error('ASSESSMENT_REQUIRED');
  if (assessment.mode === 'DIRECT') {
    const core = { schema:ROADMAP_SCHEMA, version:VERSION, taskId:assessment.task.id, goal:assessment.task.goal, mode:'DIRECT', state:'DIRECT_NO_SEPARATE_PLAN',
      summary:null, assumptions:[], milestones:[], risks:[], questions:[], stopConditions:[], capabilityGapReport:null, proposalLineage:[], truth:{publicPlanNotNeeded:true,noAutomaticExecution:true,silentPlanRewrite:false}, authority:'NONE' };
    return Object.freeze({ ...core, roadmapSha256:sha(core), creationHandoff:Object.freeze({schema:CREATION_HANDOFF_SCHEMA,taskId:assessment.task.id,roadmapSha256:null,target:'bounded-creation-program-planner-v1',state:'NOT_REQUIRED',authority:'NONE'}) });
  }
  if (!proposal || proposal.schema !== PLAN_SCHEMA || proposal.taskId !== assessment.task.id || proposal.mode !== assessment.mode) throw new Error('VISIBLE_PLAN_REQUIRED');
  const requirements = proposal.milestones.map(m => ({ id:m.id, capabilities:m.requiredCapabilities, required:true, gapType:'HAND' })).filter(r => r.capabilities.length);
  const gap = requirements.length ? CapabilityGap.compare(requirements, capabilityInventory) : { schema:CapabilityGap.SCHEMA, overall:'READY', requirements:[], missingCapabilities:[], proposedContracts:[], automaticInstall:false, automaticPermission:false, automaticQualityReduction:false };
  const milestones = proposal.milestones.map(m => ({ ...clone(m), acceptance:m.acceptanceClaims.map(c => ({ ...clone(c), route:EvidenceRouter.route(c) })) }));
  const state = gap.overall === 'BLOCKED' ? 'HELD_CAPABILITY_GAP' : gap.overall === 'UNKNOWN' ? 'HELD_CAPABILITY_UNKNOWN' : 'READY_FOR_REVIEW';
  const lineage = Array.isArray(proposalHistory) && proposalHistory.length ? proposalHistory.slice() : [proposal.proposalSha256];
  if (lineage.some(x => !/^[a-f0-9]{64}$/.test(String(x || ''))) || lineage[lineage.length - 1] !== proposal.proposalSha256) throw new Error('PROPOSAL_LINEAGE_INVALID');
  const core = { schema:ROADMAP_SCHEMA, version:VERSION, taskId:assessment.task.id, goal:assessment.task.goal, mode:assessment.mode, state,
    summary:proposal.summary, assumptions:proposal.assumptions.slice(), milestones, risks:proposal.risks.slice(), questions:proposal.questions.slice(),
    stopConditions:proposal.stopConditions.slice(), capabilityGapReport:gap, proposalLineage:lineage,
    truth:{ publicPlanNotChainOfThought:true, planNotAuthority:true, noAutomaticExecution:true, noAutomaticPromotion:true, neuralPlanIsCandidate:true, evidenceRoutesAreUntestedByDefault:true, silentPlanRewrite:false }, authority:'NONE' };
  const roadmapSha256 = sha(core);
  const creationHandoff = Object.freeze({ schema:CREATION_HANDOFF_SCHEMA, taskId:core.taskId, goal:core.goal, roadmapSha256, planningMode:core.mode,
    milestoneIds:milestones.map(m => m.id), constraints:[...core.stopConditions,...core.risks], target:'bounded-creation-program-planner-v1',
    state:state === 'READY_FOR_REVIEW' ? 'CANDIDATE_FOR_CREATION_PLANNER' : 'HELD', authority:'NONE' });
  return Object.freeze({ ...core, roadmapSha256, creationHandoff });
}

function hermesQueuePayloads(roadmap) {
  if (!roadmap || roadmap.schema !== ROADMAP_SCHEMA) throw new Error('ROADMAP_REQUIRED');
  return roadmap.milestones.map((m,i) => ({ source:'waldo-visible-roadmap', kind:'roadmap-milestone', title:'['+(i+1)+'/'+roadmap.milestones.length+'] '+m.title,
    text:['Roadmap: '+roadmap.roadmapSha256,'Milestone: '+m.id,'Objective: '+m.objective,'Depends on: '+(m.dependsOn.join(', ')||'none'),
      'Required capabilities: '+(m.requiredCapabilities.join(', ')||'none'),'Outputs: '+(m.outputs.join(', ')||'none'),'Acceptance claims: '+(m.acceptance.map(a=>a.claim).join(' | ')||'none'),
      'This is a proposal queue item, not execution authority.'].join('\n'), tags:['waldo-roadmap','roadmap:'+roadmap.roadmapSha256.slice(0,16),'milestone:'+m.id].slice(0,12),
    axm_rule:'proposal only; no direct edits; review before apply' }));
}

function progressView(roadmap, receipts = []) {
  if (!roadmap || roadmap.schema !== ROADMAP_SCHEMA) throw new Error('ROADMAP_REQUIRED');
  if (!Array.isArray(receipts) || receipts.length > 512) throw new Error('PROGRESS_RECEIPTS_INVALID');
  const milestones = roadmap.milestones.map(m => ({ id:m.id, dependsOn:m.dependsOn.slice(), acceptanceCount:m.acceptance.length, state:'PENDING', evidenceDigests:[], note:null }));
  const byId = new Map(milestones.map(m => [m.id,m])), seen = new Set();
  for (const raw of receipts) {
    if (!raw || raw.roadmapSha256 !== roadmap.roadmapSha256) throw new Error('PROGRESS_ROADMAP_DIGEST_MISMATCH');
    const mid = id(raw.milestoneId,'PROGRESS_MILESTONE'); if (!byId.has(mid)) throw new Error('PROGRESS_UNKNOWN_MILESTONE:'+mid);
    const state = choice(raw.state,['PENDING','IN_PROGRESS','BLOCKED','FAILED','DONE'],'PROGRESS_STATE');
    const evidence = list(raw.evidenceDigests,'PROGRESS_EVIDENCE',32,100).map(x => { const d=x.toLowerCase().replace(/^sha256:/,''); if(!/^[a-f0-9]{64}$/.test(d)) throw new Error('PROGRESS_EVIDENCE_DIGEST_INVALID'); return d; });
    const signature = mid+':'+state+':'+evidence.join(',')+':'+String(raw.note||''); if (seen.has(signature)) continue; seen.add(signature);
    const m = byId.get(mid);
    if (state === 'DONE') { m.dependsOn.forEach(dep => { if (byId.get(dep).state !== 'DONE') throw new Error('MILESTONE_DONE_BEFORE_DEPENDENCY:'+dep); }); if (m.acceptanceCount && !evidence.length) throw new Error('MILESTONE_DONE_REQUIRES_EVIDENCE:'+mid); }
    m.state=state; m.evidenceDigests=evidence; m.note=raw.note==null?null:text(raw.note,'PROGRESS_NOTE',1200);
  }
  const state = milestones.length && milestones.every(m=>m.state==='DONE') ? 'COMPLETE' : milestones.some(m=>m.state==='FAILED') ? 'FAILED' : milestones.some(m=>m.state==='BLOCKED') ? 'BLOCKED' : 'ACTIVE';
  const core = { schema:PROGRESS_SCHEMA, roadmapSha256:roadmap.roadmapSha256, state, milestones, authority:'NONE' };
  return Object.freeze({ ...core, progressSha256:sha(core) });
}

async function runVisiblePlanning({ task, generate, capabilityInventory = [] } = {}) {
  const assessment = assess(task);
  if (assessment.mode === 'DIRECT') return { assessment, proposal:null, proposalHistory:[], roadmap:compileRoadmap({assessment}), neuralPlannerCalls:0 };
  if (typeof generate !== 'function') throw new Error('VISIBLE_PLAN_GENERATOR_REQUIRED');
  let proposal = parseProposal(await generate(plannerPrompt(assessment), {maxTokens:assessment.maxTokensPerPlannerCall,temperature:0.2,purpose:'PUBLIC_PLAN'}), assessment);
  const history = [proposal.proposalSha256]; let calls = 1;
  if (assessment.revisionPass) { proposal = parseProposal(await generate(revisionPrompt(assessment,proposal), {maxTokens:assessment.maxTokensPerPlannerCall,temperature:0.15,purpose:'PUBLIC_PLAN_REVISION'}), assessment); history.push(proposal.proposalSha256); calls++; }
  return { assessment, proposal, proposalHistory:history.slice(), roadmap:compileRoadmap({assessment,proposal,capabilityInventory,proposalHistory:history}), neuralPlannerCalls:calls };
}

module.exports = { VERSION, ASSESSMENT_SCHEMA, PLAN_SCHEMA, ROADMAP_SCHEMA, PROGRESS_SCHEMA, CREATION_HANDOFF_SCHEMA, MODES, MODE_LIMITS,
  assess, plannerPrompt, revisionPrompt, parseProposal, compileRoadmap, hermesQueuePayloads, progressView, runVisiblePlanning };
