package repository

import (
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

var _ port.PlatformRepository = (*MyPlatformRepo)(nil)

type MyPlatformRepo struct{ engine *xorm.Engine }

func NewPlatformRepo(engine *xorm.Engine) *MyPlatformRepo { return &MyPlatformRepo{engine: engine} }
