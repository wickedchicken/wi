// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package editor

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/maruel/wi/wicore"
)

// Plugin represents a live plugin process.
type Plugin interface {
	io.Closer

	wicore.PluginRPC
}

// pluginProcess represents an out-of-process plugin.
type pluginProcess struct {
	proc        *os.Process
	client      *rpc.Client
	pid         int    // Also stored here in case proc is nil. It is not reset even when the process is closed.
	name        string // Self-published plugin name.
	initialized bool
}

func (p *pluginProcess) Close() error {
	var err error
	if p.client != nil {
		tmp := 0
		_ = p.Quit(0, &tmp)
		_ = p.client.Close()
		p.client = nil
	}
	if p.proc != nil {
		err = p.proc.Kill()
		p.proc = nil
	}
	log.Printf("Plugin(%s, %d).Close()", p.name, p.pid)
	return err
}

func (p *pluginProcess) GetInfo(in int, out *wicore.PluginDetails) error {
	return p.client.Call("PluginRPC.GetInfo", in, out)
}

func (p *pluginProcess) OnStart(in int, out *int) error {
	return errors.New("unexpected sync call")
}

func (p *pluginProcess) Quit(in int, out *int) error {
	return p.client.Call("PluginRPC.Quit", in, out)
}

// Plugins is the collection of Plugin instances, it represents all the live
// plugin processes.
type Plugins []Plugin

// Close implements io.Closer.
func (p Plugins) Close() error {
	var out error
	for _, instance := range p {
		if err := instance.Close(); err != nil {
			out = err
		}
	}
	return out
}

// loadPlugin starts a plugin and returns the process.
func loadPlugin(cmdLine []string) (Plugin, error) {
	log.Printf("loadPlugin(%v)", cmdLine)
	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	cmd.Env = append(os.Environ(), "WI=plugin")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	first := make(chan error)

	// Fail on any write to Stderr.
	wicore.Go("stderrReader", func() {
		buf := make([]byte, 2048)
		n, _ := stderr.Read(buf)
		if n != 0 {
			first <- fmt.Errorf("plugin %v failed: %s", cmdLine, buf[:n])
		}
	})

	wicore.Go("stdoutReader", func() {
		// Before starting the RPC, ensures the version matches.
		expectedVersion := wicore.CalculateVersion()
		b := make([]byte, len(expectedVersion))
		if _, err := stdout.Read(b); err != nil {
			first <- err
		}
		actualVersion := string(b)
		if expectedVersion != actualVersion {
			first <- fmt.Errorf("unexpected wicore version; expected %s, got %s", expectedVersion, actualVersion)
		}
		first <- nil
	})

	err = <-first
	if err != nil {
		return nil, err
	}

	conn := wicore.MakeReadWriteCloser(stdout, stdin)
	client := rpc.NewClient(conn)
	p := &pluginProcess{
		cmd.Process,
		client,
		cmd.Process.Pid,
		"<unknown>",
		false,
	}
	out := wicore.PluginDetails{}
	if err = p.GetInfo(0, &out); err != nil {
		return nil, err
	}
	p.name = out.Name
	log.Printf("Plugin(%s, %d) is now functional", p.name, p.pid)
	ignored := 0
	call := p.client.Go("PluginRPC.OnStart", 0, &ignored, nil)
	wicore.Go("PluginRPC.OnStart", func() {
		// TODO(maruel): Handle error.
		_ = <-call.Done
		// TODO(maruel): Synchronization via lock.
		p.initialized = true
	})
	return p, nil
}

func parseDir(i string) (string, error) {
	abs, err := filepath.Abs(i)
	if err != nil {
		return "", fmt.Errorf("invalid path %s: %s", i, err)
	}
	f, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("could not stat %s: %s", i, err)
	}
	if !f.IsDir() {
		return "", fmt.Errorf("%s is not a directory", i)
	}
	return abs, nil
}

