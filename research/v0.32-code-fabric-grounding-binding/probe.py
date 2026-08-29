#!/usr/bin/env python3
from __future__ import annotations
from typing import Any, Dict

DISCOVERY_SCHEMA="axm.code-native-discovery-seam-report/v1"
DISCIPLINE_ROUTE_SCHEMA="axm.code.human-discipline-route.v1"
INTERSECTION_SCHEMA="axm.code.discipline-grammar-intersection.v1"
TEMPLATE_SCHEMA="axm.code.template-selection.v1"
VERIFIER_SCHEMA="axm.waldo.current-verifier-observation/v0.32"

class BindingError(ValueError): pass

def _require_none_authority(obj: Dict[str,Any]) -> None:
    auth=obj.get("authority")
    if isinstance(auth,dict):
        forbidden=[k for k,v in auth.items() if v]
        if forbidden: raise BindingError("DONOR_AUTHORITY_WIDENED:"+",".join(sorted(forbidden)))
    elif auth not in (None,"NONE"):
        raise BindingError("DONOR_AUTHORITY_NOT_NONE")

def _packet(kind, source_schema, *, effect, facts=None, candidates=None, notes=None):
    return {"schema":"axm.waldo.grounding-packet/v0.32","kind":kind,"sourceSchema":source_schema,
            "reasoningEffect":effect,"facts":facts or [],"candidates":candidates or [],
            "notes":notes or [],"authority":"NONE","executionPermissionCreated":False}

def translate(payload):
    if not isinstance(payload,dict): raise BindingError("PAYLOAD_NOT_OBJECT")
    schema=payload.get("schema"); _require_none_authority(payload)

    if schema==DISCOVERY_SCHEMA:
        summary=payload.get("summary") or {}; _require_none_authority(summary)
        out=[]
        for c in payload.get("topCandidates") or []:
            _require_none_authority(c); state=c.get("state"); lang=c.get("languageId")
            if state=="NOT_RELEVANT": continue
            if state=="NATIVE_REVIEW": out.append({"class":"NATIVE_REVIEW_CONTEXT","languageId":lang,"state":state})
            elif state=="DISCOVERY_CANDIDATE": out.append({"class":"DISCOVERY_CANDIDATE","languageId":lang,"state":state})
            elif state=="WEAK_SIGNAL": out.append({"class":"WEAK_SIGNAL","languageId":lang,"state":state})
            else: raise BindingError("UNKNOWN_DISCOVERY_STATE:"+str(state))
        effect="COUPLED_CONTEXT" if any(x["class"]=="NATIVE_REVIEW_CONTEXT" for x in out) else "COUPLED_UNCERTAINTY"
        return _packet("FABRIC_DISCOVERY_TRANSLATION",schema,effect=effect,candidates=out,
                       notes=["specialist output is context/candidate, not grounding authority"])

    if schema==DISCIPLINE_ROUTE_SCHEMA:
        rows=[]
        for lane in ("active","candidates","weak"):
            for x in payload.get(lane) or []:
                rows.append({"perspectiveId":x.get("perspectiveId"),"state":x.get("state"),"lane":lane})
        return _packet("DISCIPLINE_TRANSLATION",schema,effect="COUPLED_CONTEXT",candidates=rows,
                       notes=["perspective multiplicity is not evidence multiplicity"])

    if schema==INTERSECTION_SCHEMA:
        layers=payload.get("disciplineLayers") or []
        return _packet("INTERSECTION_TRANSLATION",schema,effect="COUPLED_CONTEXT",
                       candidates=[{"perspectiveId":x.get("perspectiveId"),"state":x.get("state")} for x in layers],
                       notes=["grammar x discipline intersection is a review direction, not proof"])

    if schema==TEMPLATE_SCHEMA:
        selected=[]
        for x in payload.get("selected") or []:
            lane=x.get("lane")
            if lane=="VERIFIED_VAULT":
                selected.append({"class":"VERIFIED_MECHANIC_REFERENCE","templateId":x.get("templateId"),"lane":lane})
            elif lane=="PATTERN_NURSERY":
                selected.append({"class":"MINED_UNVERIFIED_CANDIDATE","candidateSha256":x.get("candidateSha256"),"lane":lane})
            else: raise BindingError("UNKNOWN_TEMPLATE_LANE:"+str(lane))
        return _packet("TEMPLATE_TRANSLATION",schema,effect="COUPLED_CONTEXT",candidates=selected,
                       notes=["verified template mechanics do not prove instantiated source correctness"])

    if schema==VERIFIER_SCHEMA:
        if payload.get("authority")!="NONE": raise BindingError("VERIFIER_AUTHORITY_NOT_NONE")
        fresh=payload.get("freshness")=="CURRENT"; provenance=payload.get("provenanceVerified") is True
        verdict=payload.get("verdict")
        if not fresh:
            return _packet("VERIFIER_TRANSLATION",schema,effect="COUPLED_UNCERTAINTY",
                           candidates=[{"verdict":verdict}],notes=["stale verification cannot define current state"])
        if not provenance:
            return _packet("VERIFIER_TRANSLATION",schema,effect="COUPLED_UNCERTAINTY",
                           candidates=[{"verdict":verdict}],notes=["unbound verifier observation"])
        if verdict=="CONTRADICTS_CURRENT":
            return _packet("VERIFIER_TRANSLATION",schema,effect="PASS_CHANGE",
                           facts=[{"verdict":verdict,"verifierId":payload.get("verifierId")}],
                           notes=["passes evidence into coupled reasoning; still no execution authority"])
        if verdict=="SUPPORTS_CURRENT":
            return _packet("VERIFIER_TRANSLATION",schema,effect="CONFIRM_CURRENT",
                           facts=[{"verdict":verdict,"verifierId":payload.get("verifierId")}])
        raise BindingError("UNKNOWN_VERIFIER_VERDICT:"+str(verdict))
    raise BindingError("UNSUPPORTED_SCHEMA:"+str(schema))

def execution_guard(packet, permit):
    if not permit: return "HOLD_NO_PERMIT"
    if permit.get("revoked"): return "REFUSE_REVOKED"
    if permit.get("environment")!="sim": return "REFUSE_ENVIRONMENT"
    if permit.get("capability")!="sim.commit": return "REFUSE_CAPABILITY"
    return "ALLOW"

def sample_payloads():
    return [
      {"schema":DISCOVERY_SCHEMA,"summary":{"authority":"NONE"},
       "topCandidates":[{"state":"DISCOVERY_CANDIDATE","languageId":"rust","authority":"NONE"},
                        {"state":"WEAK_SIGNAL","languageId":"vhdl","authority":"NONE"}],"authority":"NONE"},
      {"schema":DISCIPLINE_ROUTE_SCHEMA,"active":[{"perspectiveId":"security-engineering","state":"NATIVE_DISCIPLINE_REVIEW"}],
       "candidates":[],"weak":[],"authority":"NONE"},
      {"schema":INTERSECTION_SCHEMA,"disciplineLayers":[{"perspectiveId":"interactive-game-systems","state":"NATIVE_DISCIPLINE_REVIEW","authority":"NONE"}],
       "authority":"NONE"},
      {"schema":TEMPLATE_SCHEMA,"selected":[{"lane":"VERIFIED_VAULT","templateId":"code.template.rust.safe-refactor-preflight.v1","authority":"NONE"},
                                            {"lane":"PATTERN_NURSERY","candidateSha256":"abc","authority":"NONE"}],
       "authority":"NONE"},
    ]
