'use strict';

const assert = require('node:assert/strict');
const Service = require('./service-mode-composer.js');

let passed = 0;
function test(name, fn) {
  fn();
  passed += 1;
  process.stdout.write('PASS ' + name + '\n');
}
function request(overrides = {}) {
  return {
    schema: Service.REQUEST_SCHEMA,
    subject: 'Build a website with repository delivery.',
    modeId: 'software-creation',
    hostApps: [
      { id:'drive', label:'Drive-like files', connectionState:'CONNECTED', capabilities:['file.read','file.write'] },
      { id:'github', label:'Repository app', connectionState:'CONNECTED', capabilities:['repository.read','repository.write','issue.read'] },
      { id:'search', label:'Search app', connectionState:'CONNECTED', capabilities:['web.search'] },
      { id:'deploy', label:'Deployment app', connectionState:'AVAILABLE', capabilities:['deployment.write'] }
    ],
    consequence: 'LOW',
    uncertainty: 'MEDIUM',
    estimatedSteps: 6,
    artifactCount: 3,
    cycle: 0,
    consent: { hermesReasoning:false, discovery:false },
    runtime: { hermesReasoningAvailable:false },
    resourceObservation: { note:'synthetic service-mode selftest' },
    allowNeural: false,
    neuralSockets: [
      { id:'neural-a', state:'CONNECTED' },
      { id:'neural-b', state:'DISCONNECTED' }
    ],
    ...overrides
  };
}

test('snapshot exposes nine service modes and no authority', () => {
  const snapshot = Service.snapshot();
  assert.equal(snapshot.modeCount, 9);
  assert.equal(snapshot.authority, 'NONE');
  assert.equal(snapshot.automaticAppActivation, false);
});

test('explicit software mode is selected without heuristic ambiguity', () => {
  const plan = Service.planService(request());
  assert.equal(plan.classification.state, 'SERVICE_MODE_SELECTED_EXPLICITLY');
  assert.equal(plan.mode.id, 'software-creation');
});

test('research subject is heuristically classified', () => {
  const classified = Service.classifyService('Research primary sources and compare the evidence.');
  assert.equal(classified.selectedMode.id, 'research-analysis');
  assert.equal(classified.heuristic, true);
});

test('unknown subject falls back to general service', () => {
  const classified = Service.classifyService('Do the bounded thing with the supplied packet.');
  assert.equal(classified.state, 'SERVICE_MODE_SELECTED_GENERAL_FALLBACK');
  assert.equal(classified.selectedMode.id, 'general-service');
});

test('connected relevant apps are discovered', () => {
  const plan = Service.planService(request());
  const ids = plan.appPlan.relevantConnectedApps.map(app => app.id);
  assert(ids.includes('drive'));
  assert(ids.includes('github'));
  assert(ids.includes('search'));
});

test('AVAILABLE app is not treated as connected', () => {
  const plan = Service.planService(request());
  assert(!plan.appPlan.relevantConnectedApps.some(app => app.id === 'deploy'));
});

test('required software output capability is satisfied by connected apps', () => {
  const plan = Service.planService(request());
  assert.equal(plan.appPlan.missingRequiredCapabilityGroups.length, 0);
  assert(plan.appPlan.primaryAppIds.length >= 1);
});

test('missing required app capability is explicit but does not block specialist composition', () => {
  const plan = Service.planService(request({
    modeId:'communication-outreach',
    subject:'Send an outreach email.',
    hostApps:[{ id:'contacts', label:'Contacts', connectionState:'CONNECTED', capabilities:['contacts.read'] }]
  }));
  assert.equal(plan.state, 'DEGRADED_MISSING_APP_CAPABILITIES');
  assert.equal(plan.appPlan.missingRequiredCapabilityGroups.length, 1);
  assert(plan.specialistPlan.team);
});

test('connection never becomes permission or activation authority', () => {
  const plan = Service.planService(request());
  assert.equal(plan.appPlan.activationAuthority, false);
  assert.equal(plan.appPlan.permissionGrant, false);
  assert.equal(plan.truth.noAppWasActivated, true);
});

test('service profile views resolve into real specialist masks', () => {
  const plan = Service.planService(request());
  assert.equal(plan.specialistPlan.specialistProfileGaps.length, 0);
  assert.equal(plan.specialistPlan.resolvedMaskIds.MIRROR.length, 4);
  assert.equal(plan.specialistPlan.resolvedMaskIds.WALDO.length, 4);
});

