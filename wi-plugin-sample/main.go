// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This plugin serves two purposes:
// - Ensure that the plugin system is actually working.
// - Serve as a copy-pastable skeleton to help people who would like to write a
//   plugin.
package main

import (
	"github.com/maruel/wi/wi-plugin"
)

func main() {
	// This starts the control loop. See its doc for more up-to-date details.
	wi.Main()
}