// Copyright 2013 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package editor

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/maruel/wi/wi_core"
)

var singleBorder = []rune{'\u2500', '\u2502', '\u250D', '\u2510', '\u2514', '\u2518'}
var doubleBorder = []rune{'\u2550', '\u2551', '\u2554', '\u2557', '\u255a', '\u255d'}

type drawnBorder int

const (
	// TODO(maruel): For combo box (e.g. drop down list of suggestions), it
	// should be drawBorderLeftBottomRight.

	drawnBorderNone drawnBorder = iota
	drawnBorderLeft
	drawnBorderRight
	drawnBorderTop
	drawnBorderBottom
	drawnBorderAll
)

// window implements wi_core.Window. It keeps its own buffer of its display.
type window struct {
	id              int // window ID relative to the parent.
	lastChildID     int // last ID used for a children window.
	parent          *window
	cd              wi_core.CommandDispatcherFull
	childrenWindows []*window
	windowBuffer    *wi_core.Buffer // includes the border
	rect            wi_core.Rect    // Window Rect as described in wi_core.Window.Rect().
	clientAreaRect  wi_core.Rect    // Usable area within the Window, the part not obscured by borders.
	viewRect        wi_core.Rect    // Window View Rect, which is the client area not used by childrenWindows.
	view            wi_core.View    // View that renders the content. It may be nil if this Window has no content.
	docking         wi_core.DockingType
	border          wi_core.BorderType
	effectiveBorder drawnBorder // effectiveBorder automatically collapses borders when the Window Rect is too small and is based on docking.
	fg              wi_core.RGB // Default text color, to be used in borders.
	bg              wi_core.RGB // Default background color, to be used in borders
}

func (w *window) String() string {
	return fmt.Sprintf("Window(%s, %s, %v)", w.ID(), w.View().Title(), w.Rect())
}

func (w *window) PostCommands(cmds [][]string) {
	w.cd.PostCommands(cmds)
}

func (w *window) ID() string {
	if w.parent == nil {
		// editor.rootWindow.id is always 0.
		return fmt.Sprintf("%d", w.id)
	}
	return fmt.Sprintf("%s:%d", w.parent.ID(), w.id)
}

// Returns a string representing the tree.
func (w *window) Tree() string {
	// Not the most performant implementation but does the job.
	out := w.String() + "\n"
	for _, child := range w.childrenWindows {
		for _, line := range strings.Split(child.Tree(), "\n") {
			if line != "" {
				out += ("  " + line + "\n")
			}
		}
	}
	return out
}

func (w *window) Parent() wi_core.Window {
	// TODO(maruel): Understand why this is necessary at all.
	if w.parent != nil {
		return w.parent
	}
	return nil
}

func (w *window) ChildrenWindows() []wi_core.Window {
	out := make([]wi_core.Window, len(w.childrenWindows))
	for i, v := range w.childrenWindows {
		out[i] = v
	}
	return out
}

// Recursively detach a window tree.
func detachRecursively(w *window) {
	for _, c := range w.childrenWindows {
		detachRecursively(c)
	}
	w.parent = nil
	w.childrenWindows = nil
}

func recurseIDToWindow(w *window, fullID string) *window {
	parts := strings.SplitN(fullID, ":", 2)
	intID, err := strconv.Atoi(parts[0])
	if err != nil {
		// Element is not a valid number, it's an invalid reference.
		return nil
	}
	for _, child := range w.childrenWindows {
		if child.id == intID {
			if len(parts) == 2 {
				return recurseIDToWindow(child, parts[1])
			}
			return child
		}
	}
	return nil
}

// Converts a wi_core.Window.ID() to a window pointer. Returns nil if invalid.
//
// "0" is the special reference to the root window.
func (e *editor) idToWindow(id string) *window {
	cur := e.rootWindow
	if id != "0" {
		if !strings.HasPrefix(id, "0:") {
			log.Printf("Invalid id: %s", id)
			return nil
		}
		cur = recurseIDToWindow(cur, id[2:])
	}
	return cur
}

func (w *window) Rect() wi_core.Rect {
	return w.rect
}

