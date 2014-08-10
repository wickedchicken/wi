// Copyright 2014 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package wi

import (
	"testing"
)

func TestCalculateVersion(t *testing.T) {
	v := CalculateVersion()
	if v != "6bef60c9b4f7f514b2cc7ba66ecd26610e7b80e1" {
		t.Fatal(v)
	}
}
