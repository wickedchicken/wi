# Copyright 2014 Marc-Antoine Ruel. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

set -e

go build
go test
# go install github.com/kisielk/errcheck
errcheck
