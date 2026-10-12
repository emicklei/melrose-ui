package ui

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

const windowWidth, windowHeight = 900, 640

var mainEditor = NewEditor("Untitled\n\nStart writing here.", &Config{
	EditorFlashDuration: 200 * time.Millisecond,
})

func rootView() {
	window := Window{Label: "Melrose", Editor: mainEditor}
	window.Draw()
}

func TestWindowConsoleWorkflow(t *testing.T) {
	shirei.InitFontSubsystem()
	shirei.ResetInputSession()
	t.Cleanup(shirei.ResetInputSession)
	window := &Window{
		Label:   "Melrose",
		Editor:  NewEditor("document", nil),
		Console: "console output",
	}
	scope := new(int)
	frameView := func() {
		shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), window.Draw)
	}
	host := shirei.GetHost()
	host.WindowSize = shirei.Vec2{windowWidth, windowHeight}
	for frame := 0; frame < 3; frame++ {
		shirei.RunFrameFn(frameView)
	}
	host.Input.Modifiers = shirei.ModNone
	host.FrameInput = shirei.FrameInputData{Text: "x"}
	shirei.RunFrameFn(frameView)
	if window.Console != "console output" {
		t.Errorf("typing changed console to %q", window.Console)
	}
	image := shirei.RenderToImage(windowWidth, windowHeight, frameView)
	pixelScale := float32(image.Bounds().Dx()) / windowWidth * shirei.ComfortScale()
	padding := int(20 * pixelScale)
	bottom := image.Bounds().Dy() - padding
	top := bottom - 1
	for top > padding {
		color := image.RGBAAt(padding+1, top-1)
		if color.R != 0 || color.G != 0 || color.B != 0 {
			break
		}
		top--
	}
	for left := padding + 1; left < image.Bounds().Dx()-padding-1; left++ {
		color := image.RGBAAt(left, bottom-4)
		if color.R != 0 || color.G != 0 || color.B != 0 {
			t.Fatalf("console background at x=%d is %v, want black across the full width", left, color)
		}
	}
	grayPixels := 0
	for vertical := top + 1; vertical < bottom-1; vertical++ {
		for horizontal := padding + 1; horizontal < image.Bounds().Dx()-padding-1; horizontal++ {
			color := image.RGBAAt(horizontal, vertical)
			if color.R > 150 && color.R == color.G && color.G == color.B {
				grayPixels++
			}
		}
	}
	if grayPixels == 0 {
		t.Error("console did not render light gray text")
	}
}

func TestWindowPaneProportionsWorkflow(t *testing.T) {
	shirei.InitFontSubsystem()
	t.Cleanup(shirei.ResetInputSession)
	for _, scenario := range []struct {
		name   string
		width  int
		height int
		config Config
	}{
		{"defaults", 900, 640, Config{}},
		{"custom", 900, 640, Config{VariableListWidthPercentage: 35, EditorHeightPercentage: 60}},
		{"smaller window", 600, 480, Config{VariableListWidthPercentage: 35, EditorHeightPercentage: 60}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			shirei.ResetInputSession()
			window := &Window{
				Config: &scenario.config,
				Label:  "Melrose",
				Editor: NewEditor("document", nil),
				List:   VariableList{Items: []string{"first item", "second item"}},
			}
			scope := new(int)
			frameView := func() {
				shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), window.Draw)
			}
			shirei.GetHost().WindowSize = shirei.Vec2{float32(scenario.width), float32(scenario.height)}
			for frame := 0; frame < 3; frame++ {
				shirei.RunFrameFn(frameView)
			}
			if window.Editor.Config != window.Config {
				t.Fatal("window and editor do not share one config")
			}
			image := shirei.RenderToImage(scenario.width, scenario.height, frameView)
			scale := float32(image.Bounds().Dx()) / float32(scenario.width) * shirei.ComfortScale()
			padding, gap := int(20*scale), int(12*scale)
			right, bottom := image.Bounds().Dx()-padding, image.Bounds().Dy()-padding
			top, end := -1, -1
			for vertical := padding; vertical < bottom; vertical++ {
				color := image.RGBAAt(right-2, vertical)
				if color.R >= 239 && color.R <= 240 && color.G == color.R && color.B == color.R {
					if top == -1 {
						top = vertical
					}
					end = vertical + 1
				}
			}
			if top == -1 {
				t.Fatal("right-hand list is not visible")
			}
			left := right - 2
			for left > padding {
				color := image.RGBAAt(left-1, (top+end)/2)
				if color.R < 239 || color.R > 240 || color.G != color.R || color.B != color.R {
					break
				}
				left--
			}
			widthPercent, heightPercent := scenario.config.VariableListWidthPercentage, scenario.config.EditorHeightPercentage
			if widthPercent == 0 {
				widthPercent = 25
			}
			if heightPercent == 0 {
				heightPercent = 75
			}
			wantWidth := int(float32(right-padding-gap) * widthPercent / 100)
			wantHeight := int(float32(bottom-top-gap) * heightPercent / 100)
			if difference := right - left - wantWidth; difference < -3 || difference > 3 {
				t.Errorf("list width = %d pixels, want %d", right-left, wantWidth)
			}
			if difference := end - top - wantHeight; difference < -3 || difference > 3 {
				t.Errorf("pane height = %d pixels, want %d", end-top, wantHeight)
			}
			for _, vertical := range []int{top + 4, end - 4} {
				color := image.RGBAAt(padding+10, vertical)
				if color.R < 245 || color.G < 245 || color.B < 245 {
					t.Errorf("editor does not share list height at y=%d: %v", vertical, color)
				}
			}
		})
	}
}

