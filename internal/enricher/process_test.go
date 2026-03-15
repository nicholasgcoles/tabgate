package enricher

import "testing"

func TestParsePsOutput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name: "shell is foreground (idle)",
			input: `12345 12345 zsh
12345 12346 ps`,
			expect: "zsh (idle)",
		},
		{
			name: "nvim is foreground",
			input: `12346 12345 zsh
12346 12346 nvim`,
			expect: "nvim",
		},
		{
			name: "login shell is foreground",
			input: `99999 99999 -zsh
99999 10000 ps`,
			expect: "-zsh (idle)",
		},
		{
			name: "multiple processes correct tpgid match",
			input: `  501  400 /bin/zsh
  501  401 node
  501  501 python3`,
			expect: "python3",
		},
		{
			name:   "empty output",
			input:  "",
			expect: "",
		},
		{
			name:   "malformed output",
			input:  "garbage data",
			expect: "",
		},
		{
			name: "process with path",
			input: `100 100 /usr/local/bin/vim
100 101 zsh`,
			expect: "vim",
		},
		{
			name: "bash idle",
			input: `555 555 bash`,
			expect: "bash (idle)",
		},
		{
			name: "fish idle",
			input: `777 777 fish
777 778 ps`,
			expect: "fish (idle)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePsOutput(tt.input)
			if got != tt.expect {
				t.Errorf("expected %q, got %q", tt.expect, got)
			}
		})
	}
}
