package picolm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	// "log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/picolm/picolm-server/pkg/config"
	"github.com/picolm/picolm-server/pkg/types"
)

type Client struct {
	config config.PicoLMConfig
}

func NewClient(cfg config.PicoLMConfig) *Client {
	return &Client{config: cfg}
}

type Provider interface {
	Chat(ctx context.Context, req *types.ChatCompletionRequest) (*ChatResult, error)
	StreamChat(ctx context.Context, req *types.ChatCompletionRequest, handler StreamHandler) error
	GetDefaultModel() string
	Validate() error
}

var _ Provider = (*Client)(nil)

type ChatResult struct {
	Content      string
	ToolCalls    []types.ToolCall
	FinishReason string
	Usage        types.Usage
}

func (c *Client) Chat(ctx context.Context, req *types.ChatCompletionRequest) (*ChatResult, error) {
	if c.config.Binary == "" {
		return nil, fmt.Errorf("picolm binary not configured")
	}
	if c.config.ModelPath == "" {
		return nil, fmt.Errorf("picolm model path not configured")
	}

	prompt := c.buildPrompt(req.Messages, req.Tools)

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	temperature := c.config.Temperature
	if req.Temperature > 0 {
		temperature = req.Temperature
	}

	topP := c.config.TopP
	if req.TopP > 0 {
		topP = req.TopP
	}

	args := []string{
		c.config.ModelPath,
		"-n", fmt.Sprintf("%d", maxTokens),
		"-j", fmt.Sprintf("%d", c.config.Threads),
		"-t", fmt.Sprintf("%.1f", temperature),
		"-k", fmt.Sprintf("%.1f", topP),
	}

	if len(req.Tools) > 0 {
		args = append(args, "--json")
	}

	timeout := c.calculateTimeout(maxTokens)
	inferenceCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(inferenceCtx, c.config.Binary, args...)
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	elapsed := time.Since(startTime)

	if inferenceCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("picolm inference timed out after %v (max_tokens: %d)", timeout, maxTokens)
	}

	if inferenceCtx.Err() == context.Canceled {
		return nil, fmt.Errorf("request cancelled (client disconnected or timeout)")
	}

	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("picolm error: %s", stderr.String())
		}
		return nil, fmt.Errorf("picolm error: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return &ChatResult{
			Content:      "",
			FinishReason: "stop",
			Usage:        types.Usage{},
		}, nil
	}

	toolCalls := c.extractToolCalls(output)
	finishReason := "stop"
	content := c.cleanResponse(output)

	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = c.stripToolCalls(output)
	}

	approxTokens := len(output) / 4
	usage := types.Usage{
		PromptTokens:     len(prompt) / 4,
		CompletionTokens: approxTokens,
		TotalTokens:      (len(prompt) + len(output)) / 4,
	}

	_ = elapsed

	return &ChatResult{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

type StreamHandler func(content string, finishReason string) error

func (c *Client) StreamChat(ctx context.Context, req *types.ChatCompletionRequest, handler StreamHandler) error {
	if c.config.Binary == "" {
		return fmt.Errorf("picolm binary not configured")
	}
	if c.config.ModelPath == "" {
		return fmt.Errorf("picolm model path not configured")
	}

	prompt := c.buildPrompt(req.Messages, req.Tools)

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	temperature := c.config.Temperature
	if req.Temperature > 0 {
		temperature = req.Temperature
	}

	topP := c.config.TopP
	if req.TopP > 0 {
		topP = req.TopP
	}

	args := []string{
		c.config.ModelPath,
		"-n", fmt.Sprintf("%d", maxTokens),
		"-j", fmt.Sprintf("%d", c.config.Threads),
		"-t", fmt.Sprintf("%.1f", temperature),
		"-k", fmt.Sprintf("%.1f", topP),
	}

	if len(req.Tools) > 0 {
		args = append(args, "--json")
	}

	timeout := c.calculateTimeout(maxTokens)
	inferenceCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(inferenceCtx, c.config.Binary, args...)
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	defer stderr.Close()

	var wg sync.WaitGroup
	var stderrBuf strings.Builder

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text())
			stderrBuf.WriteString("\n")
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start picolm: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	finishReason := "stop"

	for scanner.Scan() {
		select {
		case <-inferenceCtx.Done():
			cmd.Process.Kill()
			if inferenceCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("picolm inference timed out after %v (max_tokens: %d)", timeout, maxTokens)
			}
			return fmt.Errorf("request cancelled (client disconnected or timeout)")
		default:
		}

		token := scanner.Text()

		if containsSpecialToken(token) {
			break
		}

		output.WriteString(token)

		if err := handler(token, ""); err != nil {
			cmd.Process.Kill()
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		cmd.Process.Kill()
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("picolm error: %s", stderrBuf.String())
		}
		return fmt.Errorf("error reading stdout: %w", err)
	}

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if inferenceCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("picolm inference timed out after %v (max_tokens: %d)", timeout, maxTokens)
		}
		if inferenceCtx.Err() == context.Canceled {
			return fmt.Errorf("request cancelled (client disconnected or timeout)")
		}
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("picolm error: %s", stderrBuf.String())
		}
		return fmt.Errorf("picolm error: %w", err)
	}

	outputStr := strings.TrimSpace(output.String())
	if outputStr == "" {
		return handler("", "stop")
	}

	toolCalls := c.extractToolCalls(outputStr)
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	return handler("", finishReason)
}

