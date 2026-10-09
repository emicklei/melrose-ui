package ui

import (
	"testing"
	"time"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

const windowWidth, windowHeight = 900, 640

var mainEditor = NewMelroseEditor("Untitled\n\nStart writing here.", Config{
	FlashDuration: 200 * time.Millisecond,
})

func rootView() {
	mainEditor.DrawRoot()
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
		modifiers   shirei.Modifiers
		focused     bool
		wantHandled int
		wantText    string
	}{
		{"Cmd+3", shirei.ModCmd, true, 1, "text"},
		{"plain 3", shirei.ModNone, true, 0, "3text"},
		{"Cmd+Shift+3", shirei.ModCmd | shirei.ModShift, true, 0, "text"},
		{"unfocused Cmd+3", shirei.ModCmd, false, 0, "text"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			mainEditor = NewMelroseEditor("text", originalEditor.Config)
			handled := 0
			mainEditor.Bindings = map[shirei.KeyCombo]func(string, widgets.TextInputState){
				shirei.Combo(shirei.Key3, shirei.ModCmd): func(string, widgets.TextInputState) { handled++ },
			}
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
			host.FrameInput = shirei.FrameInputData{Key: shirei.Key3}
			if scenario.modifiers == shirei.ModNone || scenario.wantHandled != 0 {
				host.FrameInput.Text = "3"
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
	mainEditor = NewMelroseEditor("first\nsecond", originalEditor.Config)
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
			mainEditor = NewMelroseEditor("first\n\nsecond", Config{FlashDuration: 500 * time.Millisecond})
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
			if mainEditor.flash.until.Before(started.Add(mainEditor.Config.FlashDuration)) || mainEditor.flash.until.After(time.Now().Add(mainEditor.Config.FlashDuration)) {
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
