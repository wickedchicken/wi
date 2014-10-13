// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Use "go build -tags debug" to have access to the code in this file.

// +build debug

package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/nsf/termbox-go"
)

var (
	crash = flag.Duration("crash", 0, "Crash after specified duration")
	prof  = flag.String("prof", "", "Start a profiling web server; access via /debug/pprof")
)

func DebugHook() {
	if *crash > 0 {
		// Crashes but ensure that the terminal is closed first. It's useful to
		// figure out what's happening with an infinite loop for example.
		time.AfterFunc(*crash, func() {
			termbox.Close()
			panic("Timeout")
		})
	}

	if len(*prof) == 0 {
		go func() {
			log.Println(http.ListenAndServe(*prof, nil))
		}()
	}
}
