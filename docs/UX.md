# Command guide

This guide describes the current CLI. `waldo <command> --help` is authoritative
for flags and detailed behavior.

## Command tree

```text
waldo
├── advisor
├── mirror
│   ├── reason
│   ├── ground
│   │   └── verify
│   └── experience
│       └── observe
├── status
├── config
│   ├── show
│   ├── get
│   ├── set
│   └── unset
├── index
│   ├── init
│   ├── pull
│   ├── list
│   ├── show
│   ├── summary
│   ├── verify
│   ├── audit
│   ├── bom
│   ├── ingest
│   ├── update
│   └── export
├── shard
│   ├── summary
│   ├── audit
│   ├── bom
│   ├── list-records
│   └── export-record
├── lookaside
│   ├── login
│   ├── logout
│   ├── list
│   ├── status
│   ├── verify
│   ├── mirror
│   └── rm
├── model
│   ├── advisor
│   ├── init
│   ├── pull
│   ├── list
│   ├── summary
│   ├── bom
│   ├── forecast
│   ├── train
│   ├── continue
│   ├── export
│   ├── chat
│   └── rm
└── completion
```

There is no top-level `waldo bom` command. Corpus BOM operations are under
`waldo index bom`; model BOM operations are under `waldo model bom`.

## Local readiness

```bash
waldo status
```

`status` inspects local resources, the selected index, lookaside paths, and the
training harness chosen by the same environment resolver used by `model
train`. It performs no network requests and changes no state. Each unavailable
workflow reports `Ready: no` and explains why. Global `--json` returns the
same structured facts for automation.

## Index selection

When `config.index` is unset, read-only index commands use a managed checkout
at `~/.waldo/index`. WALDO creates it on first use and safely fast-forwards it
when the checkout is clean and behind its tracking branch. If upstream history
was rewritten, WALDO also resets this clean, read-only managed checkout to the
new canonical branch. It never applies that recovery to a configured
contributor checkout.

Set `config.index` to use a writable contributor checkout:

```bash
waldo config set index /path/to/waldo-index
```

Logical index paths resolve beneath the selected checkout. Existing filesystem
paths and paths beginning with `./`, `../`, `/`, or `~/` explicitly select the
exact local checkout and never trigger Git network access. Omitting a path
selects the whole resolved index. `waldo index verify --offline` additionally
disables object-storage verification and performs local structural validation.

## Inspect and verify data

```bash
waldo index list
waldo index show core/example
waldo index summary
waldo index verify --offline
waldo index verify core/example
waldo index verify core/example --objects
waldo index audit core/example --workers 8
```

Default verification checks metadata and canonical object availability without
downloading object bodies. `--objects` downloads and hashes every selected
object. `audit` additionally validates canonical shard contents. Its worker
count controls concurrent fetch-and-audit operations.

Local Parquet files can be inspected without an index:

```bash
waldo shard summary shard.parquet
waldo shard audit shard.parquet
waldo shard bom shard.parquet
waldo shard list-records shard.parquet
```

## Contribute data

Authoring commands refuse the managed read-only checkout. Clone the index and
configure a writable checkout first:

```bash
git clone https://github.com/openwaldo/waldo-index.git
waldo config set index /path/to/waldo-index
waldo config set lookaside file:///tmp/waldo-lookaside
```

Inspect the complete ingest interface before use:

```bash
waldo index ingest --help
```

Ingestion accepts supported local files or a reviewed ingest recipe. After all
objects are audited and published, it retains a contribution overlay and
atomically applies those metadata changes to the selected index working tree
for normal Git review. WALDO does not commit, push, or open a pull request.

## Export data and inspect its BOM

```bash
waldo index export core/example /path/to/export
waldo index bom /path/to/export
waldo index verify /path/to/export
```

An export contains `EXPORT.json` plus the selected data files. `index verify`
checks the persisted BOM and hashes each exported file.

## Lookaside storage

```bash
waldo lookaside status
waldo lookaside list
waldo lookaside verify
waldo lookaside mirror --help
waldo lookaside rm --help
```

For S3 publication, configure the destination and store bucket-scoped
credentials outside corpus metadata:

```bash
waldo config set lookaside s3://bucket/prefix
waldo config set lookaside.region us-west-2
waldo lookaside login
```

Removal requires explicit object names. Review command help before any
consequential lookaside operation.

## Models

The model lifecycle can start from an index selection, a compose file, or a
supported open-weight origin. Available training backends depend on the host.

