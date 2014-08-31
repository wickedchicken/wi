// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package editor

import (
	"fmt"
	"github.com/maruel/wi/wi-plugin"
	"log"
	"time"
)

type commands struct {
	commands map[string]wi.Command
}

func (c *commands) Register(cmd wi.Command) bool {
	name := cmd.Name()
	_, ok := c.commands[name]
	c.commands[name] = cmd
	return !ok
}

func (c *commands) Get(cmd string) wi.Command {
	return c.commands[cmd]
}

func makeCommands() wi.Commands {
	return &commands{make(map[string]wi.Command)}
}

// Default commands

func cmdAlert(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	// TODO(maruel): Create an infobar that automatically dismiss itself after 5s.
	// TODO(maruel): Use a 5 seconds infobar.
	wi.RootWindow(w).NewChildWindow(makeAlertView(args[0]), wi.DockingFloating)
	log.Printf("Tree:\n%s", wi.RootWindow(w).Tree())
	//w2.Activate()
	go func() {
		<-time.After(5 * time.Second)
		// TODO(maruel): Dismiss.
	}()
}

func cmdBootstrapUI(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	statusWindowRoot := w.NewChildWindow(makeStatusViewRoot(), wi.DockingBottom)
	statusWindowRoot.NewChildWindow(makeStatusViewName(), wi.DockingLeft)
	statusWindowRoot.NewChildWindow(makeStatusViewPosition(), wi.DockingRight)
}

func cmdDocumentNew(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	if len(args) != 0 {
		cmd := wi.GetCommand(cd, w, "alias")
		cd.ExecuteCommand(w, "alert", cmd.LongDesc(cd, w))
	} else {
		w.NewChildWindow(makeView("New doc", -1, -1), wi.DockingFill)
	}
}

func cmdDocumentOpen(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	// The Window and View are created synchronously. The View is populated
	// asynchronously.
	log.Printf("Faking opening a file: %s", args)
}

func isDirtyRecurse(cd wi.CommandDispatcherFull, w wi.Window) bool {
	for _, child := range w.ChildrenWindows() {
		if isDirtyRecurse(cd, child) {
			return true
		}
		v := child.View()
		if v.IsDirty() {
			cd.ExecuteCommand(w, "alert", fmt.Sprintf(wi.GetStr(cd.CurrentLanguage(), viewDirty), v.Title()))
			return true
		}
	}
	return false
}

func cmdQuit(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	// TODO(maruel): For all the View, question if fine to quit via
	// view.IsDirty(). If not fine, "prompt" y/n to force quit. If n, stop there.
	// - Send a signal to each plugin.
	// - Send a signal back to the main loop.
	root := wi.RootWindow(w)
	if !isDirtyRecurse(cd, root) {
		quitFlag = true
		// PostDraw wakes up the command event loop so it detects it's time to
		// quit.
		cd.PostDraw()
	}
}

func cmdAlias(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	if args[0] == "window" {
	} else if args[0] == "global" {
		w = wi.RootWindow(w)
	} else {
		cmd := wi.GetCommand(cd, w, "alias")
		cd.ExecuteCommand(w, "alert", cmd.LongDesc(cd, w))
		return
	}
	alias := &wi.CommandAlias{args[1], args[2]}
	w.View().Commands().Register(alias)
}

func cmdKeyBind(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	location := args[0]
	modeName := args[1]
	keyName := args[2]
	cmdName := args[3]

	if location == "global" {
		w = wi.RootWindow(w)
	} else if location != "window" {
		cmd := wi.GetCommand(cd, w, "keybind")
		cd.ExecuteCommand(w, "alert", cmd.LongDesc(cd, w))
		return
	}

	var mode wi.KeyboardMode
	if modeName == "command" {
		mode = wi.CommandMode
	} else if modeName == "edit" {
		mode = wi.CommandMode
	} else if modeName == "all" {
		mode = wi.AllMode
	} else {
		cmd := wi.GetCommand(cd, w, "keybind")
		cd.ExecuteCommand(w, "alert", cmd.LongDesc(cd, w))
		return
	}
	w.View().KeyBindings().Set(mode, keyName, cmdName)
}

