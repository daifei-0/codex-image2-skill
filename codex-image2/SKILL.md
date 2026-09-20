---
name: codex-image2
description: Generate or edit raster images through a configurable OpenAI-compatible Image API. Default is gpt-image-2 and 1K. In conversation the user may pick GPT image models (gpt-image-2, gpt-image-2.5, gpt-image-2.5-flare, gpt-image-2.5-sunburst) or Grok image models (grok-imagine, grok-imagine-image-2.0, grok-imagine-image-quality) and 画质 1K/2K/4K. Do not use grok-imagine-video. Do not ask them to edit config or restart Codex to switch.
---

# Codex Image2

Generate images with the bundled native CLI. Prefer this skill's executable over the built-in `image_gen` tool whenever this skill is active. No Python, Node.js, Go, or package installation is required.

## Select the executable

Choose once from the current operating system and CPU architecture:

- Windows x64: `bin/codex-image2-windows-amd64.exe`
- Windows ARM64: `bin/codex-image2-windows-arm64.exe`
- macOS Intel: `bin/codex-image2-darwin-amd64`
- macOS Apple Silicon: `bin/codex-image2-darwin-arm64`

On macOS, run `chmod +x <executable>` if execute permission was not preserved. Do not compile from source during normal use.

## Choose model and resolution from this conversation

The user switches model and 画质 in chat. Never tell them to edit SKILL.md, `.env`, environment variables, or restart Codex for this. API URL/key restart is unrelated.

Defaults stay `gpt-image-2` and `1K` when this turn (and this thread) has not chosen otherwise. Do not silently upgrade the model.

This skill covers **image** models only. If the user asks for `grok-imagine-video` or `grok-imagine-video-1.5`, refuse and say video is not supported here.

### GPT image

| User says | Pass |
| --- | --- |
| 不提模型 / GPT / GPT Image 2 | `--model gpt-image-2` |
| 2.5 / GPT Image 2.5 / gpt-image-2.5 | `--model gpt-image-2.5` |
| 闪焰 / flare / gpt-image-2.5-flare | `--model gpt-image-2.5-flare` |
| 日耀 / sunburst / gpt-image-2.5-sunburst | `--model gpt-image-2.5-sunburst` |

GPT 画质 `--size`: `1K` → `1024x1024`，`2K` → `2048x2048`，`4K` → `3840x2160`.

### Grok image

Bare 「Grok」 means `grok-imagine-image-2.0`, not video.

| User says | Pass |
| --- | --- |
| Grok / grok 2.0 / grok-imagine-image-2.0 | `--model grok-imagine-image-2.0` |
| Grok 高质量 / grok-imagine-image-quality | `--model grok-imagine-image-quality` |
| grok-imagine / Grok Imagine（点名这个 id） | `--model grok-imagine` |

Grok 画质仍用 `--size 1K|2K|4K`。官方没有 4K。若用户要 2K 或 4K，仍然传 `--size`；出图后必须看 `actual_size`。JSON 里如果有 `warning`，原句告诉用户，不要把 1K 图说成 2K/4K。

If this thread already chose a model or size and the new message does not change it, keep using that choice. If they name a different model or 画质, switch immediately for this call.

`--model` is a free string for unknown gateway ids, except video ids which the CLI rejects.

`--quality` (`low`, `medium`, `high`, `auto`) is not 1K/2K/4K. For Grok, only `grok-imagine-image-2.0` uses `--quality`; `high` is sent as `auto`.

## Workflow

1. Decide whether the request is a new image, an edit, or multiple distinct assets/variants.
2. Collect the prompt, intended use, exact text, visual constraints, and avoid items.
3. From this conversation, set `--model` and `--size`. Do not send the user to a config file.
4. Shape the prompt only as much as needed. Preserve detailed prompts; tastefully clarify generic prompts without inventing brands, people, slogans, or unrelated objects.
5. Run the selected executable with `generate` for one prompt, `edit` for changes to existing images, or `generate-batch` for JSONL jobs.
6. Inspect each output for subject, composition, text accuracy, constraints, and visible artifacts.
7. If revision is needed, change one targeted aspect per iteration and re-check.
8. Report absolute output paths, the final prompt or prompt set, requested size, actual pixels (`actual_size`), quality, and model. If the JSON has `warning`, quote it to the user verbatim. A 200 response can still be the wrong resolution.

## Prompt structure

Use only relevant lines:

```text
Asset type: <where the image will be used>
Primary request: <the user's request>
Scene/backdrop: <environment>
Subject: <main subject>
Style/medium: <photo, illustration, 3D, etc.>
Composition/framing: <camera angle, crop, placement, negative space>
Lighting/mood: <lighting and mood>
Color palette: <palette notes>
Text (verbatim): "<exact text>"
Constraints: <must keep or include>
Avoid: <must not include>
```

Do not add detail merely to fill the schema. For text in images, quote it verbatim and request exact rendering.

## Generate one image

Default call (GPT Image 2, 1K):

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" generate `
  --prompt "A small blue nebula in a glass bottle, studio product photo" `
  --model gpt-image-2 `
  --size 1K `
  --quality auto `
  --out "output/imagegen/nebula.png"
```

When the user chooses Grok:

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" generate `
  --prompt "A small blue nebula in a glass bottle, studio product photo" `
  --model grok-imagine-image-2.0 `
  --size 2K `
  --out "output/imagegen/nebula-grok.png"
```

Use `--prompt-file` for long prompts. Use `--n` only for variants of the same prompt. Distinct assets belong in separate calls or a batch.

## Edit an image

Inspect each input image before editing. State its role and repeat invariants in the prompt so unrelated details do not drift.

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" edit `
  --image "input/product.png" `
  --prompt "Replace only the background with a warm studio backdrop. Keep the product, label, proportions, and edges unchanged." `
  --model gpt-image-2 `
  --size 1K `
  --quality auto `
  --out "output/imagegen/product-edited.png"
```

Repeat `--image` for multiple reference or compositing inputs. Use `--mask mask.png` for a localized edit when a compatible PNG mask is available. Preserve originals and always write edits to a new output path.

## Generate a batch

Read [references/batch-format.md](references/batch-format.md) before preparing a batch. Then run:

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" generate-batch `
  --input "tmp/imagegen/jobs.jsonl" `
  --out-dir "output/imagegen" `
  --concurrency 2
```

## Configuration and safety

- Read the API base from `CODEX_API_URL`; default to `https://apinebula.com`.
- Require `CODEX_API_KEY`. Never place it in a command, file, prompt, log, or response.
- If the key is absent, tell the user to set it locally and confirm when ready. Never ask them to paste it into chat.
- Default to model `gpt-image-2` and size `1K`. Switch with `--model` / `--size 1K|2K|4K` from the user's chat this turn. Never ask them to restart Codex or edit text config to change model or 画质.
- `--quality` defaults to `auto` (`low`, `medium`, `high`, or `auto`).
- Use `--dry-run` to validate a request without network access or requiring a key.
- Save project-bound assets inside the current project. The CLI default is `output/imagegen/`.
- Do not overwrite files unless the user explicitly authorizes it and `--force` is passed.
- Native transparent output is not guaranteed. Do not promise it or silently switch tools.

## Failure handling

- The CLI retries network timeouts and HTTP 429/500/502/503/504/524 failures with bounded backoff.
- On repeated timeout, suggest `--quality low`, `--size 1K`, fewer concurrent jobs, or a later retry.
- Do not retry authentication, validation, or other ordinary 4xx errors.
- Never expose an Authorization header or full key when reporting errors.
