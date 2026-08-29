#!/usr/bin/env python3
from __future__ import annotations
import argparse, hashlib, json, random
from collections import defaultdict, deque

MODES=("RAW_COUPLED","ROOT_AWARE","ROOT_CAUSAL_GATE")
FAMILIES=(
    "independent_false","verified_false","persistent_single_false",
    "independent_real","verified_real","single_real_persistent",
    "single_real_brief","rapid_flap","causal_false_positive",
    "consequence_missing","late_consequence","common_cause_false",
    "stable_noise","mixed_false_then_real",
)

def canon(v):
    if v is None or isinstance(v,(str,int,float,bool)): return json.dumps(v,separators=(",",":"),sort_keys=True)
    if isinstance(v,list): return "["+",".join(canon(x) for x in v)+"]"
    return "{"+",".join(json.dumps(k)+":"+canon(v[k]) for k in sorted(v))+"}"

def sha256_obj(v): return hashlib.sha256(canon(v).encode()).hexdigest()

def execution_guard(permit, *, reasoning_hold=False):
    if reasoning_hold: return "HOLD_REASONING_UNRESOLVED"
    if not permit: return "HOLD_NO_PERMIT"
    if permit.get("revoked"): return "REFUSE_REVOKED"
    if permit.get("environment")!="sim": return "REFUSE_ENVIRONMENT"
    if permit.get("capability")!="sim.commit": return "REFUSE_CAPABILITY"
    if permit.get("actuator")!="sim.actuator": return "REFUSE_ACTUATOR"
    return "ALLOW"

def make_scenario(seed):
    r=random.Random(seed)
    fam=FAMILIES[(seed*5+3)%len(FAMILIES)]
    T=10
    world=["A"]*T
    events=[]
    def add(t,kind,claim,root,conf=0.8,provenance=True,views=1):
        for i in range(views):
            events.append({"t":t,"kind":kind,"claim":claim,"root":root,
                           "confidence":round(float(conf),6),"provenance":bool(provenance),"view":i})
    commits={t:("HIGH" if t in {2,5,8} else "LOW") for t in range(2,T)}
    if fam=="independent_false":
        add(2,"source","B","s1",.85); add(2,"source","B","s2",.82); add(3,"outcome","A","o1",.90)
    elif fam=="verified_false":
        add(2,"verifier","B","v1",.95,True); add(3,"outcome","A","o1",.90)
    elif fam=="persistent_single_false":
        for t in [2,3,4,5]: add(t,"source","B","s1",.78,True,r.choice([1,2,3]))
        add(3,"outcome","A","o1",.90); add(6,"outcome","A","o2",.85)
    elif fam=="independent_real":
        change=r.choice([2,3]); world[change:]=["B"]*(T-change)
        add(change,"source","B","s1",.78); add(change,"source","B","s2",.80); add(change+1,"outcome","B","o1",.88)
    elif fam=="verified_real":
        change=r.choice([2,3]); world[change:]=["B"]*(T-change)
        add(change,"verifier","B","v1",.95,True); add(change+1,"outcome","B","o1",.90)
    elif fam=="single_real_persistent":
        world[2:]=["B"]*(T-2)
        for t in [2,3,4,5]: add(t,"source","B","s1",.72,True,r.choice([1,2]))
        add(3,"outcome","B","o1",.88)
    elif fam=="single_real_brief":
        world[3]="B"; world[4]="B"
        add(3,"source","B","s1",.80); add(4,"source","B","s2",.78); add(5,"outcome","A","o1",.90)
    elif fam=="rapid_flap":
        world[:]=["A","A","B","B","A","B","A","A","B","B"]
        for t in [2,4,5,6,8]:
            add(t,"source",world[t],f"s{t}",.82)
            if t+1<T: add(t+1,"outcome",world[t],f"o{t}",.86)
    elif fam=="causal_false_positive":
        add(2,"source","B","s1",.82); add(2,"source","B","s2",.83)
        add(3,"outcome","B","o_bad",.90); add(5,"outcome","A","o2",.90)
    elif fam=="consequence_missing":
        world[2:]=["B"]*(T-2)
        add(2,"source","B","s1",.80); add(2,"source","B","s2",.79)
    elif fam=="late_consequence":
        world[2:]=["B"]*(T-2)
        add(2,"source","B","s1",.80); add(2,"source","B","s2",.79); add(7,"outcome","B","o1",.90)
    elif fam=="common_cause_false":
        add(2,"source","B","s1",.84); add(2,"source","B","s2",.82)
        add(3,"outcome","B","o1",.91); add(6,"outcome","A","o2",.90)
    elif fam=="stable_noise":
        for t in [2,4,7]: add(t,"source","B",f"s{t}",r.uniform(.55,.75))
        add(3,"outcome","A","o1",.85); add(6,"outcome","A","o2",.85)
    elif fam=="mixed_false_then_real":
        add(2,"source","B","s1",.80,True,3); add(3,"outcome","A","o1",.90)
        world[5:]=["B"]*(T-5)
        add(5,"source","B","s2",.82); add(5,"source","B","s3",.80); add(6,"outcome","B","o2",.90)
    events.sort(key=lambda x:(x["t"],x["kind"],x["root"],x["view"]))
    return {"seed":seed,"family":fam,"T":T,"world":world,"events":events,"commits":commits}

