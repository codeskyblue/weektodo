package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

// Client calls OpenAI-compatible chat completion endpoints.
type Client struct {
	http *http.Client
}

// New creates a client with a generous timeout for report generation.
func New() *Client {
	return &Client{http: &http.Client{Timeout: 90 * time.Second}}
}

// ValidateEndpoint checks the URL and credentials needed to reach the API.
func ValidateEndpoint(cfg model.AIConfig, lang i18n.Language) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("%s", i18n.T(lang, "请先填写 AI Base URL", "Fill in the AI Base URL first"))
	}
	parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s", i18n.T(lang, "AI Base URL 必须是 http:// 或 https:// 地址", "AI Base URL must be an http:// or https:// address"))
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("%s", i18n.T(lang, "请先填写 API Key", "Fill in the API key first"))
	}
	return nil
}

// ValidateConfig checks the complete chat endpoint configuration.
func ValidateConfig(cfg model.AIConfig, lang i18n.Language) error {
	if err := ValidateEndpoint(cfg, lang); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("%s", i18n.T(lang, "请先填写模型名称", "Fill in the model name first"))
	}
	return nil
}

// Models fetches the OpenAI-compatible model list from the configured endpoint.
func (c *Client) Models(ctx context.Context, cfg model.AIConfig, lang i18n.Language) ([]string, error) {
	if err := ValidateEndpoint(cfg, lang); err != nil {
		return nil, err
	}
	endpoint := modelsEndpoint(cfg.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(lang, "构建模型列表请求失败", "Failed to build the model list request"), err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(lang, "获取模型列表失败", "Failed to fetch the model list"), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 300 {
			message = message[:300] + "…"
		}
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("%s %d: %s", i18n.T(lang, "模型列表接口返回", "Model list endpoint returned"), resp.StatusCode, message)
	}
	ids := parseModelIDs(raw)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s", i18n.T(lang, "接口没有返回模型列表", "The endpoint returned no models"))
	}
	return ids, nil
}

func modelsEndpoint(base string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint = strings.TrimSuffix(endpoint, "/chat/completions")
	if strings.HasSuffix(endpoint, "/models") {
		return endpoint
	}
	return endpoint + "/models"
}

func parseModelIDs(raw []byte) []string {
	var payload struct {
		Data   json.RawMessage `json:"data"`
		Models json.RawMessage `json:"models"`
	}
	_ = json.Unmarshal(raw, &payload)
	seen := map[string]bool{}
	out := []string{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	collect := func(data json.RawMessage) {
		if len(data) == 0 {
			return
		}
		var objects []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &objects) == nil {
			for _, item := range objects {
				if item.ID != "" {
					add(item.ID)
				} else {
					add(item.Name)
				}
			}
			return
		}
		var stringsList []string
		if json.Unmarshal(data, &stringsList) == nil {
			for _, item := range stringsList {
				add(item)
			}
		}
	}
	collect(payload.Data)
	collect(payload.Models)
	sort.Strings(out)
	return out
}

// Polish generates a weekly report from the local report and completed task tree.
func (c *Client) Polish(ctx context.Context, cfg model.AIConfig, week *model.Week, report string, lang i18n.Language) (string, error) {
	prompt := WeeklyPrompt(week, report, lang)
	system := i18n.T(lang, "你是一名严谨、务实的中文周报助手。", "You are a rigorous and practical weekly report assistant.")
	return c.complete(ctx, cfg, system, prompt, 1800, lang)
}

// Test performs a tiny completion to verify the endpoint and credentials.
func (c *Client) Test(ctx context.Context, cfg model.AIConfig, lang i18n.Language) (string, error) {
	return c.complete(ctx, cfg, i18n.T(lang, "你是连接测试助手。", "You are a connection test assistant."), i18n.T(lang, "只回复 OK。", "Reply with OK only."), 8, lang)
}

func (c *Client) complete(ctx context.Context, cfg model.AIConfig, system, user string, maxTokens int, lang i18n.Language) (string, error) {
	if err := ValidateConfig(cfg, lang); err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	body := map[string]any{
		"model": strings.TrimSpace(cfg.Model),
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.3,
		"max_tokens":  maxTokens,
		"stream":      false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(lang, "构建 AI 请求失败", "Failed to build the AI request"), err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(lang, "连接 AI 接口失败", "Failed to connect to the AI endpoint"), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 500 {
			message = message[:500] + "…"
		}
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("%s %d: %s", i18n.T(lang, "AI 接口返回", "AI endpoint returned"), resp.StatusCode, message)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(lang, "解析 AI 响应失败", "Failed to parse the AI response"), err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("%s: %s", i18n.T(lang, "AI 接口返回错误", "AI endpoint error"), parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%s", i18n.T(lang, "AI 接口没有返回内容", "The AI endpoint returned no content"))
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("%s", i18n.T(lang, "AI 接口返回了空内容", "The AI endpoint returned empty content"))
	}
	return content, nil
}

// WeeklyPrompt builds the prompt used for report polishing in the selected language.
func WeeklyPrompt(week *model.Week, report string, lang i18n.Language) string {
	tree := truncate(todo.DoneText(week), 12000, lang)
	if lang == i18n.En {
		return fmt.Sprintf(`Generate a concise, factual English weekly report from the completed tasks below.

Requirements:
1. Use Markdown, 200-400 words.
2. Group related work by theme. Do not list every task and do not invent anything.
3. Keep key outcomes, numbers and important details.
4. Output only the report body, without a preamble.

Period: %s ~ %s (%s)
Completed tasks (hierarchy):
%s

Local summary:
%s`, week.Start, week.End, week.ID, tree, report)
	}
	return fmt.Sprintf(`请根据下面这一周的任务完成情况，生成一份简洁、专业、真实的中文周报。

要求：
1. 使用 Markdown，200-400 字。
2. 按工作主题归纳同类任务，不要逐条罗列，不要虚构未出现的内容。
3. 保留关键成果、数字和重要细节。
4. 不要输出“以下是周报”之类的开场白，只输出周报正文。

周期：%s ~ %s（%s）
本周完成情况（层级）：
%s

本地汇总：
%s`, week.Start, week.End, week.ID, tree, report)
}

func truncate(value string, limit int, lang i18n.Language) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "\n" + i18n.T(lang, "…（内容过长已截断）", "… (truncated)")
}
