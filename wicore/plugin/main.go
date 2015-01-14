// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package plugin implements the common code to implement a wi plugin.
package plugin

import (
	"fmt"
	"io"
	"net/rpc"
	"os"

	"github.com/maruel/wi/wicore"
	"github.com/maruel/wi/wicore/lang"
)

// PluginImpl is the base implementation of interface wicore.Plugin. Embed this
// structure and override the functions desired.
type PluginImpl struct {
	Name        string
	Description lang.Map
}

func (p *PluginImpl) String() string {
	return fmt.Sprintf("Plugin(%s, %d)", p.Name, os.Getpid())
}

func (p *PluginImpl) Details() wicore.PluginDetails {
	return wicore.PluginDetails{
		p.Name,
		p.Description.String(),
	}
}

func (p *PluginImpl) Init(e wicore.Editor) {
}

func (p *PluginImpl) Close() error {
	return nil
}

// pluginRPC implements wicore.PluginRPC and implement common bookeeping.
type pluginRPC struct {
	conn         io.Closer
	langListener wicore.EventListener
	plugin       wicore.Plugin
	e            *editorProxy
}

func (p *pluginRPC) GetInfo(l lang.Language, out *wicore.PluginDetails) error {
	lang.Set(l)
	*out = p.plugin.Details()
	return nil
}

func (p *pluginRPC) OnStart(details wicore.EditorDetails, ignored *int) error {
	p.e.id = details.ID
	p.e.version = details.Version
	p.langListener = p.e.RegisterEditorLanguage(func(l lang.Language) {
		// Propagate the information.
		lang.Set(l)
	})
	p.plugin.Init(p.e)
	return nil
}

func (p *pluginRPC) Quit(int, *int) error {
	// TODO(maruel): Is it really worth cancelling event listeners? It's just
	// unnecessary slow down, we should favor performance in the shutdown code.
	if p.langListener != nil {
		_ = p.langListener.Close()
		p.langListener = nil
	}
	p.e = nil
	err := p.plugin.Close()
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	return err
}

// editorProxy is an experimentation.
type editorProxy struct {
	wicore.EventRegistry
	deferred     chan func()
	id           string
	activeWindow wicore.Window
	factoryNames []string
	keyboardMode wicore.KeyboardMode
	version      string
}

func (e *editorProxy) ID() string {
	return e.id
}

func (e *editorProxy) ActiveWindow() wicore.Window {
	return e.activeWindow
}

func (e *editorProxy) ViewFactoryNames() []string {
	out := make([]string, len(e.factoryNames))
	for i, v := range e.factoryNames {
		out[i] = v
	}
	return out
}

func (e *editorProxy) AllDocuments() []wicore.Document {
	return nil
}

func (e *editorProxy) AllPlugins() []wicore.PluginDetails {
	return nil
}

func (e *editorProxy) KeyboardMode() wicore.KeyboardMode {
	return e.keyboardMode
}

func (e *editorProxy) Version() string {
	return e.version
}

// Main is the function to call from your plugin to initiate the communication
// channel between wi and your plugin.
func Main(plugin wicore.Plugin) {
	if os.ExpandEnv("${WI}") != "plugin" {
		fmt.Fprint(os.Stderr, "This is a wi plugin. This program is only meant to be run through wi itself.\n")
		os.Exit(1)
	}
	// TODO(maruel): Take garbage from os.Stdin, put garbage in os.Stdout.
	fmt.Print(wicore.CalculateVersion())

	conn := wicore.MakeReadWriteCloser(os.Stdin, os.Stdout)
	server := rpc.NewServer()
	reg, deferred := wicore.MakeEventRegistry()
	e := &editorProxy{
		reg,
		deferred,
		"",
		nil,
		[]string{},
		wicore.Normal,
		"",
	}
	p := &pluginRPC{
		e:      e,
		conn:   os.Stdin,
		plugin: plugin,
	}
	// Statically assert the interface is correctly implemented.
	var _ wicore.PluginRPC = p
	var _ wicore.EventRegistryRPC = e
	if err := server.RegisterName("PluginRPC", p); err != nil {
		panic(err)
	}
	if err := server.RegisterName("EventRegistryRPC", p.e); err != nil {
		panic(err)
	}
	server.ServeConn(conn)
	os.Exit(0)
}
