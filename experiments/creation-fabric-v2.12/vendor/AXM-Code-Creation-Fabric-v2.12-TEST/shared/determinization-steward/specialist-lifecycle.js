'use strict';
const crypto=require('node:crypto');
const VERSION='0.53.0';
function canon(v){if(v===null||typeof v!=='object')return JSON.stringify(v);if(Array.isArray(v))return'['+v.map(canon).join(',')+']';return'{'+Object.keys(v).sort().map(k=>JSON.stringify(k)+':'+canon(v[k])).join(',')+'}'}
function sha(v){return crypto.createHash('sha256').update(canon(v)).digest('hex')}
function clone(v){return JSON.parse(JSON.stringify(v))}
function controller(v){const x=String(v||'').toUpperCase();if(!['MIRROR','WALDO'].includes(x))throw Error('ONLY_MIRROR_OR_WALDO_MAY_PIN');return x}
function activeSeat(s){return s&&s.state==='ACTIVE'&&s.owner&&['MIRROR','WALDO'].includes(s.owner)}
function pin(seat,byController,reason='CONTROLLER_PIN'){
  if(!activeSeat(seat))throw Error('ACTIVE_MIRROR_OR_WALDO_SEAT_REQUIRED');const by=controller(byController);if(by!==seat.owner)throw Error('CONTROLLER_MAY_ONLY_PIN_OWN_SEAT');
  const next=clone(seat);next.pin={locked:true,byController:by,reason:String(reason).slice(0,500)};delete next.seatDigest;next.seatDigest=sha(next);return Object.freeze(next)
}
function complete(seat,outcomeRef='TASK_COMPLETE'){
  if(!activeSeat(seat))throw Error('ACTIVE_MIRROR_OR_WALDO_SEAT_REQUIRED');
  if(seat.pin&&seat.pin.locked===true){const next=clone(seat);next.state='IDLE_LOCKED';next.lastOutcomeRef=String(outcomeRef).slice(0,500);delete next.seatDigest;next.seatDigest=sha(next);return Object.freeze({state:'IDLE_LOCKED',seat:Object.freeze(next),receipt:Object.freeze({schema:'axm.specialist-idle-lock-receipt/v0.53',seatId:seat.id,owner:seat.owner,packageFingerprint:seat.packageFingerprint,compiledPackageBytesRetained:seat.compiledPackageBytes,rawPackageRetained:true,memoryPromotion:'NONE',authority:'NONE'})})}
  return revoke(seat,'TASK_COMPLETE_UNPINNED:'+String(outcomeRef).slice(0,300));
}
function resume(seat,byController,taskRef){if(!seat||seat.state!=='IDLE_LOCKED'||!seat.pin||seat.pin.locked!==true)throw Error('IDLE_LOCKED_SEAT_REQUIRED');const by=controller(byController);if(by!==seat.owner)throw Error('CONTROLLER_MAY_ONLY_RESUME_OWN_SEAT');const next=clone(seat);next.state='ACTIVE';next.resumeTaskRef=String(taskRef||'next-task').slice(0,500);delete next.seatDigest;next.seatDigest=sha(next);return Object.freeze(next)}
function unpin(seat,byController,reason='CONTROLLER_UNPIN'){if(!seat||!['ACTIVE','IDLE_LOCKED'].includes(seat.state)||!seat.pin||seat.pin.locked!==true)throw Error('PINNED_SEAT_REQUIRED');const by=controller(byController);if(by!==seat.owner)throw Error('CONTROLLER_MAY_ONLY_UNPIN_OWN_SEAT');return revoke(seat,reason)}
function revoke(seat,reason='TASK_OR_LEASE_COMPLETE'){if(!seat||!['ACTIVE','IDLE_LOCKED'].includes(seat.state))throw Error('RESIDENT_SEAT_REQUIRED');const receipt={schema:'axm.ephemeral-specialist-revocation/v0.53',seatId:seat.id,owner:seat.owner,specialistId:seat.specialistId,profileId:seat.profileId,packageFingerprint:seat.packageFingerprint,compiledPackageBytesReleased:seat.compiledPackageBytes,reason:String(reason).slice(0,500),rawSpecialistPackageRetained:false,memoryPromotion:'NONE',authority:'NONE'};receipt.revocationDigest=sha(receipt);return Object.freeze({state:'REVOKED',seat:null,receipt:Object.freeze(receipt)})}
function residentCount(seats){return(Array.isArray(seats)?seats:[]).filter(s=>s&&['ACTIVE','IDLE_LOCKED'].includes(s.state)).length}
function canAllocate(seats,cap=5){return residentCount(seats)<cap}
function reusable(seats,profileId){return(Array.isArray(seats)?seats:[]).find(s=>s&&['ACTIVE','IDLE_LOCKED'].includes(s.state)&&s.profileId===profileId)||null}
module.exports=Object.freeze({VERSION,pin,complete,resume,unpin,revoke,residentCount,canAllocate,reusable,sha});
