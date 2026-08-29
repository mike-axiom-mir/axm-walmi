#!/usr/bin/env python3
"""AXM WALDO v0.33 — specialist corroboration independence probe.

Question: when one caller observation fans out across many grammar eyes,
discipline lenses, template routes, and machine heuristics, does "many agreeing
views" get mistaken for independent evidence?

The root-aware membrane keeps coupled reasoning as default. It collapses
derived views by provenanceRoot before corroboration. A bounded persistence
fallback may resolve one-root uncertainty after 3 observations. Authority is
NONE throughout; execution remains a separate simulated guard.
"""
from __future__ import annotations
from dataclasses import dataclass
import random, json, hashlib

MODES=("RAW_COUPLED","VIEW_COUNT_CORROBORATION","ROOT_AWARE_MEMBRANE")
FAMILIES=("fanout-false","persistent-fanout-false","independent-real","single-real-long",
          "single-real-brief","alias-false","independent-false","mixed-false-then-real",
          "rapid-flap","stable-noise","verified-real","verified-false")

def report(root, support, strength="MODERATE", views=1, *, provenance=None, verified=False, obs=None):
    return {"rootId":root,"provenanceRoot":provenance or root,"observationId":obs or root,
            "support":support,"strength":strength,"derivedViews":views,
            "verifiedNative":bool(verified),"authority":"NONE"}

def scenario(seed:int):
    r=random.Random(seed); fam=FAMILIES[seed%len(FAMILIES)]
    ticks=[]
    def add(t, world, reps, consequence=None):
        ticks.append({"tick":t,"world":world,"reports":reps,"consequence":consequence or ("HIGH" if t in (3,6) else "LOW")})
    if fam=="fanout-false":
        for t in range(1,7):
            reps=[report("false-A","NEW","STRONG",r.randint(4,12),obs=f"fa-{t}")] if t in (2,3) else []
            add(t,"CURRENT",reps)
    elif fam=="persistent-fanout-false":
        for t in range(1,7):
            reps=[report("false-A","NEW","STRONG",r.randint(4,12),obs=f"pfa-{t}")] if t>=2 else []
            add(t,"CURRENT",reps)
    elif fam=="independent-real":
        for t in range(1,7):
            world="NEW" if t>=2 else "CURRENT"
            reps=[] if t<2 else [report("sensor-A","NEW","MODERATE",r.randint(1,4),obs=f"ira-{t}"),
                                  report("sensor-B","NEW","MODERATE",r.randint(1,4),obs=f"irb-{t}")]
            add(t,world,reps)
    elif fam=="single-real-long":
        for t in range(1,7):
            world="NEW" if t>=2 else "CURRENT"
            reps=[] if t<2 else [report("sensor-A","NEW","STRONG",r.randint(2,8),obs=f"srl-{t}")]
            add(t,world,reps)
    elif fam=="single-real-brief":
        for t in range(1,7):
            world="NEW" if t>=5 else "CURRENT"
            reps=[] if t<5 else [report("sensor-A","NEW","STRONG",r.randint(3,9),obs=f"srb-{t}")]
            add(t,world,reps)
    elif fam=="alias-false":
        for t in range(1,7):
            reps=[]
            if t in (2,3):
                reps=[report("eye-rust","NEW","STRONG",3,provenance="caller-obs-X",obs=f"alias-r-{t}"),
                      report("discipline-security","NEW","STRONG",2,provenance="caller-obs-X",obs=f"alias-s-{t}"),
                      report("template-rust","NEW","MODERATE",2,provenance="caller-obs-X",obs=f"alias-t-{t}")]
            add(t,"CURRENT",reps)
    elif fam=="independent-false":
        for t in range(1,7):
            reps=[] if t<2 else [report("false-A","NEW","MODERATE",2,obs=f"ifa-{t}"),
                                  report("false-B","NEW","MODERATE",2,obs=f"ifb-{t}")]
            add(t,"CURRENT",reps)
    elif fam=="mixed-false-then-real":
        for t in range(1,7):
            world="NEW" if t>=4 else "CURRENT"
            if t==2: reps=[report("false-A","NEW","STRONG",8,obs="mix-false")]
            elif t>=4: reps=[report("sensor-A","NEW","MODERATE",2,obs=f"mix-a-{t}"),
                              report("sensor-B","NEW","MODERATE",2,obs=f"mix-b-{t}")]
            else: reps=[]
            add(t,world,reps)
    elif fam=="rapid-flap":
        worlds={1:"CURRENT",2:"NEW",3:"NEW",4:"CURRENT",5:"CURRENT",6:"NEW"}
        for t in range(1,7):
            w=worlds[t]
            reps=[report("sensor-A",w,"STRONG",r.randint(2,7),obs=f"flap-{t}")]
            add(t,w,reps)
    elif fam=="stable-noise":
        for t in range(1,7):
            reps=[report("noise-A","NEW","WEAK",r.randint(1,5),obs=f"noise-{t}")] if t in (2,4,6) else []
            add(t,"CURRENT",reps)
    elif fam=="verified-real":
        for t in range(1,7):
            world="NEW" if t>=2 else "CURRENT"
            reps=[] if t<2 else [report("native-verifier","NEW","STRONG",1,verified=True,obs=f"vr-{t}")]
            add(t,world,reps)
    elif fam=="verified-false":
        for t in range(1,7):
            reps=[report("native-verifier","NEW","STRONG",1,verified=True,obs=f"vf-{t}")] if t==2 else []
            add(t,"CURRENT",reps)
    return {"seed":seed,"family":fam,"ticks":ticks}

