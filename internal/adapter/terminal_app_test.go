package adapter

import (
	"testing"
)

func TestParseListTabsOutput(t *testing.T) {
	tests := []struct {
		name string
		input string
		want []rawTab
	}{
		{
			name:  "single tab",
			input: "123|1|/dev/ttys001\n",
			want: []rawTab{
				{WindowID: "123", TabIndex: "1", TTY: "/dev/ttys001"},
			},
		},
		{
			name: "multiple tabs across windows",
			input: "100|1|/dev/ttys001\n100|2|/dev/ttys002\n200|1|/dev/ttys003\n",
			want: []rawTab{
				{WindowID: "100", TabIndex: "1", TTY: "/dev/ttys001"},
				{WindowID: "100", TabIndex: "2", TTY: "/dev/ttys002"},
				{WindowID: "200", TabIndex: "1", TTY: "/dev/ttys003"},
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
			input: "bad line\n100|1|/dev/ttys001\n||\nalso bad\n",
			want: []rawTab{
				{WindowID: "100", TabIndex: "1", TTY: "/dev/ttys001"},
			},
		},
		{
			name:  "line with only two fields is skipped",
			input: "100|1\n",
			want:  nil,
		},
		{
			name:  "trailing newlines handled",
			input: "100|1|/dev/ttys001\n\n\n",
			want: []rawTab{
				{WindowID: "100", TabIndex: "1", TTY: "/dev/ttys001"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListTabsOutput(tt.input)

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

func TestParseLsofOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "typical lsof output",
			input: "p12345\nfcwd\nn/Users/nic/Projects/tabgate\n",
			want:  "/Users/nic/Projects/tabgate",
		},
		{
			name:  "path with spaces",
			input: "p99999\nfcwd\nn/Users/nic/My Projects/cool app\n",
			want:  "/Users/nic/My Projects/cool app",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "no n-line",
			input: "p12345\nfcwd\n",
			want:  "",
		},
		{
			name:  "multiple n-lines takes last",
			input: "p12345\nfcwd\nn/first/path\np67890\nfcwd\nn/second/path\n",
			want:  "/second/path",
		},
		{
			name:  "root directory",
			input: "p1\nfcwd\nn/\n",
			want:  "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLsofOutput(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