// SetRect sets the rect of this Window, based on the parent's Window own
// Rect(). It updates Rect() and synchronously updates the child Window that
// are not DockingFloating.
func (w *window) SetRect(rect wi_core.Rect) {
	// SetRect() recreates the buffer and immediately draws the borders.
	if !w.rect.Eq(rect) {
		w.rect = rect
		// Internal consistency check.
		if w.parent != nil {
			if !w.rect.In(w.parent.clientAreaRect) {
				panic(fmt.Sprintf("Child %v doesn't fit parent's client area %v: %v", w, w.parent, w.parent.clientAreaRect))
			}
		}

		w.windowBuffer = wi_core.NewBuffer(w.rect.Width, w.rect.Height)
		w.updateBorder()
	}
	// Still flow the call through children Window, so DockingFloating are
	// properly updated.
	w.resizeChildren()
}

// calculateEffectiveBorder calculates window.effectiveBorder.
func calculateEffectiveBorder(r wi_core.Rect, d wi_core.DockingType) drawnBorder {
	switch d {
	case wi_core.DockingFill:
		return drawnBorderNone

	case wi_core.DockingFloating:
		if r.Width >= 5 && r.Height >= 3 {
			return drawnBorderAll
		}
		return drawnBorderNone

	case wi_core.DockingLeft:
		if r.Width > 1 && r.Height > 0 {
			return drawnBorderRight
		}
		return drawnBorderNone

	case wi_core.DockingRight:
		if r.Width > 1 && r.Height > 0 {
			return drawnBorderLeft
		}
		return drawnBorderNone

	case wi_core.DockingTop:
		if r.Height > 1 && r.Width > 0 {
			return drawnBorderBottom
		}
		return drawnBorderNone

	case wi_core.DockingBottom:
		if r.Height > 1 && r.Width > 0 {
			return drawnBorderTop
		}
		return drawnBorderNone

	default:
		panic("Unknown DockingType")
	}
}

// resizeChildren() resizes all the children Window.
func (w *window) resizeChildren() {
	log.Printf("%s.resizeChildren()", w)
	// When borders are used, w.clientAreaRect.X and .Y are likely 1.
	remaining := w.clientAreaRect
	var fill *window
	for _, child := range w.childrenWindows {
		switch child.Docking() {
		case wi_core.DockingFill:
			fill = child

		case wi_core.DockingFloating:
			// Floating uses its own thing.
			// TODO(maruel): Not clean. Doesn't handle root Window resize properly.
			child.SetRect(child.Rect())

		case wi_core.DockingLeft:
			width, _ := child.View().NaturalSize()
			if width >= remaining.Width {
				width = remaining.Width
			} else if child.border != wi_core.BorderNone {
				width++
			}
			tmp := remaining
			tmp.Width = width
			remaining.X += width
			remaining.Width -= width
			child.SetRect(tmp)

		case wi_core.DockingRight:
			width, _ := child.View().NaturalSize()
			if width >= remaining.Width {
				width = remaining.Width
			} else if child.border != wi_core.BorderNone {
				width++
			}
			tmp := remaining
			tmp.X += (remaining.Width - width)
			tmp.Width = width
			remaining.Width -= width
			child.SetRect(tmp)

		case wi_core.DockingTop:
			_, height := child.View().NaturalSize()
			if height >= remaining.Height {
				height = remaining.Height
			} else if child.border != wi_core.BorderNone {
				height++
			}
			tmp := remaining
			tmp.Height = height
			remaining.Y += height
			remaining.Height -= height
			child.SetRect(tmp)

		case wi_core.DockingBottom:
			_, height := child.View().NaturalSize()
			if height >= remaining.Height {
				height = remaining.Height
			} else if child.border != wi_core.BorderNone {
				height++
			}
			tmp := remaining
			tmp.Y += (remaining.Height - height)
			tmp.Height = height
			remaining.Height -= height
			child.SetRect(tmp)

		default:
			panic("Fill me")
		}
	}
	if fill != nil {
		fill.SetRect(remaining)
		w.viewRect.X = 0
		w.viewRect.Y = 0
		w.viewRect.Width = 0
		w.viewRect.Height = 0
		w.view.SetSize(0, 0)
	} else {
		w.viewRect = remaining
		w.view.SetSize(w.viewRect.Width, w.viewRect.Height)
	}
	wi_core.PostCommand(w, "editor_redraw")
}