test('service mode uses existing Mirror and WALDO pools', () => {
  const plan = Service.planService(request());
  assert.equal(plan.specialistPlan.team.pools.MIRROR.seats.length, 4);
  assert.equal(plan.specialistPlan.team.pools.WALDO.seats.length, 4);
  assert(plan.specialistPlan.team.pools.MIRROR.seats.every(seat => seat.owner === 'MIRROR'));
  assert(plan.specialistPlan.team.pools.WALDO.seats.every(seat => seat.owner === 'WALDO'));
});

test('service specialists remain authority-free ephemeral packages', () => {
  const plan = Service.planService(request());
  const seats = [...plan.specialistPlan.team.pools.MIRROR.seats, ...plan.specialistPlan.team.pools.WALDO.seats];
  assert(seats.every(seat => seat.authority === 'NONE'));
  assert(seats.every(seat => seat.innerSettings.memoryPolicy === 'EPHEMERAL'));
});

test('Hermes remains trigger-only in service mode', () => {
  const plan = Service.planService(request({
    consent:{hermesReasoning:true, discovery:true},
    runtime:{hermesReasoningAvailable:true}
  }));
  assert.equal(plan.hermesPolicy.mode, 'TRIGGER_ONLY');
  assert.equal(plan.specialistPlan.team.hermes.activeRequests.length, 0);
});

test('neural sockets are optional and no call is made by default', () => {
  const plan = Service.planService(request());
  assert.deepEqual(plan.neuralPlan.suggestedSocketIds, []);
  assert.equal(plan.neuralPlan.automaticCall, false);
});

test('caller may make connected neural socket eligible without making a call', () => {
  const plan = Service.planService(request({allowNeural:true}));
  assert.deepEqual(plan.neuralPlan.suggestedSocketIds, ['neural-a']);
  assert.equal(plan.neuralPlan.automaticCall, false);
  assert.equal(plan.truth.noNeuralCallWasMade, true);
});

test('travel mode selects travel-relevant connected apps only', () => {
  const plan = Service.planService(request({
    modeId:'travel-planning',
    subject:'Plan a trip and hotel itinerary.',
    hostApps:[
      {id:'search',label:'Search',connectionState:'CONNECTED',capabilities:['web.search']},
      {id:'calendar',label:'Calendar',connectionState:'CONNECTED',capabilities:['calendar.read','calendar.write']},
      {id:'repo',label:'Repository',connectionState:'CONNECTED',capabilities:['repository.read']}
    ]
  }));
  const ids = plan.appPlan.relevantConnectedApps.map(app => app.id);
  assert(ids.includes('search'));
  assert(ids.includes('calendar'));
  assert(!ids.includes('repo'));
});

test('data analysis mode composes numerical perspective', () => {
  const plan = Service.planService(request({
    modeId:'data-analysis',
    subject:'Analyze this spreadsheet forecast.',
    hostApps:[{id:'sheet',label:'Sheet',connectionState:'CONNECTED',capabilities:['spreadsheet.read','spreadsheet.write','compute']}]
  }));
  assert(plan.specialistPlan.requestedProfiles.WALDO.includes('numerical'));
});

test('host app descriptors reject hidden extra fields', () => {
  assert.throws(() => Service.planService(request({
    hostApps:[{id:'bad',label:'Bad',connectionState:'CONNECTED',capabilities:['file.read'],secret:'no'}]
  })), /HOST_APP_0_FIELDS_INVALID/);
});

test('ambiguous heuristic subject holds instead of silently selecting', () => {
  const result = Service.classifyService('Email calendar');
  assert.equal(result.state, 'SERVICE_MODE_SELECTION_REQUIRED');
  assert.equal(result.selectedMode, null);
});

test('held service selection composes no apps or specialists', () => {
  const plan = Service.planService(request({
    modeId:null,
    subject:'Email calendar'
  }));
  assert.equal(plan.state, 'HELD_SERVICE_SELECTION_REQUIRED');
  assert.equal(plan.appPlan, null);
  assert.equal(plan.specialistPlan, null);
});

test('same request produces deterministic plan digest', () => {
  const first = Service.planService(request());
  const second = Service.planService(request());
  assert.equal(first.planDigest, second.planDigest);
});

test('service mode grants no authority', () => {
  const plan = Service.planService(request());
  assert.equal(plan.authority, 'NONE');
  assert.equal(plan.specialistPlan.team.authority, 'NONE');
});

process.stdout.write('RESULT ' + passed + '/22 PASS\n');
