package authncore

import "fmt"

type Locker interface {
	Lock(ctx *AuthContext) error
}

type IsLocker interface {
	IsLocked(ctx *AuthContext) error
}

type HasLock interface {
	IsLocked() bool
}

func LockCheck(checker IsLocker) AuthOption {
	if checker != nil {
		return AuthOptionFunc(func(auth *AuthService) error {
			auth.OnBeforeLoad(AuthFunc(func(ctx *AuthContext) error {
				return checker.IsLocked(ctx)
			}))
			return nil
		})
	}
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeAuth(AuthFunc(func(ctx *AuthContext) error {
			if ctx.Authentication == nil {
				return nil
			}
			u, ok := ctx.Authentication.(HasLock)
			if !ok {
				return fmt.Errorf("user is unsupported for the error lock - %T", ctx.Authentication)
			}
			if u.IsLocked() {
				return ErrUserLocked
			}
			return nil

			// if u.LockedAt.IsZero() || u.Name == "admin" {
			// 	return nil
			// }
			//
			// if u.LockedTimeExpires == 0 {
			// 	return ErrUserLocked
			// }
			// if time.Now().Before(u.LockedAt.Add(u.LockedTimeExpires)) {
			// 	return ErrUserLocked
			// }
			// return nil
		}))
		return nil
	})
}
