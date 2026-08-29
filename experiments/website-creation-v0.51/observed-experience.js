'use strict';

const crypto = require('crypto');
const probeModule = require('./website-creation-probe.js');

function sealObservedExperience(observedAt) {
  const timestamp = new Date(observedAt);
  if (Number.isNaN(timestamp.getTime())) throw new Error('OBSERVED_AT_INVALID');
  const createdAt = new Date(timestamp.getTime() + 1000);
  const rfc3339 = value => value.toISOString().replace('.000Z', 'Z');
  const probe = probeModule.runWebsiteCreationProbe();
  const record = {
    schema: 'axm.waldo.mirror-ground-record/v0.39',
    id: 'observed-website-creation-v0.51-source-hold',
    dataClass: 'OBSERVED_EXECUTION_TRACE',
    synthetic: false,
    sourceReceiptSha256: probe.receiptSha256,
    observedAt: rfc3339(timestamp),
    createdAt: rfc3339(createdAt),
    rootIds: ['continuity', 'truth', 'wisdom-over-speed'],
    challengeKind: 'WEBSITE_CREATION',
    evidenceSignal: 'INCONCLUSIVE',
    trainingDisposition: 'MEMORY_ONLY',
    messages: [
      { role: 'user', content: probe.brief },
      { role: 'assistant', content: 'HELD_WALDO_SOURCE_CANDIDATE_ABSENT: the Creation Fabric produced a website plan, but no WALDO neural source candidate was available, so no website was written.' }
    ],
    rationale: 'Observed local benchmark execution. Route and planning passed; source, render, verification, repair, and model-quality scoring were blocked before a WALDO neural candidate existed. Retain the failure as experience without rewarding it as a positive target.',
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

module.exports = Object.freeze({ sealObservedExperience });

if (require.main === module) {
  if (!process.argv[2]) throw new Error('OBSERVED_AT_REQUIRED');
  process.stdout.write(`${JSON.stringify(sealObservedExperience(process.argv[2]))}\n`);
}
