# Codex Image2 Skill（鹊桥转存）

让 Codex 通过自定义 API 地址和密钥生成或编辑图片。默认模型是 GPT Image 2（`gpt-image-2`），默认画质是 `1K`。对话里可以换成 GPT 或 Grok 的生图模型，以及 `1K` / `2K` / `4K`。**不含 Video。** 不要改配置文件，也不用重启客户端。

## 为什么做这个 Skill

最近使用 API 中转服务时，我发现不少中转站已经把 `gpt-image-2` 从常规模型列表中移出，导致 Codex 无法像以前一样直接发现并调用生图模型。

于是我写了这个 Skill。原理很简单：

1. 从环境变量读取 API 地址和密钥；
2. 直接调用 OpenAI 兼容的图片生成或编辑接口；
3. 将返回的图片保存到项目中；
4. 让 Codex 检查图片并展示最终结果。

仓库已经提供 Windows 和 macOS 的原生可执行文件。普通用户不需要安装 Python、Node.js、Go 或其他依赖。

## 功能

- 文生图
- 单图或多图编辑
- **对话里切换生图模型（不含 Video）**
  - GPT：`gpt-image-2`（默认）、`gpt-image-2.5`、`gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`
  - Grok：`grok-imagine`、`grok-imagine-image-2.0`（说「Grok」时默认这个）、`grok-imagine-image-quality`
- **对话里切换画质**：不说就用 `1K`。GPT 支持 `1K` / `2K` / `4K`；Grok 官方是 `1K` / `2K`，说 4K 会打到上游最高档（约 2816×1584，不是真 4K）
- 不用改配置文件，也不用重启 Codex
- 可选 PNG Mask 局部编辑
- JSONL 并发批量生图，单条任务可覆盖 `model` / `size`
- 支持 Base64 和 URL 两种图片响应
- 自动重试网络超时、429、5xx 和 524 错误
- 输出文件覆盖保护
- API Key 脱敏，不写入 Skill 或日志
- 内置 Windows x64/ARM64 与 macOS Intel/Apple Silicon 可执行文件

## 如何使用

这个项目包含：

- 开源项目：**codex-image2-skill**
- Skill 名称：**codex-image2**

### 1. 安装 Skill

最简单的方式是把本项目地址发给 Codex，让它帮你安装：

```text
请帮我安装这个 Skill：
https://github.com/daifei-0/codex-image2-skill
```

也可以手动安装。

Windows PowerShell：

```powershell
git clone https://github.com/daifei-0/codex-image2-skill.git
Copy-Item codex-image2-skill\codex-image2 "$HOME\.codex\skills\codex-image2" -Recurse
```

macOS / Linux：

```bash
git clone https://github.com/daifei-0/codex-image2-skill.git
cp -R codex-image2-skill/codex-image2 ~/.codex/skills/codex-image2
```

### 2. 配置 API 地址和密钥

在 PowerShell 中执行下面两条命令，可将环境变量永久保存到当前 Windows 用户：

```powershell
[Environment]::SetEnvironmentVariable("CODEX_API_URL", "你的API地址", "User")
[Environment]::SetEnvironmentVariable("CODEX_API_KEY", "你的API密钥", "User")
```

例如，你的 API 地址可能是：

```text
https://example.com
```

既可以填写服务根地址，也可以填写以 `/v1` 结尾的地址，Skill 会自动整理接口路径。

