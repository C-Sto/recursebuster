package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/c-sto/recursebuster/pkg/recursebuster"

	ui "github.com/jroimartin/gocui"
)

var version = "UNSET"

//StartUI is called to begin the UI... stuff
func StartUI(uiWG *sync.WaitGroup, quitChan chan struct{}, s *recursebuster.State) {
	version = s.Cfg.Version
	g, err := ui.NewGui(ui.OutputNormal)
	if err != nil {
		panic(err)
	}
	//s.ui
	s.SetUI(g)
	defer func() {
		StopUI(g)
		close(quitChan)
	}()
	g.SetManagerFunc(layout)

	err = g.SetKeybinding("", ui.KeyCtrlX, ui.ModNone, s.HandleX)
	if err != nil {
		panic(err)
	}

	err = g.SetKeybinding("", ui.KeyCtrlC, ui.ModNone, quit)
	if err != nil {
		panic(err)
	}

	err = g.SetKeybinding("", ui.KeyPgup, ui.ModNone, pgUp)
	if err != nil {
		panic(err)
	}
	err = g.SetKeybinding("", ui.KeyPgdn, ui.ModNone, pgDown)
	if err != nil {
		panic(err)
	}
	err = g.SetKeybinding("", ui.KeyArrowUp, ui.ModNone, scrollUp)
	if err != nil {
		panic(err)
	}
	err = g.SetKeybinding("", ui.KeyArrowDown, ui.ModNone, scrollDown)
	if err != nil {
		panic(err)
	}

	err = g.SetKeybinding("", ui.KeyCtrlT, ui.ModNone, s.AddWorker)
	if err != nil {
		panic(err)
	}

	err = g.SetKeybinding("", ui.KeyCtrlY, ui.ModNone, s.StopWorker) //wtf? no shift modifier??
	if err != nil {
		panic(err)
	}
	/* Mouse stuff broke copying out of the terminal... not ideal
	err = s.ui.SetKeybinding("", ui.MouseWheelUp, ui.ModNone, scrollUp)
	if err != nil {
		panic(err
	}
	err = s.ui.SetKeybinding("", ui.MouseWheelDown, ui.ModNone, scrollDown)
	if err != nil {
		panic(err)
	}*/
	uiWG.Done()
	err = g.MainLoop()
	if err != nil && err != ui.ErrQuit {
		panic(err)
	}
}

//StopUI should be called when closing the program. It prints out the lines in the main view buffer to stdout, and closes the ui object
func StopUI(u *ui.Gui) {
	p, _ := u.View("Main")
	lines := p.ViewBuffer()
	u.Close()
	fmt.Print(lines)
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
	v.Title = "~Recursebuster V" + version + " by C_Sto (@C__Sto)~"
	_, err = g.SetView("Status", 0, mY-6, mX-1, mY-1)
	if err != nil && err != ui.ErrUnknownView {
		return err
	}
	return nil
}

//https://github.com/mephux/komanda-cli/blob/4b3c83ae8946d6eaf607d6d74158ff4a06343009/komanda/util.go
func scrollUp(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	scrollView(v, g, -1)
	return nil
}

// ScrollDown view by one
func scrollDown(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	scrollView(v, g, 1)
	return nil
}

func pgUp(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	scrollView(v, g, -10)
	return nil
}

// ScrollDown view by one
func pgDown(g *ui.Gui, cv *ui.View) error {
	v, _ := g.View("Main")
	scrollView(v, g, 10)
	return nil
}

func scrollView(v *ui.View, g *ui.Gui, dy int) {
	// Grab the view that we want to scroll.
	// Get the size and position of the view.
	_, y := v.Size()
	ox, oy := v.Origin()
	v.Autoscroll = false
	e := v.SetOrigin(ox, oy+dy)
	if e != nil {
		return
		//appease error check static analysis
	}
	if oy+dy > strings.Count(v.ViewBuffer(), "\n")-y-1 {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	}
}
