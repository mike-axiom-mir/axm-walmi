'use strict';

const crypto = require('crypto');
const route = require('../creation-fabric-v2.12/vendor/AXM-Code-Creation-Fabric-v2.12-TEST/shared/code-capability-fabric/language-organs/code-route-knowledge.js');
const planner = require('../creation-fabric-v2.12/vendor/AXM-Code-Creation-Fabric-v2.12-TEST/shared/code-capability-fabric/language-organs/code-creation-session-planner.js');

const SCHEMA = 'axm.waldo.website-creation-probe/v0.51';
const DEFAULT_BRIEF = 'Create a website.';
const ACTIVE_LANGUAGES = Object.freeze(['html', 'css', 'javascript']);

function canon(value) {
  if (value === null || typeof value !== 'object') return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canon).join(',')}]`;
  return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${canon(value[key])}`).join(',')}}`;
}

function hash(value) {
  return crypto.createHash('sha256').update(typeof value === 'string' ? value : canon(value)).digest('hex');
}

function evidenceDigest(label, brief) {
  return `sha256:${hash(`${label}\n${brief}`)}`;
}

function deriveRequiredBindings(recipe) {
  const bindings = {};
  for (const role of recipe.roles.filter(entry => !entry.optional)) {
    const candidate = role.candidates.find(entry => entry.active);
    if (!candidate) return null;
    bindings[role.role] = candidate.languageId;
  }
  return bindings;
}

function runWebsiteCreationProbe(brief = DEFAULT_BRIEF) {
  if (typeof brief !== 'string' || !brief.trim()) throw new Error('BRIEF_REQUIRED');
  const exactBrief = brief.trim();
  const routePlan = route.planGoal({
    goal: exactBrief,
    activeLanguageIds: ACTIVE_LANGUAGES,
    preferredLanguageIds: [],
    topRecipes: 5
  });
  const recipe = routePlan.recipes[0] || null;
  const roleBindings = recipe && deriveRequiredBindings(recipe);
  if (!recipe || recipe.id !== 'web-application' || !roleBindings) {
    const held = {
      schema: SCHEMA,
      result: 'HELD_WEB_APPLICATION_ROUTE_UNRESOLVED',
      brief: exactBrief,
      briefSha256: hash(exactBrief),
      routeResult: routePlan.result,
      selectedRecipe: recipe && recipe.id || null,
      sourceGenerated: false,
      writerInvoked: false,
      websiteCreated: false,
      trainingInvoked: false,
      weightsChanged: false,
      authority: 'NONE'
    };
    return Object.freeze({ ...held, receiptSha256: hash(held) });
  }

  const primaryRole = recipe.roles.find(entry => !entry.optional).role;
  const request = planner.createRequest({
    projectId: 'waldo-website-challenge-v0.51',
    goal: exactBrief,
    primaryLanguageId: roleBindings[primaryRole],
    primaryRole,
    activeLanguageIds: ACTIVE_LANGUAGES,
    preferredLanguageIds: [],
    roleBindings,
    intent: 'build',
    observation: {
      goals: [exactBrief],
      capabilities: ['creation fabric route planning', 'exact-byte candidate writer'],
      gaps: ['WALDO neural source candidate not supplied to this run'],
      constraints: ['no substitute model', 'no working-chat authored website source'],
      risks: ['hidden source authorship', 'unverified rendered behavior'],
      requirements: ['WALDO-authored exact file bytes', 'tests and render evidence before completion'],
      signals: ['website', 'browser', 'verification'],
      factCodes: ['ONE_HUMAN_BRIEF'],
      requestedPerspectives: ['browser-runtime', 'verification'],
      roleIds: Object.keys(roleBindings),
      domainOverlays: ['web']
    },
    direction: {
      actorClass: 'MACHINE',
      title: 'WALDO website challenge',
      directionalPrompt: exactBrief,
      roadmap: ['route', 'author', 'write', 'verify', 'repair'],
      constraints: ['source bytes must come from WALDO neural output', 'writer may apply but may not author'],
      sourceLinks: []
    },
    production: {
      draftCount: 1,
      variantAxes: [],
      budgetCeilings: {
        maxTotalDraftRevisions: 8,
        maxRevisionsPerDraft: 8,
        maxArtifactBuilds: 8,
        maxAdmissionChecks: 8,
        maxQuickTests: 8,
        maxHeavyVerifierRuns: 4
      }
    }
  });
  const roots = planner.ROOTS.map(root => ({
    root,
    status: 'PASS',
    evidenceDigest: evidenceDigest(root, exactBrief),
    reasonCodes: ['bounded-local-probe']
  }));
  const rootGate = planner.evaluateRootGate({ request, roots });
  const creationPlan = planner.plan({ request, rootGate });
  const readyForSource = creationPlan.result === 'CREATION_SESSION_PLAN_READY_NO_EXECUTION';
  const core = {
    schema: SCHEMA,
    result: readyForSource ? 'HELD_WALDO_SOURCE_CANDIDATE_ABSENT' : 'HELD_CREATION_PLAN_NOT_READY',
    brief: exactBrief,
    briefSha256: hash(exactBrief),
    oneHumanBrief: true,
    activeLanguages: [...ACTIVE_LANGUAGES],
    route: {
      result: routePlan.result,
      selectedRecipe: recipe.id,
      selectedScore: recipe.score,
      roleBindings
    },
    creation: {
      result: creationPlan.result,
      requestSha256: request.requestSha256,
      rootGateSha256: rootGate.gateSha256,
      planSha256: creationPlan.planSha256,
      readyForSource,
      selectedLanguage: creationPlan.selectedLanguage && creationPlan.selectedLanguage.languageId || null
    },
    source: {
      requiredAuthor: 'WALDO_NEURAL',
      substituteModelAllowed: false,
      workingChatAuthorshipAllowed: false,
      candidateSupplied: false,
      sourceGenerated: false
    },
    writer: {
      command: 'waldo-axm-mirror apply-candidate <request.json> <workspace-root> <receipt.json>',
      requestSchema: 'axm.waldo.candidate-write-request/v0.51',
      role: 'EXACT_BYTE_APPLIER_NOT_AUTHOR',
      state: 'READY_NOT_INVOKED_WITHOUT_SOURCE',
      invoked: false
    },
    verification: {
      state: 'NOT_RUN_NO_ARTIFACT',
      rendered: false,
      repairsApplied: false,
      websiteCreated: false
    },
    benchmark: {
      kind: 'STAGED_OBSERVED_EXECUTION',
      simulation: false,
      completionOnlyScoring: false,
      stages: [
        { id: 'route', state: 'PASS' },
        { id: 'plan', state: 'PASS' },
        { id: 'source', state: 'INFRASTRUCTURE_BLOCKED' },
        { id: 'write', state: 'NOT_RUN' },
        { id: 'render-verify', state: 'NOT_RUN' },
        { id: 'repair', state: 'NOT_RUN' }
      ],
      passedStages: 2,
      blockedStages: 1,
      notRunStages: 3,
      modelQualityScore: null,
      modelQualityScoreReason: 'NO_WALDO_NEURAL_CANDIDATE_EXISTED'
    },
    experience: {
      observedExecutionTrace: true,
      simulation: false,
      benchmarkState: 'INFRASTRUCTURE_BLOCKED',
      retainedDisposition: 'MEMORY_ONLY',
      retainedLesson: 'Planning and writing are connected, but a WALDO neural source candidate or trainable WALDO checkpoint must be present before source, render, repair, or quality scoring can occur.',
      futureContextEligible: true,
      positiveTrainingEligible: false,
      positiveTrainingBlock: 'NO_VERIFIED_HELPFUL_OR_CORRECTED_SOURCE_OUTPUT',
      trajectoryTrainingEligible: true,
      trajectoryTarget: 'OUTCOME_CONDITIONED_REFLECTION',
      completionRequiredForLearning: false,
      failedAttemptSupervised: false,
      trainingInvoked: false,
      weightsChanged: false
    },
    authority: {
      network: false,
      install: false,
      deployment: false,
      merge: false,
      promotion: false,
      canon: false
    }
  };
  return Object.freeze({ ...core, receiptSha256: hash(core) });
}

module.exports = Object.freeze({ SCHEMA, DEFAULT_BRIEF, ACTIVE_LANGUAGES, runWebsiteCreationProbe, canon, hash });

if (require.main === module) {
  process.stdout.write(`${JSON.stringify(runWebsiteCreationProbe(process.argv[2] || DEFAULT_BRIEF), null, 2)}\n`);
}
