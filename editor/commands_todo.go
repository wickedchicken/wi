// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package editor

import (
	"log"

	"github.com/maruel/wi/wi_core"
)

func cmdShell(c *wi_core.CommandImpl, cd wi_core.CommandDispatcherFull, w wi_core.Window, args ...string) {
	log.Printf("Faking opening a new shell: %s", args)
}

func cmdDoc(c *wi_core.CommandImpl, cd wi_core.CommandDispatcherFull, w wi_core.Window, args ...string) {
	// TODO(maruel): Grab the current word under selection if no args is
	// provided. Pass this token to shell.
	docArgs := make([]string, len(args)+1)
	docArgs[0] = "doc"
	copy(docArgs[1:], args)
	//dispatcher.Execute(w, "shell", docArgs...)
}

func cmdHelp(c *wi_core.CommandImpl, cd wi_core.CommandDispatcherFull, w wi_core.Window, args ...string) {
	// TODO(maruel): Creates a new Window with a ViewHelp.
	log.Printf("Faking help: %s", args)
}

// RegisterTodoCommands registers the top-level native commands that are yet to
// be implemented.
//
// TODO(maruel): Implement these commands properly and move to the right place.
func RegisterTodoCommands(dispatcher wi_core.Commands) {
	cmds := []wi_core.Command{
		&wi_core.CommandImpl{
			"doc",
			-1,
			cmdDoc,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Search godoc documentation",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Uses the 'doc' tool to get documentation about the text under the cursor.",
			},
		},
		&wi_core.CommandImpl{
			"help",
			-1,
			cmdHelp,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Prints help",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Prints general help or help for a particular command.",
			},
		},
		&wi_core.CommandImpl{
			"shell",
			-1,
			cmdShell,
			wi_core.WindowCategory,
			wi_core.LangMap{
				wi_core.LangEn: "Opens a shell process",
			},
			wi_core.LangMap{
				wi_core.LangEn: "Opens a shell process in a new buffer.",
			},
		},
	}
	for _, cmd := range cmds {
		dispatcher.Register(cmd)
	}
}