```bash
waldo model forecast composes/0000-canary.yaml
waldo model train canary composes/0000-canary.yaml
waldo model summary canary
waldo model bom canary
waldo model chat canary
waldo model export --help
```

`forecast` reports readiness and runtime for the current host. Add
`--compare-hosts` to include the versioned hardware comparison. Run `forecast`
before allocating substantial compute. Training and generation
fail when the selected host lacks a compatible runtime or artifacts.
Models with a declared `interaction.template` automatically receive the
matching prompt format and multi-turn history in `model chat`; models without
one remain raw causal-continuation models.

## Experimental AXM Mirror reasoning

The experimental branch exposes deterministic-primary Mirror reasoning through a strict request file:

```bash
waldo mirror reason request.json
waldo mirror reason request.json --neural --model local-model
waldo mirror reason request.json --experience-ledger private-experience.jsonl
waldo mirror experience observe outcome.json --ledger private-experience.jsonl \
  --learn-to private-training.jsonl --hermes-to private-hermes-memory.jsonl
```

Stable requests resolve without opening a model. Uncertain or conflicting requests HOLD unless local neural escalation is explicitly opted into. Neural responses remain candidates with authority `NONE`. `--trace` writes hash-only receipts; `--learn-to` can visibly capture chat-derived learning input as either review-required or explicitly approved for later training.

`--experience-ledger` adds an inspectable learning-by-experience loop. Every reasoning call appends what was sensed and how the stack reacted. `mirror experience observe` later appends the observed outcome and a retained lesson. Completed episodes are exposed as visible context on later unresolved neural calls. Helpful reactions and explicit corrections project response targets. Harmful and inconclusive outcomes do not become positive response targets; instead, they project an outcome-conditioned trajectory in which the failed attempt is tool evidence and only the retained lesson is supervised. Completion is therefore not the learning gate.

The standalone WALMI Windows host adds `WALMI_EXPERIENCE_OBSERVE.cmd` and `WALMI_AUTOLEARN.cmd`. The former closes an episode into the portable direct-experience, training-projection, and Hermes ledgers and then invokes the controller. The controller waits for eight new projections by default, normalizes both response targets and failure reflections into a conversational corpus, replays the accumulated derived corpus, and persists a local checkpoint. The source experience ledger is never rewritten. `WALMI_AUTOLEARN.cmd --status` reports the threshold without training. A new weight hash proves mutation and persistence; behavior quality remains unknown until a separate held-out evaluation supports it.

`--hermes-to` writes the same completed episode as a portable Hermes memory capsule. This is a handoff to AXM's separate Hermes runtime lane, not a claim that WALDO installed, launched, or silently mutated Hermes. All three private ledgers are forced to mode `0600`.

Verify a visible positive-ground or observed-learning dataset before it reaches
training intake:

```bash
waldo mirror ground verify ground.jsonl
waldo mirror ground verify positive-ground.jsonl --positive-seed \
  --training-to private-structured-training.jsonl
```

`SYNTHETIC_SEED`, `OBSERVED_CHAT`, and `OBSERVED_EXECUTION_TRACE` remain
distinct source classes. Synthetic records name their generator and cannot
claim observation evidence. Observed records require a timestamp and source
receipt. Only curated, helpful, or corrected records can project positive
structured response targets; harmful and inconclusive Mirror episodes may
project assistant-supervised reflections while their attempts remain unsupervised evidence. Unreviewed ground records are
retained without becoming positive targets. `--training-to` writes a new mode
`0600` file and never mutates weights by itself.

### Bounded project workspace

Create a strict request such as:

```json
{
  "schema": "axm.waldo.workspace-loop-request/v0.2",
  "prompt": "Inspect this project, repair the failing test, verify it, and explain the result.",
  "max_turns": 16,
  "verify_commands": [["go", "test", "./..."]]
}
```

Then explicitly select the only visible project root and local model:

```bash
waldo mirror workspace C:\Projects\MyGame workspace-request.json --model local-model
```

That invocation is read-only. Add `--allow-write` only when the bounded transactional writer may change files inside the selected root. Verification commands come only from the request JSON and never pass through a shell. `--experience-to private-workspace.jsonl` retains a review-required receipt; it does not automatically train or change weights.

Reference composes under `composes/` are test and experiment inputs, not model
quality guarantees. See the [model compose guide](MODEL-COMPOSE.md) for the
complete schema, defaults, profiles, and validation rules.

## Output and help

Human-readable output is the default. Global `--json` emits structured results
while progress remains on standard error.

```bash
waldo --json config get
waldo index verify --help
waldo completion --help
```
