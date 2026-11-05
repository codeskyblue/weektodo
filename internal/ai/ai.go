package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

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

// ValidateConfig checks the user supplied endpoint configuration.
func ValidateConfig(cfg model.AIConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("请先填写 AI Base URL")
	}
	parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("AI Base URL 必须是 http:// 或 https:// 地址")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("请先填写模型名称")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("请先填写 API Key")
	}
	return nil
}

// Polish generates a weekly report from the local report and completed task tree.
func (c *Client) Polish(ctx context.Context, cfg model.AIConfig, week *model.Week, report string) (string, error) {
	prompt := WeeklyPrompt(week, report)
	return c.complete(ctx, cfg, "你是一名严谨、务实的中文周报助手。", prompt, 1800)
}

// Test performs a tiny completion to verify the endpoint and credentials.
func (c *Client) Test(ctx context.Context, cfg model.AIConfig) (string, error) {
	return c.complete(ctx, cfg, "你是连接测试助手。", "只回复 OK。", 8)
}

func (c *Client) complete(ctx context.Context, cfg model.AIConfig, system, user string, maxTokens int) (string, error) {
	if err := ValidateConfig(cfg); err != nil {
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
		return "", fmt.Errorf("构建 AI 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("连接 AI 接口失败: %w", err)
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
		return "", fmt.Errorf("AI 接口返回 %d: %s", resp.StatusCode, message)
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
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("AI 接口返回错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("AI 接口没有返回内容")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("AI 接口返回了空内容")
	}
	return content, nil
}

// WeeklyPrompt builds the Chinese prompt used for report polishing.
func WeeklyPrompt(week *model.Week, report string) string {
	tree := truncate(todo.DoneText(week), 12000)
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

func truncate(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "\n…（内容过长已截断）"
}
