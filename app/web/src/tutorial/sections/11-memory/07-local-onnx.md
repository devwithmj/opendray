# Local ONNX embeddings

opendray's default memory embedder is BM25 — keyword matching with
no model files. For semantic search **without an HTTP backend**
(no ollama, no OpenAI, fully air-gapped), build with the
`local_onnx` tag and point at a sentence-transformer ONNX model on
disk.

This is **opt-in** because the build pulls cgo + native libraries
that not every operator wants. The default `go build` produces a
cgo-free binary with the LocalONNX path stubbed out.

## What you need

| Artifact | Where to get it | Approx. size |
|---|---|---|
| `libonnxruntime.dylib` (macOS) or `.so` (Linux) | `brew install onnxruntime` / package manager | ~25 MB |
| `libtokenizers.a` | <https://github.com/daulet/tokenizers/releases/latest> (per-platform tarball) | ~37 MB |
| `model.onnx` | HuggingFace ONNX export of any sentence-transformer (BGE-m3, BGE-small, nomic-embed) | 90 MB - 2 GB |
| `tokenizer.json` | Same HuggingFace repo as the model | 1-17 MB |

For BGE-m3 specifically, download `Xenova/bge-m3` from HuggingFace:

```bash
mkdir -p ~/.opendray/models/bge-m3
cd ~/.opendray/models/bge-m3

curl -L -o tokenizer.json \
  https://huggingface.co/Xenova/bge-m3/resolve/main/tokenizer.json
curl -L -o model.onnx \
  https://huggingface.co/Xenova/bge-m3/resolve/main/onnx/model.onnx
# Optional INT8 quantized version (~565 MB instead of ~2 GB):
# curl -L -o model.onnx \
#   https://huggingface.co/Xenova/bge-m3/resolve/main/onnx/model_quantized.onnx
```

## Build with the tag

```bash
brew install onnxruntime           # one-time
mkdir -p ~/.opendray/deps
curl -L https://github.com/daulet/tokenizers/releases/latest/download/libtokenizers.darwin-arm64.tar.gz \
  | tar -xz -C ~/.opendray/deps    # produces libtokenizers.a

CGO_LDFLAGS="-L/opt/homebrew/opt/onnxruntime/lib -L$HOME/.opendray/deps" \
DYLD_LIBRARY_PATH="/opt/homebrew/opt/onnxruntime/lib" \
go build -tags local_onnx -o opendray ./cmd/opendray
```

Set `DYLD_LIBRARY_PATH` (macOS) or `LD_LIBRARY_PATH` (Linux) at
**runtime** too — the dynamic linker needs to find
`libonnxruntime.dylib` at process start. systemd-style deployments
usually put this in the service unit's `Environment=` directive.

## Configure

```toml
[memory]
backend = "local"

[memory.local]
model           = "bge-m3"   # cosmetic — appears in logs / UI
library_path    = "/opt/homebrew/opt/onnxruntime/lib"
model_path      = "~/.opendray/models/bge-m3/model.onnx"
tokenizer_path  = "~/.opendray/models/bge-m3/tokenizer.json"
max_seq_len     = 512
```

Restart opendray. The startup log should show:

```
INFO memory ready  embedder=local-onnx:model.onnx  dimensions=1024
```

Settings → Server → Memory → Inspector → status strip mirrors the
same info. Click "Test embedder" to round-trip a sample text and
confirm everything wires up.

## Performance expectations

| Model | Dimensions | macOS arm64 latency (avg per call) |
|---|---|---|
| `bge-m3` (FP32) | 1024 | ~150 ms |
| `bge-m3` (INT8 quantized) | 1024 | ~80 ms |
| `bge-small-en-v1.5` | 384 | ~30 ms |

Storage in pgvector scales linearly with dimensions — 1024 dims
costs ~4 KB/row plus index. For 10K memories: ~40 MB plus HNSW
index. Negligible.

## Switching back

Change `[memory.backend]` to anything else (`bm25` / `http`),
restart opendray. Existing memories embedded with the local model
stay in pgvector but won't be returned by future searches —
opendray filters by `embedder` name to keep cosine comparisons
honest. To make them searchable again, switch the backend back, or
delete + re-embed via Inspector.

## What this build skips

- `go:embed` of the model bytes — keeps binary small at the cost
  of operator setup. Phase 2.B.2 will optionally embed bge-small
  (120 MB) for true offline / single-binary deployments.
- GitHub Actions cross-platform matrix — macOS arm64 is verified;
  amd64 and Linux are likely to work but untested. File issues as
  you find platform breaks.
- INT8 quantization of arbitrary models on the fly — pre-quantize
  in HuggingFace + download.

## When to bother

Use BM25 (default) when:
- You're testing or doing keyword-shaped recall (file names, code
  identifiers).

Use HTTP backend (ollama / OpenAI compat) when:
- You can run ollama locally — easiest path to high-quality
  semantic search, no rebuild needed.

Use LocalONNX (this) when:
- Air-gapped deployment with no network egress.
- You've benchmarked BGE / nomic / mxbai variants and want a
  specific model with no daemon.
- You want to ship opendray as a single binary (with `go:embed`
  in phase 2.B.2).
