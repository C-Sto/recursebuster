package librecursebuster

import (
	"fmt"
	"os"

	ui "github.com/jroimartin/gocui"
)

func (s *State) StartUI() {
	g, err := ui.NewGui(ui.OutputNormal)

	if err != nil {
		panic(err)
		return
	}
	s.ui = g
	defer s.ui.Close()

	s.ui.SetManagerFunc(layout)

	err = s.ui.SetKeybinding("", ui.KeyCtrlX, ui.ModNone, handleX)
	if err != nil {
		panic(err)
		return
	}

	err = s.ui.SetKeybinding("", ui.KeyCtrlC, ui.ModNone, quit)
	if err != nil {
		panic(err)
		return
	}

	err = s.ui.MainLoop()
	if err != nil && err != ui.ErrQuit {
		panic(err)
		return
	}

}

func handleX(g *ui.Gui, v *ui.View) error {
	vi, _ := g.View("Main")
	fmt.Fprintln(vi, "asdf")
	//fmt.Fprintln(v, "X!!!")
	return nil
}

func quit(g *ui.Gui, v *ui.View) error {
	os.Exit(1)
	return ui.ErrQuit
}

func layout(g *ui.Gui) error {
	mX, mY := g.Size()
	_, err := g.SetView("Main", 0, 0, mX-1, mY-1)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	return nil
}