func (w *window) Buffer() *wi_core.Buffer {
	// TODO(maruel): Redo API.
	// Opportunistically refresh the view buffer.
	if w.viewRect.Width != 0 && w.viewRect.Height != 0 {
		b := w.windowBuffer.SubBuffer(w.viewRect)
		b.Blit(w.view.Buffer())
	}
	return w.windowBuffer
}

func (w *window) Docking() wi_core.DockingType {
	return w.docking
}

func (w *window) SetView(view wi_core.View) {
	if view != w.view {
		w.view = view
		b := w.windowBuffer.SubBuffer(w.viewRect)
		b.Fill(w.cell(' '))
		wi_core.PostCommand(w, "editor_redraw")
	}
	panic("To test")
}

// updateBorder calculates w.effectiveBorder, w.clientAreaRect and draws the
// borders right away in the Window's buffer.
//
// It's called by SetRect() and will be called by SetBorder (if ever
// implemented).
func (w *window) updateBorder() {
	if w.border == wi_core.BorderNone {
		w.effectiveBorder = drawnBorderNone
	} else {
		w.effectiveBorder = calculateEffectiveBorder(w.rect, w.docking)
	}

	s := doubleBorder
	if w.border == wi_core.BorderSingle {
		s = singleBorder
	}

	switch w.effectiveBorder {
	case drawnBorderNone:
		w.clientAreaRect = wi_core.Rect{0, 0, w.rect.Width, w.rect.Height}

	case drawnBorderLeft:
		w.clientAreaRect = wi_core.Rect{1, 0, w.rect.Width - 1, w.rect.Height}
		w.windowBuffer.SubBuffer(wi_core.Rect{0, 0, 1, w.rect.Height}).Fill(w.cell(s[1]))

	case drawnBorderRight:
		w.clientAreaRect = wi_core.Rect{0, 0, w.rect.Width - 1, w.rect.Height}
		w.windowBuffer.SubBuffer(wi_core.Rect{w.rect.Width - 1, 0, 1, w.rect.Height}).Fill(w.cell(s[1]))

	case drawnBorderTop:
		w.clientAreaRect = wi_core.Rect{0, 1, w.rect.Width, w.rect.Height - 1}
		w.windowBuffer.SubBuffer(wi_core.Rect{0, 0, w.rect.Width, 1}).Fill(w.cell(s[0]))

	case drawnBorderBottom:
		w.clientAreaRect = wi_core.Rect{0, 0, w.rect.Width, w.rect.Height - 1}
		w.windowBuffer.SubBuffer(wi_core.Rect{0, w.rect.Height - 1, w.rect.Width, 1}).Fill(w.cell(s[0]))

	case drawnBorderAll:
		w.clientAreaRect = wi_core.Rect{1, 1, w.rect.Width - 2, w.rect.Height - 2}
		// Corners.
		w.windowBuffer.Set(0, 0, w.cell(s[2]))
		w.windowBuffer.Set(0, w.rect.Height-1, w.cell(s[4]))
		w.windowBuffer.Set(w.rect.Width-1, 0, w.cell(s[3]))
		w.windowBuffer.Set(w.rect.Width-1, w.rect.Height-1, w.cell(s[5]))
		// Lines.
		w.windowBuffer.SubBuffer(wi_core.Rect{1, 0, w.rect.Width - 2, 1}).Fill(w.cell(s[0]))
		w.windowBuffer.SubBuffer(wi_core.Rect{1, w.rect.Height - 1, w.rect.Width - 2, w.rect.Height - 1}).Fill(w.cell(s[0]))
		w.windowBuffer.SubBuffer(wi_core.Rect{0, 1, 1, w.rect.Height - 2}).Fill(w.cell(s[1]))
		w.windowBuffer.SubBuffer(wi_core.Rect{w.rect.Width - 1, 1, w.rect.Width - 1, w.rect.Height - 2}).Fill(w.cell(s[1]))

	default:
		panic("Unknown drawnBorder")
	}

	if w.clientAreaRect.Width < 0 {
		w.clientAreaRect.Width = 0
		panic("Fix this case")
	}
	if w.clientAreaRect.Height < 0 {
		w.clientAreaRect.Height = 0
		panic("Fix this case")
	}
}

func (w *window) cell(r rune) wi_core.Cell {
	return wi_core.MakeCell(r, w.fg, w.bg)
}

