package boo

import (
	"context"
	"net/http"
	"strings"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/services/authn"
	jwt "github.com/golang-jwt/jwt/v4"
)

type TokenFindFunc func(r *http.Request) string
type TokenCheckFunc func(ctx context.Context, req *http.Request, tokenStr string) (context.Context, error)

// TokenVerify http middleware handler will verify a Token string from a http request.
//
// TokenVerify will search for a token in a http request, in the order:
//  1. 'token' URI query parameter
//  2. 'Authorization: BEARER T' request header
//  3. Cookie 'token' value
//
// example:
//   TokenVerify([]TokenFindFunc{
//       TokenFromQuery,
//   }, []TokenCheckFunc{
//       JWTCheck(NewJWTAuth(...), TokenToUser),
//   })
//   ......
//
//   func TokenToUser(ctx context.Context, token *jwt.Token) (authn.User, error) {
// 	    claims, ok := token.Claims.(*jwt.StandardClaims)
// 	    if !ok {
// 	    	return nil, errors.New("claims isnot jwt.StandardClaims")
// 	    }
//
// 	    ss := strings.SplitN(claims.Audience, " ", 2)
// 	    if len(ss) < 2 {
// 	    	return nil, errors.New("Audience '" + claims.Audience + "' is invalid")
// 	    }
//
// 	    userid, cerr := strconv.ParseInt(ss[0], 10, 64)
// 	    if cerr != nil {
// 	    	return nil, errors.New("Audience '" + claims.Audience + "' is invalid")
// 	    }
// 	    if userid == 0 {
// 	    	if defaultUser == nil {
// 			    return nil, errors.New("Audience '" + claims.Audience + "' is invalid, userid is missing")
// 		    }
// 		    username := ss[1]
// 		    return warpTokenUser(defaultUser, username, token), nil
// 	    }
// 	    return usermanager.UserByID(ctx, userid)
//    }
func TokenVerify(findTokenFns []TokenFindFunc, checkTokenFns []TokenCheckFunc) authn.AuthValidateFunc {
	return func(ctx context.Context, req *http.Request) (context.Context, error) {
		var tokenStr string

		// Extract token string from the request by calling token find functions in
		// the order they where provided. Further extraction stops if a function
		// returns a non-empty string.
		for _, fn := range findTokenFns {
			tokenStr = fn(req)
			if tokenStr != "" {
				break
			}
		}
		if tokenStr == "" {
			return nil, authn.ErrTokenNotFound
		}

		for _, fn := range checkTokenFns {
			c, err := fn(ctx, req, tokenStr)
			if err == nil || err != authn.ErrSkipped {
				return c, err
			}
		}

		return nil, authn.ErrUnauthorized
	}
}

func JWTCheck(ja *JWTAuth, readUser func(ctx context.Context, token *jwt.Token) (authn.AuthUser, error)) TokenCheckFunc {
	return func(ctx context.Context, req *http.Request, tokenStr string) (context.Context, error) {
		// Verify the token
		token, err := ja.Decode(tokenStr)
		if err != nil {
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					// Token is either expired or not active yet
					err = authn.ErrTokenExpired
				} else {
					err = errors.WithHTTPCode(err, http.StatusUnauthorized)
				}
			} else if strings.HasPrefix(err.Error(), "token is expired") {
				err = authn.ErrTokenExpired
			} else {
				err = errors.WithHTTPCode(err, http.StatusUnauthorized)
			}
			return nil, err
		}

		if token == nil || !token.Valid || token.Method != ja.signer {
			return nil, authn.ErrUnauthorized
		}

		if token.Claims == nil {
			return nil, authn.ErrUnauthorized
		}

		if err = token.Claims.Valid(); err != nil {
			return nil, err
		}

		return authn.ContextWithReadCurrentUser(ctx, authn.ReadCurrentUserFunc(func(ctx context.Context) (authn.AuthUser, error) {
			return readUser(ctx, token)
		})), nil
	}
}

type JWTAuth struct {
	signKey   interface{}
	verifyKey interface{}
	signer    jwt.SigningMethod
	parser    *jwt.Parser
}

// NewJWTAuth creates a JWTAuth authenticator instance that provides middleware handlers
// and encoding/decoding functions for JWT signing.
func NewJWTAuth(alg string, signKey interface{}, verifyKey interface{}) *JWTAuth {
	return NewJWTAuthWithParser(alg, &jwt.Parser{}, signKey, verifyKey)
}

// NewJWTAuthWithParser is the same as New, except it supports custom parser settings
// introduced in jwt-go/v2.4.0.
//
// We explicitly toggle `SkipClaimsValidation` in the `jwt-go` parser so that
// we can control when the claims are validated - in our case, by the Verifier
// http middleware handler.
func NewJWTAuthWithParser(alg string, parser *jwt.Parser, signKey interface{}, verifyKey interface{}) *JWTAuth {
	parser.SkipClaimsValidation = true
	return &JWTAuth{
		signKey:   signKey,
		verifyKey: verifyKey,
		signer:    jwt.GetSigningMethod(alg),
		parser:    parser,
	}
}

func (ja *JWTAuth) Encode(claims *jwt.StandardClaims) (t *jwt.Token, tokenString string, err error) {
	t = jwt.New(ja.signer)
	t.Claims = claims
	tokenString, err = t.SignedString(ja.signKey)
	t.Raw = tokenString
	return
}

func (ja *JWTAuth) Decode(tokenString string) (*jwt.Token, error) {
	return ja.parser.ParseWithClaims(tokenString, &jwt.StandardClaims{}, ja.keyFunc)
}

func (ja *JWTAuth) Signer() jwt.SigningMethod {
	return ja.signer
}

func (ja *JWTAuth) keyFunc(t *jwt.Token) (interface{}, error) {
	if ja.verifyKey != nil {
		return ja.verifyKey, nil
	}
	return ja.signKey, nil
}

// TokenFromCookie tries to retreive the token string from a cookie named
// "token".
func TokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// TokenFromHeader tries to retreive the token string from the
// "Authorization" reqeust header: "Authorization: BEARER T".
func TokenFromHeader(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}
	return ""
}

// TokenFromQuery tries to retreive the token string from the "token" URI
// query parameter.
func TokenFromQuery(r *http.Request) string {
	// Get token from query param named "token".
	return r.URL.Query().Get("token")
}