func TestEditorKeyboardBindings(t *testing.T) {
	shirei.InitFontSubsystem()
	originalEditor := mainEditor
	t.Cleanup(func() {
		mainEditor = originalEditor
		shirei.ResetInputSession()
	})
	for _, scenario := range []struct {
		name        string
		key         shirei.KeyCode
		input       string
		modifiers   shirei.Modifiers
		focused     bool
		wantHandled int
		wantText    string
	}{
		{"Cmd+2", shirei.Key2, "2", shirei.ModCmd, true, 1, "text"},
		{"plain 2", shirei.Key2, "2", shirei.ModNone, true, 0, "2text"},
		{"Cmd+Shift+2", shirei.Key2, "2", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+2", shirei.Key2, "2", shirei.ModCmd, false, 0, "text"},
		{"Cmd+E", shirei.KeyE, "e", shirei.ModCmd, true, 1, "text"},
		{"plain e", shirei.KeyE, "e", shirei.ModNone, true, 0, "etext"},
		{"Cmd+Shift+E", shirei.KeyE, "e", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+E", shirei.KeyE, "e", shirei.ModCmd, false, 0, "text"},
		{"Cmd+3", shirei.Key3, "3", shirei.ModCmd, true, 1, "text"},
		{"plain 3", shirei.Key3, "3", shirei.ModNone, true, 0, "3text"},
		{"Cmd+Shift+3", shirei.Key3, "3", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+3", shirei.Key3, "3", shirei.ModCmd, false, 0, "text"},
		{"Cmd+4", shirei.Key4, "4", shirei.ModCmd, true, 1, "text"},
		{"plain 4", shirei.Key4, "4", shirei.ModNone, true, 0, "4text"},
		{"Cmd+Shift+4", shirei.Key4, "4", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+4", shirei.Key4, "4", shirei.ModCmd, false, 0, "text"},
		{"Cmd+K", shirei.KeyK, "k", shirei.ModCmd, true, 1, "text"},
		{"plain k", shirei.KeyK, "k", shirei.ModNone, true, 0, "ktext"},
		{"Cmd+Shift+K", shirei.KeyK, "k", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+K", shirei.KeyK, "k", shirei.ModCmd, false, 0, "text"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			mainEditor = NewEditor("text", originalEditor.Config)
			handled := 0
			combo := shirei.Combo(scenario.key, shirei.ModCmd)
			if mainEditor.Bindings[combo] == nil {
				t.Fatal("shortcut is not registered")
			}
			mainEditor.Bindings[combo] = func(string, widgets.TextInputState) { handled++ }
			shirei.ResetInputSession()
			shirei.GetHost().WindowSize = shirei.Vec2{windowWidth, windowHeight}
			for frame := 0; frame < 3; frame++ {
				shirei.RunFrameFn(rootView)
			}
			if !scenario.focused {
				shirei.ResetInputSession()
			}
			host := shirei.GetHost()
			host.Input.Modifiers = scenario.modifiers
			host.FrameInput = shirei.FrameInputData{Key: scenario.key}
			if scenario.modifiers == shirei.ModNone || scenario.wantHandled != 0 {
				host.FrameInput.Text = scenario.input
			}
			shirei.RunFrameFn(rootView)
			if handled != scenario.wantHandled {
				t.Errorf("handled %d times, want %d", handled, scenario.wantHandled)
			}
			if mainEditor.Text != scenario.wantText {
				t.Errorf("document = %q, want %q", mainEditor.Text, scenario.wantText)
			}
		})
	}
}

func TestEditorShortcutLoggingWorkflow(t *testing.T) {
	shirei.InitFontSubsystem()
	originalEditor := mainEditor
	t.Cleanup(func() {
		mainEditor = originalEditor
		shirei.ResetInputSession()
	})
	for _, key := range []shirei.KeyCode{shirei.Key2, shirei.KeyE, shirei.Key3, shirei.Key4, shirei.KeyK} {
		for _, scenario := range []struct {
			name   string
			anchor int
			cursor int
			want   string
		}{
			{"caret", 8, 8, "Line 2: second\n"},
			{"selection", 7, 10, "Line 2: eco\n"},
		} {
			t.Run(fmt.Sprint(key)+"/"+scenario.name, func(t *testing.T) {
				mainEditor = NewEditor("first\nsecond", originalEditor.Config)
				mainEditor.Bindings[shirei.Combo(shirei.Key1, shirei.ModCmd)] = func(string, widgets.TextInputState) {
					widgets.EditorSetSelection(shirei.CurrentId(), scenario.anchor, scenario.cursor)
				}
				scope := new(int)
				frameView := func() {
					shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), rootView)
				}
				shirei.ResetInputSession()
				shirei.GetHost().WindowSize = shirei.Vec2{windowWidth, windowHeight}
				for frame := 0; frame < 3; frame++ {
					shirei.RunFrameFn(frameView)
				}
				host := shirei.GetHost()
				host.Input.Modifiers = shirei.ModCmd
				host.FrameInput = shirei.FrameInputData{Key: shirei.Key1}
				shirei.RunFrameFn(frameView)
				reader, writer, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				defer writer.Close()
				originalStdout := os.Stdout
				defer func() { os.Stdout = originalStdout }()
				os.Stdout = writer
				host.FrameInput = shirei.FrameInputData{Key: key}
				shirei.RunFrameFn(frameView)
				os.Stdout = originalStdout
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				output, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				if string(output) != scenario.want {
					t.Errorf("output = %q, want %q", output, scenario.want)
				}
			})
		}
	}
}

func TestEditorSelection(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		document string
		anchor   int
		cursor   int
		wantLine int
		wantText string
	}{
		{"caret in first line", "first\nsecond", 2, 2, 1, "first"},
		{"caret in second line", "first\nsecond", 8, 8, 2, "second"},
		{"selection", "first\nsecond", 7, 10, 2, "eco"},
		{"reverse selection", "first\nsecond", 10, 7, 2, "eco"},
		{"multiline selection", "first\nsecond", 3, 9, 1, "st\nsec"},
		{"empty line", "first\n\nthird", 6, 6, 2, ""},
		{"trailing empty line", "first\n", 6, 6, 2, ""},
		{"caret at newline", "first\nsecond", 5, 5, 1, "first"},
		{"end of document", "first\nsecond", 12, 12, 2, "second"},
		{"unicode selection", "caf\u00e9\n\u65e5\u672c\u8a9e", 5, 7, 2, "\u65e5\u672c"},
		{"unicode line", "caf\u00e9\n\u65e5\u672c\u8a9e", 6, 6, 2, "\u65e5\u672c\u8a9e"},
		{"empty document", "", 0, 0, 1, ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			lineNumber, content := editorSelection(scenario.document, widgets.TextInputState{
				Anchor: scenario.anchor, Cursor: scenario.cursor,
			})
			if lineNumber != scenario.wantLine || content != scenario.wantText {
				t.Errorf("selection = (%d, %q), want (%d, %q)", lineNumber, content, scenario.wantLine, scenario.wantText)
			}
		})
	}
}