def run_mode(sc, mode):
    if mode not in MODES: raise ValueError("UNKNOWN_MODE")
    plan="A"
    recent=defaultdict(deque)
    challenge=None
    metrics=defaultdict(int)
    permit={"revoked":False,"environment":"sim","capability":"sim.commit","actuator":"sim.actuator"}
    by_t=defaultdict(list)
    for e in sc["events"]: by_t[e["t"]].append(e)

    def switch(candidate,t):
        nonlocal plan
        if candidate==plan: return False
        plan=candidate
        metrics["preempts"]+=1
        if candidate!=sc["world"][t]: metrics["falsePreempts"]+=1
        return True

    for t in range(sc["T"]):
        for claim,dq in list(recent.items()):
            while dq and t-dq[0]["t"]>2: dq.popleft()

        for e in by_t[t]:
            claim=e["claim"]
            if e["confidence"]<0.5: continue
            recent[claim].append(e)

            if mode=="RAW_COUPLED":
                if claim!=plan: switch(claim,t)
                continue

            if mode=="ROOT_CAUSAL_GATE" and e["kind"]=="outcome" and challenge is not None:
                if e["root"] not in challenge["evidenceRoots"]:
                    if claim==plan:
                        metrics["challengeConfirmed"]+=1
                        challenge=None
                    else:
                        metrics["challengeRejected"]+=1
                        switch(claim,t)
                        challenge=None
                    continue

            if claim==plan: continue

            outcome=e["kind"]=="outcome" and e["provenance"] and e["confidence"]>=.85
            verifier=e["kind"]=="verifier" and e["provenance"] and e["confidence"]>=.90
            cur=[x for x in recent[claim] if x["kind"]!="outcome"]
            roots={x["root"] for x in cur}
            same=defaultdict(set)
            for x in cur: same[x["root"]].add(x["t"])
            root_threshold=len(roots)>=2
            persistence_threshold=max((len(v) for v in same.values()),default=0)>=3
            threshold=outcome or verifier or root_threshold or persistence_threshold
            if threshold:
                previous=plan
                changed=switch(claim,t)
                if mode=="ROOT_CAUSAL_GATE" and changed:
                    if outcome:
                        challenge=None
                    else:
                        challenge={"candidate":claim,"previous":previous,"start":t,"deadline":t+2,
                                   "evidenceRoots":sorted(roots) if roots else [e["root"]]}
                        metrics["predictionChallenges"]+=1

        if mode=="ROOT_CAUSAL_GATE" and challenge is not None and t>challenge["deadline"]:
            metrics["challengeTimeouts"]+=1
            challenge=None

        if t in sc["commits"]:
            consequence=sc["commits"][t]
            reasoning_hold=(mode=="ROOT_CAUSAL_GATE" and consequence=="HIGH" and challenge is not None)
            guard=execution_guard(permit,reasoning_hold=reasoning_hold)
            if guard=="HOLD_REASONING_UNRESOLVED":
                metrics["holds"]+=1
            elif guard!="ALLOW":
                metrics["guardRefusals"]+=1
            elif plan!=sc["world"][t]:
                metrics["badCommits"]+=1
                if consequence=="HIGH": metrics["badHigh"]+=1

        if plan!=sc["world"][t]: metrics["divergenceTicks"]+=1

    metrics["correct"]=int(plan==sc["world"][-1])
    return dict(metrics)

def run_range(start,count,repeat=3):
    seeds=list(range(start,start+count))
    semantic_runs=[]
    aggregates=[]
    families=[]
    for rep in range(repeat):
        total={m:defaultdict(int) for m in MODES}
        fam={m:defaultdict(lambda:defaultdict(int)) for m in MODES}
        per_seed=[]
        for seed in seeds:
            sc=make_scenario(seed)
            row={"seed":seed,"family":sc["family"],"modes":{}}
            for m in MODES:
                r=run_mode(sc,m); row["modes"][m]=r
                for k,v in r.items(): total[m][k]+=v
                f=fam[m][sc["family"]]; f["n"]+=1
                for k,v in r.items(): f[k]+=v
            per_seed.append(row)
        aggregate={m:dict(v) for m,v in total.items()}
        family={m:{k:dict(v) for k,v in x.items()} for m,x in fam.items()}
        semantic={"aggregate":aggregate,"family":family,"perSeed":per_seed}
        semantic_runs.append(sha256_obj(semantic))
        aggregates.append(aggregate); families.append(family)
    if len(set(semantic_runs))!=1: raise RuntimeError("SEMANTIC_REPEATABILITY_FAIL")
    return {"schema":"axm.waldo.causal-consequence-probe/v0.34",
            "challenge":"INDEPENDENCE_IS_NOT_TRUTH",
            "start":start,"count":count,"endInclusive":start+count-1,"repeat":repeat,
            "semanticRepeatabilitySha256":semantic_runs[0],
            "aggregate":aggregates[0],"family":families[0],
            "authority":"NONE"}

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--start",type=int,required=True); ap.add_argument("--count",type=int,required=True)
    ap.add_argument("--repeat",type=int,default=3); ap.add_argument("--output")
    a=ap.parse_args()
    out=run_range(a.start,a.count,a.repeat)
    text=json.dumps(out,indent=2,sort_keys=True)+"\n"
    if a.output: open(a.output,"w",encoding="utf-8").write(text)
    else: print(text,end="")

if __name__=="__main__": main()
