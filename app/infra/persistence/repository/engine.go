package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"context"
	"errors"

	"xorm.io/xorm"
)

func repoSession(engine *xorm.Engine, ctx context.Context, operation string) (*xorm.Session, error) {
	if engine == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// engine.Context returns an auto-closing session.  That is convenient for
	// one-shot reads, but it is unsafe for callers that begin a transaction:
	// xorm's Get/Find will close that session before the caller can Commit.
	// Repository methods own the lifecycle and close this explicit session
	// themselves, so transactional Claim/Checkpoint paths remain atomic.
	return engine.NewSession().Context(ctx), nil
}

func repoTx(engine *xorm.Engine, ctx context.Context, operation string, fn func(*xorm.Session) error) (err error) {
	if engine == nil {
		return apperrors.Unavailable(operation+".database", nil)
	}
	if fn == nil {
		return apperrors.Invalid(operation+".callback", "transaction callback is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	session := engine.NewSession().Context(ctx)
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			closeFailure := apperrors.Unavailable(operation+".close", closeErr)
			if err == nil {
				err = closeFailure
			} else {
				err = errors.Join(err, closeFailure)
			}
		}
	}()
	if err = session.Begin(); err != nil {
		return apperrors.Unavailable(operation+".begin", err)
	}

	committed := false
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = session.Rollback()
			panic(recovered)
		}
		if committed {
			return
		}
		if rollbackErr := session.Rollback(); rollbackErr != nil {
			rollbackFailure := apperrors.Unavailable(operation+".rollback", rollbackErr)
			if err == nil {
				err = rollbackFailure
			} else {
				err = errors.Join(err, rollbackFailure)
			}
		}
	}()

	if err = fn(session); err != nil {
		if apperrors.KindOf(err) == apperrors.KindInternal {
			err = apperrors.WrapUnavailable(operation, err)
		}
		return err
	}
	if err = session.Commit(); err != nil {
		return apperrors.Unavailable(operation+".commit", err)
	}
	committed = true
	return nil
}
