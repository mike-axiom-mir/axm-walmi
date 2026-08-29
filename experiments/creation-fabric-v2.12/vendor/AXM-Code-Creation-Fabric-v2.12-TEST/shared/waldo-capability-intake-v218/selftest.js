'use strict';

const assert = require('node:assert/strict');
const Intake = require('./waldo-capability-intake-v218.js');

let pass = 0;
function test(name, fn) { fn(); pass += 1; console.log('PASS', name); }

test('PR67 and PR68 builders are active in WALDO body', () => {
  const inventory = Intake.inventory();
  assert.equal(inventory.allActive, true);
  assert.deepEqual(inventory.entries.map((entry) => entry.id).sort(), ['bounded-record-query-v1','closed-object-contract-adapter-v2']);
  assert.equal(inventory.specialistBuilderVersion, '2.5.0');
});

test('strict object-contract adapter builds through exact Code Specialist lane', () => {
  const built = Intake.buildDetached('closed-object-contract-adapter-v2');
  assert.equal(built.result.status, 'COMPLETE_DETACHED_CANDIDATE');
  assert.equal(built.verification.pass, true);
  assert.equal(built.result.recipeRef.id, 'closed-object-contract-adapter');
  assert.equal(built.candidateExecuted, false);
  assert.equal(built.generatedSelftestExecuted, false);
});

test('bounded record query builds through exact Code Specialist lane', () => {
  const built = Intake.buildDetached('bounded-record-query-v1');
  assert.equal(built.result.status, 'COMPLETE_DETACHED_CANDIDATE');
  assert.equal(built.verification.pass, true);
  assert.equal(built.result.recipeRef.id, 'bounded-record-query');
  assert.equal(built.candidateExecuted, false);
  assert.equal(built.generatedSelftestExecuted, false);
});

test('new deterministic body capabilities retain no lifecycle authority', () => {
  for (const id of Intake.BUILDER_IDS) {
    const built = Intake.buildDetached(id);
    assert.equal(built.installed, false);
    assert.equal(built.integrated, false);
    assert.equal(built.published, false);
    assert.equal(built.promoted, false);
    assert.equal(built.canonChanged, false);
    assert.equal(built.authority, 'NONE');
  }
});

test('existing WALDO Workshop integration sees both new builders', () => {
  const snap = Intake.snapshot();
  assert(snap.workshopCreationIds.includes('closed-object-contract-adapter-v2'));
  assert(snap.workshopCreationIds.includes('bounded-record-query-v1'));
  assert.equal(snap.runtimeDependencyOnAXMRepository, false);
});

console.log('RESULT', pass + '/5 PASS');
