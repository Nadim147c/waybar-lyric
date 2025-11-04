package str

import (
	"testing"

	"github.com/Nadim147c/waybar-lyric/internal/config"
)

func TestCensorText(t *testing.T) {
	profanity = `
# comments

badword|worseword
`

	tests := []struct {
		input    string
		kind     string
		expected string
	}{
		{"this is a badword", "full", "this is a *******"},
		{"a worseword is here", "full", "a ********* is here"},
		{"badword worseword", "partial", "b*****d w*******d"},
		{"badword worseword", "invalid-type", "badword worseword"},
		{"no bad content", "full", "no bad content"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			config.FilterProfanityType = tt.kind
			output := CensorText(tt.input)
			if output != tt.expected {
				t.Errorf(
					"CensorText(%q, %v) = %q; want %q",
					tt.input,
					tt.kind,
					output,
					tt.expected,
				)
			}
		})
	}
}
