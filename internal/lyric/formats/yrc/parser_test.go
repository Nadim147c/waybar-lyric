package yrc

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Nadim147c/waybar-lyric/internal/lyric/models"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestParseWordTimedLine(t *testing.T) {
	lines, err := ParseText(`{"t":0,"c":[{"tx":"credits"}]}
[27360,1290](27360,240,0)I've (27600,90,0)been (27690,360,0)tryna (28050,600,0)call
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 { // the empty first line and the lyric line
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	line := lines[1]
	if line.Timestamp != ms(27_360) {
		t.Errorf("timestamp = %v, want 27.36s", line.Timestamp)
	}
	if line.Text != "I've been tryna call" {
		t.Errorf("text = %q", line.Text)
	}

	// four words with an untimed space between each of them
	if len(line.Words) != 7 {
		t.Fatalf("got %d words, want 7", len(line.Words))
	}
	if line.Words[0].Text != "I've" {
		t.Errorf("first word = %q", line.Words[0].Text)
	}
	if line.Words[1].Start != -1 || line.Words[1].Text != " " {
		t.Errorf("second word = %+v, want an untimed space", line.Words[1])
	}
	if line.Words[2].Start != ms(27_600) {
		t.Errorf("third word starts at %v", line.Words[2].Start)
	}
	if line.Words[6].End != ms(28_650) {
		t.Errorf("last word ends at %v", line.Words[6].End)
	}
}

func TestParseKeepsUnspacedWordsAdjacent(t *testing.T) {
	lines, err := ParseText("[0,1000](0,500,0)我(500,500,0)爱\n")
	if err != nil {
		t.Fatal(err)
	}

	line := lines[1]
	if line.Text != "我爱" {
		t.Errorf("text = %q", line.Text)
	}
	if len(line.Words) != 2 {
		t.Errorf("got %d words, want 2 without a space between them", len(line.Words))
	}
}

func TestParseKeepsUntimedParentheses(t *testing.T) {
	lines, err := ParseText("[1000,2000](1000,500,0)la (la) (1500,500,0)again\n")
	if err != nil {
		t.Fatal(err)
	}

	line := lines[1]
	if line.Text != "la (la) again" {
		t.Errorf("text = %q", line.Text)
	}
	if len(line.Words) != 3 || line.Words[0].Text != "la (la)" {
		t.Errorf("words = %+v", line.Words)
	}
}

func TestParseWithoutLyricLines(t *testing.T) {
	_, err := ParseText(`{"t":0,"c":[{"tx":"credits"}]}` + "\n")
	if !errors.Is(err, models.ErrLyricsNotSynced) {
		t.Errorf("err = %v, want ErrLyricsNotSynced", err)
	}
}

func TestWritersFromCreditHeaders(t *testing.T) {
	text := `{"t":0,"c":[{"tx":"作词: "},{"tx":"Abel Tesfaye"},{"tx":"/"},{"tx":"Max Martin"}]}
{"t":1,"c":[{"tx":"作曲: "},{"tx":"Max Martin"}]}
{"t":2,"c":[{"tx":"制作人: "},{"tx":"Oscar Holter"}]}
[1000,2000](1000,500,0)la
`

	got := Writers(text)
	want := []string{"Abel Tesfaye", "Max Martin"}
	if !slices.Equal(got, want) {
		t.Errorf("writers = %q, want %q", got, want)
	}
}
