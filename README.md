# WALMI

WALMI is a local-first personal AI experiment built from the OpenWALDO / AXM
Mirror stack. Its central question is practical: can one person own an AI whose
future behaviour grows from preserved, inspectable experience rather than from
disposable chat sessions alone?

This repository is the clean **standalone source boundary**. It contains the
complete implementation files needed to inspect, build, test, and package the
current experiment. It intentionally contains no personal saves, learned model
weights, raw experience, secrets, caches, or installed runtime. Those belong to
each WALMI instance and are ignored by Git.

Current grounded capabilities include:

- the complete WALDO model, corpus, provenance, Mirror, and Hermes-facing source;
- thresholded outcome-conditioned experience growth without rewriting direct
  experience;
- persistent named saves for Theme Park, Factual Space, and Living City;
- paced neural play through visible bounded action menus;
- offline browser-cartridge and Windows PC packaging sources;
- deterministic source manifests and one-file transport bundles.

The project is experimental and makes no blanket safety or intelligence claim.
Receipts prove bounded facts such as artifact identity, accepted actions, save
continuity, and changed weights. They do not by themselves prove good strategy,
general reasoning, or safe behaviour.

Start with [the standalone architecture](docs/WALMI_STANDALONE.md) and
[`portable/walmi-pc-package`](portable/walmi-pc-package). Run
`python tools/seal_source_tree.py --verify` to verify this source checkout. Run
`python tools/build_source_bundle.py` to turn the repository into one portable
ZIP while retaining every source file and its hash.

On Windows, run `powershell -ExecutionPolicy Bypass -File
tools/test_walmi_windows.ps1`. It verifies the WALMI integration packages,
cross-builds the browser WebAssembly entrypoint, runs the portable Python tests,
and checks the sealed source tree. The upstream all-package Go suite also
contains POSIX permission and optional external-tool fixtures; those are not
misreported as native Windows WALMI failures.

## Included WALDO engine

WALDO is a command-line tool for building auditable AI training datasets and
models. It connects a Git-governed corpus index, content-addressed Parquet
objects, and model artifacts with verifiable bills of materials (BOMs).

WALDO is under active development. It is being opened early so users and
contributors can help shape the project through frequent, incremental
releases. Expect interfaces and formats outside the documented compatibility
contract to evolve.

## What works today

- Inspect, verify, audit, ingest, update, and export indexed corpora.
- Read and publish canonical Parquet objects through local or S3 lookaside
  storage.
- Create corpus, training-run, model, and release provenance records.
- Forecast and train models through MLX, PyTorch, or single-node TorchTitan
  when the required runtime is installed.
- Export native WALDO, Hugging Face, MLX, GGUF, and Ollama packages.

WALDO does not prove that a license assertion is legally correct, that model
output is safe, or that a generated disclosure alone establishes regulatory
compliance. It records attributable facts and verifies artifact identity.

## Build

WALDO requires Go 1.25 or newer.

```bash
git clone https://github.com/openwaldo/waldo.git
cd waldo
go install ./cmd/waldo
waldo --help
```

`go install` writes the executable to `GOBIN`, or to `GOPATH/bin` when
`GOBIN` is unset. That directory must be on `PATH`.

## First steps

With no configured index, read-only index commands use a managed checkout at
`~/.waldo/index`:

```bash
waldo status
waldo index list
waldo index summary
waldo index verify --offline
```

Contributing data requires a separate writable index checkout:

```bash
git clone https://github.com/openwaldo/waldo-index.git
waldo config set index /path/to/waldo-index
waldo config set lookaside file:///tmp/waldo-lookaside
waldo index ingest --help
```

Use `waldo <command> --help` as the authoritative command reference.

## Documentation

Start with the [documentation index](docs/README.md). In particular:

- [Command guide](docs/UX.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Testing](docs/TESTING.md)
- [Open-source release plan](docs/RELEASING.md)

Early feedback, testing, documentation improvements, and focused code
contributions are welcome. See [Contributing](docs/CONTRIBUTING.md).

## Development

```bash
./testing/unit.sh
./testing/vet.sh
```

The full end-to-end suite is described in [docs/TESTING.md](docs/TESTING.md).

## License

WALDO is licensed under the [Apache License 2.0](LICENSE). See [NOTICE](NOTICE)
for attribution notices.
