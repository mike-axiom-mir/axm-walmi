#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

function fail(error) {
  process.stderr.write(`WALMI_ASSET_HAND_ERROR: ${error && error.stack ? error.stack : error}\n`);
  process.exit(1);
}

function loadRequest(file) {
  const text = file === '-' ? fs.readFileSync(0, 'utf8') : fs.readFileSync(file, 'utf8');
  const request = JSON.parse(text);
  if (!request || request.schema !== 'axm.walmi.asset-hand-request/v0.2') {
    throw new Error('request schema must be axm.walmi.asset-hand-request/v0.2');
  }
  return request;
}

async function main() {
  if (process.argv.length < 3 || process.argv.length > 4) {
    throw new Error('usage: node walmi_asset_hand_runner.js <request.json|-> [response.json]');
  }
  const handsRoot = path.resolve(__dirname, '..', 'body', 'shared', 'asset-hands');
  const Hands = require(path.join(handsRoot, 'asset-hands.js'));
  const request = loadRequest(process.argv[2]);
  const host = request.host || {
    capabilities: ['svg', 'json', 'canvas-2d'],
    permissions: [],
    accepts: [
      Hands.RESULT_SCHEMA, 'image/svg+xml', 'image/png', 'image/apng', 'image/ktx2',
      'image/jpeg', 'image/webp', 'application/json', 'application/pdf',
      'application/dxf', 'application/mtlx+xml', 'application/vnd.opentimelineio+json',
      'text/css', 'text/plain', 'model/obj', 'model/gltf-binary'
    ]
  };
  const action = String(request.action || '').toLowerCase();
  let result;
  switch (action) {
    case 'status':
      result = {
        version: Hands.VERSION,
        hands: Hands.list().length,
        kinds: Hands.KINDS,
        target_canvas_mediums: Hands.TARGET_CANVAS_MEDIUMS,
        network_used: false,
        workshop_required: false
      };
      break;
    case 'list':
      result = Hands.list();
      break;
    case 'diagnose':
      result = Hands.diagnose(request.brief, host);
      break;
    case 'create':
      result = await Hands.createAsync(request.brief, Object.assign({}, request.options || {}, { host }));
      break;
    case 'create-family':
      result = await Hands.createFamilyAsync(request.brief, Object.assign({ maxHands: 8 }, request.options || {}, { host }));
      break;
    default:
      throw new Error(`unsupported action ${JSON.stringify(request.action)}`);
  }
  const response = {
    schema: 'axm.walmi.asset-hand-response/v0.2',
    state: 'READY',
    action,
    asset_hands_version: Hands.VERSION,
    authority: 'CANDIDATE_ONLY',
    network_used: false,
    workshop_required: false,
    result
  };
  const payload = `${JSON.stringify(response, null, 2)}\n`;
  if (process.argv[3]) fs.writeFileSync(process.argv[3], payload, 'utf8');
  else process.stdout.write(payload);
}

main().catch(fail);
