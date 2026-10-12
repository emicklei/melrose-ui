package ui

import (
	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

type Window struct {
	Config  *Config
	Label   string
	Editor  *Editor
	List    VariableList
	Console string
}

func (window *Window) Draw() {
	if window.Config == nil {
		window.Config = window.Editor.Config
	}
	window.Editor.Config = window.Config
	listWidthPercentage := window.Config.VariableListWidthPercentage
	if listWidthPercentage == 0 {
		listWidthPercentage = 25
	}
	listWidthPercentage = min(max(listWidthPercentage, 1), 99)
	editorHeightPercentage := window.Config.EditorHeightPercentage
	if editorHeightPercentage == 0 {
		editorHeightPercentage = 75
	}
	editorHeightPercentage = min(max(editorHeightPercentage, 1), 99)
	shirei.Container(shirei.Attrs(shirei.Viewport, shirei.Pad(20), shirei.Gap(12), shirei.Background(0, 0, 97, 1)), func() {
		shirei.Label(window.Label, shirei.FontSize(22), shirei.FontWeight(shirei.WeightBold))
		shirei.Container(shirei.Attrs(shirei.Expand, shirei.Row, shirei.Gap(12),
			shirei.Grow(editorHeightPercentage), shirei.MinHeight(0), func(attrs *shirei.AttrSet) {
				attrs.ExtrinsicSize = true
			}), func() {
			shirei.Container(shirei.Attrs(shirei.Expand, shirei.Grow(100-listWidthPercentage),
				shirei.MinWidth(0), func(attrs *shirei.AttrSet) {
					attrs.ExtrinsicSize = true
				}), window.Editor.Draw)
			shirei.NextAccessName("list")
			shirei.ContainerWithKey("list", shirei.Attrs(
				shirei.Grow(listWidthPercentage), shirei.Expand, shirei.MinWidth(0),
				func(attrs *shirei.AttrSet) { attrs.ExtrinsicSize = true },
				shirei.Clip, shirei.Background(0, 0, 94, 1)), window.List.Draw)
		})
		shirei.NextAccessName("console")
		shirei.ContainerWithKey("console", shirei.Attrs(shirei.Expand,
			shirei.Grow(100-editorHeightPercentage), shirei.MinHeight(0),
			shirei.MinWidth(0), shirei.Clip, shirei.Background(0, 0, 0, 1), func(attrs *shirei.AttrSet) {
				attrs.ExtrinsicSize = true
			}), func() {
			shirei.NextAccessRole("text")
			shirei.NextAccessEditable(false, true)
			shirei.NextAccessValue(window.Console)
			shirei.AssignAccess()
			shirei.Container(shirei.Attrs(shirei.Expand, shirei.Grow(1), shirei.Pad(8), shirei.Clip), func() {
				widgets.LargeText(window.Console, shirei.FontSize(12), shirei.TextColor(0, 0, 80, 1))
			})
		})
	})
}
