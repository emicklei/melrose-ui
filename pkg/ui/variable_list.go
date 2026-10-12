package ui

import (
	"go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

type VariableList struct {
	Items []string
}

func (list *VariableList) Draw() {
	shirei.NextAccessRole("list")
	shirei.AssignAccess()
	widgets.VirtualListView(list, len(list.Items), func(index int) any {
		return index
	}, nil, func(index int, width float32) {
		shirei.Container(shirei.Attrs(shirei.FixSize(width, 28*shirei.ComfortScale()),
			shirei.Pad(6), shirei.Clip), func() {
			shirei.Label(list.Items[index], shirei.FontSize(14))
		})
	})
}
