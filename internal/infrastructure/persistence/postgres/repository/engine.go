package repository

import (
	apperrors "benetnasch/internal/domain/errors"
	"context"

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
		return apperrors.Unavailable(operation+".begin", err)
	}
	if err := fn(session); err != nil {
		_ = session.Rollback()
		if apperrors.KindOf(err) != apperrors.KindInternal {
			return err
		}
		return apperrors.Wrap(apperrors.KindUnavailable, operation, err)
	}
	if err := session.Commit(); err != nil {
		return apperrors.Unavailable(operation+".commit", err)
	}
	return nil
}