func (w *window) View() wi_core.View {
	return w.view
}

func makeWindow(parent *window, view wi_core.View, docking wi_core.DockingType) *window {
	log.Printf("makeWindow(%s, %s, %s)", parent, view.Title(), docking)
	var cd wi_core.CommandDispatcherFull
	id := 0
	if parent != nil {
		cd = parent.cd
		parent.lastChildID++
		id = parent.lastChildID
	}
	// It's more complex than that but it's a fine default.
	border := wi_core.BorderNone
	if docking == wi_core.DockingFloating {
		border = wi_core.BorderDouble
	}
	return &window{
		id:      id,
		parent:  parent,
		cd:      cd,
		view:    view,
		docking: docking,
		border:  border,
		fg:      wi_core.White,
		bg:      wi_core.Black,
	}
}

// drawRecurse recursively draws the Window tree into buffer out.
func drawRecurse(w *window, offsetX, offsetY int, out *wi_core.Buffer) {
	log.Printf("drawRecurse(%s, %d, %d); %v", w.View().Title(), offsetX, offsetY, w.Rect())
	if w.Docking() == wi_core.DockingFloating {
		// Floating Window are relative to the screen, not the parent Window.
		offsetX = 0
		offsetY = 0
	}
	// TODO(maruel): Only draw non-occuled Windows!
	dest := w.Rect()
	dest.X += offsetX
	dest.Y += offsetY
	out.SubBuffer(dest).Blit(w.Buffer())

	fillFound := false
	for _, child := range w.childrenWindows {
		// In the case of DockingFill, only the first one should be drawn. In
		// particular, the DockingFloating child of an hidden DockingFill will not
		// be drawn.
		if child.docking == wi_core.DockingFill {
			if fillFound {
				continue
			}
			fillFound = true
		}
		drawRecurse(child, dest.X, dest.Y, out)
	}
}

// Commands

func cmdWindowActivate(c *privilegedCommandImpl, e *editor, w *window, args ...string) {
	windowName := args[0]

	child := e.idToWindow(windowName)
	if child == nil {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), isNotValidWindow), windowName))
		return
	}
	e.activateWindow(child)
}

func cmdWindowClose(c *privilegedCommandImpl, e *editor, w *window, args ...string) {
	windowName := args[0]

	child := e.idToWindow(windowName)
	if child == nil {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), isNotValidWindow), windowName))
		return
	}
	for i, v := range child.parent.childrenWindows {
		if v == child {
			copy(w.childrenWindows[i:], w.childrenWindows[i+1:])
			w.childrenWindows[len(w.childrenWindows)-1] = nil
			w.childrenWindows = w.childrenWindows[:len(w.childrenWindows)-1]
			detachRecursively(v)
			wi_core.PostCommand(e, "editor_redraw")
			return
		}
	}
}

func cmdWindowLog(c *wi_core.CommandImpl, cd wi_core.CommandDispatcherFull, w wi_core.Window, args ...string) {
	root := wi_core.RootWindow(w)
	log.Printf("Window tree:\n%s", root.Tree())
}

func cmdWindowNew(c *privilegedCommandImpl, e *editor, w *window, args ...string) {
	windowName := args[0]
	dockingName := args[1]
	viewFactoryName := args[2]

	parent := e.idToWindow(windowName)
	if parent == nil {
		if viewFactoryName != "infobar_alert" {
			e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), isNotValidWindow), windowName))
		}
		return
	}

	docking := wi_core.StringToDockingType(dockingName)
	if docking == wi_core.DockingUnknown {
		if viewFactoryName != "infobar_alert" {
			e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), invalidDocking), dockingName))
		}
		return
	}
	// TODO(maruel): Only the first child Window with DockingFill is visible.
	// TODO(maruel): Reorder .childrenWindows with
	// CommandDispatcherFull.ActivateWindow() but only with DockingFill.
	// TODO(maruel): Also allow DockingFloating.
	//if docking != wi_core.DockingFill {
	for _, child := range parent.childrenWindows {
		if child.Docking() == docking {
			if viewFactoryName != "infobar_alert" {
				e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), cantAddTwoWindowWithSameDocking), docking))
			}
			return
		}
	}
	//}

	viewFactory, ok := e.viewFactories[viewFactoryName]
	if !ok {
		if viewFactoryName != "infobar_alert" {
			e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), invalidViewFactory), viewFactoryName))
		}
		return
	}
	view := viewFactory(args[3:]...)

	child := makeWindow(parent, view, docking)
	if docking == wi_core.DockingFloating {
		width, height := view.NaturalSize()
		if child.border != wi_core.BorderNone {
			width += 2
			height += 2
		}
		// TODO(maruel): Handle when width or height > scren size.
		// TODO(maruel): Not clean. Doesn't handle root Window resize properly.
		rootRect := e.rootWindow.Rect()
		child.rect.X = (rootRect.Width - width - 1) / 2
		child.rect.Y = (rootRect.Height - height - 1) / 2
		child.rect.Width = width
		child.rect.Height = height
	}
	parent.childrenWindows = append(parent.childrenWindows, child)
	parent.resizeChildren()
	e.activateWindow(child)
}