const defaultSystemPrompt = "You are a helpful assistant."

func (c *Client) buildPrompt(messages []types.ChatMessage, tools []types.ToolDefinition) string {
	var sb strings.Builder

	var systemParts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemParts = append(systemParts, msg.Content)
		}
	}

	if len(tools) > 0 {
		systemParts = append(systemParts, c.buildToolsPrompt(tools))
	}

	// Use default system prompt if none provided
	if len(systemParts) == 0 {
		systemParts = append(systemParts, defaultSystemPrompt)
	}

	// Write system prompt
	sb.WriteString("<|system|>\n")
	sb.WriteString(strings.Join(systemParts, "\n\n"))
	sb.WriteString("</s>\n")

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// Already handled above
		case "user":
			sb.WriteString("<|user|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "assistant":
			sb.WriteString("<|assistant|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "tool":
			sb.WriteString("<|user|>\n")
			sb.WriteString(fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
			sb.WriteString("</s>\n")
		}
	}

	// No newline after assistant - matches working format
	sb.WriteString("<|assistant|>")

	// log.Printf("[DEBUG] buildPrompt: %q", sb.String())

	return sb.String()
}

func (c *Client) buildToolsPrompt(tools []types.ToolDefinition) string {
	var sb strings.Builder

	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("When you need to use a tool, respond with ONLY a JSON object:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"tool_calls":[{"id":"call_xxx","type":"function","function":{"name":"tool_name","arguments":"{...}"}}]}`)
	sb.WriteString("\n```\n\n")
	sb.WriteString("CRITICAL: The 'arguments' field MUST be a JSON-encoded STRING.\n\n")
	sb.WriteString("### Tool Definitions:\n\n")

	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		sb.WriteString(fmt.Sprintf("#### %s\n", tool.Function.Name))
		if tool.Function.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Function.Description))
		}
		if len(tool.Function.Parameters) > 0 {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			sb.WriteString(fmt.Sprintf("Parameters:\n```json\n%s\n```\n", string(paramsJSON)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (c *Client) extractToolCalls(text string) []types.ToolCall {
	var result struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil
	}

	if len(result.ToolCalls) == 0 {
		return nil
	}

	toolCalls := make([]types.ToolCall, len(result.ToolCalls))
	for i, tc := range result.ToolCalls {
		toolCalls[i] = types.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: types.CallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return toolCalls
}

func (c *Client) stripToolCalls(text string) string {
	var result struct {
		ToolCalls any    `json:"tool_calls"`
		Content   string `json:"content"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return text
	}

	if result.Content != "" {
		return result.Content
	}

	return text
}

func (c *Client) cleanResponse(output string) string {
	specialTokens := []string{"<|user|>", "<|assistant|>", "</s>"}

	minIdx := len(output)
	for _, token := range specialTokens {
		if idx := strings.Index(output, token); idx != -1 && idx < minIdx {
			minIdx = idx
		}
	}

	if minIdx < len(output) {
		output = output[:minIdx]
	}

	output = strings.ReplaceAll(output, "<|user|>", "")
	output = strings.ReplaceAll(output, "<|assistant|>", "")
	output = strings.ReplaceAll(output, "</s>", "")
	output = strings.ReplaceAll(output, "<|system|>", "")
	output = strings.ReplaceAll(output, "<|end|>", "")

	return strings.TrimSpace(output)
}

func containsSpecialToken(token string) bool {
	specialTokens := []string{"<|user|>", "<|assistant|>", "</s>", "<|system|>", "<|end|>"}
	for _, t := range specialTokens {
		if strings.Contains(token, t) {
			return true
		}
	}
	return false
}

func (c *Client) Validate() error {
	if c.config.Binary == "" {
		return fmt.Errorf("binary path is required")
	}

	info, err := os.Stat(c.config.Binary)
	if err != nil {
		return fmt.Errorf("binary not found at %q: %w", c.config.Binary, err)
	}
	if info.IsDir() {
		return fmt.Errorf("binary path %q is a directory", c.config.Binary)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary %q is not executable", c.config.Binary)
	}

	if c.config.ModelPath == "" {
		return fmt.Errorf("model path is required")
	}

	info, err = os.Stat(c.config.ModelPath)
	if err != nil {
		return fmt.Errorf("model not found at %q: %w", c.config.ModelPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("model path %q is a directory", c.config.ModelPath)
	}

	return nil
}

func (c *Client) calculateTimeout(maxTokens int) time.Duration {
	if c.config.TimeoutSeconds > 0 {
		return time.Duration(c.config.TimeoutSeconds) * time.Second
	}

	baseTimeout := 60 * time.Second
	perTokenTimeout := 500 * time.Millisecond

	estimatedTime := baseTimeout + time.Duration(maxTokens)*perTokenTimeout

	maxTimeout := 10 * time.Minute
	if estimatedTime > maxTimeout {
		return maxTimeout
	}

	return estimatedTime
}

func (c *Client) GetDefaultModel() string {
	return "picolm-local"
}
