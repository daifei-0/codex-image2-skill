package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIURL    = "https://apinebula.com"
	defaultModel     = "gpt-image-2.5"
	defaultGrokModel = "grok-imagine-image-2.0"
	defaultSize      = "1K"
	defaultQuality   = "auto"
	defaultOutDir    = "output/imagegen"
)

// sizePresets 把用户说的 1K/2K/4K 换成 Images API 真正认的像素。
// 4K 用 3840x2160：官方长边上限 3840，正方形 3840x3840 会超过像素总量上限。
var sizePresets = map[string]string{
	"1K": "1024x1024",
	"2K": "2048x2048",
	"4K": "3840x2160",
}

var retryable = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true, 524: true}

// 对话别名 → 网关模型 id。更具体的写在前面。不含 Video。
var modelAliases = []struct {
	alias string
	id    string
}{
	{"gpt-image-2.5-flare", "gpt-image-2.5-flare"},
	{"gpt image 2.5 flare", "gpt-image-2.5-flare"},
	{"2.5 flare", "gpt-image-2.5-flare"},
	{"image 2.5 flare", "gpt-image-2.5-flare"},
	{"image 2.5 闪焰", "gpt-image-2.5-flare"},
	{"flare", "gpt-image-2.5-flare"},
	{"闪焰", "gpt-image-2.5-flare"},
	{"闪焰版", "gpt-image-2.5-flare"},
	{"gpt-image-2.5-sunburst", "gpt-image-2.5-sunburst"},
	{"gpt image 2.5 sunburst", "gpt-image-2.5-sunburst"},
	{"2.5 sunburst", "gpt-image-2.5-sunburst"},
	{"image 2.5 sunburst", "gpt-image-2.5-sunburst"},
	{"image 2.5 日耀", "gpt-image-2.5-sunburst"},
	{"sunburst", "gpt-image-2.5-sunburst"},
	{"日耀", "gpt-image-2.5-sunburst"},
	{"日耀版", "gpt-image-2.5-sunburst"},
	{"gpt-image-2.5", "gpt-image-2.5"},
	{"gpt image 2.5", "gpt-image-2.5"},
	{"gptimage 2.5", "gpt-image-2.5"},
	{"image 2.5", "gpt-image-2.5"},
	{"image2.5", "gpt-image-2.5"},
	{"img 2.5", "gpt-image-2.5"},
	{"image 二点五", "gpt-image-2.5"},
	{"二点五", "gpt-image-2.5"},
	{"2.5", "gpt-image-2.5"},
	{"gpt-image-2", "gpt-image-2"},
	{"gpt image 2", "gpt-image-2"},
	{"gptimage 2", "gpt-image-2"},
	{"gptimage2", "gpt-image-2"},
	{"image 2", "gpt-image-2"},
	{"image2", "gpt-image-2"},
	{"image 2.0", "gpt-image-2"},
	{"gpt image 2.0", "gpt-image-2"},
	{"img 2", "gpt-image-2"},
	{"image 二", "gpt-image-2"},
	{"2", "gpt-image-2"},
	{"gpt", defaultModel},
	{"gpt image", defaultModel},
	{"grok-imagine-image-quality", "grok-imagine-image-quality"},
	{"grok imagine image quality", "grok-imagine-image-quality"},
	{"grok imagine quality", "grok-imagine-image-quality"},
	{"grok quality", "grok-imagine-image-quality"},
	{"imagine quality", "grok-imagine-image-quality"},
	{"grok 高质量", "grok-imagine-image-quality"},
	{"grok 高质量版", "grok-imagine-image-quality"},
	{"grok 质量版", "grok-imagine-image-quality"},
	{"grok-imagine-image-2.0", "grok-imagine-image-2.0"},
	{"grok imagine image 2.0", "grok-imagine-image-2.0"},
	{"grok imagine 2.0", "grok-imagine-image-2.0"},
	{"grok imagine image 2", "grok-imagine-image-2.0"},
	{"grok 2.0", "grok-imagine-image-2.0"},
	{"grok 2", "grok-imagine-image-2.0"},
	{"imagine 2.0", "grok-imagine-image-2.0"},
	{"grok-imagine", "grok-imagine"},
	{"grok imagine", "grok-imagine"},
	{"grok", defaultGrokModel},
}

