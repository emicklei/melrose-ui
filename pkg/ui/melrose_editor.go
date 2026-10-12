package ui

import (
	"fmt"
	"strings"
	"time"

	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

type Editor struct {
	Config   *Config
	Text     string
	Bindings map[shirei.KeyCombo]func(string, widgets.TextInputState)

	markedLines map[int]bool
	flash       struct {
		document string
		start    int
		end      int
		top      float32
		until    time.Time
	}
}

func NewEditor(text string, config *Config) *Editor {
	if config == nil {
		config = &Config{}
	}
	editor := &Editor{Text: text, Config: config, markedLines: map[int]bool{}}
	editor.Bindings = map[shirei.KeyCombo]func(string, widgets.TextInputState){
		shirei.Combo(shirei.Key2, shirei.ModCmd): editor.handleInspect,
		shirei.Combo(shirei.KeyE, shirei.ModCmd): editor.handleEvaluate,
		shirei.Combo(shirei.Key3, shirei.ModCmd): editor.printSelection,
		shirei.Combo(shirei.Key4, shirei.ModCmd): editor.handleKill,
		shirei.Combo(shirei.KeyK, shirei.ModCmd): editor.handleKill,
	}
	return editor
}

func (editor *Editor) handleInspect(document string, state widgets.TextInputState) {
	logSelection(document, state)
}

func (editor *Editor) handleEvaluate(document string, state widgets.TextInputState) {
	logSelection(document, state)
}

func (editor *Editor) handleKill(document string, state widgets.TextInputState) {
	logSelection(document, state)
}

func (editor *Editor) printSelection(document string, state widgets.TextInputState) {
	logSelection(document, state)
	editor.flash.document = document
	editor.flash.start, editor.flash.end = editorRange(document, state)
	editor.flash.top = state.CaretPos[1]
	editor.flash.until = time.Now().Add(editor.Config.EditorFlashDuration)
}

func (editor *Editor) markLine(document string, state widgets.TextInputState) {
	logSelection(document, state)
	state.Anchor = state.Cursor
	lineNumber, _ := editorSelection(document, state)
	editor.markedLines[lineNumber] = true
}

func logSelection(document string, state widgets.TextInputState) {
	lineNumber, content := editorSelection(document, state)
	fmt.Printf("Line %d: %s\n", lineNumber, content)
}

func editorSelection(document string, state widgets.TextInputState) (int, string) {
	start, end := editorRange(document, state)
	runes := []rune(document)
	lineNumber := strings.Count(string(runes[:start]), "\n") + 1
	return lineNumber, string(runes[start:end])
}

func editorRange(document string, state widgets.TextInputState) (int, int) {
	runes := []rune(document)
	cursor := min(max(state.Cursor, 0), len(runes))
	anchor := min(max(state.Anchor, 0), len(runes))
	start, end := min(cursor, anchor), max(cursor, anchor)
	if start == end {
		for start > 0 && runes[start-1] != '\n' {
			start--
		}
		for end < len(runes) && runes[end] != '\n' {
			end++
		}
	}
	return start, end
}

func (editor *Editor) Draw() {
	shirei.ContainerWithKey(editor, shirei.Attrs(shirei.Expand, shirei.Grow(1)), func() {
		style := widgets.CurrentColorScheme.TextInput
		lineCount := strings.Count(editor.Text, "\n") + 1
		gutterWidth := float32(len(fmt.Sprint(lineCount)))*16 + 28
		cfg := widgets.TextInputConfigWithStyle(widgets.TextInputConfig{
			FontSize: 16,
			Padding:  shirei.N4(12),
			Wrap:     true,
		}, style)
		cfg.Padding[shirei.PAD_LEFT] += gutterWidth
		padding := cfg.Padding
		for index := range padding {
			padding[index] *= shirei.ComfortScale()
		}
		shirei.NextAccessName("Document")
		shirei.Container(shirei.Attrs(shirei.Expand, shirei.Grow(1), shirei.MinWidth(0),
			shirei.Focusable, shirei.Clip, shirei.PadVec(padding), shirei.Corners(4),
			shirei.BackgroundVec(style.Background), shirei.BorderWidth(1), shirei.BorderColorVec(style.Border)), func() {
			var binding func(string, widgets.TextInputState)
			if shirei.HasFocus() && shirei.GetInputState().Composition == "" {
				if handle, ok := editor.Bindings[shirei.ActiveCombo()]; ok {
					shirei.GetHost().FrameInput.Key = shirei.KeyCodeNone
					shirei.GetHost().FrameInput.Text = ""
					binding = handle
				}
			}
			state := widgets.ProcessTextInput(&editor.Text, cfg)
			if binding != nil && state.HasFocus {
				binding(editor.Text, state)
			}
			shirei.NextAccessRole("text")
			shirei.NextAccessEditable(true, true)
			shirei.NextAccessValue(editor.Text)
			shirei.AssignAccess()
			if state.HasFocus {
				shirei.ModAttrs(shirei.BorderColorVec(widgets.CurrentColorScheme.FocusRing))
			}
			paintState, paintConfig := state, cfg
			if editor.flash.document == editor.Text && !state.Composing && time.Now().Before(editor.flash.until) {
				paintState.SelFrom, paintState.SelTo = editor.flash.start, editor.flash.end
				paintConfig.SelectionColor = shirei.Vec4{130, 65, 78, 1}
				if editor.flash.start == editor.flash.end {
					scroll := *shirei.Use[shirei.Vec2]("ti-scroll")
					shirei.Element(shirei.Attrs(shirei.Float(padding[shirei.PAD_LEFT], padding[shirei.PAD_TOP]+editor.flash.top-scroll[1]),
						shirei.FixSize(shirei.GetContentWidth(), cfg.FontSize*shirei.ComfortScale()),
						shirei.BackgroundVec(paintConfig.SelectionColor), shirei.ClickThrough))
				}
				shirei.RequestNextFrame()
			}
			widgets.DrawTextInputPlain(paintState, paintConfig)
			display := editor.Text
			if state.Composing {
				runes := []rune(display)
				cursor := min(max(state.Cursor, 0), len(runes))
				display = string(runes[:cursor]) + shirei.GetInputState().Composition + string(runes[cursor:])
			}
			editor.drawLineNumbers(display, cfg, gutterWidth)
		})
	})
}

func (editor *Editor) drawLineNumbers(document string, cfg widgets.TextInputConfig, gutterWidth float32) {
	scale := shirei.ComfortScale()
	fontSize := cfg.FontSize * scale
	style := shirei.DefaultTextStyle()
	style.FontSize = fontSize
	contentRect := shirei.GetContentRect()
	scroll := *shirei.Use[shirei.Vec2]("ti-scroll")
	shaped := shirei.ShapeTextMax(document, style, contentRect.Size[0])
	if document == "" {
		shaped = shirei.ShapeTextMax(" ", style, contentRect.Size[0])
	}
	shirei.Container(shirei.Attrs(shirei.Float(4*scale, cfg.Padding[shirei.PAD_TOP]*scale),
		shirei.FixSize(gutterWidth*scale-8*scale, contentRect.Size[1]), shirei.Clip, shirei.ClickThrough), func() {
		var top float32
		fallbackStart, countedTo, lineNumber, previousNumber := 0, 0, 1, 0
		for _, visualLine := range shaped.Lines {
			start := len(shaped.Runes)
			for _, segment := range visualLine.Segments {
				for _, glyph := range segment.Glyphs {
					cluster := int(glyph.Cluster)
					if cluster < len(shaped.Runes) && shaped.Runes[cluster] != '\n' {
						start = min(start, cluster)
					}
				}
			}
			if start == len(shaped.Runes) {
				start = fallbackStart
			}
			for countedTo < start {
				if shaped.Runes[countedTo] == '\n' {
					lineNumber++
				}
				countedTo++
			}
			fallbackStart = start
			for fallbackStart < len(shaped.Runes) {
				fallbackStart++
				if shaped.Runes[fallbackStart-1] == '\n' {
					break
				}
			}
			if lineNumber != previousNumber && top-scroll[1]+fontSize > 0 && top-scroll[1] < contentRect.Size[1] {
				shirei.Container(shirei.Attrs(shirei.Float(0, top-scroll[1]), shirei.Row, shirei.MainAlign(shirei.AlignEnd), shirei.CrossMid, shirei.Gap(8*scale),
					shirei.FixSize(gutterWidth*scale-8*scale, fontSize), shirei.ClickThrough), func() {
					markerColor := shirei.Vec4{0, 85, 50, 0}
					if editor.markedLines[lineNumber] {
						markerColor[3] = 1
					}
					shirei.Element(shirei.Attrs(shirei.FixSize(8*scale, 8*scale), shirei.Corners(4*scale),
						shirei.BackgroundVec(markerColor), shirei.ClickThrough))
					shirei.Label(fmt.Sprint(lineNumber), shirei.FontSize(fontSize), shirei.TextColor(0, 0, 55, 1))
				})
			}
			previousNumber = lineNumber
			top += visualLine.Height
		}
	})
}
