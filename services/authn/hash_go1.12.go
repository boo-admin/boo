//go:build !go1.16
// +build !go1.16

package authn

import (
	"hash"

	"github.com/boo-admin/boo/errors"
)

func getOtherHash(alg string) (func() hash.Hash, error) {
	return nil, errors.New("hash '" + alg + "' unsupport")
}
