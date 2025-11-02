package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/Nadim147c/fang"
	"github.com/Nadim147c/waybar-lyric/cmd"
	"github.com/Nadim147c/waybar-lyric/internal/config"
)

var Version = "dev-build"

var version = fmt.Sprintf("waybar-lyric %s (https://github.com/Nadim147c/waybar-lyric)", Version)

func main() {
	config.Version = version

	err := fang.Execute(
		context.Background(),
		cmd.Command,
		fang.WithFlagTypes(),
		fang.WithNotifySignal(os.Interrupt, os.Kill, syscall.SIGKILL, syscall.SIGTERM),
		fang.WithVersion(version),
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
		fang.WithErrorHandler(func(w io.Writer, styles fang.Styles, err error) {
			if errors.Is(err, context.Canceled) {
				err = errors.New("Closed by user")
			}
			fang.DefaultErrorHandler(w, styles, err)
		}),
	)
	if err != nil {
		os.Exit(1)
	}
}
