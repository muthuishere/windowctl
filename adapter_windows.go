//go:build windows

package windowctl

import winadapter "github.com/muthuishere/windowctl/internal/adapter/windows"

func newPlatformAdapter() Adapter { return winadapter.New() }
