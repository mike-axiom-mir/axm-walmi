# WALDO Witness Mirror synthetic examples

These files are public-safe synthetic inputs. They do not describe a real
trained model or a real independent evaluation.

The checked-in corpus, run, release, comparison, behavior, sensory, skill
continuity, discovery, inner-asset, and capability-spine drafts are synthetic
contract fixtures. The complete gated, situated, inner-asset, and synthetic
LIVE capability CLI flows are exercised by `TestGatedCloneFoundationCLIFlow`,
`TestInnerAssetCLIFlow`, `TestPortableCapabilitySpineCLIFlow`, and the focused
tests under `internal/axmmirror`.

`corpus-bom.json` intentionally remains a schema-1 shard fixture. The current
v0.2 lens reports its record filter as not declared, its assessment as legacy,
and its privacy-redaction evidence as not recorded. Current schema-2/v9
compatibility is tested against the real upstream Go BOM type rather than
misrepresenting this legacy example as freshly redacted.

The separate experimental CLI now supports:

```text
waldo-axm-mirror lens-corpus
waldo-axm-mirror witness-run
waldo-axm-mirror profile-contract
waldo-axm-mirror anchor
waldo-axm-mirror lock
waldo-axm-mirror contamination
waldo-axm-mirror context
waldo-axm-mirror gate-claims
waldo-axm-mirror seal-evaluation
waldo-axm-mirror intake-sensory
waldo-axm-mirror assess-skills
waldo-axm-mirror discover
waldo-axm-mirror situated-context
waldo-axm-mirror forge-asset
waldo-axm-mirror verify-asset
waldo-axm-mirror census-capabilities
waldo-axm-mirror intake-capabilities
waldo-axm-mirror plan-handoff
waldo-axm-mirror seal-translation
waldo-axm-mirror verify-handoff-return
waldo-axm-mirror seal-gated
waldo-axm-mirror seal-situated
waldo-axm-mirror verify
```

`seal-evaluation` is a pre-execution operation in a real experiment. The test
constructs all deterministic inputs in one process for replay convenience, but
the final gated seal still requires the output and claim assessment to bind the
exact precommitted protocol digest.

The situated examples distinguish three surfaces that must not be collapsed:

- a Sensorium receipt is typed evidence supplied to WALDO, not proof that
  WALDO captured an image, stream, host, or environment itself;
- a skill continuity request compares exact current and backup manifests and
  may only propose a human-reviewed recovery candidate;
- AI-native seams are primary machine judgement while human-native discovery
  remains explicit secondary advice.

All thirteen Sensorium contracts are supported by the intake catalog. A
specific situated request still names only the senses the task actually
requires; support never implies silent invocation or permission.

`inner-asset-recipe.json` is a two-frame synthetic witness-orb recipe. The
foundry compiles it into separate editable recipe/grid, primary PNG, preview
PNG, and sprite-atlas artifacts inside one deterministic `.axmasset` candidate.
The fixture pins the complete portable bundle digest, but remains visually
UNREVIEWED and carries no install, approval, promotion, or CANON authority.

The portable capability fixtures intentionally demonstrate a held real-source
path:

- `self-capability-census-request.json` binds a synthetic externally observed
  build and emits the clone's compiled declaration catalog;
- `external-capability-snapshot.json` pins four exact public platform contracts
  but labels them `SOURCE_ONLY`, never LIVE;
- `capability-gap-request.json` asks for the declared platform translation
  contract using exact input and output schemas;
- `capability-translation-declaration.json` records that no schema translation
  is required, without generating or invoking an adapter;
- `capability-return-draft.json` is synthetic declared return metadata. It is
  necessarily `HOLD_RETURN_PLAN` because the source-only plan was never live or
  executable.

This chain pins the initial durable receipt digests and proves that source
knowledge cannot silently become runtime readiness. Separate tests construct a
fresh synthetic LIVE provider and reach `RETURN_BINDINGS_VERIFIED`; that still
proves bindings only, not returned bytes, content quality, permission
enforcement, installation, promotion, or CANON.

HOLD and REFUSED receipts are written before the CLI returns a nonzero status,
so unresolved evidence remains inspectable. Output paths are atomic no-replace
writes and are never silently overwritten.
