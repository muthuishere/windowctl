//go:build darwin

package windowctl

import darwinadapter "github.com/muthuishere/windowctl/src/internal/adapter/darwin"

func newPlatformAdapter() Adapter { return darwinadapter.New() }
