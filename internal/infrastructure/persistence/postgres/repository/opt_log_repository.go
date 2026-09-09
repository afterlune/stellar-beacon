package repository

import (
	"benetnasch/internal/domain/entity"
	"benetnasch/internal/infrastructure/persistence/postgres/orm"
	"context"
	"fmt"
)

func SaveOptLog(ctx context.Context, optLog entity.TOperationLog) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	session := ormInit.GetEngine().NewSession().Context(ctx)
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close operation log session: %w", closeErr)
		}
	}()
	_, err = session.Insert(&optLog)
	return err
}

func SaveExLog(ctx context.Context, exLog entity.TExceptionLog) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	session := ormInit.GetEngine().NewSession().Context(ctx)
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close exception log session: %w", closeErr)
		}
	}()
	_, err = session.Insert(&exLog)
	return err
}