func TestEditorMarkerWorkflow(t *testing.T) {
	shirei.InitFontSubsystem()
	originalEditor := mainEditor
	t.Cleanup(func() {
		mainEditor = originalEditor
		shirei.ResetInputSession()
	})
	mainEditor = NewEditor("first\nsecond", originalEditor.Config)
	mainEditor.Bindings[shirei.Combo(shirei.Key4, shirei.ModCmd)] = mainEditor.markLine
	scope := new(int)
	frameView := func() {
		shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), rootView)
	}
	shirei.ResetInputSession()
	shirei.GetHost().WindowSize = shirei.Vec2{windowWidth, windowHeight}
	for frame := 0; frame < 3; frame++ {
		shirei.RunFrameFn(frameView)
	}
	host := shirei.GetHost()
	host.Input.Modifiers = shirei.ModCmd
	host.FrameInput = shirei.FrameInputData{Key: shirei.Key4, Text: "4"}
	shirei.RunFrameFn(frameView)
	if !mainEditor.markedLines[1] || len(mainEditor.markedLines) != 1 {
		t.Fatalf("markers = %v, want only line 1 marked", mainEditor.markedLines)
	}
	if mainEditor.Text != "first\nsecond" {
		t.Fatalf("shortcut changed document to %q", mainEditor.Text)
	}
	image := shirei.RenderToImage(windowWidth, windowHeight, frameView)
	redPixels := 0
	for top := 130; top < 170; top++ {
		for left := 72; left < 90; left++ {
			color := image.RGBAAt(left, top)
			if color.R > 180 && color.G < 100 && color.B < 100 {
				redPixels++
			}
		}
	}
	if redPixels == 0 {
		t.Error("no red marker rendered to the left of the first line number")
	}
}

