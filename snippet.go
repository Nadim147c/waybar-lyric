package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func PrintSnippet() {
	fmt.Fprintln(os.Stderr, `Put the following object in your waybar config:`)

	snippet := fmt.Sprintf(`
"custom/lyrics": {
	"return-type": "json",
	"format": "{icon} {0}",
	"hide-empty-text": true,
	"format-icons": {
		"playing": "",
		"paused": "",
		"lyric": "",
		"music": "󰝚",
		"no_lyric": ""
	},
	"exec-if": "which waybar-lyric",
	"exec": "waybar-lyric --quiet -m%d",
	"on-click": "waybar-lyric --toggle",
},
`, MaxTextLength)

	cmd := exec.Command("which", "bat")
	if err := cmd.Run(); err == nil {
		batCmd := exec.Command("bat", "-pljson")

		reader := strings.NewReader(snippet)
		batCmd.Stdin = reader
		batCmd.Stdout = os.Stdout

		if err := batCmd.Run(); err != nil {
			fmt.Println(snippet)
		}
	} else {
		fmt.Println(snippet)
	}
}
