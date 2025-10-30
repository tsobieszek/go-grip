package pkg

import (
	"bytes"
	"testing"
)

func TestIsListStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"unordered dash", "- Item", true},
		{"unordered asterisk", "* Item", true},
		{"unordered plus", "+ Item", true},
		{"ordered with dot", "1. Item", true},
		{"ordered with paren", "1) Item", true},
		{"multi-digit ordered", "123. Item", true},
		{"blockquote with list", "> - Item", true},
		{"blockquote ordered", "> 1. Item", true},
		{"not a list - no space", "-Item", false},
		{"not a list - just dash", "-", false},
		{"not a list - text", "Regular text", false},
		{"empty line", "", false},
		{"dash in text", "This - is text", false},
		{"number dot in text", "Version 1.5 released", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isListStart([]byte(tt.line))
			if result != tt.expected {
				t.Errorf("isListStart(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestPreprocessMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "add blank line before unordered list",
			input: `Some text.
- Item 1
- Item 2`,
			expected: `Some text.

- Item 1
- Item 2`,
		},
		{
			name: "add blank line before ordered list",
			input: `Some text.
1. Item 1
2. Item 2`,
			expected: `Some text.

1. Item 1
2. Item 2`,
		},
		{
			name: "preserve existing blank line",
			input: `Some text.

- Item 1
- Item 2`,
			expected: `Some text.

- Item 1
- Item 2`,
		},
		{
			name: "no blank line between list items",
			input: `- Item 1
- Item 2
- Item 3`,
			expected: `- Item 1
- Item 2
- Item 3`,
		},
		{
			name: "preserve code blocks",
			input: "```\nText\n- Not a list\n```",
			expected: "```\nText\n- Not a list\n```",
		},
		{
			name: "list after code block",
			input: "```\ncode\n```\n- Item 1",
			expected: "```\ncode\n```\n\n- Item 1",
		},
		{
			name: "blockquote with list",
			input: "> Text\n> - Item 1\n> - Item 2",
			expected: "> Text\n\n> - Item 1\n> - Item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessMarkdown([]byte(tt.input))
			if !bytes.Equal(result, []byte(tt.expected)) {
				t.Errorf("preprocessMarkdown() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					tt.input, tt.expected, string(result))
			}
		})
	}
}

func TestPreprocessMarkdownCodeBlocks(t *testing.T) {
	input := "```\nText\n- Not a list\n1. Not a list\n```\nText\n- Real list"
	result := preprocessMarkdown([]byte(input))
	expected := "```\nText\n- Not a list\n1. Not a list\n```\nText\n\n- Real list"

	if !bytes.Equal(result, []byte(expected)) {
		t.Errorf("Code block handling failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
			input, expected, string(result))
	}
}
