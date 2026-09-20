# Batch JSONL format

Write one JSON object per line. Each job requires `prompt` and may override generation or output settings.

```jsonl
{"prompt":"A blue ceramic mug on white","out":"mug.png"}
{"prompt":"A red paper kite in a clear sky","model":"gpt-image-2.5","size":"2K","quality":"low","n":2,"out":"kite.png"}
{"prompt":"A wide landscape poster","size":"4K","out":"poster.png"}
```

Supported fields are `prompt`, `size`, `quality`, `n`, `out`, and `model`. `size` accepts `1K`, `2K`, `4K`, `auto`, or `WIDTHxHEIGHT`. Omit `model` and `size` to keep the CLI defaults (`gpt-image-2`, `1K`). Relative `out` paths are resolved under `--out-dir`. Blank lines are ignored.

Use unique output names. When `n` is greater than one, the CLI adds `-1`, `-2`, and so on before the extension. The batch command prints a JSON summary and exits nonzero if any job fails.
