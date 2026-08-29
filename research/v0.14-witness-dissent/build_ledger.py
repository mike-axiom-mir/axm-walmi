from __future__ import annotations
import hashlib, json
from pathlib import Path
R=Path(__file__).resolve().parent

def canonical(obj): return json.dumps(obj,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
def dh(obj): return 'sha256:'+hashlib.sha256(canonical(obj)).hexdigest()
def load(n): return json.loads((R/n).read_text())
def load_ring(): return json.loads((R/'../../internal/axmmirror/testdata/peer-audit-ring-v0.13.json').resolve().read_text())
def entry(seq,phase,w,supersedes,prev):
    e={'sequence':seq,'phase':phase,'witnessId':w['witnessId'],'implementation':w['implementation'],'witnessReceiptDigest':w['receiptDigest'],'evidenceViewDigest':w['evidenceViewDigest'],'verdict':w['verdict'],'findings':w['findings'],'authority':'NONE','previousEntryDigest':prev,'supersedesEntryDigest':supersedes}
    e['entryDigest']=dh(e); return e
ring=load_ring(); a=load('witness-a-t1.json'); b1=load('witness-b-t1.json'); b2=load('witness-b-t2.json')
peer_holds=[x for x in ring['peerRing']['audits'] if x['verdict']=='HOLD']
e1=entry(1,'T1',a,'','GENESIS'); e2=entry(2,'T1',b1,'',e1['entryDigest']); e3=entry(3,'T2',b2,e2['entryDigest'],e2['entryDigest'])
receipt={'schema':'axm.waldo-mirror.v0.14-witness-dissent-ledger/v0.1','status':'OBSERVED_APPEND_ONLY_DISSENT','source':{'v013ReceiptDigest':ring['receiptDigest'],'v013WitnessDigest':'1113f359f149214e48c20902a73aaffca612b0a54d35c79e8499806b91845739','peerAuditPackDigest':ring['peerAuditPack']['digest'],'peerPasses':ring['peerRing']['evidencePasses'],'peerHolds':ring['peerRing']['holds'],'independentWitnessSuppliedAtPeerVote':ring['peerRing']['independentWitnessSupplied'],'guardedPeerHolds':[{k:x[k] for k in ['auditorStrategy','targetAuditorStrategy','verdict','findings','authority']} for x in peer_holds]},'independentWitnesses':{'witnessA':{'receiptDigest':a['receiptDigest'],'implementation':a['implementation'],'verdictAtT1':a['verdict']},'witnessB':{'holdReceiptDigest':b1['receiptDigest'],'passReceiptDigest':b2['receiptDigest'],'implementation':b1['implementation'],'verdictAtT1':b1['verdict'],'verdictAtT2':b2['verdict']}},'appendOnlyLedger':[e1,e2,e3],'resolution':{'t1WitnessDisagreementObserved':True,'witnessBHoldResolvedByAdditionalEvidence':True,'witnessBHistoricalHoldPreserved':True,'peerGuardedHoldsPreserved':True,'peerHistoryRewritten':False,'witnessHistoryRewritten':False,'majorityAuthorityGranted':False,'finalJudgeAssigned':False,'finalVerdict':'NONE','authority':'NONE'},'truth':{'witnessImplementationsIndependent':True,'witnessEvidenceViewsDifferentAtT1':True,'liveAIProviderCalled':False,'networkRequestedByHarness':False,'targetCodeMutated':False,'peerReceiptMutated':False,'installationPerformed':False,'registrationPerformed':False,'stagingPerformed':False,'promotionPerformed':False,'canonChanged':False},'authority':'NONE'}
receipt['receiptDigest']=dh(receipt)
(R/'witness-dissent-ledger.full.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(receipt['receiptDigest'])
