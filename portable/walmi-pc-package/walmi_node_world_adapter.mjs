#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { pathToFileURL } from "node:url";

const REQUEST_SCHEMA = "axm.walmi.simulator-adapter-request/v1";
const RESPONSE_SCHEMA = "axm.walmi.simulator-adapter-response/v1";
const MAX_INPUT_BYTES = 256 * 1024;

function fail(message) {
  throw new Error(message);
}

function sha256(text) {
  return crypto.createHash("sha256").update(text, "utf8").digest("hex");
}

function atomicWrite(filePath, text, exclusive = false) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  if (exclusive) {
    fs.writeFileSync(filePath, text, { encoding: "utf8", flag: "wx" });
    return;
  }
  const temporary = `${filePath}.${process.pid}.${crypto.randomUUID()}.tmp`;
  fs.writeFileSync(temporary, text, { encoding: "utf8", flag: "wx" });
  fs.renameSync(temporary, filePath);
}

function readRequest() {
  const text = fs.readFileSync(0, "utf8");
  if (Buffer.byteLength(text, "utf8") > MAX_INPUT_BYTES) fail("adapter request is too large");
  const request = JSON.parse(text);
  if (!request || Array.isArray(request) || typeof request !== "object") fail("request must be a JSON object");
  if (request.schema !== REQUEST_SCHEMA) fail(`request schema must be ${REQUEST_SCHEMA}`);
  if (!["create", "observe", "act"].includes(request.command)) fail("command must be create, observe, or act");
  if (!["theme-park", "living-city"].includes(request.world)) fail("world must be theme-park or living-city");
  for (const name of ["worldRoot", "statePath"]) {
    if (typeof request[name] !== "string" || !request[name].trim()) fail(`${name} is required`);
  }
  if (request.command === "act" && (typeof request.actionId !== "string" || !request.actionId.trim())) {
    fail("actionId is required for act");
  }
  return request;
}

function compactAction(id, label, description, nativeAction) {
  return { id, label, description, nativeAction };
}