func cmdShowCommandWindow(c *wi.CommandImpl, cd wi.CommandDispatcherFull, w wi.Window, args ...string) {
	// Create the Window with the command view and attach it to the currently
	// focused Window.
	cmdWindow := makeCommandView()
	w.NewChildWindow(cmdWindow, wi.DockingFloating)
}

// Native commands.
var defaultCommands = []wi.Command{
	&wi.CommandImpl{
		"alert",
		1,
		cmdAlert,
		wi.WindowCategory,
		wi.LangMap{
			wi.LangEn: "Shows a modal message",
		},
		wi.LangMap{
			wi.LangEn: "Prints a message in a modal dialog box.",
		},
	},
	&wi.CommandImpl{
		"bootstrap_ui",
		0,
		cmdBootstrapUI,
		wi.WindowCategory,
		wi.LangMap{
			wi.LangEn: "Bootstraps the editor's UI",
		},
		wi.LangMap{
			wi.LangEn: "Bootstraps the editor's UI. This command is automatically run on startup and cannot be executed afterward. It adds the standard status bar. This command exists so it can be overriden by a plugin, so it can create its own status bar.",
		},
	},
	&wi.CommandImpl{
		"document_new",
		-1,
		cmdDocumentNew,
		wi.WindowCategory,
		wi.LangMap{
			wi.LangEn: "Create a new buffer",
		},
		wi.LangMap{
			wi.LangEn: "Create a new buffer.",
		},
	},
	&wi.CommandImpl{
		"document_open",
		-1,
		cmdDocumentOpen,
		wi.WindowCategory,
		wi.LangMap{
			wi.LangEn: "Opens a file in a new buffer",
		},
		wi.LangMap{
			wi.LangEn: "Opens a file in a new buffer.",
		},
	},
	&wi.CommandImpl{
		"quit",
		0,
		cmdQuit,
		wi.WindowCategory,
		wi.LangMap{
			wi.LangEn: "Quits",
		},
		wi.LangMap{
			wi.LangEn: "Quits the editor. Optionally bypasses writing the files to disk.",
		},
	},

	&wi.CommandImpl{
		"alias",
		3,
		cmdAlias,
		wi.CommandsCategory,
		wi.LangMap{
			wi.LangEn: "Binds an alias to another command",
		},
		wi.LangMap{
			// TODO(maruel): For complex aliasing, use macro?
			wi.LangEn: "Usage: alias [window|global] <alias> <name>\nBinds an alias to another command. The alias can either be local to the window or global",
		},
	},
	&wi.CommandImpl{
		"keybind",
		4,
		cmdKeyBind,
		wi.CommandsCategory,
		wi.LangMap{
			wi.LangEn: "Binds a keyboard mapping to a command",
		},
		wi.LangMap{
			wi.LangEn: "Usage: keybind [window|global] [command|edit|all] <key> <command>\nBinds a keyboard mapping to a command. The binding can be to the active view for view-specific key binding or to the root view for global key bindings.",
		},
	},
	&wi.CommandImpl{
		"show_command_window",
		0,
		cmdShowCommandWindow,
		wi.CommandsCategory,
		wi.LangMap{
			wi.LangEn: "Shows the interactive command window",
		},
		wi.LangMap{
			wi.LangEn: "This commands exists so it can be bound to a key to pop up the interactive command window.",
		},
	},

	&wi.CommandAlias{"new", "document_new"},
	&wi.CommandAlias{"open", "document_open"},
	&wi.CommandAlias{"q", "quit"},

	// DIRECTION = up/down/left/right
	// window_DIRECTION
	// cursor_move_DIRECTION
	// add_text/insert/delete
	// undo/redo
	// verb/movement/multiplier
	// Modes, select (both column and normal), command.
	// ...
}

// RegisterDefaultCommands registers the top-level native commands. This
// includes the window management commands, opening a new file buffer (it's a
// text editor after all) and help, quitting, etc. It doesn't includes handling
// a file buffer itself, it's up to the relevant view to add the corresponding
// commands. For example, "open" is implemented but "write" is not!
func RegisterDefaultCommands(dispatcher wi.Commands) {
	for _, cmd := range defaultCommands {
		dispatcher.Register(cmd)
	}
	for _, cmd := range windowCommands {
		dispatcher.Register(cmd)
	}
}
