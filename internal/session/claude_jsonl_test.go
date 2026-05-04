package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchesClaudeProjectDir(t *testing.T) {
	cases := []struct {
		name    string
		encoded string
		cwd     string
		want    bool
	}{
		{
			name:    "exact path components",
			encoded: "-Users-alice-projects-opendray-v2",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    true,
		},
		{
			name:    "underscore→dash normalisation",
			encoded: "-Users-alice-Documents-Claude-Workspace-opendray-v2",
			cwd:     "/Users/alice/Documents/Claude_Workspace/opendray-v2",
			want:    true,
		},
		{
			name:    "missing component",
			encoded: "-Users-alice-projects-different-app",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    false,
		},
		{
			name:    "out-of-order parts",
			encoded: "-Users-alice-v2-opendray-projects",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parts := splitPathParts(c.cwd)
			got := matchesClaudeProjectDir(c.encoded, parts)
			if got != c.want {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}

func TestRenderAssistantBlocks_BulletText(t *testing.T) {
	blocks := []claudeContentBlock{
		{Type: "text", Text: "Here is your answer.\n\nIt has two paragraphs."},
	}
	var b strings.Builder
	renderAssistantBlocks(&b, blocks, map[string]string{})
	out := b.String()
	if !strings.HasPrefix(out, "● ") {
		t.Errorf("text bullet missing:\n%s", out)
	}
	if !strings.Contains(out, "Here is your answer.") || !strings.Contains(out, "two paragraphs") {
		t.Errorf("text body lost:\n%s", out)
	}
}

func TestRenderAssistantBlocks_DropsThinking(t *testing.T) {
	blocks := []claudeContentBlock{
		{Type: "thinking", Text: "<internal reasoning the user shouldn't see>"},
		{Type: "text", Text: "Visible reply."},
	}
	var b strings.Builder
	renderAssistantBlocks(&b, blocks, map[string]string{})
	out := b.String()
	if strings.Contains(out, "internal reasoning") {
		t.Errorf("thinking leaked: %s", out)
	}
	if !strings.Contains(out, "Visible reply") {
		t.Errorf("text dropped: %s", out)
	}
}

func TestFormatToolUse_FilePathTitle(t *testing.T) {
	b := claudeContentBlock{
		Type: "tool_use",
		Name: "Write",
		Input: newToolInput(t, map[string]string{
			"file_path": "docs/ANDROID_PORT_SPEC.md",
		}),
	}
	got := formatToolUse(b)
	if got != "Write(docs/ANDROID_PORT_SPEC.md)" {
		t.Errorf("format = %q", got)
	}
}

func TestFormatToolUse_BashCommandWithDesc(t *testing.T) {
	b := claudeContentBlock{
		Type: "tool_use",
		Name: "Bash",
		Input: newToolInput(t, map[string]string{
			"command":     "rg 'TODO' --type go",
			"description": "find outstanding TODOs",
		}),
	}
	got := formatToolUse(b)
	if !strings.Contains(got, "Bash(rg 'TODO' --type go)") || !strings.Contains(got, "find outstanding") {
		t.Errorf("format = %q", got)
	}
}

func TestRenderToolResults_AttachedAfterToolUse(t *testing.T) {
	useID := "tool_abc"
	summary := map[string]string{useID: "Write(README.md)"}
	resultBlocks := []claudeContentBlock{{
		Type:       "tool_result",
		ToolUseID:  useID,
		RawContent: mustRawString(t, "Wrote 734 lines to README.md\n\n  1 # Hello\n  2 World"),
	}}
	var b strings.Builder
	renderToolResults(&b, resultBlocks, summary)
	out := b.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "└ Wrote 734 lines") {
		t.Errorf("result not rendered with └ prefix:\n%s", out)
	}
	// summary[useID] should be cleared after the result is consumed.
	if _, still := summary[useID]; still {
		t.Errorf("summary not cleared after tool_result attach")
	}
}

func TestRenderClaudeRecentTurn_FullFlow(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")

	lines := []string{
		mustEntry(t, "user", "create the Android spec doc"),
		assistantWithTool(t, "我将创建 Android 移植规格文档到 docs/ANDROID_PORT_SPEC.md。",
			"tool_use_1", "Write", map[string]string{
				"file_path": "docs/ANDROID_PORT_SPEC.md",
			}),
		userToolResult(t, "tool_use_1", "Wrote 734 lines to docs/ANDROID_PORT_SPEC.md"),
		mustEntry(t, "assistant", "已创建 Android 移植规格文档：docs/ANDROID_PORT_SPEC.md"),
	}
	if err := os.WriteFile(jsonl, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := renderClaudeRecentTurn(jsonl, 30)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"● 我将创建 Android 移植规格文档",
		"● Write(docs/ANDROID_PORT_SPEC.md)",
		"└ Wrote 734 lines",
		"● 已创建 Android 移植规格文档",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q in:\n%s", want, got)
		}
	}
}

func TestLastAssistantText_PicksMostRecent_Compat(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	lines := []string{
		mustEntry(t, "user", "first user message"),
		mustEntry(t, "assistant", "old reply (should be replaced)"),
		mustEntry(t, "user", "follow-up question"),
		mustEntry(t, "assistant", "newest reply — this should win"),
	}
	if err := os.WriteFile(jsonl, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := renderClaudeRecentTurn(jsonl, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "newest reply") {
		t.Errorf("did not pick most recent: %q", got)
	}
	if strings.Contains(got, "old reply") {
		t.Errorf("returned stale assistant entry: %q", got)
	}
}

// newToolInput materialises the anonymous Input struct used inside
// claudeContentBlock from a string-keyed map. Test helper.
func newToolInput(t *testing.T, kv map[string]string) *struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Path        string `json:"path,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
	Query       string `json:"query,omitempty"`
	Question    string `json:"question,omitempty"`
	URL         string `json:"url,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
} {
	t.Helper()
	in := &struct {
		Command     string `json:"command,omitempty"`
		Description string `json:"description,omitempty"`
		Content     string `json:"content,omitempty"`
		FilePath    string `json:"file_path,omitempty"`
		Path        string `json:"path,omitempty"`
		Pattern     string `json:"pattern,omitempty"`
		Query       string `json:"query,omitempty"`
		Question    string `json:"question,omitempty"`
		URL         string `json:"url,omitempty"`
		Prompt      string `json:"prompt,omitempty"`
	}{}
	in.Command = kv["command"]
	in.Description = kv["description"]
	in.FilePath = kv["file_path"]
	in.Path = kv["path"]
	in.Pattern = kv["pattern"]
	in.Query = kv["query"]
	in.Question = kv["question"]
	in.URL = kv["url"]
	in.Prompt = kv["prompt"]
	return in
}

func mustRawString(t *testing.T, s string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// assistantWithTool builds a JSONL line for an assistant entry that
// contains one text block followed by a single tool_use block with
// the given id, name, and input map.
func assistantWithTool(t *testing.T, text, useID, toolName string, input map[string]string) string {
	t.Helper()
	textBlock := map[string]any{"type": "text", "text": text}
	inputObj := map[string]any{}
	for k, v := range input {
		inputObj[k] = v
	}
	toolBlock := map[string]any{
		"type":  "tool_use",
		"id":    useID,
		"name":  toolName,
		"input": inputObj,
	}
	// JSONL stores tool_use_id under the actual JSON key the
	// production decoder expects ("tool_use_id" on tool_result;
	// tool_use entries set "id"). For tool_use blocks we mirror the
	// id under both keys so the unmarshaler picks it up regardless
	// of which field the runtime ends up using.
	toolBlock["tool_use_id"] = useID
	content, _ := json.Marshal([]any{textBlock, toolBlock})
	entry := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"role":    "assistant",
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// userToolResult builds a JSONL line for a user entry whose content
// is a single tool_result block addressing the given tool_use_id.
func userToolResult(t *testing.T, useID, body string) string {
	t.Helper()
	resultBlock := map[string]any{
		"type":         "tool_result",
		"tool_use_id":  useID,
		"content":      body,
	}
	content, _ := json.Marshal([]any{resultBlock})
	entry := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestResolveLatestClaudeJSONL_RealFile(t *testing.T) {
	// Build a fake ~/.claude/projects/<encoded>/<sid>.jsonl tree under TempDir.
	tmpHome := t.TempDir()
	encoded := "-tmp-fake-cwd"
	pdir := filepath.Join(tmpHome, ".claude", "projects", encoded)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := filepath.Join(pdir, "session-abc.jsonl")
	body := mustEntry(t, "assistant", "hello from JSONL") + "\n"
	if err := os.WriteFile(jsonl, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpHome)
	got := resolveLatestClaudeJSONL("/tmp/fake/cwd")
	if got != jsonl {
		t.Errorf("resolveLatestClaudeJSONL = %q, want %q", got, jsonl)
	}
	resp := claudeRecentResponse("/tmp/fake/cwd")
	if !strings.Contains(resp, "hello from JSONL") {
		t.Errorf("claudeRecentResponse = %q", resp)
	}
}

// mustEntry serialises one JSONL line for a user/assistant message.
// Keeps tests succinct (each line is one assistant turn with one
// text block).
func mustEntry(t *testing.T, role, text string) string {
	t.Helper()
	content, err := json.Marshal([]claudeContentBlock{{Type: "text", Text: text}})
	if err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
		"type": role,
		"message": map[string]any{
			"role":    role,
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
