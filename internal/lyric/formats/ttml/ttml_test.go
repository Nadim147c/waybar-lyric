package ttml

import (
	"bytes"
	_ "embed"
	"testing"
)

//go:embed line_level_testdata.xml
var lineLevelTestdata []byte

//go:embed word_level_testdata.xml
var wordLevelTestdata []byte

//go:embed ado.ttml
var ado []byte

func TestParse(t *testing.T) {
	t.Run("ado", func(t *testing.T) {
		lines, err := Parse(bytes.NewReader(ado))
		if err != nil {
			t.Error(err)
		}
		t.Log(len(lines))
	})

	t.Run("line level", func(t *testing.T) {
		lines, err := Parse(bytes.NewReader(lineLevelTestdata))
		if err != nil {
			t.Error(err)
		}
		t.Log(len(lines))
	})

	t.Run("word level", func(t *testing.T) {
		lines, err := Parse(bytes.NewReader(wordLevelTestdata))
		if err != nil {
			t.Error(err)
		}
		t.Log(len(lines))
	})
}
