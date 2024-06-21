//go:build go1.16
// +build go1.16

package httpext

import (
	"net"

	"gitee.com/Trisia/gotlcp/tlcp"
	"github.com/boo-admin/boo/errors"
)

func (r *Runner) enableTlcp(listener net.Listener) (net.Listener, error) {
	sigCertificate, err := tlcp.LoadX509KeyPair(r.TLCP.SigCertFile, r.TLCP.SigKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "加载 sig 证书失败")
	}

	encCertificate, err := tlcp.LoadX509KeyPair(r.TLCP.EncCertFile, r.TLCP.EncKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "加载 enc 证书失败")
	}

	tlcpconfig := &tlcp.Config{Certificates: []tlcp.Certificate{
		sigCertificate,
		encCertificate,
	}}

	return tlcp.NewListener(listener, tlcpconfig), nil
}
