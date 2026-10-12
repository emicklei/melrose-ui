# Melrose UI

A basic native Go window using [Shirei](https://github.com/hasenj/go-shirei).
The window contains a full-width text editor with line numbers.
Documents stay in memory and are not saved when the window closes.

Requires Go 1.25 or newer. Run from this directory:

```sh
go run ./cmd/melrose-ui
```

The text field supports selection, clipboard operations, and undo/redo through
Shirei's native editing widget. Cmd+3 prints `Line N: content` to the terminal
without changing the document. With a selection, it prints the selected text
and its starting line number. Otherwise, it prints the entire caret line.
The reported line or selection flashes green for 200 ms without changing
the document, caret, or selection.
Cmd+2 invokes inspect, Cmd+E invokes evaluate, and Cmd+4 or Cmd+K invokes kill.
These handlers currently log the line number and selected content only.
Additional shortcuts can be registered in `Editor.Bindings`.

The widget owns its text, markers, keyboard handlers, and flash state.
Create it once and call `Draw()` from the view function:

```go
config := &ui.Config{
	VariableListWidthPercentage: 25,
	EditorHeightPercentage:     75,
	EditorFlashDuration:        200 * time.Millisecond,
}
editor := ui.NewEditor("Start writing here.", config)
window := &ui.Window{
	Config: config,
	Label:  "Melrose",
	Editor: editor,
	List:   ui.VariableList{Items: []string{"First item", "Second item"}},
}
```

Import `melrose-ui/pkg/ui` to use the widget. `Window` owns the label and
outer layout. Call `window.Draw()` for the complete application view, or
`editor.Draw()` to embed the editor without a label.
The executable entry point lives in `cmd/melrose-ui`.

`VariableList` owns the scrolling list's items and rendering. Update
`window.List.Items` to change its contents. The list sits to the right of
the editor at the same height.
`Config.VariableListWidthPercentage` divides the usable row width, excluding the gap.
`EditorHeightPercentage` divides the remaining height below the title,
excluding gaps; the console fills the rest. Zero values default to 25 and 75,
respectively. Other values are clamped to 1-99. Set `window.Console` to update
the full-width read-only console beneath the editor and list.

The window and editor share the same `*Config`. An omitted window config
uses the editor's config; `NewEditor(text, nil)` creates a default config.
Change `config.EditorFlashDuration` to adjust subsequent flashes.

Build or render a headless snapshot:

```sh
go build -o /tmp/melrose-ui ./cmd/melrose-ui
go run ./cmd/melrose-ui -png /tmp/melrose-ui.png
```