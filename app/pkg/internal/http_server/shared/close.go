package shared

import (
	"errors"
	"net"
)

func IsClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed)
}
