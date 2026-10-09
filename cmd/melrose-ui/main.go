package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"melrose-ui/pkg/ui"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/app"
)

const windowWidth, windowHeight = 900, 640

func main() {
	mainEditor := ui.NewMelroseEditor("Untitled\n\nStart writing here.", ui.Config{
		FlashDuration: 200 * time.Millisecond,
	})
	snapshot := flag.String("png", "", "render a window snapshot to a PNG and exit")
	flag.Parse()
	if *snapshot != "" {
		if err := shirei.RenderToPNG(*snapshot, windowWidth, windowHeight, mainEditor.DrawRoot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app.SetupWindow("Melrose - Editor", windowWidth, windowHeight)
	app.Run(mainEditor.DrawRoot)
}
