package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
	"strconv"
)

func CountJobs(vo *model.JobSearchVO) (count int) {
	s := ""
	if vo.JobName != "" {
		s += " where j.job_name like '%" + vo.JobName + "%'"
	}
	if vo.JobGroup != "" && s != "" {
		s += " and j.job_group = " + vo.JobGroup
	} else if vo.JobGroup != "" && s == "" {
		s += " where j.job_group = " + vo.JobGroup
	}
	if vo.Status != 0 && s != "" {
		s += " and j.status = " + strconv.Itoa(vo.Status)
	} else if vo.Status != 0 && s == "" {
		s += " where j.status = " + strconv.Itoa(vo.Status)
	}
	s = fmt.Sprintf(pgsql.CountJobs, s)
	engine := ormInit.GetEngine()
	_, err := engine.SQL(s).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}

	return count
}

func ListJobs(current, size int, vo *model.JobSearchVO) []*model.JobDTO {
	s := ""
	if vo.JobName != "" {
		s += " where j.job_name like '%" + vo.JobName + "%'"
	}
	if vo.JobGroup != "" && s != "" {
		s += " and j.job_group = " + vo.JobGroup
	} else if vo.JobGroup != "" && s == "" {
		s += " where j.job_group = " + vo.JobGroup
	}
	if vo.Status != 0 && s != "" {
		s += " and j.status = " + strconv.Itoa(vo.Status)
	} else if vo.Status != 0 && s == "" {
		s += " where j.status = " + strconv.Itoa(vo.Status)
	}
	s = fmt.Sprintf(pgsql.ListJobs, s, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var jobs []*model.JobDTO
	err := engine.SQL(s).Find(&jobs)
	if err != nil {
		zlog.Error(err.Error())
	}

	return jobs
}

func ListJobGroups() (s []string) {
	engine := ormInit.GetEngine()
	err := engine.SQL(pgsql.ListJobGroups).Find(&s)
	if err != nil {
		zlog.Error(err.Error())
	}

	return s
}
