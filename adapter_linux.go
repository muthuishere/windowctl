//go:build linux

package windowctl

import linuxadapter "github.com/muthuishere/windowctl/internal/adapter/linux"

func newPlatformAdapter() Adapter { return linuxadapter.New() }
