package adapter

import (
	"testing"
)

func TestParseGhosttyListOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ghosttyRawTab
	}{
		{
			name:  "single tab",
			input: "tab-group-abc|tab-001|term-uuid-1|/Users/nic/project|zsh\n",
			want: []ghosttyRawTab{
				{WindowID: "tab-group-abc", TabID: "tab-001", TerminalID: "term-uuid-1", WorkingDir: "/Users/nic/project", Name: "zsh"},
			},
		},
		{
			name: "multiple windows and tabs",
			input: "win-1|tab-a|term-1|/Users/nic/foo|vim\n" +
				"win-1|tab-b|term-2|/Users/nic/bar|zsh\n" +
				"win-2|tab-c|term-3|/tmp|bash\n",
			want: []ghosttyRawTab{
				{WindowID: "win-1", TabID: "tab-a", TerminalID: "term-1", WorkingDir: "/Users/nic/foo", Name: "vim"},
				{WindowID: "win-1", TabID: "tab-b", TerminalID: "term-2", WorkingDir: "/Users/nic/bar", Name: "zsh"},
				{WindowID: "win-2", TabID: "tab-c", TerminalID: "term-3", WorkingDir: "/tmp", Name: "bash"},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "only whitespace",
			input: "   \n  \n",
			want:  nil,
		},
		{
			name:  "malformed lines are skipped",
			input: "bad line\nwin-1|tab-a|term-1|/dir|name\ntoo|few|fields\n",
			want: []ghosttyRawTab{
				{WindowID: "win-1", TabID: "tab-a", TerminalID: "term-1", WorkingDir: "/dir", Name: "name"},
			},
		},
		{
			name:  "empty window or tab ID skipped",
			input: "|tab-a|term-1|/dir|name\nwin-1||term-1|/dir|name\n",
			want:  nil,
		},
		{
			name:  "trailing newlines handled",
			input: "win-1|tab-a|term-1|/dir|name\n\n\n",
			want: []ghosttyRawTab{
				{WindowID: "win-1", TabID: "tab-a", TerminalID: "term-1", WorkingDir: "/dir", Name: "name"},
			},
		},
		{
			name:  "name with pipe character",
			input: "win-1|tab-a|term-1|/dir|name|with|pipes\n",
			want: []ghosttyRawTab{
				{WindowID: "win-1", TabID: "tab-a", TerminalID: "term-1", WorkingDir: "/dir", Name: "name|with|pipes"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGhosttyListOutput(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("len: got %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tab[%d]: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
