package librecursebuster

import (
	"fmt"
	"strings"
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

	err = s.ui.SetKeybinding("", ui.KeyPgup, ui.ModNone, pgUp)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.KeyPgdn, ui.ModNone, pgDown)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.KeyArrowUp, ui.ModNone, scrollUp)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.KeyArrowDown, ui.ModNone, scrollDown)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.MouseWheelUp, ui.ModNone, scrollUp)
	if err != nil {
		panic(err)
	}
	err = s.ui.SetKeybinding("", ui.MouseWheelDown, ui.ModNone, scrollUp)
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

	v, err := g.SetView("Main", 0, 0, mX-1, mY-7)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	_, y := v.Size()
	_, oy := v.Origin()
	if oy > strings.Count(v.ViewBuffer(), "\n")-y-1 {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	}
	v.Title = "~Recursebuster V" + gState.Version + " by C_Sto (@C__Sto)~"
	_, err = g.SetView("Status", 0, mY-6, mX-1, mY-1)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	return nil
}

//https://github.com/mephux/komanda-cli/blob/4b3c83ae8946d6eaf607d6d74158ff4a06343009/komanda/util.go
func scrollUp(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, g, -1)
	return nil
}

// ScrollDown view by one
func scrollDown(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, g, 1)
	return nil
}

func pgUp(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, g, -10)
	return nil
}

// ScrollDown view by one
func pgDown(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	ScrollView(v, g, 10)
	return nil
}

func ScrollView(v *ui.View, g *ui.Gui, dy int) {
	// Grab the view that we want to scroll.
	// Get the size and position of the view.
	_, y := v.Size()
	ox, oy := v.Origin()
	v.Autoscroll = false
	e := v.SetOrigin(ox, oy+dy)
	if e != nil {
		//appease error check static analysis
	}
	if oy+dy > strings.Count(v.ViewBuffer(), "\n")-y-1 {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	}
}
