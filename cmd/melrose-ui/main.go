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
	config := &ui.Config{
		VariableListWidthPercentage: 25,
		EditorHeightPercentage:      75,
		EditorFlashDuration:         200 * time.Millisecond,
	}
	mainWindow := &ui.Window{
		Config: config,
		Label:  "Melrose",
		Editor: ui.NewEditor("Untitled\n\nStart writing here.", config),
	}
	snapshot := flag.String("png", "", "render a window snapshot to a PNG and exit")
	flag.Parse()
	if *snapshot != "" {
		if err := shirei.RenderToPNG(*snapshot, windowWidth, windowHeight, mainWindow.Draw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app.SetupWindow("Melrose - Editor", windowWidth, windowHeight)
	app.Run(mainWindow.Draw)
}
