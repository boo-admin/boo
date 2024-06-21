//go:build !go1.16
// +build !go1.16

package httpext

import (
	"net"

	"github.com/boo-admin/boo/errors"
)

func (r *Runner) enableTlcp(listener net.Listener) (net.Listener, error) {
	return nil, errors.New("本版本不支持国密 tlcp")
}
