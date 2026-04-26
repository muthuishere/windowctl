//go:build darwin

package windowctl

import darwinadapter "github.com/muthuishere/windowctl/internal/adapter/darwin"

func newPlatformAdapter() Adapter { return darwinadapter.New() }