func TestEditorHighlightWorkflow(t *testing.T) {
	shirei.InitFontSubsystem()
	originalEditor := mainEditor
	t.Cleanup(func() {
		mainEditor = originalEditor
		shirei.ResetInputSession()
	})
	for _, scenario := range []struct {
		name      string
		anchor    int
		cursor    int
		wantStart int
		wantEnd   int
	}{
		{"line", 2, 2, 0, 5},
		{"selection", 1, 4, 1, 4},
		{"multiline selection", 3, 9, 3, 9},
		{"blank line", 6, 6, 6, 6},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			mainEditor = NewEditor("first\n\nsecond", &Config{EditorFlashDuration: 500 * time.Millisecond})
			mainEditor.Bindings = map[shirei.KeyCombo]func(string, widgets.TextInputState){
				shirei.Combo(shirei.Key3, shirei.ModCmd): mainEditor.printSelection,
				shirei.Combo(shirei.Key2, shirei.ModCmd): func(string, widgets.TextInputState) {
					widgets.EditorSetSelection(shirei.CurrentId(), scenario.anchor, scenario.cursor)
				},
			}
			scope := new(int)
			frameView := func() {
				shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), rootView)
			}
			shirei.ResetInputSession()
			shirei.GetHost().WindowSize = shirei.Vec2{windowWidth, windowHeight}
			for frame := 0; frame < 3; frame++ {
				shirei.RunFrameFn(frameView)
			}
			host := shirei.GetHost()
			host.Input.Modifiers = shirei.ModCmd
			host.FrameInput = shirei.FrameInputData{Key: shirei.Key2}
			shirei.RunFrameFn(frameView)
			host.FrameInput = shirei.FrameInputData{Key: shirei.Key3}
			started := time.Now()
			shirei.RunFrameFn(frameView)
			if mainEditor.flash.until.Before(started.Add(mainEditor.Config.EditorFlashDuration)) || mainEditor.flash.until.After(time.Now().Add(mainEditor.Config.EditorFlashDuration)) {
				t.Error("flash deadline does not use the configured duration")
			}
			if mainEditor.flash.start != scenario.wantStart || mainEditor.flash.end != scenario.wantEnd {
				t.Fatalf("flash range = [%d, %d), want [%d, %d)", mainEditor.flash.start, mainEditor.flash.end, scenario.wantStart, scenario.wantEnd)
			}
			countGreen := func() int {
				image := shirei.RenderToImage(windowWidth, windowHeight, frameView)
				green := 0
				for top := 130; top < 250; top++ {
					for left := 150; left < 350; left++ {
						color := image.RGBAAt(left, top)
						if int(color.G) > int(color.R)+20 && int(color.G) > int(color.B)+20 {
							green++
						}
					}
				}
				return green
			}
			if countGreen() == 0 {
				t.Error("Cmd+3 did not render a green background")
			}
			mainEditor.flash.until = time.Now().Add(-time.Second)
			if countGreen() != 0 {
				t.Error("green background remained after the flash expired")
			}
			if mainEditor.Text != "first\n\nsecond" {
				t.Errorf("shortcut changed document to %q", mainEditor.Text)
			}
		})
	}
}