def _candidate_counts(reports):
    counts={}
    for x in reports:
        if x["strength"]=="WEAK": continue
        counts[x["support"]]=counts.get(x["support"],0)+int(x.get("derivedViews",1))
    return counts

def _root_counts(reports):
    roots={}
    for x in reports:
        if x["strength"]=="WEAK": continue
        roots.setdefault(x["support"],set()).add(x["provenanceRoot"])
    return {k:len(v) for k,v in roots.items()}

def run_one(s,mode):
    plan="CURRENT"; false_preempts=0; divergence=0; bad=0; bad_high=0; holds=0; fallback=0
    persistence={}  # (support, provenanceRoot) -> consecutive observation count
    prev_seen=set()
    ledger=[]
    for tick in s["ticks"]:
        world=tick["world"]; reps=tick["reports"]; consequence=tick["consequence"]
        for x in reps:
            if x.get("authority")!="NONE": raise AssertionError("AUTHORITY_WIDENED")
        target=None; unresolved=False
        if mode=="RAW_COUPLED":
            candidates=[x for x in reps if x["strength"]!="WEAK" and x["support"]!=plan]
            if candidates:
                candidates.sort(key=lambda x:({"MODERATE":1,"STRONG":2}.get(x["strength"],0),x["derivedViews"]),reverse=True)
                target=candidates[0]["support"]
        elif mode=="VIEW_COUNT_CORROBORATION":
            counts=_candidate_counts(reps)
            if counts:
                best=max(counts,key=lambda k:(counts[k],k))
                if counts[best]>=2 and best!=plan: target=best
        elif mode=="ROOT_AWARE_MEMBRANE":
            # A verified-native observation can excite coupling immediately, but still is not truth.
            verified=[x for x in reps if x["verifiedNative"] and x["strength"]!="WEAK" and x["support"]!=plan]
            if verified:
                target=verified[0]["support"]
            else:
                rc=_root_counts(reps)
                eligible=[(n,k) for k,n in rc.items() if n>=2 and k!=plan]
                if eligible:
                    eligible.sort(reverse=True); target=eligible[0][1]
                else:
                    # One-root candidate remains uncertainty. View fan-out does not raise independence.
                    different=[x for x in reps if x["strength"]!="WEAK" and x["support"]!=plan]
                    if different:
                        unresolved=True
                        current={(x["support"],x["provenanceRoot"]) for x in different}
                        for key in current:
                            persistence[key]=persistence.get(key,0)+1 if key in prev_seen else 1
                        prev_seen=current
                        best=max(current,key=lambda k:persistence[k])
                        if persistence[best]>=3:
                            fallback+=1; target=best[0]; unresolved=False
                    else:
                        prev_seen=set()
        else: raise ValueError(mode)

        if target and target!=plan:
            if target!=world: false_preempts+=1
            plan=target
        if plan!=world: divergence+=1
        held=False
        if mode=="ROOT_AWARE_MEMBRANE" and unresolved and consequence=="HIGH":
            holds+=1; held=True
        if not held and plan!=world:
            bad+=1
            if consequence=="HIGH": bad_high+=1
        ledger.append({"tick":tick["tick"],"world":world,"plan":plan,"target":target,
                       "unresolved":unresolved,"held":held,"authority":"NONE"})
    return {"finalCorrect":plan==s["ticks"][-1]["world"],"falsePreempts":false_preempts,
            "divergenceTicks":divergence,"badCommits":bad,"badHigh":bad_high,
            "holds":holds,"fallbackActivations":fallback,"ledger":ledger}

def benchmark(seeds):
    out={m:{"correct":0,"falsePreempts":0,"divergenceTicks":0,"badCommits":0,"badHigh":0,"holds":0,"fallbackActivations":0} for m in MODES}
    fam={}
    for seed in seeds:
        s=scenario(seed); fam.setdefault(s["family"],{m:{"n":0,"correct":0,"falsePreempts":0,"badCommits":0,"divergenceTicks":0} for m in MODES})
        for m in MODES:
            r=run_one(s,m); a=out[m]; a["correct"]+=int(r["finalCorrect"])
            for k in ("falsePreempts","divergenceTicks","badCommits","badHigh","holds","fallbackActivations"): a[k]+=r[k]
            f=fam[s["family"]][m]; f["n"]+=1; f["correct"]+=int(r["finalCorrect"]); f["falsePreempts"]+=r["falsePreempts"]; f["badCommits"]+=r["badCommits"]; f["divergenceTicks"]+=r["divergenceTicks"]
    return {"seedCount":len(seeds),"modes":out,"families":fam}

def semantic_digest(seeds):
    body=[]
    for seed in seeds:
        s=scenario(seed)
        body.append({"seed":seed,"family":s["family"],"runs":{m:run_one(s,m) for m in MODES}})
    raw=json.dumps(body,sort_keys=True,separators=(",",":")).encode()
    return hashlib.sha256(raw).hexdigest()

if __name__=="__main__":
    import argparse
    p=argparse.ArgumentParser(); p.add_argument("--start",type=int,required=True); p.add_argument("--count",type=int,required=True)
    a=p.parse_args(); seeds=list(range(a.start,a.start+a.count))
    print(json.dumps({"benchmark":benchmark(seeds),"semanticSha256":semantic_digest(seeds)},indent=2,sort_keys=True))
