//go:build windows

package windowctl

import winadapter "github.com/muthuishere/windowctl/src/internal/adapter/windows"

func newPlatformAdapter() Adapter { return winadapter.New() }
