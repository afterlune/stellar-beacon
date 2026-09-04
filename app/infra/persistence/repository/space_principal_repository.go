package repository

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"xorm.io/xorm"
)

var _ port.SpacePrincipalRepository = (*MySpacePrincipalRepository)(nil)

// MySpacePrincipalRepository reads the machine identities created by the
// explicit space migration. It is intentionally separate from human user and
// Casbin role repositories.
type MySpacePrincipalRepository struct {
	engine *xorm.Engine
}

func NewSpacePrincipalRepository(engine *xorm.Engine) *MySpacePrincipalRepository {
	return &MySpacePrincipalRepository{engine: engine}
}

type spacePrincipalRow struct {
	ID            string `xorm:"id"`
	PrincipalType string `xorm:"principal_type"`
	DisplayName   string `xorm:"display_name"`
	Scopes        []byte `xorm:"scopes"`
	Enabled       bool   `xorm:"enabled"`
}

func (r *MySpacePrincipalRepository) Get(ctx context.Context, id string) (port.SpacePrincipalRecord, error) {
	if r == nil {
		return port.SpacePrincipalRecord{}, apperrors.Unavailable("space.principal.get.database", nil)
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n\x00") {
		return port.SpacePrincipalRecord{}, apperrors.Invalid("space.principal.get", "principal id is invalid")
	}
	session, err := repoSession(r.engine, ctx, "space.principal.get")
	if err != nil {
		return port.SpacePrincipalRecord{}, err
	}
	defer session.Close()

	var row spacePrincipalRow
	found, err := session.SQL(`
SELECT id, principal_type, display_name, scopes, enabled
FROM t_agent_principal
WHERE id = ?`, id).Get(&row)
	if err != nil {
		return port.SpacePrincipalRecord{}, apperrors.Unavailable("space.principal.get", err)
	}
	if !found {
		return port.SpacePrincipalRecord{}, apperrors.NotFound("space.principal.get")
	}
	principal := port.SpacePrincipal{ID: row.ID, Type: row.PrincipalType}
	if len(row.Scopes) > 0 {
		if err := json.Unmarshal(row.Scopes, &principal.Scopes); err != nil {
			return port.SpacePrincipalRecord{}, apperrors.Unavailable("space.principal.get.decode", err)
		}
	}
	if err := principal.Validate(); err != nil {
		return port.SpacePrincipalRecord{}, apperrors.Unavailable("space.principal.get.decode", err)
	}
	return port.SpacePrincipalRecord{
		Principal:   principal,
		DisplayName: strings.TrimSpace(row.DisplayName),
		Enabled:     row.Enabled,
	}, nil
}