![配置 Codex Image2 环境变量](http://image.fengfengzhidao.com/fengfeng_110920260715224031.png?key=fengfengbuzhidao)

> **重启只针对 API 地址和密钥。** 配完这两项后完全退出再打开 Codex，密钥才会生效。之后换模型、换画质都在对话里说，不要去改配置文件，也不要再重启。

macOS / Linux 用户可以将以下内容加入自己的 shell 配置文件：

```bash
export CODEX_API_URL="你的API地址"
export CODEX_API_KEY="你的API密钥"
```

### 3. 对话里生图：怎么切换模型、怎么切换画质

安装好 Skill、配好密钥并重启过一次 Codex 之后，日常用法就是跟它说话。**不要把模型或画质写进配置文件。** 每次请求点名即可，Skill 会给这一次调用带上对应参数。

默认（这句话里不提模型和画质）：

- 模型：`gpt-image-2`（GPT Image 2）
- 画质：`1K`

可选生图模型如下，**不含** `grok-imagine-video` / `grok-imagine-video-1.5`。

**GPT**

| 你在对话里说 | 这一次实际用 |
| --- | --- |
| 不提 / GPT / GPT Image 2 | `gpt-image-2` |
| 2.5 / GPT Image 2.5 | `gpt-image-2.5` |
| 闪焰 / flare | `gpt-image-2.5-flare` |
| 日耀 / sunburst | `gpt-image-2.5-sunburst` |

**Grok**

| 你在对话里说 | 这一次实际用 |
| --- | --- |
| Grok（只说 Grok） | `grok-imagine-image-2.0` |
| Grok 2.0 / grok-imagine-image-2.0 | `grok-imagine-image-2.0` |
| Grok 高质量 / grok-imagine-image-quality | `grok-imagine-image-quality` |
| grok-imagine（点名这个 id） | `grok-imagine` |

画质单独说：

| 你在对话里说 | GPT | Grok |
| --- | --- | --- |
| 不提画质 | `1K`（约 1024×1024） | `1k`（实测 1024×1024） |
| 画质 2K | `2048×2048` | `2048×2048`（`resolution=2k` + `size=2048x2048`） |
| 画质 4K | 约 `3584×2016`（4K 档，不是严格 3840×2160） | 官方无 4K；会打到上游最高档，约 `2816×1584`，Skill 会说明不是真 4K |

同一段对话里下一句换说法就行，立即生效，不用重开客户端。

默认生图：

```text
使用 $codex-image2 生成一张图片：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

换成 GPT Image 2.5：

```text
使用 $codex-image2，模型用 2.5，生成一张图片：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

换成 GPT 闪焰 / 日耀：

```text
使用 $codex-image2，用闪焰，画质 2K：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

换成 Grok：

```text
使用 $codex-image2，用 Grok，画质 2K：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

Grok 高质量档：

```text
使用 $codex-image2，用 Grok 高质量：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

换成 4K 画质（GPT 才是真 4K；Grok 会落到 2K）：

```text
使用 $codex-image2，画质 4K，生成一张图片：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

模型和画质一起改：

```text
使用 $codex-image2，模型 gpt-image-2.5，画质 4K：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

同一轮对话里接着换：

```text
还是这只猫，改用 Grok，画质 2K 再出一张。
```

改图也可以带上模型和画质：

```text
使用 $codex-image2，用 Grok，画质 2K，修改这张图片：
只把背景替换成雪山，人物、服装、姿势和构图保持不变。
```

![使用 Codex Image2 生图](http://image.fengfengzhidao.com/fengfeng_110920260715224141.png?key=fengfengbuzhidao)

## CLI 用法

通常直接在 Codex 对话里指定 `$codex-image2`，并在同一句话里说模型和画质。不需要手动执行 CLI，也不需要为切换模型和画质改任何文本配置。

下面的命令只适合调试或自动化。

选择与你的系统匹配的文件：

| 系统 | 可执行文件 |
| --- | --- |
| Windows x64 | `codex-image2/bin/codex-image2-windows-amd64.exe` |
| Windows ARM64 | `codex-image2/bin/codex-image2-windows-arm64.exe` |
| macOS Intel | `codex-image2/bin/codex-image2-darwin-amd64` |
| macOS Apple Silicon | `codex-image2/bin/codex-image2-darwin-arm64` |

macOS 如果提示没有执行权限，运行：

```bash
chmod +x codex-image2/bin/codex-image2-darwin-*
```

生成图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" generate `
  --prompt "A tiny blue nebula inside a glass bottle" `
  --model gpt-image-2 `
  --size 1K `
  --quality auto `
  --out "output/imagegen/nebula.png"
```

换成 2.5 和 4K：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" generate `
  --prompt "A tiny blue nebula inside a glass bottle" `
  --model gpt-image-2.5 `
  --size 4K `
  --quality auto `
  --out "output/imagegen/nebula-4k.png"
```

换成 Grok 2K：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" generate `
  --prompt "A tiny blue nebula inside a glass bottle" `
  --model grok-imagine-image-2.0 `
  --size 2K `
  --out "output/imagegen/nebula-grok.png"
```

GPT 的 `--size` 对照：`1K` → `1024x1024`，`2K` → `2048x2048`，`4K` → `3840x2160`（上游常见落地 `3584x2016`）。Grok 同时发 `resolution` 和 `size`：`1K` → `1k`/`1024x1024`，`2K` → `2k`/`2048x2048`，`4K` → `2k`/`3840x2160`（6rr 最高约 `2816x1584`）。`--quality` 仍是 `low|medium|high|auto`，和画质档位不是一回事。

编辑图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" edit `
  --image "input.png" `
  --prompt "Replace only the background with a warm studio backdrop" `
  --out "output/imagegen/edited.png"
```

批量任务格式和完整工作流请查看 [`codex-image2/SKILL.md`](codex-image2/SKILL.md) 和 [`batch-format.md`](codex-image2/references/batch-format.md)。

## 从源码构建

普通用户不需要执行这一步。开发者安装 Go 后，可以运行：

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o codex-image2/bin/codex-image2-windows-amd64.exe codex-image2/src/image_gen.go
```

源码只使用 Go 标准库。

## 超简单的方式

如果觉得安装 Skill 和配置环境变量还是太麻烦，也可以直接使用鹊桥的在线生图页面：

### [https://cdn.5202828.xyz](https://cdn.5202828.xyz)

登录鹊桥后打开 [在线生图](https://cdn.5202828.xyz/draw/)，支持文生图和图生图，同样使用 `gpt-image-2`，打开网页即可使用，不需要安装任何东西。

![鹊桥在线生图](docs/images/queqiao-draw.jpg)

本 Skill 配置里的 `CODEX_API_URL` 也可以直接填鹊桥地址 `https://cdn.5202828.xyz/v1`，密钥使用在鹊桥创建的令牌。

## 常见问题

### 配置后仍提示没有 API Key

完全退出 Codex 后重新启动。已经打开的 Codex 进程不会自动读取新设置的用户环境变量。换模型、换画质不走这条路径，对话里说就行。

### 切换模型和画质要不要重启？

不要。只有第一次配置 `CODEX_API_URL` / `CODEX_API_KEY` 才需要重启。之后在对话里说「用 2.5」「用 Grok」「用闪焰」「画质 4K」即可。

### 支持哪些模型？Video 呢？

支持截图里的生图模型：GPT 四个（`gpt-image-2`、`gpt-image-2.5`、`gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`）和 Grok 三个（`grok-imagine`、`grok-imagine-image-2.0`、`grok-imagine-image-quality`）。**不支持** `grok-imagine-video` 和 `grok-imagine-video-1.5`。

### Grok 能开 2K / 4K 吗？

Grok Imagine 官方分辨率只有 `1k` 和 `2k`，没有 4K。鹊桥侧已经打开 Grok 通道的 body 透传，并把 OpenAI 的 `size` 补成 6rr 认的 `resolution`。现在：

- 2K 会出 **2048×2048**
- 4K 没有真 4K，会出上游最高档，约 **2816×1584**。Skill 仍会提示这不是 4K，不会把 2K 图说成 4K

GPT 四个生图模型 1K / 2K 按像素出图；请求 4K 时 OpenAI/6rr 会落到约 `3584×2016`（仍按 4K 档计，不是严格 3840×2160）。这不是鹊桥把 `size` 剥掉，也不是肥嘟嘟改了尺寸。

### 接口返回 524 或超时

这通常表示中转服务的图片生成耗时超过了网关限制。可以尝试 `--quality low`、`--size 1K`、减少批量并发，或稍后重试。

### 是否支持所有中转站

中转服务需要兼容以下接口，并提供你实际传入的生图模型（默认 `gpt-image-2`，也可传 GPT 2.5/闪焰/日耀或 Grok Imagine 生图）：

```text
POST /v1/images/generations
POST /v1/images/edits
```

不同服务的参数支持和稳定性可能存在差异。

## 安全说明

- 不要把真实 API Key 提交到 GitHub。
- 不要把 Key 写进 Skill、提示词、截图或聊天消息。
- 建议为不同服务使用独立密钥，并定期轮换。
- 本 Skill 只从 `CODEX_API_KEY` 环境变量读取密钥，不会主动保存密钥。

## License

[MIT](LICENSE)

## 关于本仓库

本仓库是 **鹊桥** 对开源项目 [fengfengzhidao/codex-image2-skill](https://github.com/fengfengzhidao/codex-image2-skill) 的 **转存备份**。

- 原作者：[fengfengzhidao](https://github.com/fengfengzhidao)，原仓库地址：https://github.com/fengfengzhidao/codex-image2-skill
- 转存目的：鹊桥的用户会长期通过本地址安装这个 Skill，为避免原仓库失效、删除或不可访问导致安装失败，这里做了一份完整转存。
- 转存内容：Skill 目录、可执行文件、源码和许可证来自原仓库；仓库名和 Skill 名（`codex-image2`）未改。本 README 补充了转存说明、安装地址，以及对话里切换生图模型和 1K/2K/4K 画质的用法。
- 许可证：沿用原项目的 [MIT](LICENSE) 许可，版权归原作者所有。

如果你需要最新版本或想参与开发，请优先访问原仓库。
