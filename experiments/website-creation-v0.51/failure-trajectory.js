'use strict';

const crypto = require('crypto');
const probeModule = require('./website-creation-probe.js');

function sealFailureTrajectory(observedAt) {
  const timestamp = new Date(observedAt);
  if (Number.isNaN(timestamp.getTime())) throw new Error('OBSERVED_AT_INVALID');
  const rfc3339 = value => value.toISOString().replace('.000Z', 'Z');
  const probe = probeModule.runWebsiteCreationProbe();
  const record = {
    schema: 'axm.waldo.experience-trajectory/v0.51',
    id: 'website-v051-source-hold',
    sourceReceiptSha256: probe.receiptSha256,
    observedAt: rfc3339(timestamp),
    outcomeSignal: 'INCONCLUSIVE',
    prompt: probe.brief,
    attemptTrace: 'The web-application route and Creation Fabric plan completed. The exact-byte writer was ready. No WALDO neural source candidate or trainable WALDO checkpoint was present, so source, render, verification, and repair did not run.',
    observedOutcome: 'No website artifact existed. The benchmark was infrastructure-blocked at the source-author step.',
    lesson: 'Treat the incomplete attempt as experience. Preserve the passed route and plan, restore a real WALDO neural source-author runtime or checkpoint, and resume at source generation instead of restarting or substituting another model.',
    targetKind: 'OUTCOME_CONDITIONED_REFLECTION',
    trainingObjective: 'assistant-response-modeling',
    completionRequired: false,
    failedAttemptSupervised: false,
    reflectionTargetSupervised: true,
    authority: {
      tool_execution: false,
      training: false,
      promotion: false,
      canon: false,
      world_action: false
    },
    recordSha256: ''
  };
  record.recordSha256 = crypto.createHash('sha256').update(JSON.stringify(record)).digest('hex');
  return record;
}

module.exports = Object.freeze({ sealFailureTrajectory });

if (require.main === module) {
  if (!process.argv[2]) throw new Error('OBSERVED_AT_REQUIRED');
  process.stdout.write(`${JSON.stringify(sealFailureTrajectory(process.argv[2]), null, 2)}\n`);
}