// getPluginsPaths returns the search paths for plugins.
//
// Currently look at ".", each element of $GOPATH/bin and in $WIPLUGINSPATH.
func getPluginsPaths() []string {
	out := []string{}
	for _, i := range filepath.SplitList(os.Getenv("GOPATH")) {
		abs, err := parseDir(filepath.Join(i, "bin"))
		if err != nil {
			log.Printf("GOPATH contains invalid %s: %s", i, err)
			continue
		}
		out = append(out, abs)
	}
	for _, i := range filepath.SplitList(os.Getenv("WIPLUGINSPATH")) {
		abs, err := parseDir(i)
		if err != nil {
			log.Printf("WIPLUGINSPATH contains invalid %s: %s", i, err)
			continue
		}
		out = append(out, abs)
	}
	log.Printf("getPluginsPaths() = %v", out)
	return out
}

// enumPlugins enumerate the plugins that should be loaded.
//
// It returns the command lines to use to start the processes. It support
// normal executable, standalone source file and directory containing multiple
// source files.
//
// Source files will incur a ~500ms to ~1s compilation overhead, so they should
// eventually be compiled. Still, it's very useful for quick prototyping.
func enumPlugins(searchDirs []string) ([][]string, error) {
	out := [][]string{}
	var err error
	for _, searchDir := range searchDirs {
		files, err2 := ioutil.ReadDir(searchDir)
		if err2 != nil {
			err = err2
		}
		if len(files) == 0 {
			continue
		}

		for _, f := range files {
			name := f.Name()
			if !strings.HasPrefix(name, "wi-plugin-") {
				continue
			}
			filePath := filepath.Join(searchDir, name)

			if f.IsDir() {
				// Compile on-the-fly a directory of source files.
				// TODO(maruel): When built with -tags debug, pass it along.
				files, err2 := filepath.Glob(filepath.Join(filePath, "*.go"))
				if len(files) == 0 || err2 != nil {
					continue
				}
				i := []string{"go", "run"}
				for _, t := range files {
					i = append(i, t)
				}
				out = append(out, i)
				continue
			}

			if strings.HasSuffix(name, ".go") {
				// Compile on-the-fly a source file.
				// TODO(maruel): When built with -tags debug, pass it along.
				out = append(out, []string{"go", "run", filePath})
				continue
			}

			// Crude check for executable test.
			if runtime.GOOS == "windows" {
				if !strings.HasSuffix(name, ".exe") {
					continue
				}
			} else {
				if f.Mode()&0111 == 0 {
					continue
				}
			}
			out = append(out, []string{filePath})
		}
	}
	return out, err
}

func loadPlugins(pluginExecutables [][]string) (Plugins, error) {
	type x struct {
		Plugin
		error
	}
	c := make(chan x)
	wicore.Go("loadPlugins", func() {
		var wg sync.WaitGroup
		for _, cmd := range pluginExecutables {
			wg.Add(1)
			wicore.Go("loadPlugin", func() {
				func(n []string) {
					defer wg.Done()
					if p, err := loadPlugin(n); err != nil {
						c <- x{error: fmt.Errorf("failed to load %v: %s", n, err)}
					} else {
						c <- x{Plugin: p}
					}
				}(cmd)
			})
		}
		// Wait for all the plugins to be loaded.
		wg.Wait()
		close(c)
	})

	// Convert to a slice.
	var wg sync.WaitGroup
	out := make(Plugins, 0, len(pluginExecutables))
	errs := make([]error, 0)
	wg.Add(1)
	wicore.Go("pluginReaper", func() {
		defer wg.Done()
		for i := range c {
			if i.error != nil {
				errs = append(errs, i.error)
			} else {
				out = append(out, i.Plugin)
			}
		}
	})
	wg.Wait()

	var err error
	if len(errs) != 0 {
		tmp := ""
		for _, e := range errs {
			tmp += e.Error() + "\n"
		}
		err = errors.New(tmp[:len(tmp)-1])
	}
	return out, err
}
