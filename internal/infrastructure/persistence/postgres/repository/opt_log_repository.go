package repository

import (
	"context"
	"fmt"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
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
