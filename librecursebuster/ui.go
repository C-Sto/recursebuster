package librecursebuster

import (
	"fmt"
	"sync"

	ui "github.com/jroimartin/gocui"
)

func (s *State) StartUI(wg *sync.WaitGroup, quitChan chan struct{}) {
	g, err := ui.NewGui(ui.OutputNormal)

	if err != nil {
		panic(err)
	}
	s.ui = g
	defer func() {
		p, _ := s.ui.View("Main")
		lines := p.ViewBuffer()
		s.ui.Close()
		fmt.Print(lines)
		close(quitChan)
	}()
	s.ui.SetManagerFunc(layout)

	err = s.ui.SetKeybinding("", ui.KeyCtrlX, ui.ModNone, handleX)
	if err != nil {
		panic(err)
	}

	err = s.ui.SetKeybinding("", ui.KeyCtrlC, ui.ModNone, quit)
	if err != nil {
		panic(err)
	}

	err = s.ui.SetKeybinding("", ui.KeyPgup, ui.ModNone, ScrollUp)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.KeyPgdn, ui.ModNone, ScrollDown)
	if err != nil {
		panic(err)
	}
	wg.Done()
	err = s.ui.MainLoop()
	if err != nil && err != ui.ErrQuit {
		panic(err)
	}

}

func handleX(g *ui.Gui, v *ui.View) error {
	//vi, _ := g.View("Main")
	//close(gState.StopDir)
	select { //lol dope hack to stop it blocking
	case gState.StopDir <- struct{}{}:
	default:
	}
	//gState.StopDir <- struct{}{}
	//fmt.Fprintln(v, "X!!!")
	return nil
}

func quit(g *ui.Gui, v *ui.View) error {
	return ui.ErrQuit
}

func layout(g *ui.Gui) error {
	mX, mY := g.Size()

	_, err := g.SetView("Main", 0, 0, mX-1, mY-7)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	v, _ := g.View("Main")
	v.Autoscroll = true
	_, err = g.SetView("Status", 0, mY-6, mX-1, mY-1)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	return nil
}

//https://github.com/mephux/komanda-cli/blob/4b3c83ae8946d6eaf607d6d74158ff4a06343009/komanda/util.go
func ScrollUp(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, -1)
	return nil
}

// ScrollDown view by one
func ScrollDown(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, 1)
	return nil
}

// ScrollView by a given offset
func ScrollView(v *ui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+dy); err != nil {
			return err
		}
	}

	return nil
}
