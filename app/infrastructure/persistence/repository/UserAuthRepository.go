package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
	"strconv"
)

func ListUsers(current, size int, vo *model.ConditionVO) []*model.UserAdminDTO {
	s := ""
	if vo.LonginType != 0 {
		s += " where id in (SELECT user_info_id FROM t_user_auth WHERE login_type = " + strconv.Itoa(vo.LonginType) + ")"
	}
	if vo.Keywords != "" && s == "" {
		s += " where nickname like '%" + vo.Keywords + "%'"
	} else if vo.Keywords != "" && s != "" {
		s += " and nickname like '%" + vo.Keywords + "%'"
	}
	sql1 := fmt.Sprintf(pgsql.ListUsers, s, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var usersAdmin []*model.UserAdminDTO
	err := engine.SQL(sql1).Find(&usersAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range usersAdmin {
		var roles []model.UserRoleDTO
		sql2 := fmt.Sprintf(pgsql.RolesByUserId, s, size, (current-1)*size, v.Id)
		err := engine.SQL(sql2).Find(&roles)
		if err != nil {
			zlog.Error(err.Error())
		}
		v.Roles = roles
	}
	return usersAdmin
}

func CountUser(vo *model.ConditionVO) (count int64) {
	s := ""
	if vo.Keywords != "" {
		s += " where nickname like '%" + vo.Keywords + "%'"
	}
	if vo.LonginType != 0 && s != "" {
		s += " and login_type = " + strconv.Itoa(vo.LonginType)
	} else if vo.LonginType != 0 && s == "" {
		s += " where login_type = " + strconv.Itoa(vo.LonginType)
	}
	s = fmt.Sprintf(pgsql.CountUser, s)
	engine := ormInit.GetEngine()
	_, err := engine.SQL(s).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}

	return count
}
