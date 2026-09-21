---
name: codex-image2-5
description: Generate or edit images through a configurable Image API. Codex Image 2.5 defaults to gpt-image-2.5 and 1K; users can say Image 2, Image 2.5, 二点五, 闪焰, 日耀, or Grok and switch 1K/2K/4K during a conversation. Preserve explicit model choices and prior settings. Image tasks only, not video.
---

# Codex Image 2.5

Generate images with the bundled native CLI. Prefer this skill's executable over the built-in `image_gen` tool whenever this skill is active. No Python, Node.js, Go, or package installation is required.

## Select the executable

Choose once from the current operating system and CPU architecture:

- Windows x64: `bin/codex-image2-5-windows-amd64.exe`
- Windows ARM64: `bin/codex-image2-5-windows-arm64.exe`
- macOS Intel: `bin/codex-image2-5-darwin-amd64`
- macOS Apple Silicon: `bin/codex-image2-5-darwin-arm64`

On macOS, run `chmod +x <executable>` if execute permission was not preserved. Do not compile from source during normal use.

## Choose model and resolution from this conversation

Treat model, resolution (`--size`), and rendering quality (`--quality`) as separate choices. Extract them from the user's generation/editing instructions; do not send a whole sentence as `--model`.

For each option, use this order: explicit choice in the current request → most recent choice in this conversation → default. Defaults for a new conversation are `gpt-image-2.5`, `1K`, and `auto`. Pass the selected values explicitly on every CLI call; the CLI itself does not remember conversation state.

The skill's name, `$codex-image2-5` or "Codex Image 2.5", is an invocation, not a request to reset the model. If the user says "使用 Codex Image 2.5，用 Image 2", use `gpt-image-2`. Do not select models from quoted image text, the image subject, an aspect ratio, image numbers, or a resolution such as 2K.

Users switch in chat; do not ask them to edit files, environment variables or restart the client for model/resolution changes. Initial API URL/key setup is separate.

### Model names in ordinary language

Recognize case differences, spaces, hyphens, underscores, full-width letters/digits and these short names:

| User says | Model |
| --- | --- |
| GPT / GPT Image (no version) | `gpt-image-2.5` |
| Image 2 / image2 / GPT Image 2 / img2 / Image 二 | `gpt-image-2` |
| Image 2.5 / image2.5 / 2.5 / 二点五 | `gpt-image-2.5` |
| 闪焰 / 闪焰版 / flare / Image 2.5 flare | `gpt-image-2.5-flare` |
| 日耀 / 日耀版 / sunburst / Image 2.5 sunburst | `gpt-image-2.5-sunburst` |
| Grok / Grok 2 / Grok 2.0 | `grok-imagine-image-2.0` |
| Grok 高质量 / Grok 质量版 / Grok quality | `grok-imagine-image-quality` |
| grok-imagine / Grok Imagine (explicit name) | `grok-imagine` |

"用 Image 2" means Image 2, even though the skill is named Image 2.5. "用二点五" and "换成2.5模型" mean Image 2.5. Prefer the more specific variant when named: "Image 2.5 闪焰" is flare.

"高质量一点" or "高清一点" alone preserves the current model and resolution; use `--quality high` where supported. Never turn that phrase alone into the Grok quality model or silently choose 4K. Only explicit "Grok 高质量" selects that model. If a requested model name has genuinely conflicting interpretations, ask one short clarification before a paid call.

Unknown explicit gateway model IDs may pass through unchanged. Video IDs, including Grok video, are unsupported.

### Resolution and conversation continuity

Map "1K/1k/1 K/一K" to `1K`, "2K/2k/2 K/两K/二K" to `2K`, and "4K/4k/4 K/四K" to `4K`. Full-width forms such as "２Ｋ" are equivalent.

- "这张改成4K，模型不变": keep the previous model, use 4K.
- "换Image 2.5，分辨率不变": change only the model.
- "用Grok，2K": change both.
- "还是这个模型再画一张": retain model, resolution and quality.
- "恢复默认模型": use Image 2.5, preserve resolution.
- "全部恢复默认": use Image 2.5 + 1K + auto.

Example sequence: "用Image 2，2K画海边" → Image 2 + 2K; "改4K" → Image 2 + 4K; "换二点五" → Image 2.5 + 4K; "用Grok，2K" → Grok + 2K.

GPT request sizes: 1K → 1024×1024, 2K → 2048×2048, 4K → 3840×2160. These are requested pixels, not a guarantee of the upstream result.

Grok's current adapter uses 1k/2k resolution plus size. Its 4K option requests the upstream's highest slot; it is not guaranteed native 4K and may produce about 2816×1584. Inspect `actual_size` for every image. Report any `warning` verbatim; do not call a lower-resolution image 4K.

`--quality low|medium|high|auto` is independent of resolution. In the current Grok adapter, only `grok-imagine-image-2.0` sends quality, and high maps to auto.

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

Default call (GPT Image 2.5, 1K):

```powershell
& "<skill-dir>\bin\codex-image2-5-windows-amd64.exe" generate `
  --prompt "A small blue nebula in a glass bottle, studio product photo" `
  --model gpt-image-2.5 `
  --size 1K `
  --quality auto `
  --out "output/imagegen/nebula.png"
```

When the user chooses Grok:

```powershell
& "<skill-dir>\bin\codex-image2-5-windows-amd64.exe" generate `
  --prompt "A small blue nebula in a glass bottle, studio product photo" `
  --model grok-imagine-image-2.0 `
  --size 2K `
  --out "output/imagegen/nebula-grok.png"
```

Use `--prompt-file` for long prompts. Use `--n` only for variants of the same prompt. Distinct assets belong in separate calls or a batch.

## Edit an image

Inspect each input image before editing. State its role and repeat invariants in the prompt so unrelated details do not drift.

```powershell
& "<skill-dir>\bin\codex-image2-5-windows-amd64.exe" edit `
  --image "input/product.png" `
  --prompt "Replace only the background with a warm studio backdrop. Keep the product, label, proportions, and edges unchanged." `
  --model gpt-image-2.5 `
  --size 1K `
  --quality auto `
  --out "output/imagegen/product-edited.png"
```

Repeat `--image` for multiple reference or compositing inputs. Use `--mask mask.png` for a localized edit when a compatible PNG mask is available. Preserve originals and always write edits to a new output path.

## Generate a batch

Read [references/batch-format.md](references/batch-format.md) before preparing a batch. Then run:

```powershell
& "<skill-dir>\bin\codex-image2-5-windows-amd64.exe" generate-batch `
  --input "tmp/imagegen/jobs.jsonl" `
  --out-dir "output/imagegen" `
  --concurrency 2
```

## Configuration and safety

- Read the API base from `CODEX_API_URL`; default to `https://apinebula.com`.
- Require `CODEX_API_KEY`. Never place it in a command, file, prompt, log, or response.
- If the key is absent, tell the user to set it locally and confirm when ready. Never ask them to paste it into chat.
- Default to model `gpt-image-2.5` and size `1K` when this conversation has not selected them. Switch with `--model` / `--size 1K|2K|4K` from the user's chat this turn. Never ask them to restart Codex or edit text config to change model or 画质.
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
