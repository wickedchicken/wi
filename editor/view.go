// Copyright 2013 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package editor

import (
	"log"
	"time"
	"unicode/utf8"

	"github.com/maruel/wi/wi_core"
)

// TODO(maruel): Plugable drawing function.
type drawInto func(v wi_core.View, buffer wi_core.Buffer)

type view struct {
	commands      wi_core.Commands
	keyBindings   wi_core.KeyBindings
	title         string
	isDirty       bool
	isDisabled    bool
	naturalX      int
	naturalY      int
	actualX       int
	actualY       int
	onAttach      func(v *view, w wi_core.Window)
	defaultFormat wi_core.CellFormat
	buffer        *wi_core.Buffer
}

// wi_core.View interface.

func (v *view) Commands() wi_core.Commands {
	return v.commands
}

func (v *view) KeyBindings() wi_core.KeyBindings {
	return v.keyBindings
}

func (v *view) Title() string {
	return v.title
}

func (v *view) IsDirty() bool {
	return v.isDirty
}

func (v *view) IsDisabled() bool {
	return v.isDisabled
}

func (v *view) NaturalSize() (x, y int) {
	return v.naturalX, v.naturalY
}

func (v *view) SetSize(x, y int) {
	log.Printf("View(%s).SetSize(%d, %d)", v.Title(), x, y)
	v.actualX = x
	v.actualY = y
	v.buffer = wi_core.NewBuffer(x, y)
}

func (v *view) OnAttach(w wi_core.Window) {
	if v.onAttach != nil {
		v.onAttach(v, w)
	}
}

func (v *view) DefaultFormat() wi_core.CellFormat {
	// TODO(maruel): if v.defaultFormat.Empty() { return v.Window().Parent().DefaultFormat() }
	return v.defaultFormat
}

// A disabled static view.
type staticDisabledView struct {
	view
}

func (v *staticDisabledView) Buffer() *wi_core.Buffer {
	// TODO(maruel): Use the parent view format by default. No idea how to
	// surface this information here. Cost is at least a RPC, potentially
	// multiple when multiple plugins are involved in the tree.
	v.buffer.Fill(wi_core.Cell{' ', v.defaultFormat})
	v.buffer.DrawString(v.Title(), 0, 0, v.defaultFormat)
	return v.buffer
}

// Empty non-editable window.
func makeStaticDisabledView(title string, naturalX, naturalY int) *staticDisabledView {
	return &staticDisabledView{
		view{
			commands:      makeCommands(),
			keyBindings:   makeKeyBindings(),
			title:         title,
			isDisabled:    true,
			naturalX:      naturalX,
			naturalY:      naturalY,
			defaultFormat: wi_core.CellFormat{Fg: wi_core.Red, Bg: wi_core.Black},
		},
	}
}

// The status line is a hierarchy of Window, one for each element, each showing
// a single item.
func statusRootViewFactory(args ...string) wi_core.View {
	// TODO(maruel): OnResize(), query the root Window size, if y<=5 or x<=15,
	// set the root status Window to y=0, so that it becomes effectively
	// invisible when the editor window is too small.
	v := makeStaticDisabledView("Status Root", 1, 1)
	v.defaultFormat.Bg = wi_core.LightGray
	return v
}

func statusNameViewFactory(args ...string) wi_core.View {
	// View name.
	// TODO(maruel): Register events of Window activation, make itself Invalidate().
	v := makeStaticDisabledView("Status Name", 15, 1)
	// TODO(maruel): Set to black and have it use the parent's colors.
	v.defaultFormat.Bg = wi_core.LightGray
	return v
}

func statusPositionViewFactory(args ...string) wi_core.View {
	// Position, % of file.
	// TODO(maruel): Register events of movement, make itself Invalidate().
	v := makeStaticDisabledView("Status Position", 15, 1)
	// TODO(maruel): Set to black and have it use the parent's colors.
	v.defaultFormat.Bg = wi_core.LightGray
	return v
}

type commandView struct {
	view
}

func (v *commandView) Buffer() *wi_core.Buffer {
	v.buffer.Fill(wi_core.Cell{' ', v.defaultFormat})
	v.buffer.DrawString(v.Title(), 0, 0, v.defaultFormat)
	return v.buffer
}

// The command dialog box.
// TODO(maruel): Position it 5 lines below the cursor in the parent Window's
// View. Do this via onAttach.
func commandViewFactory(args ...string) wi_core.View {
	return &commandView{
		view{
			commands:      makeCommands(),
			keyBindings:   makeKeyBindings(),
			title:         "Command",
			naturalX:      30,
			naturalY:      1,
			defaultFormat: wi_core.CellFormat{Fg: wi_core.Green, Bg: wi_core.Black},
		},
	}
}

type documentView struct {
	view
}

func (v *documentView) Buffer() *wi_core.Buffer {
	v.buffer.Fill(wi_core.Cell{' ', v.defaultFormat})
	v.buffer.DrawString(v.Title(), 0, 0, v.defaultFormat)
	return v.buffer
}

func documentViewFactory(args ...string) wi_core.View {
	// TODO(maruel): Sort out "use max space".
	//onAttach
	return &documentView{
		view{
			commands:      makeCommands(),
			keyBindings:   makeKeyBindings(),
			title:         "<Empty document>",
			naturalX:      100,
			naturalY:      100,
			defaultFormat: wi_core.CellFormat{Fg: wi_core.BrightYellow, Bg: wi_core.Black},
		},
	}
}

func infobarAlertViewFactory(args ...string) wi_core.View {
	out := "Alert: " + args[0]
	l := utf8.RuneCountInString(out)
	v := makeStaticDisabledView(out, l, 1)
	v.onAttach = func(v *view, w wi_core.Window) {
		go func() {
			// Dismiss after 5 seconds.
			<-time.After(5 * time.Second)
			wi_core.PostCommand(w, "window_close", w.ID())
		}()
	}
	return v
}

// RegisterDefaultViewFactories registers the builtins views factories.
func RegisterDefaultViewFactories(e Editor) {
	e.RegisterViewFactory("command", commandViewFactory)
	e.RegisterViewFactory("infobar_alert", infobarAlertViewFactory)
	e.RegisterViewFactory("new_document", documentViewFactory)
	e.RegisterViewFactory("status_name", statusNameViewFactory)
	e.RegisterViewFactory("status_position", statusPositionViewFactory)
	e.RegisterViewFactory("status_root", statusRootViewFactory)
}

// Commands

// RegisterViewCommands registers view-related commands
func RegisterViewCommands(dispatcher wi_core.Commands) {
	cmds := []wi_core.Command{}
	for _, cmd := range cmds {
		dispatcher.Register(cmd)
	}
}
