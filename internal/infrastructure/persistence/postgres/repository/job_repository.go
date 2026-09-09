package repository

import (
	"benetnasch/internal/infrastructure/persistence/postgres/orm"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
	"benetnasch/internal/interfaces/http/model"
	"log/slog"
)

type JobRepo interface {
	CountJobs(vo *model.JobSearchVO) (count int)
	ListJobs(current, size int, vo *model.JobSearchVO) []*model.JobDTO
	ListJobGroups() (s []string)
}

type MyJobRepo struct{}

func jobFilters(vo *model.JobSearchVO) (string, []interface{}) {
	query := ""
	args := make([]interface{}, 0, 3)
	if vo.JobName != "" {
		query += " WHERE j.job_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.JobName))
	}
	if vo.JobGroup != "" {
		if query == "" {
			query = " WHERE "
		} else {
			query += " AND "
		}
		query += "j.job_group = ?"
		args = append(args, vo.JobGroup)
	}
	if vo.Status != 0 {
		if query == "" {
			query = " WHERE "
		} else {
			query += " AND "
		}
		query += "j.status = ?"
		args = append(args, vo.Status)
	}
	return query, args
}

func (j *MyJobRepo) CountJobs(vo *model.JobSearchVO) (count int) {
	filters, args := jobFilters(vo)
	query := "SELECT count(DISTINCT j.id) FROM t_job j" + filters
	if _, err := ormInit.GetEngine().SQL(query, args...).Get(&count); err != nil {
		slog.Error("count jobs failed", "error", err)
	}
	return count
}

func (j *MyJobRepo) ListJobs(current, size int, vo *model.JobSearchVO) []*model.JobDTO {
	limit, offset := pgsql.Page(current, size)
	filters, args := jobFilters(vo)
	query := "SELECT * FROM t_job j" + filters + " ORDER BY j.status DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var jobs []*model.JobDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&jobs); err != nil {
		slog.Error("list jobs failed", "error", err)
	}
	return jobs
}

func (j *MyJobRepo) ListJobGroups() []string {
	var groups []string
	if err := ormInit.GetEngine().SQL(pgsql.ListJobGroups).Find(&groups); err != nil {
		slog.Error("list job groups failed", "error", err)
	}
	return groups
}
