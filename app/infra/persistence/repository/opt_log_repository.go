package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/row"
	"context"

	"xorm.io/xorm"
)

func saveOptLog(engine *xorm.Engine, ctx context.Context, optLog port.TOperationLog) (err error) {
	session, err := repoSession(engine, ctx, "operation_log.save")
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = apperrors.Unavailable("operation_log.close", closeErr)
		}
	}()
	optLogRow := row.ToOperationLog(optLog)
	if _, err = session.Insert(&optLogRow); err != nil {
		return apperrors.Unavailable("operation_log.save", err)
	}
	return nil
}

func saveExLog(engine *xorm.Engine, ctx context.Context, exLog port.TExceptionLog) (err error) {
	session, err := repoSession(engine, ctx, "exception_log.save")
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = apperrors.Unavailable("exception_log.close", closeErr)
		}
	}()
	exLogRow := row.ToExceptionLog(exLog)
	if _, err = session.Insert(&exLogRow); err != nil {
		return apperrors.Unavailable("exception_log.save", err)
	}
	return nil
}
