from __future__ import annotations
import hashlib, json, sys
from pathlib import Path


def receipt_digest(obj: dict) -> str:
    base = {k: v for k, v in obj.items() if k != 'receiptDigest'}
    raw = json.dumps(base, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
    return 'sha256:' + hashlib.sha256(raw).hexdigest()


def main() -> int:
    view_path, out_path = map(Path, sys.argv[1:3])
    view = json.loads(view_path.read_text())
    included = view['includedArtifacts']
    required = view['requiredArtifacts']
    missing = [name for name in required if name not in included]
    bad: list[str] = []

    ring_meta = included.get('peerRing')
    if ring_meta:
        p = (view_path.parent / ring_meta['path']).resolve()
        if not p.exists():
            bad.append('peerRing')
        else:
            ring = json.loads(p.read_text())
            if (
                ring.get('receiptDigest') != view['sourceReceiptDigest']
                or ring.get('authority') != 'NONE'
                or ring.get('peerAuditPack', {}).get('digest') != view['peerAuditPackDigest']
            ):
                bad.append('peerRingBoundary')

    projection_meta = included.get('peerAuditProjection')
    if projection_meta:
        p = (view_path.parent / projection_meta['path']).resolve()
        if not p.exists():
            bad.append('peerAuditProjection')
        else:
            projection = json.loads(p.read_text())
            if (
                projection.get('schema') != 'axm.peer-audit-pack-public-projection/v0.1'
                or projection.get('id') != 'peer-evidence-audit'
                or projection.get('fullPackDigest') != view['peerAuditPackDigest']
                or projection.get('authorityCeiling') != 'NONE'
            ):
                bad.append('peerAuditProjectionBoundary')

    verdict = 'HOLD' if missing or bad else 'EVIDENCE_PASS'
    findings: list[str] = []
    if missing:
        findings.append('MATERIAL_EVIDENCE_VIEW_INCOMPLETE')
    if bad:
        findings.append('MATERIAL_EVIDENCE_MISMATCH')
    view_digest = 'sha256:' + hashlib.sha256(json.dumps(view, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    out = {
        'schema': 'axm.waldo-independent-witness/v0.14',
        'witnessId': 'witness-b-python-material',
        'implementation': 'python-material-v1',
        'sourceReceiptDigest': view['sourceReceiptDigest'],
        'evidenceViewDigest': view_digest,
        'verdict': verdict,
        'findings': findings,
        'authority': 'NONE',
    }
    out['receiptDigest'] = receipt_digest(out)
    out_path.write_text(json.dumps(out, indent=2) + '\n')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