func cmdWindowSetDocking(c *privilegedCommandImpl, e *editor, w *window, args ...string) {
	windowName := args[0]
	dockingName := args[1]

	child := e.idToWindow(windowName)
	if child == nil {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), isNotValidWindow), windowName))
		return
	}
	docking := wi_core.StringToDockingType(dockingName)
	if docking == wi_core.DockingUnknown {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), invalidDocking), dockingName))
		return
	}
	if w.docking != docking {
		// TODO(maruel): Check no other parent's child window have the same dock.
		w.docking = docking
		w.parent.resizeChildren()
		wi_core.PostCommand(w, "editor_redraw")
	}
}

func cmdWindowSetRect(c *privilegedCommandImpl, e *editor, w *window, args ...string) {
	windowName := args[0]

	child := e.idToWindow(windowName)
	if child == nil {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), isNotValidWindow), windowName))
		return
	}
	r := wi_core.Rect{}
	var err1, err2, err3, err4 error
	r.X, err1 = strconv.Atoi(args[1])
	r.Y, err2 = strconv.Atoi(args[2])
	r.Width, err3 = strconv.Atoi(args[3])
	r.Height, err4 = strconv.Atoi(args[4])
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		e.ExecuteCommand(w, "alert", fmt.Sprintf(wi_core.GetStr(e.CurrentLanguage(), invalidRect), args[1], args[2], args[3], args[4]))
		return
	}
	child.SetRect(r)
}

// RegisterWindowCommands registers all the commands relative to window
// management.
func RegisterWindowCommands(dispatcher wi_core.Commands) {
	var windowCommands = []wi_core.Command{
		&privilegedCommandImpl{
			"window_activate",
			1,
			cmdWindowActivate,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Activate a window",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Active a window. This means the Window will have keyboard focus.",
			},
		},
		&privilegedCommandImpl{
			"window_close",
			1,
			cmdWindowClose,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Closes a window",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Closes a window. Note that any window can be closed and all the child window will be destroyed at the same time.",
			},
		},
		&wi_core.CommandImpl{
			"window_log",
			0,
			cmdWindowLog,
			wi_core.DebugCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Logs the window tree",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Logs the window tree, this is only relevant if -verbose is used.",
			},
		},
		&privilegedCommandImpl{
			"window_new",
			-1,
			cmdWindowNew,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Creates a new window",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Usage: window_new <parent> <docking> <view name> <view args...>\nCreates a new window. The new window is created as a child to the specified parent. It creates inside the window the view specified. The Window is activated. It is invalid to add a child Window with the same docking as one already present.",
			},
		},
		&privilegedCommandImpl{
			"window_set_docking",
			2,
			cmdWindowSetDocking,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Change the docking of a window",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Changes the docking of this Window relative to the parent window. This will forces an invalidation and a redraw.",
			},
		},
		&privilegedCommandImpl{
			"window_set_rect",
			5,
			cmdWindowSetRect,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Move a window",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Usage: window_set_rect <window> <x> <y> <w> <h>\nMoves a Window relative to the parent window, unless it is floating, where it is relative to the view port.",
			},
		},
		// 'screenshot', mainly for unit test; open a new buffer with the screenshot, so it can be saved with 'w'.
	}
	for _, cmd := range windowCommands {
		dispatcher.Register(cmd)
	}
}
