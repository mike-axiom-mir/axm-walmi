'use strict';

const Builders = require('../capability-fabric/builder-registry.js');
const Specialist = require('../code-capability-fabric/code-specialist-capability-builder-v1.js');
const Workshop = require('../waldo-workshop-integration/waldo-workshop-integration.js');

const VERSION = '0.1.0';
const SCHEMA = 'axm.waldo.capability-intake-v218/v1';
const BUILDER_IDS = Object.freeze([
  'closed-object-contract-adapter-v2',
  'bounded-record-query-v1'
]);

function inventory() {
  const entries = BUILDER_IDS.map((id) => {
    const descriptor = Builders.describe(id);
    return Object.freeze({
      id,
      active: !!descriptor && descriptor.status === Builders.ACTIVE,
      descriptor,
      authority: 'NONE'
    });
  });
  return Object.freeze({
    schema: 'axm.waldo.capability-intake-v218-inventory/v1',
    entries,
    allActive: entries.every((entry) => entry.active),
    specialistBuilderVersion: Specialist.VERSION,
    authority: 'NONE'
  });
}

function requestFor(id) {
  if (id === 'closed-object-contract-adapter-v2') return Specialist.buildContractAdapterExampleRequest();
  if (id === 'bounded-record-query-v1') return Specialist.buildRecordQueryExampleRequest();
  throw new Error('CAPABILITY_NOT_IN_V218_INTAKE');
}

function buildDetached(id) {
  if (!BUILDER_IDS.includes(id)) throw new Error('CAPABILITY_NOT_IN_V218_INTAKE');
  const descriptor = Builders.describe(id);
  if (!descriptor || descriptor.status !== Builders.ACTIVE) throw new Error('CAPABILITY_NOT_ACTIVE');
  const request = requestFor(id);
  const result = Specialist.generate(request);
  const verification = Specialist.verify(result, request);
  if (!verification.pass) throw new Error('SPECIALIST_RESULT_REBUILD_FAILED');
  return Object.freeze({
    schema: 'axm.waldo.capability-intake-v218-build/v1',
    id,
    descriptor,
    requestDigest: request.requestDigest,
    result,
    verification,
    candidateExecuted: result.resourceObservation.candidateExecuted,
    generatedSelftestExecuted: result.resourceObservation.generatedSelftestExecuted,
    installed: result.truth.installed,
    integrated: result.truth.integrated,
    published: result.truth.published,
    promoted: result.truth.promoted,
    canonChanged: result.truth.canonChanged,
    authority: 'NONE'
  });
}

function snapshot() {
  const workshop = Workshop.snapshot();
  return Object.freeze({
    schema: SCHEMA,
    version: VERSION,
    inventory: inventory(),
    workshopCreationIds: workshop.creation.entries.map((entry) => entry.id).sort(),
    sourcePlatformCommit: 'c0d57d4ea0403b5d0932e4abf177d207bc87791b',
    sourcePlatformPrStack: [67, 68],
    runtimeDependencyOnAXMRepository: false,
    automaticExecution: false,
    automaticInstallation: false,
    automaticIntegration: false,
    authority: 'NONE'
  });
}

module.exports = Object.freeze({ VERSION, SCHEMA, BUILDER_IDS, inventory, buildDetached, snapshot });