async function openTheme(request) {
  const root = path.resolve(request.worldRoot);
  const runtimeUrl = pathToFileURL(path.join(root, "runtime", "headless-simulator.js")).href;
  const simulationUrl = pathToFileURL(path.join(root, "playable_3d", "src", "core", "simulation.js")).href;
  const catalogUrl = pathToFileURL(path.join(root, "playable_3d", "src", "core", "catalog.js")).href;
  const [{ HeadlessSimulator }, { canPlace }, { CATALOG, GRID_SIZE }] = await Promise.all([
    import(runtimeUrl),
    import(simulationUrl),
    import(catalogUrl)
  ]);
  const statePath = path.resolve(request.statePath);
  const simulator = request.command === "create"
    ? HeadlessSimulator.create({ seed: String(request.seed || "WALMI-THEME-001"), mode: "sandbox", name: "Walmi Park" })
    : HeadlessSimulator.fromSerialized(fs.readFileSync(statePath, "utf8"));

  function actionMenu() {
    const state = simulator.state;
    const result = [
      compactAction("advance:60", "Run park for 1 hour", "Observe one simulated operating hour.", { type: "advance", minutes: 60 }),
      compactAction("advance:240", "Run park for 4 hours", "Observe four simulated operating hours.", { type: "advance", minutes: 240 }),
      compactAction("park:toggle", state.park.open ? "Close park" : "Open park", "Change gate state.", { type: "togglePark" })
    ];
    const ticket = Number(state.park.ticketPrice || 0);
    for (const value of [...new Set([Math.max(0, ticket - 2), ticket + 2])]) {
      result.push(compactAction(`ticket:${value}`, `Set ticket price to ${value}`, "Change the visible gate price.", { type: "setTicketPrice", value }));
    }
    for (const entity of state.world.entities.slice(0, 12)) {
      const label = CATALOG[entity.catalogId]?.label || entity.catalogId;
      const identity = `${label} (${entity.id})`;
      result.push(compactAction(`maintain:${entity.id}`, `Maintain ${identity}`, "Spend cash to restore condition while preserving history.", { type: "maintain", entityId: entity.id }));
      result.push(compactAction(`toggle:${entity.id}`, `${entity.open ? "Close" : "Open"} ${identity}`, "Change this element's operating state.", { type: "toggleEntity", entityId: entity.id }));
    }
    let offeredBuilds = 0;
    for (const definition of Object.values(CATALOG)) {
      if (offeredBuilds >= 8 || Number(definition.cost) > Number(state.economy.cash)) continue;
      let placement = null;
      for (let z = 0; z < GRID_SIZE && !placement; z += 1) {
        for (let x = 0; x < GRID_SIZE; x += 1) {
          if (canPlace(state, definition.id, x, z, 0).ok) {
            placement = { x, z };
            break;
          }
        }
      }
      if (!placement) continue;
      const nativeAction = { type: "build", catalogId: definition.id, x: placement.x, z: placement.z, rotation: 0 };
      result.push(compactAction(`build:${definition.id}:${placement.x}:${placement.z}`, `Build ${definition.label}`, `Cost ${definition.cost}; placement pre-validated by the world.`, nativeAction));
      offeredBuilds += 1;
    }
    return result.slice(0, 64);
  }

  function observation() {
    const summary = simulator.summary();
    return {
      schema: "axm.walmi.simulator-observation/v1",
      world: "theme-park",
      version: summary.playableVersion,
      stateDigest: summary.stateHash,
      visible: {
        parkName: summary.parkName,
        mode: summary.mode,
        day: summary.day,
        minute: summary.minute,
        cash: summary.cash,
        visitorsPresent: summary.visitorsPresent,
        lifetimeVisitors: summary.lifetimeVisitors,
        entities: summary.entities,
        entityStates: summary.entityStates,
        entityStatesTruncated: summary.entityStatesTruncated,
        paths: summary.paths,
        ticketPrice: simulator.state.park.ticketPrice,
        parkOpen: simulator.state.park.open,
        progression: summary.progression,
        dayReport: summary.dayReport
      },
      actions: actionMenu().map(({ nativeAction: _native, ...publicAction }) => publicAction)
    };
  }

  const before = observation();
  let outcome = { ok: true, reason: request.command === "create" ? "world created" : "world observed" };
  if (request.command === "act") {
    const selected = actionMenu().find((action) => action.id === request.actionId);
    if (!selected) {
      outcome = { ok: false, reason: "actionId is not present in the current visible menu" };
    } else if (selected.nativeAction.type === "advance") {
      simulator.advance(selected.nativeAction.minutes);
      outcome = { ok: true, reason: `advanced ${selected.nativeAction.minutes} minutes` };
    } else {
      outcome = simulator.apply(selected.nativeAction);
    }
  }
  const after = observation();
  if (request.command === "create") atomicWrite(statePath, simulator.serialize(), true);
  if (request.command === "act" && outcome.ok) atomicWrite(statePath, simulator.serialize());
  return { before, after, outcome };
}

