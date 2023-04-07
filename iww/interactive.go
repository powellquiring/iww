package iww

import "github.com/rivo/tview"

func Interactive(apikey string) error {
	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText(text)
	}
	grid := tview.NewGrid().
		SetRows(0, 3).
		SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(newPrimitive("Bodyxxx"), 0, 0, 1, 3, 0, 0, false).
		AddItem(newPrimitive("Command"), 1, 0, 1, 3, 0, 0, false)

	if err := tview.NewApplication().SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}
	return nil
}
