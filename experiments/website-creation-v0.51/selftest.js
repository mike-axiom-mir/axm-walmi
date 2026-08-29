'use strict';

const assert = require('assert');
const probe = require('./website-creation-probe.js');
const failure = require('./failure-trajectory.js');

const first = probe.runWebsiteCreationProbe();
const repeat = probe.runWebsiteCreationProbe();
let checks = 0;
function check(condition, message) {
  assert(condition, message);
  checks += 1;
}

check(first.schema === probe.SCHEMA, 'schema');
check(first.brief === 'Create a website.', 'exact brief');
check(first.oneHumanBrief === true, 'one human brief');
check(first.route.selectedRecipe === 'web-application', 'web route');
check(first.route.roleBindings['document-ui'] === 'html', 'document binding');
check(first.route.roleBindings['browser-runtime'] === 'javascript', 'runtime binding');
check(first.creation.result === 'CREATION_SESSION_PLAN_READY_NO_EXECUTION', 'creation plan');
check(first.creation.readyForSource === true, 'source handoff ready');
check(first.source.requiredAuthor === 'WALDO_NEURAL', 'WALDO author');
check(first.source.substituteModelAllowed === false, 'no substitute');
check(first.source.workingChatAuthorshipAllowed === false, 'no chat authorship');
check(first.source.sourceGenerated === false, 'no source');
check(first.writer.role === 'EXACT_BYTE_APPLIER_NOT_AUTHOR', 'writer role');
check(first.writer.invoked === false, 'writer held');
check(first.verification.websiteCreated === false, 'no website claim');
check(first.result === 'HELD_WALDO_SOURCE_CANDIDATE_ABSENT', 'exact hold');
check(first.benchmark.completionOnlyScoring === false, 'partial progress is scored');
check(first.benchmark.passedStages === 2, 'route and plan receive credit');
check(first.benchmark.modelQualityScore === null, 'infrastructure block is not a model score');
check(first.experience.observedExecutionTrace === true, 'observed experience');
check(first.experience.simulation === false, 'not simulation');
check(first.experience.benchmarkState === 'INFRASTRUCTURE_BLOCKED', 'benchmark state');
check(first.experience.retainedDisposition === 'MEMORY_ONLY', 'memory retained');
check(first.experience.futureContextEligible === true, 'future context');
check(first.experience.positiveTrainingEligible === false, 'no invalid positive target');
check(first.experience.trajectoryTrainingEligible === true, 'failure trajectory is trainable');
check(first.experience.completionRequiredForLearning === false, 'learning is not completion gated');
check(first.experience.failedAttemptSupervised === false, 'failed response is not copied');
check(first.experience.trainingInvoked === false, 'no training');
check(first.experience.weightsChanged === false, 'no weight claim');
check(first.receiptSha256 === repeat.receiptSha256, 'deterministic receipt');
const trajectory = failure.sealFailureTrajectory('2026-08-24T19:58:21Z');
check(trajectory.completionRequired === false, 'experience is not completion gated');
check(trajectory.failedAttemptSupervised === false, 'failed action is not imitated');
check(trajectory.reflectionTargetSupervised === true, 'failure reflection is trainable');
check(trajectory.targetKind === 'OUTCOME_CONDITIONED_REFLECTION', 'reflection target kind');

console.log(`WALDO website creation probe selftest PASS (${checks} checks)`);
