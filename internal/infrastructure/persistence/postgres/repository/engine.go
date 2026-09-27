package repository

import (
	"context"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"

	"xorm.io/xorm"
)

func repoSession(engine *xorm.Engine, ctx context.Context, operation string) (*xorm.Session, error) {
	if engine == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return engine.Context(ctx), nil
}

// repoTransactionSession returns a caller-owned session for explicit
// transactions. Engine.Context marks its sessions auto-close, which closes a
// transaction after the first Get/Find call.
func repoTransactionSession(engine *xorm.Engine, ctx context.Context, operation string) (*xorm.Session, error) {
	if engine == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return engine.NewSession().Context(ctx), nil
}

// repoCtxErr keeps a disconnected caller from being reported as a service
// failure. A canceled request reaches the driver as an opaque error
// (driver.ErrBadConn, "pq: canceling statement due to user request"), so the
// request context is the reliable signal rather than the error chain.
func repoCtxErr(ctx context.Context, operation string, err error) error {
	if ctx != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return apperrors.Canceled(operation, ctxErr)
		}
	}
	return err
}

func repoTx(engine *xorm.Engine, ctx context.Context, operation string, fn func(*xorm.Session) error) error {
	if engine == nil {
		return apperrors.Unavailable(operation+".database", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return repoCtxErr(ctx, operation+".begin", apperrors.Unavailable(operation+".begin", err))
	}
	if err := fn(session); err != nil {
		_ = session.Rollback()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return apperrors.Canceled(operation, ctxErr)
		}
		if apperrors.KindOf(err) != apperrors.KindInternal {
			return err
		}
		return apperrors.Wrap(apperrors.KindUnavailable, operation, err)
	}
	if err := session.Commit(); err != nil {
		return repoCtxErr(ctx, operation+".commit", apperrors.Unavailable(operation+".commit", err))
	}
	return nil
}
