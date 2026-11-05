package webdav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
)

// Config is the persisted WebDAV endpoint configuration.
type Config = model.WebDAVConfig

// Client is a small WebDAV client for JSON file synchronization.
type Client struct {
	cfg      Config
	lang     i18n.Language
	dirParts []string
	http     *http.Client
}

// New creates a client with a bounded timeout.
func New(cfg Config, lang i18n.Language) *Client {
	cfg.URL = strings.TrimSpace(cfg.URL)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.RemoteDir = strings.Trim(strings.TrimSpace(cfg.RemoteDir), "/")
	return &Client{
		cfg:      cfg,
		lang:     lang,
		dirParts: splitDir(cfg.RemoteDir),
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

func splitDir(dir string) []string {
	parts := []string{}
	for _, part := range strings.Split(dir, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

// Validate checks the endpoint configuration.
func Validate(cfg Config, lang i18n.Language) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("%s", i18n.T(lang, "请先填写 WebDAV 地址", "Fill in the WebDAV URL first"))
	}
	parsed, err := url.Parse(strings.TrimSpace(cfg.URL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s", i18n.T(lang,
			"WebDAV 地址必须是 http:// 或 https:// 地址",
			"WebDAV URL must be an http:// or https:// address"))
	}
	for _, part := range splitDir(cfg.RemoteDir) {
		if part == ".." {
			return fmt.Errorf("%s", i18n.T(lang, "WebDAV 目录不能包含 ..", "WebDAV directory cannot contain .."))
		}
	}
	return nil
}

// Test checks that the endpoint responds to a WebDAV request.
func (c *Client) Test(ctx context.Context) error {
	if err := Validate(c.cfg, c.lang); err != nil {
		return err
	}
	// Prefer checking the configured directory. If it does not exist yet,
	// fall back to the base URL so a test can still validate credentials.
	for _, target := range c.testTargets() {
		status, body, err := c.propfind(ctx, target)
		if err != nil {
			return err
		}
		switch {
		case status == http.StatusMultiStatus, status >= 200 && status < 300:
			return nil
		case status == http.StatusUnauthorized, status == http.StatusForbidden:
			return c.authError(status)
		case status == http.StatusNotFound:
			continue
		case status == http.StatusMethodNotAllowed, status == http.StatusNotImplemented:
			if optionsErr := c.options(ctx, target); optionsErr == nil {
				return nil
			}
		default:
			message := strings.TrimSpace(string(body))
			if len(message) > 200 {
				message = message[:200] + "…"
			}
			return fmt.Errorf("%s %d: %s", i18n.T(c.lang, "WebDAV 返回 HTTP", "WebDAV returned HTTP"), status, message)
		}
	}
	return fmt.Errorf("%s", i18n.T(c.lang, "WebDAV 地址不可用", "WebDAV endpoint is unavailable"))
}

// Get reads a remote file. The bool is false when the file does not exist.
func (c *Client) Get(ctx context.Context, name string) ([]byte, bool, error) {
	if err := Validate(c.cfg, c.lang); err != nil {
		return nil, false, err
	}
	endpoint, err := c.endpoint(name)
	if err != nil {
		return nil, false, err
	}
	req, err := c.request(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", i18n.T(c.lang, "下载 WebDAV 文件失败", "Failed to download the WebDAV file"), err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, c.authError(resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("%s: HTTP %d", i18n.T(c.lang, "下载 WebDAV 文件失败", "Failed to download the WebDAV file"), resp.StatusCode)
	}
	return body, true, nil
}

// Put uploads a remote file, creating the configured directory if needed.
func (c *Client) Put(ctx context.Context, name string, data []byte) error {
	if err := Validate(c.cfg, c.lang); err != nil {
		return err
	}
	if err := c.ensureDir(ctx); err != nil {
		return err
	}
	endpoint, err := c.endpoint(name)
	if err != nil {
		return err
	}
	req, err := c.request(ctx, http.MethodPut, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(c.lang, "上传 WebDAV 文件失败", "Failed to upload the WebDAV file"), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return c.authError(resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: HTTP %d %s",
			i18n.T(c.lang, "上传 WebDAV 文件失败", "Failed to upload the WebDAV file"),
			resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) ensureDir(ctx context.Context) error {
	for index := 1; index <= len(c.dirParts); index++ {
		endpoint, err := c.urlFor(c.dirParts[:index]...)
		if err != nil {
			return err
		}
		req, err := c.request(ctx, "MKCOL", endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T(c.lang, "创建 WebDAV 目录失败", "Failed to create the WebDAV directory"), err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusCreated, http.StatusNoContent, http.StatusOK, http.StatusMethodNotAllowed:
			continue
		case http.StatusUnauthorized, http.StatusForbidden:
			return c.authError(resp.StatusCode)
		}
		return fmt.Errorf("%s: HTTP %d %s",
			i18n.T(c.lang, "创建 WebDAV 目录失败", "Failed to create the WebDAV directory"),
			resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) authError(status int) error {
	if c.lang == i18n.En {
		return fmt.Errorf("%s (HTTP %d)",
			i18n.T(c.lang, "WebDAV 认证失败", "WebDAV authentication failed"), status)
	}
	return fmt.Errorf("%s（HTTP %d）",
		i18n.T(c.lang, "WebDAV 认证失败", "WebDAV authentication failed"), status)
}

func (c *Client) testTargets() []string {
	targets := []string{}
	if len(c.dirParts) > 0 {
		if endpoint, err := c.urlFor(c.dirParts...); err == nil {
			targets = append(targets, endpoint)
		}
	}
	if endpoint, err := c.urlFor(); err == nil {
		targets = append(targets, endpoint)
	}
	return targets
}

func (c *Client) propfind(ctx context.Context, endpoint string) (int, []byte, error) {
	body := strings.NewReader(`<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:prop><d:resourcetype/></d:prop></d:propfind>`)
	req, err := c.request(ctx, "PROPFIND", endpoint, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", i18n.T(c.lang, "连接 WebDAV 失败", "Failed to connect to WebDAV"), err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, data, nil
}

func (c *Client) options(ctx context.Context, endpoint string) error {
	req, err := c.request(ctx, http.MethodOptions, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("OPTIONS HTTP %d", resp.StatusCode)
}

func (c *Client) request(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "weektodo-webdav/1.0")
	if c.cfg.Username != "" || c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	return req, nil
}

func (c *Client) endpoint(name string) (string, error) {
	parts := append(append([]string{}, c.dirParts...), name)
	return c.urlFor(parts...)
}

// urlFor joins path segments under the configured base URL without escaping
// them twice; url.URL.String takes care of percent-encoding.
func (c *Client) urlFor(parts ...string) (string, error) {
	parsed, err := url.Parse(c.cfg.URL)
	if err != nil {
		return "", err
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("%s", i18n.T(c.lang, "WebDAV 路径不能包含 ..", "WebDAV path cannot contain .."))
		}
		basePath += "/" + part
	}
	if basePath == "" {
		basePath = "/"
	}
	parsed.Path = basePath
	parsed.Fragment = ""
	return parsed.String(), nil
}
