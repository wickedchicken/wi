// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// wi-plugin-sample is an example plugin for wi.
//
// This plugin serves two purposes:
//   - Ensure that the plugin system is actually working.
//   - Serve as a copy-pastable skeleton to help people who would like to write
//     a plugin.
//
// To try it out, from the wi/ directory, run `go build` then set the
// environment variable `WIPLUGINSPATH=.`, so that this directory is
// automatically compiled via `go run`. See ../editor/plugins.go for the gory
// details.
package main

import (
	"github.com/maruel/wi/wicore"
	"github.com/maruel/wi/wicore/lang"
	"github.com/maruel/wi/wicore/plugin"
)

type pluginImpl struct {
	plugin.PluginImpl
	e wicore.Editor
}

// Init is the place to do full initialization. It is not required to implement
// this function.
func (p *pluginImpl) Init(e wicore.Editor) {
	p.e = e
}

// Close is the place to do full shut down. It is not required to implement
// this function.
func (p *pluginImpl) Close() error {
	p.e = nil
	return nil
}

func main() {
	// This starts the control loop. See its doc for more up-to-date details.
	p := &pluginImpl{
		plugin.PluginImpl{
			"wi-plugin-sample",
			lang.Map{
				lang.En: "Sample plugin to be used as a template",
				lang.Fr: "Plugin exemple pour être utilisé comme modèle",
			},
		},
		nil,
	}
	plugin.Main(p)
}
