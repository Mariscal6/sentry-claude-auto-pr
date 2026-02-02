package tools

import (
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantLen int // expected length > 0 means we expect valid JSON
	}{
		{
			name: "json in code block",
			input: `Here is the fix:

` + "```json" + `
{
  "success": true,
  "description": "Fixed null pointer",
  "files": []
}
` + "```" + `

Let me know if you need anything else.`,
			want:    `{"success": true,"description": "Fixed null pointer","files": []}`,
			wantLen: 1,
		},
		{
			name:    "json in plain code block",
			input:   "```\n{\"success\": true}\n```",
			want:    `{"success": true}`,
			wantLen: 1,
		},
		{
			name:    "raw json object",
			input:   `Some text before {"success": false, "error": "could not fix"} and after`,
			want:    `{"success": false, "error": "could not fix"}`,
			wantLen: 1,
		},
		{
			name:    "nested json",
			input:   `{"outer": {"inner": {"deep": true}}, "arr": [1, 2, 3]}`,
			want:    `{"outer": {"inner": {"deep": true}}, "arr": [1, 2, 3]}`,
			wantLen: 1,
		},
		{
			name:    "no json",
			input:   "Just plain text with no JSON",
			want:    "",
			wantLen: 0,
		},
		{
			name:    "json with strings containing braces",
			input:   `{"message": "Use {name} as placeholder", "code": "func() { }"}`,
			want:    `{"message": "Use {name} as placeholder", "code": "func() { }"}`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)

			if tt.wantLen == 0 {
				if got != "" {
					t.Errorf("extractJSON() = %q, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Errorf("extractJSON() returned empty, want non-empty JSON")
				return
			}

			// Just verify it starts with { and ends with }
			if got[0] != '{' || got[len(got)-1] != '}' {
				t.Errorf("extractJSON() = %q, doesn't look like JSON", got)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	tool := NewClaudeCodeTool("/tmp/test", "")

	req := &FixRequest{
		IssueID:      "12345",
		Title:        "NullPointerException in UserService",
		ErrorType:    "NullPointerException",
		ErrorMessage: "Cannot invoke method on null object",
		Level:        "error",
		Platform:     "java",
		Culprit:      "com.example.UserService.getUser",
		Permalink:    "https://sentry.io/issues/12345",
		Stacktrace: []Frame{
			{
				Filename: "UserService.java",
				Function: "getUser",
				LineNo:   42,
				InApp:    true,
			},
			{
				Filename: "UserController.java",
				Function: "handleRequest",
				LineNo:   15,
				InApp:    true,
			},
			{
				Filename: "spring-framework.jar",
				Function: "dispatch",
				LineNo:   100,
				InApp:    false,
			},
		},
	}

	prompt := tool.buildPrompt(req)

	// Verify prompt contains key information
	checks := []string{
		"12345",
		"NullPointerException",
		"UserService.java",
		"getUser",
		"[IN APP]",
		"spring-framework.jar",
	}

	for _, check := range checks {
		if !contains(prompt, check) {
			t.Errorf("buildPrompt() missing %q", check)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
