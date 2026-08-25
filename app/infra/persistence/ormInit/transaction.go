package ormInit

import (
	"context"
	"errors"
	"fmt"

	"xorm.io/xorm"
)

// WithTx executes fn in a transaction bound to ctx.
//
// The callback must return an error when any operation fails. The helper then
// rolls the transaction back and preserves both the original and rollback
// errors where possible. A panic is rolled back and re-thrown so callers keep
// the usual panic semantics.
func WithTx(ctx context.Context, fn func(*xorm.Session) error) (err error) {
	return WithEngineTx(GetEngine(), ctx, fn)
}

// WithEngineTx executes fn using the supplied engine. Repository adapters must
// receive their engine from the composition root; a nil engine is an explicit
// configuration error rather than an implicit process-wide fallback.
func WithEngineTx(engine *xorm.Engine, ctx context.Context, fn func(*xorm.Session) error) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if fn == nil {
		return errors.New("transaction callback is nil")
	}

	if engine == nil {
		return errors.New("database engine is not initialized")
	}

	session := engine.NewSession().Context(ctx)
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close transaction session: %w", closeErr)
		}
	}()

	if err = session.Begin(); err != nil {
		return fmt.Errorf("begin transaction: %w", err)
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
			if err == nil {
				err = fmt.Errorf("rollback transaction: %w", rollbackErr)
			} else {
				err = fmt.Errorf("%w; rollback transaction: %v", err, rollbackErr)
			}
		}
	}()

	if err = fn(session); err != nil {
		return err
	}
	if err = session.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}
