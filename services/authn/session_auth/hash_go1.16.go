//go:build go1.16
// +build go1.16

package session_auth

import (
	"hash"

	"github.com/boo-admin/boo/errors"
	"github.com/emmansun/gmsm/sm3"
)

func getOtherHash(alg string) (func() hash.Hash, error) {
	switch alg {
	case "sm3":
		return sm3.New, nil
	}
	return nil, errors.New("hash '" + alg + "' unsupport")
}