type commonArgs struct {
	model       string
	size        string
	quality     string
	n           int
	outDir      string
	force       bool
	dryRun      bool
	maxAttempts int
	timeout     time.Duration
}

type apiResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}

type batchJob struct {
	Prompt  string `json:"prompt"`
	Model   string `json:"model,omitempty"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	N       int    `json:"n,omitempty"`
	Out     string `json:"out,omitempty"`
}

func endpoint(base, operation string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = defaultAPIURL
	}
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	return base + "/images/" + operation
}

// resolveSize 接受 1K/2K/4K、auto，或 WIDTHxHEIGHT。发给上游的永远是后两种。
func resolveSize(size string) (string, error) {
	size = normalizeSize(size)
	if size == "" {
		size = defaultSize
	}
	if preset, ok := sizePresets[strings.ToUpper(size)]; ok {
		return preset, nil
	}
	if strings.EqualFold(size, "auto") {
		return "auto", nil
	}
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return "", errors.New("size must be 1K, 2K, 4K, auto, or WIDTHxHEIGHT")
	}
	w, e1 := strconv.Atoi(parts[0])
	h, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || w < 1 || h < 1 {
		return "", errors.New("size must be 1K, 2K, 4K, auto, or WIDTHxHEIGHT")
	}
	return fmt.Sprintf("%dx%d", w, h), nil
}

func normalizeWidth(value string) string {
	return strings.Map(func(character rune) rune {
		if character >= '！' && character <= '～' {
			return character - 0xFEE0
		}
		return character
	}, value)
}

func normalizeKey(value string) string {
	value = strings.ToLower(normalizeWidth(value))
	value = strings.NewReplacer("-", "", "_", "").Replace(value)
	return strings.Join(strings.Fields(value), "")
}

func normalizeSize(size string) string {
	size = strings.ToUpper(strings.Join(strings.Fields(normalizeWidth(size)), ""))
	size = strings.ReplaceAll(size, "×", "X")
	switch size {
	case "一K":
		return "1K"
	case "二K", "两K":
		return "2K"
	case "四K":
		return "4K"
	}
	return size
}

func isVideoModel(model string) bool {
	key := normalizeKey(model)
	return strings.Contains(key, "video") || strings.Contains(key, "视频")
}

func isGrokImage(model string) bool {
	k := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(k, "grok-imagine") && !isVideoModel(k)
}

// resolveModel 把对话里的叫法收成网关模型名。Video 直接拒绝；认不出的原样传。
func resolveModel(model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultModel, nil
	}
	if isVideoModel(model) {
		return "", errors.New("video models are not supported; use a GPT or Grok image model")
	}
	key := normalizeKey(model)
	if key == "高质量" || key == "高清" || key == "highquality" {
		return "", errors.New("quality alone does not select a model; keep the current model and use --quality high, or explicitly choose Grok 高质量")
	}
	for _, item := range modelAliases {
		if key == normalizeKey(item.alias) {
			return item.id, nil
		}
	}
	return model, nil
}

func grokResolution(size string) (res, pixel, note string) {
	size = normalizeSize(size)
	if size == "" {
		size = defaultSize
	}
	switch strings.ToUpper(size) {
	case "1K", "AUTO":
		return "1k", "1024x1024", ""
	case "2K":
		return "2k", "2048x2048", ""
	case "4K":
		// Do not send resolution=4k: 6rr ignores it and returns 1024x1024.
		// size=3840x2160 is the highest slot (typically 2816x1584).
		return "2k", "3840x2160", "Grok Imagine has no 4K; requested the highest available slot"
	}
	resolved, err := resolveSize(size)
	if err != nil {
		return "1k", "1024x1024", ""
	}
	parts := strings.Split(strings.ToLower(resolved), "x")
	if len(parts) == 2 {
		w, e1 := strconv.Atoi(parts[0])
		h, e2 := strconv.Atoi(parts[1])
		if e1 == nil && e2 == nil && (w >= 3000 || h >= 3000) {
			return "2k", resolved, ""
		}
		if e1 == nil && e2 == nil && (w >= 1536 || h >= 1536) {
			return "2k", resolved, ""
		}
	}
	return "1k", "1024x1024", ""
}

type imageSpec struct {
	model   string
	payload map[string]any
	fields  map[string]string
	extra   map[string]any
}

func prepareImageRequest(prompt string, args commonArgs) (imageSpec, error) {
	model, err := resolveModel(args.model)
	if err != nil {
		return imageSpec{}, err
	}
	spec := imageSpec{
		model:   model,
		payload: map[string]any{"model": model, "prompt": prompt, "n": args.n},
		fields:  map[string]string{"model": model, "prompt": prompt, "n": strconv.Itoa(args.n)},
		extra:   map[string]any{"model": model, "size": args.size},
	}
	if isGrokImage(model) {
		res, pixel, note := grokResolution(args.size)
		// new-api drops unknown fields like resolution unless pass_through is on.
		// 6rr honors both resolution=2k and OpenAI size=2048x2048; send both.
		spec.payload["resolution"] = res
		spec.payload["size"] = pixel
		spec.fields["resolution"] = res
		spec.fields["size"] = pixel
		spec.extra["family"] = "grok"
		spec.extra["resolution"] = res
		spec.extra["resolved_size"] = pixel
		if note != "" {
			spec.extra["note"] = note
		}
		if model == "grok-imagine-image-2.0" {
			q := args.quality
			if q == "high" {
				q = "auto"
			}
			spec.payload["quality"] = q
			spec.fields["quality"] = q
			spec.extra["quality"] = q
		}
		return spec, nil
	}
	resolved, err := resolveSize(args.size)
	if err != nil {
		return imageSpec{}, err
	}
	spec.payload["size"] = resolved
	spec.payload["quality"] = args.quality
	spec.fields["size"] = resolved
	spec.fields["quality"] = args.quality
	spec.extra["resolved_size"] = resolved
	spec.extra["quality"] = args.quality
	if strings.HasPrefix(strings.ToLower(model), "gpt-image") {
		spec.extra["family"] = "gpt"
	} else {
		spec.extra["family"] = "other"
	}
	return spec, nil
}

func resultJSON(spec imageSpec, outputs any) map[string]any {
	out := map[string]any{}
	for k, v := range spec.extra {
		out[k] = v
	}
	out["outputs"] = outputs
	return out
}

func requestedBand(size string) string {
	size = normalizeSize(size)
	u := strings.ToUpper(size)
	if u == "1K" || u == "2K" || u == "4K" {
		return u
	}
	if strings.EqualFold(size, "auto") || size == "" {
		return "1K"
	}
	resolved, err := resolveSize(size)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.ToLower(resolved), "x")
	if len(parts) != 2 {
		return ""
	}
	w, e1 := strconv.Atoi(parts[0])
	h, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil {
		return ""
	}
	return qualityBand(w, h)
}

func qualityBand(w, h int) string {
	long := w
	if h > w {
		long = h
	}
	switch {
	case long >= 3000:
		return "4K"
	case long >= 1800:
		return "2K"
	default:
		return "1K"
	}
}

func bandRank(band string) int {
	switch band {
	case "4K":
		return 3
	case "2K":
		return 2
	default:
		return 1
	}
}

func inspectImage(data []byte) (int, int, string) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, ""
	}
	return cfg.Width, cfg.Height, format
}

func attachActual(spec *imageSpec, images [][]byte, asked string) {
	if spec == nil || len(images) == 0 {
		return
	}
	w, h, format := inspectImage(images[0])
	if w < 1 || h < 1 {
		return
	}
	actual := qualityBand(w, h)
	spec.extra["actual_size"] = fmt.Sprintf("%dx%d", w, h)
	spec.extra["actual_format"] = format
	spec.extra["actual_quality"] = actual
	askedBand := requestedBand(asked)
	if askedBand == "" || bandRank(askedBand) <= bandRank(actual) {
		return
	}
	warn := fmt.Sprintf("上游 API 不支持 %s 的 %s 画质（未按请求分辨率出图，实际 %dx%d，相当于 %s）。请改用 %s，或换 GPT 生图模型。", spec.model, askedBand, w, h, actual, actual)
	spec.extra["warning"] = warn
	fmt.Fprintln(os.Stderr, "warning:", warn)
}

func stringifyErrorValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		if msg, ok := t["message"]; ok {
			return stringifyErrorValue(msg)
		}
	}
	return ""
}

func extractJSONError(body []byte) string {
	var top map[string]any
	if err := json.Unmarshal(body, &top); err != nil {
		s := strings.TrimSpace(string(body))
		if len(s) > 240 {
			s = s[:240]
		}
		return s
	}
	if msg := stringifyErrorValue(top["error"]); msg != "" {
		return msg
	}
	return stringifyErrorValue(top["message"])
}

func formatAPIError(status int, body []byte) string {
	msg := extractJSONError(body)
	if msg == "" {
		return fmt.Sprintf("上游 API 返回 HTTP %d", status)
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "size") || strings.Contains(lower, "resolution") || strings.Contains(lower, "dimension") || strings.Contains(lower, "4k") || strings.Contains(lower, "2k") || strings.Contains(lower, "画质") {
		return fmt.Sprintf("上游 API 不支持这次请求的画质或尺寸：%s", msg)
	}
	return fmt.Sprintf("上游 API 返回 HTTP %d：%s", status, msg)
}

func validateCommon(args commonArgs) error {
	if _, err := resolveModel(args.model); err != nil {
		return err
	}
	if args.n < 1 || args.n > 10 {
		return errors.New("n must be from 1 to 10")
	}
	if args.maxAttempts < 1 {
		return errors.New("--max-attempts must be at least 1")
	}
	if args.timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if args.quality != "low" && args.quality != "medium" && args.quality != "high" && args.quality != "auto" {
		return errors.New("quality must be low, medium, high, or auto")
	}
	if _, err := resolveSize(args.size); err != nil {
		return err
	}
	return nil
}

func promptValue(prompt, promptFile string) (string, error) {
	if (prompt == "") == (promptFile == "") {
		return "", errors.New("provide exactly one of --prompt or --prompt-file")
	}
	if promptFile != "" {
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("could not read prompt file: %w", err)
		}
		prompt = string(data)
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("prompt must not be empty")
	}
	return prompt, nil
}

func outputPaths(out, outDir, prompt string, n int) []string {
	if out == "" {
		sum := sha256.Sum256([]byte(prompt))
		out = "image-" + hex.EncodeToString(sum[:6]) + ".png"
	}
	if filepath.Dir(out) == "." {
		out = filepath.Join(outDir, out)
	}
	ext := filepath.Ext(out)
	if ext == "" {
		ext = ".png"
		out += ext
	}
	if n == 1 {
		return []string{out}
	}
	base := strings.TrimSuffix(out, ext)
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("%s-%d%s", base, i+1, ext)
	}
	return paths
}

func checkOutputs(paths []string, force bool) error {
	if force {
		return nil
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing file: %s", path)
		}
	}
	return nil
}

func apiKey() (string, error) {
	key := strings.TrimSpace(os.Getenv("CODEX_API_KEY"))
	if key == "" {
		return "", errors.New("CODEX_API_KEY is not set; set it locally, then retry")
	}
	return key, nil
}

func doRequest(client *http.Client, makeRequest func() (*http.Request, error), maxAttempts int) ([]byte, error) {
	var last string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := makeRequest()
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return nil, errors.New("could not read API response")
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return body, nil
			}
			last = formatAPIError(resp.StatusCode, body)
			if !retryable[resp.StatusCode] || attempt == maxAttempts {
				return nil, errors.New(last)
			}
		} else {
			last = "API request failed: network error or timeout"
			if attempt == maxAttempts {
				return nil, errors.New(last)
			}
		}
		time.Sleep(time.Duration(750*(1<<(attempt-1))) * time.Millisecond)
	}
	return nil, errors.New(last)
}

func decodeResponse(raw []byte, client *http.Client) ([][]byte, error) {
	var result apiResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.New("API returned invalid JSON")
	}
	if len(result.Data) == 0 {
		return nil, errors.New("API response contains no image data")
	}
	images := make([][]byte, 0, len(result.Data))
	for _, item := range result.Data {
		if item.B64JSON != "" {
			data, err := base64.StdEncoding.DecodeString(item.B64JSON)
			if err != nil {
				return nil, errors.New("API returned invalid base64 image data")
			}
			images = append(images, data)
		} else if item.URL != "" {
			if _, err := url.ParseRequestURI(item.URL); err != nil {
				return nil, errors.New("API returned an invalid image URL")
			}
			resp, err := client.Get(item.URL)
			if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if resp != nil {
					resp.Body.Close()
				}
				return nil, errors.New("could not download image URL")
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, errors.New("could not read downloaded image")
			}
			images = append(images, data)
		} else {
			return nil, errors.New("API image entry contains neither b64_json nor url")
		}
	}
	return images, nil
}

func saveImages(images [][]byte, paths []string) ([]string, error) {
	if len(images) != len(paths) {
		return nil, fmt.Errorf("API returned %d image(s), expected %d", len(images), len(paths))
	}
	abs := make([]string, len(paths))
	for i, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("could not create output directory: %w", err)
		}
		if err := os.WriteFile(path, images[i], 0644); err != nil {
			return nil, fmt.Errorf("could not write output: %w", err)
		}
		abs[i], _ = filepath.Abs(path)
	}
	return abs, nil
}

func generate(prompt, out string, args commonArgs) (map[string]any, error) {
	if err := validateCommon(args); err != nil {
		return nil, err
	}
	paths := outputPaths(out, args.outDir, prompt, args.n)
	if err := checkOutputs(paths, args.force); err != nil {
		return nil, err
	}
	ep := endpoint(os.Getenv("CODEX_API_URL"), "generations")
	spec, err := prepareImageRequest(prompt, args)
	if err != nil {
		return nil, err
	}
	if args.dryRun {
		return map[string]any{"dry_run": true, "endpoint": ep, "payload": spec.payload, "outputs": paths}, nil
	}
	key, err := apiKey()
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(spec.payload)
	client := &http.Client{Timeout: args.timeout}
	raw, err := doRequest(client, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, ep, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Content-Type", "application/json")
		}
		return req, err
	}, args.maxAttempts)
	if err != nil {
		return nil, err
	}
	images, err := decodeResponse(raw, client)
	if err != nil {
		return nil, err
	}
	attachActual(&spec, images, args.size)
	outputs, err := saveImages(images, paths)
	if err != nil {
		return nil, err
	}
	return resultJSON(spec, outputs), nil
}

func edit(prompt string, imagePaths []string, mask, out string, args commonArgs) (map[string]any, error) {
	if err := validateCommon(args); err != nil {
		return nil, err
	}
	for _, path := range append(append([]string{}, imagePaths...), mask) {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return nil, fmt.Errorf("input file not found: %s", path)
		}
	}
	paths := outputPaths(out, args.outDir, prompt, args.n)
	if err := checkOutputs(paths, args.force); err != nil {
		return nil, err
	}
	ep := endpoint(os.Getenv("CODEX_API_URL"), "edits")
	spec, err := prepareImageRequest(prompt, args)
	if err != nil {
		return nil, err
	}
	if args.dryRun {
		return map[string]any{"dry_run": true, "endpoint": ep, "fields": spec.fields, "images": imagePaths, "mask": mask, "outputs": paths}, nil
	}
	key, err := apiKey()
	if err != nil {
		return nil, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range spec.fields {
		_ = writer.WriteField(name, value)
	}
	for _, path := range imagePaths {
		part, err := writer.CreateFormFile("image", filepath.Base(path))
		if err != nil {
			return nil, err
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}
	if mask != "" {
		part, err := writer.CreateFormFile("mask", filepath.Base(mask))
		if err != nil {
			return nil, err
		}
		file, err := os.Open(mask)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}
	writer.Close()
	contentType := writer.FormDataContentType()
	data := append([]byte(nil), body.Bytes()...)
	client := &http.Client{Timeout: args.timeout}
	raw, err := doRequest(client, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, ep, bytes.NewReader(data))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Content-Type", contentType)
		}
		return req, err
	}, args.maxAttempts)
	if err != nil {
		return nil, err
	}
	images, err := decodeResponse(raw, client)
	if err != nil {
		return nil, err
	}
	attachActual(&spec, images, args.size)
	outputs, err := saveImages(images, paths)
	if err != nil {
		return nil, err
	}
	return resultJSON(spec, outputs), nil
}

func printJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func parseCommon(fs *flag.FlagSet, argv []string, args *commonArgs) error {
	var timeout float64
	fs.StringVar(&args.model, "model", defaultModel, "image model (default gpt-image-2.5; Image 2, Image 2.5, GPT or Grok image ids, not video)")
	fs.StringVar(&args.size, "size", defaultSize, "1K, 2K, 4K, auto, or WIDTHxHEIGHT")
	fs.StringVar(&args.quality, "quality", defaultQuality, "low, medium, high, or auto")
	fs.IntVar(&args.n, "n", 1, "number of variants (1-10)")
	fs.StringVar(&args.outDir, "out-dir", defaultOutDir, "default output directory")
	fs.BoolVar(&args.force, "force", false, "overwrite existing outputs")
	fs.BoolVar(&args.dryRun, "dry-run", false, "validate without network access")
	fs.IntVar(&args.maxAttempts, "max-attempts", 3, "maximum attempts")
	fs.Float64Var(&timeout, "timeout", 150, "request timeout in seconds")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	args.timeout = time.Duration(timeout * float64(time.Second))
	return nil
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func runGenerate(argv []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	var prompt, promptFile, out string
	fs.StringVar(&prompt, "prompt", "", "prompt text")
	fs.StringVar(&promptFile, "prompt-file", "", "UTF-8 prompt file")
	fs.StringVar(&out, "out", "", "output path")
	var args commonArgs
	if err := parseCommon(fs, argv, &args); err != nil {
		return err
	}
	finalPrompt, err := promptValue(prompt, promptFile)
	if err != nil {
		return err
	}
	result, err := generate(finalPrompt, out, args)
	if err == nil {
		printJSON(result)
	}
	return err
}

func runEdit(argv []string) error {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	var images stringList
	var prompt, promptFile, mask, out string
	fs.Var(&images, "image", "input image; repeat for multiple images")
	fs.StringVar(&mask, "mask", "", "optional PNG mask")
	fs.StringVar(&prompt, "prompt", "", "prompt text")
	fs.StringVar(&promptFile, "prompt-file", "", "UTF-8 prompt file")
	fs.StringVar(&out, "out", "", "output path")
	var args commonArgs
	if err := parseCommon(fs, argv, &args); err != nil {
		return err
	}
	if len(images) == 0 {
		return errors.New("provide at least one --image")
	}
	finalPrompt, err := promptValue(prompt, promptFile)
	if err != nil {
		return err
	}
	result, err := edit(finalPrompt, images, mask, out, args)
	if err == nil {
		printJSON(result)
	}
	return err
}

func runBatch(argv []string) error {
	fs := flag.NewFlagSet("generate-batch", flag.ContinueOnError)
	var input string
	var concurrency int
	var failFast bool
	fs.StringVar(&input, "input", "", "JSONL input path")
	fs.IntVar(&concurrency, "concurrency", 2, "parallel jobs")
	fs.BoolVar(&failFast, "fail-fast", false, "stop scheduling after a failure")
	var args commonArgs
	if err := parseCommon(fs, argv, &args); err != nil {
		return err
	}
	if input == "" || concurrency < 1 {
		return errors.New("--input is required and --concurrency must be at least 1")
	}
	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("could not read batch input: %w", err)
	}
	defer file.Close()
	var jobs []batchJob
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var job batchJob
		if err := json.Unmarshal(scanner.Bytes(), &job); err != nil || strings.TrimSpace(job.Prompt) == "" {
			return errors.New("each JSONL line must contain a non-empty prompt")
		}
		jobs = append(jobs, job)
	}
	if err := scanner.Err(); err != nil || len(jobs) == 0 {
		return errors.New("batch input contains no valid jobs")
	}
	type result struct {
		Index int            `json:"index"`
		OK    bool           `json:"ok"`
		Data  map[string]any `json:"data,omitempty"`
		Error string         `json:"error,omitempty"`
	}
	results := make([]result, len(jobs))
	queue := make(chan int)
	var wg sync.WaitGroup
	var failed bool
	var mu sync.Mutex
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range queue {
				job := jobs[index]
				jobArgs := args
				if job.Model != "" {
					jobArgs.model = job.Model
				}
				if job.Size != "" {
					jobArgs.size = job.Size
				}
				if job.Quality != "" {
					jobArgs.quality = job.Quality
				}
				if job.N != 0 {
					jobArgs.n = job.N
				}
				data, err := generate(strings.TrimSpace(job.Prompt), job.Out, jobArgs)
				results[index] = result{Index: index + 1, OK: err == nil, Data: data}
				if err != nil {
					results[index].Error = err.Error()
					mu.Lock()
					failed = true
					mu.Unlock()
				}
			}
		}()
	}
	for index := range jobs {
		mu.Lock()
		stop := failFast && failed
		mu.Unlock()
		if stop {
			break
		}
		queue <- index
	}
	close(queue)
	wg.Wait()
	succeeded, failures := 0, 0
	for _, item := range results {
		if item.Index == 0 {
			continue
		}
		if item.OK {
			succeeded++
		} else {
			failures++
		}
	}
	printJSON(map[string]any{"jobs": results, "succeeded": succeeded, "failed": failures})
	if failures > 0 {
		return errors.New("one or more batch jobs failed")
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "Codex Image 2.5")
	fmt.Fprintln(os.Stderr, "Usage: codex-image2-5 <generate|generate-batch|edit> [options]")
	fmt.Fprintln(os.Stderr, "Defaults: --model gpt-image-2.5  --size 1K  --quality auto")
	fmt.Fprintln(os.Stderr, "GPT:   gpt-image-2  gpt-image-2.5  gpt-image-2.5-flare  gpt-image-2.5-sunburst")
	fmt.Fprintln(os.Stderr, "Grok:  grok-imagine  grok-imagine-image-2.0  grok-imagine-image-quality")
	fmt.Fprintln(os.Stderr, "Video models are not supported.")
	fmt.Fprintln(os.Stderr, "GPT size: 1K=1024x1024  2K=2048x2048  4K=3840x2160 (upstream may deliver 3584x2016)")
	fmt.Fprintln(os.Stderr, "Grok: 1K=1024  2K=2048  4K requests highest available slot (not true 4K)")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "generate":
		err = runGenerate(os.Args[2:])
	case "generate-batch":
		err = runBatch(os.Args[2:])
	case "edit":
		err = runEdit(os.Args[2:])
	case "--help", "-h", "help":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}
