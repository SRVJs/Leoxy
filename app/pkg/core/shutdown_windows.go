//go:build windows

package core

import "os"

func ShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