function openLiving(request) {
  const root = path.resolve(request.worldRoot);
  const require = createRequire(path.join(root, "runtime", "headless-simulator.js"));
  const { HeadlessSimulator } = require(path.join(root, "runtime", "headless-simulator.js"));
  const statePath = path.resolve(request.statePath);
  const simulator = request.command === "create"
    ? HeadlessSimulator.create({ seed: String(request.seed || "WALMI-LIVING-001") })
    : HeadlessSimulator.fromText(fs.readFileSync(statePath, "utf8"));

  function digest() {
    return sha256(simulator.serialize());
  }

  function actionMenu() {
    const world = simulator.world;
    const actions = [
      compactAction("advance:60", "Observe city for 1 hour", "Advance city time without selecting an activity.", { type: "advance", minutes: 60 })
    ];
    if (world.activeShift) {
      const job = simulator.axm.Content.jobById(world.activeShift.jobId);
      for (const task of job?.actions || []) {
        actions.push(compactAction(
          `work:${task.id}`,
          `Work: ${task.name}`,
          `Live one hour of the active ${job.name} shift.`,
          { type: "workTask", taskId: task.id }
        ));
      }
    } else if (simulator.axm.Systems.currentJob(world)) {
      const job = simulator.axm.Systems.currentJob(world);
      actions.push(compactAction(
        "work:start",
        `Start work at ${job.name}`,
        "Begin a real interactive workday; later choices shape each hour.",
        { type: "workStart" }
      ));
    }

    const activeAdventure = simulator.axm.Community.activeAdventureFor(world, "player");
    if (activeAdventure) {
      const stage = activeAdventure.stages[activeAdventure.stageIndex];
      if (stage && Number(world.player.money) >= Number(stage.cost)) {
        actions.push(compactAction(
          `adventure:continue:${activeAdventure.id}`,
          `Continue ${activeAdventure.title}`,
          `${stage.title}; ${stage.durationHours} hour(s), cost ${stage.cost}.`,
          { type: "adventureContinue", adventureId: activeAdventure.id }
        ));
      }
    } else {
      actions.push(compactAction(
        "adventure:discover",
        "Look for a personal adventure",
        "Discover one optional no-deadline thread grounded in this city.",
        { type: "adventureDiscover" }
      ));
    }

    const opportunities = (world.communityOpportunities || []).filter((entry) =>
      ["open", "available", "awaiting_player"].includes(entry.status)
    ).slice(0, 6);
    for (const opportunity of opportunities) {
      if (opportunity.status === "awaiting_player") {
        actions.push(compactAction(
          `community:accept:${opportunity.id}`,
          `Accept ${opportunity.title}`,
          "Accept the invitation without forcing attendance.",
          { type: "communityRespond", opportunityId: opportunity.id, response: "accept" }
        ));
        actions.push(compactAction(
          `community:decline:${opportunity.id}`,
          `Decline ${opportunity.title}`,
          "Decline without a hidden relationship or progression penalty.",
          { type: "communityRespond", opportunityId: opportunity.id, response: "decline" }
        ));
      } else if (Number(world.player.money) >= Number(opportunity.cost)) {
        actions.push(compactAction(
          `community:join:${opportunity.id}`,
          `Join ${opportunity.title}`,
          `${opportunity.durationHours} hour(s), cost ${opportunity.cost}.`,
          { type: "communityJoin", opportunityId: opportunity.id }
        ));
      }
    }

    if (world.activeTravel) {
      actions.push(compactAction(
        "travel:finish",
        "Finish the current walk",
        "Resolve the remaining real route with its full time and need cost.",
        { type: "travelFinish" }
      ));
    } else if (!world.activeShift && !world.activeEnterpriseSessionId) {
      for (const place of world.places.filter((entry) => entry.id !== world.player.locationId).slice(0, 6)) {
        actions.push(compactAction(
          `travel:${place.id}`,
          `Go to ${place.name}`,
          "Travel a connected city route with its real time and need cost.",
          { type: "travel", placeId: place.id }
        ));
      }
    }

    const activities = simulator.axm.Content.ACTIVITIES.map((activity) => compactAction(
      `activity:${activity.id}`,
      activity.name,
      `${activity.hours} hour(s), cost ${activity.cost}. ${activity.description}`,
      { type: "activity", activityId: activity.id }
    ));
    return [...actions, ...activities].slice(0, 64);
  }

  function observation() {
    const summary = simulator.summary();
    const player = simulator.world.player;
    const activeAdventure = simulator.axm.Community.activeAdventureFor(simulator.world, "player");
    const job = simulator.axm.Systems.currentJob(simulator.world);
    return {
      schema: "axm.walmi.simulator-observation/v1",
      world: "living-city",
      version: summary.version,
      stateDigest: digest(),
      visible: {
        time: summary.time,
        people: summary.people,
        places: summary.places,
        ledgerEntries: summary.ledgerEntries,
        player: {
          money: player.money,
          needs: player.needs,
          skills: player.skills,
          locationId: player.locationId,
          homePropertyId: player.homePropertyId
        },
        work: {
          job: job?.name || null,
          active: Boolean(simulator.world.activeShift),
          remainingHours: simulator.world.activeShift?.remainingHours ?? null
        },
        community: simulator.axm.Community.metrics(simulator.world),
        activeAdventure: activeAdventure ? {
          id: activeAdventure.id,
          title: activeAdventure.title,
          stage: activeAdventure.stageIndex,
          stages: activeAdventure.stages.length
        } : null,
        recentLedger: simulator.world.ledger.slice(-3).map((entry) => entry.message)
      },
      actions: actionMenu().map(({ nativeAction: _native, ...publicAction }) => publicAction)
    };
  }

  const before = observation();
  let outcome = { ok: true, reason: request.command === "create" ? "world created" : "world observed" };
  if (request.command === "act") {
    const selected = actionMenu().find((action) => action.id === request.actionId);
    if (!selected) {
      outcome = { ok: false, reason: "actionId is not present in the current visible menu" };
    } else if (selected.nativeAction.type === "advance") {
      simulator.advanceMinutes(selected.nativeAction.minutes);
      outcome = { ok: true, reason: `advanced ${selected.nativeAction.minutes} minutes` };
    } else if (selected.nativeAction.type === "workStart") {
      outcome = simulator.axm.Systems.startInteractiveShift(simulator.world);
    } else if (selected.nativeAction.type === "workTask") {
      outcome = simulator.axm.Systems.performWorkTask(simulator.world, selected.nativeAction.taskId);
    } else if (selected.nativeAction.type === "adventureDiscover") {
      outcome = simulator.axm.Community.discoverAdventure(simulator.world);
    } else if (selected.nativeAction.type === "adventureContinue") {
      outcome = simulator.axm.Community.continueAdventure(simulator.world, selected.nativeAction.adventureId);
    } else if (selected.nativeAction.type === "communityRespond") {
      outcome = simulator.axm.Community.respondToCommunityOpportunity(
        simulator.world, selected.nativeAction.opportunityId, selected.nativeAction.response
      );
    } else if (selected.nativeAction.type === "communityJoin") {
      outcome = simulator.axm.Community.participateOpportunity(simulator.world, selected.nativeAction.opportunityId);
    } else if (selected.nativeAction.type === "travel") {
      outcome = simulator.axm.Systems.startPlayerTravel(simulator.world, selected.nativeAction.placeId, "compressed");
    } else if (selected.nativeAction.type === "travelFinish") {
      outcome = simulator.axm.Systems.finishPlayerTravelCompressed(simulator.world);
    } else {
      outcome = simulator.performActivity(selected.nativeAction.activityId);
    }
    if (outcome.ok) {
      outcome = {
        ...outcome,
        action: outcome.action || selected.label,
        reason: outcome.reason || `${selected.label} completed.`
      };
    }
    if (outcome.ok) simulator.validate();
  }
  const after = observation();
  if (request.command === "create") atomicWrite(statePath, simulator.serialize(), true);
  if (request.command === "act" && outcome.ok) atomicWrite(statePath, simulator.serialize());
  return { before, after, outcome };
}

try {
  const request = readRequest();
  const result = request.world === "theme-park" ? await openTheme(request) : openLiving(request);
  process.stdout.write(`${JSON.stringify({ schema: RESPONSE_SCHEMA, world: request.world, command: request.command, ...result })}\n`);
} catch (error) {
  process.stderr.write(`WALMI simulator adapter refused: ${error.message}\n`);
  process.exitCode = 2;
}
