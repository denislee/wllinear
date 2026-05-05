package ai

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	geminiTimeout      = 30 * time.Second
	geminiBatchTimeout = 120 * time.Second
	maxDescription     = 300
	geminiModel        = "gemini-2.5-flash-lite"
)

// IssueInput carries the fields needed to categorize an issue.
type IssueInput struct {
	Identifier  string
	Title       string
	Description string
}

// AIClient defines the interface for interacting with an AI service to categorize issues.
type AIClient interface {
	CategorizeIssue(identifier, title, description string, allowedCategories []string) (string, error)
	CategorizeIssues(issues []IssueInput, allowedCategories []string) (map[string]string, error)
}

// GeminiClient is an implementation of AIClient that uses the Gemini CLI.
type GeminiClient struct{}

// NewGeminiClient creates a new GeminiClient.
func NewGeminiClient() *GeminiClient {
	return &GeminiClient{}
}

// CategorizeIssue categorizes an issue into one of the allowed categories.
func (c *GeminiClient) CategorizeIssue(identifier, title, description string, allowedCategories []string) (string, error) {
	desc := description
	if len(desc) > maxDescription {
		desc = desc[:maxDescription]
	}

	prompt := fmt.Sprintf(
		"Categorize this Linear issue into EXACTLY ONE of these categories:\n%s\n\nIssue:\nID: %s\nTitle: %s\nDescription: %s\n\nRespond ONLY with the category name. Do not include any other text.",
		strings.Join(allowedCategories, ", "),
		identifier,
		title,
		desc,
	)

	ctx, cancel := context.WithTimeout(context.Background(), geminiTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini", "-m", geminiModel, "-p", prompt)
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("gemini timed out after %s", geminiTimeout)
		}
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return "", fmt.Errorf("gemini CLI not found in PATH; install it to use AI categorization")
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("gemini failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("gemini command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CategorizeIssues sends all issues in a single Gemini call and returns a
// map of issue identifier -> matched category (empty if none matched).
// The matching logic prefers the longest allowed label present in the
// model's response line (case-insensitive), mirroring the reference
// linear_labeler.py behavior.
func (c *GeminiClient) CategorizeIssues(issues []IssueInput, allowedCategories []string) (map[string]string, error) {
	if len(issues) == 0 {
		return map[string]string{}, nil
	}
	if len(allowedCategories) == 0 {
		return nil, fmt.Errorf("no allowed categories provided")
	}

	var issuesText strings.Builder
	for _, it := range issues {
		desc := it.Description
		if len(desc) > maxDescription {
			desc = desc[:maxDescription]
		}
		fmt.Fprintf(&issuesText, "ID: %s\nTitle: %s\nDescription: %s\n---\n", it.Identifier, it.Title, desc)
	}

	exampleA := allowedCategories[0]
	exampleB := exampleA
	if len(allowedCategories) > 1 {
		exampleB = allowedCategories[1]
	}

	prompt := fmt.Sprintf(
		"Categorize the following Linear issues into EXACTLY ONE of these categories:\n%s\n\nIssues:\n%s\n\nRespond ONLY with a list of \"ID: Category\", one per line. Do not include any other text or markdown.\nExample:\n1Q25-1234: %s\n1Q25-5678: %s\n",
		strings.Join(allowedCategories, ", "),
		issuesText.String(),
		exampleA,
		exampleB,
	)

	ctx, cancel := context.WithTimeout(context.Background(), geminiBatchTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini", "-m", geminiModel, "-p", prompt)
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("gemini timed out after %s", geminiBatchTimeout)
		}
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, fmt.Errorf("gemini CLI not found in PATH; install it to use AI categorization")
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("gemini failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("gemini command failed: %w", err)
	}

	// Longest-first, so "Feature Improvement" is preferred over "Feature".
	sortedAllowed := make([]string, len(allowedCategories))
	copy(sortedAllowed, allowedCategories)
	sort.SliceStable(sortedAllowed, func(i, j int) bool {
		return len(sortedAllowed[i]) > len(sortedAllowed[j])
	})

	results := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		identifier := strings.TrimSpace(line[:idx])
		categoryRaw := strings.ToLower(strings.TrimSpace(line[idx+1:]))
		if identifier == "" || categoryRaw == "" {
			continue
		}
		for _, candidate := range sortedAllowed {
			if strings.Contains(categoryRaw, strings.ToLower(candidate)) {
				results[identifier] = candidate
				break
			}
		}
	}
	return results, nil
}
