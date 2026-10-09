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
Cmd+4 adds a red dot to the left of the caret line's number. Multiple lines
can be marked; markers are kept by line number for the current session.
Additional shortcuts can be registered in `MelroseEditor.Bindings`.

The widget owns its text, markers, keyboard handlers, and flash state.
Create it once and call `Draw()` from the view function:

```go
editor := ui.NewMelroseEditor("Start writing here.", ui.Config{
	FlashDuration: 200 * time.Millisecond,
})
```

Import `melrose-ui/pkg/ui` to use the widget. Call `editor.DrawRoot()` for
the complete application view, or `editor.Draw()` to embed the editor.
The executable entry point lives in `cmd/melrose-ui`.

Change `editor.Config.FlashDuration` to adjust subsequent flashes.

Build or render a headless snapshot:

```sh
go build -o /tmp/melrose-ui ./cmd/melrose-ui
go run ./cmd/melrose-ui -png /tmp/melrose-ui.png
```