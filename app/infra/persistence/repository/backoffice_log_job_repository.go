package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/row"
	"context"
	"strings"
	"time"

	"xorm.io/xorm"
)

var _ port.ErrorLogRepository = (*MyErrorLogRepo)(nil)
var _ port.OperationLogRepository = (*MyOperationLogRepo)(nil)
var _ port.JobLogRepository = (*MyJobLogRepo)(nil)
var _ port.JobRepository = (*MyJobRepository)(nil)

type MyErrorLogRepo struct{ engine *xorm.Engine }

func NewErrorLogRepo(engine *xorm.Engine) *MyErrorLogRepo { return &MyErrorLogRepo{engine: engine} }

func (r *MyErrorLogRepo) List(ctx context.Context, current, size int, keywords string) ([]port.TExceptionLog, int64, error) {
	return listExceptionLogs(r.engine, ctx, current, size, keywords)
}

func (r *MyErrorLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "error_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&row.TExceptionLog{})
		return err
	})
}

type MyOperationLogRepo struct{ engine *xorm.Engine }

func NewOperationLogRepo(engine *xorm.Engine) *MyOperationLogRepo {
	return &MyOperationLogRepo{engine: engine}
}

func (r *MyOperationLogRepo) List(ctx context.Context, current, size int, keywords string) ([]port.TOperationLog, int64, error) {
	session, err := repoSession(r.engine, ctx, "operation_log.list")
	if err != nil {
		return nil, 0, err
	}
	where, args := operationLogFilter(keywords)
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_operation_log"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("operation_log.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var logs []row.TOperationLog
	args = append(args, limit, offset)
	query := "SELECT * FROM t_operation_log" + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	if err := session.SQL(query, args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("operation_log.list", err)
	}
	return row.FromOperationLogs(logs), count, nil
}

func (r *MyOperationLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "operation_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&row.TOperationLog{})
		return err
	})
}

type MyJobLogRepo struct{ engine *xorm.Engine }

func NewJobLogRepo(engine *xorm.Engine) *MyJobLogRepo { return &MyJobLogRepo{engine: engine} }

func (r *MyJobLogRepo) List(ctx context.Context, current, size int, filter port.JobLogFilter) ([]port.TJobLog, int64, error) {
	session, err := repoSession(r.engine, ctx, "job_log.list")
	if err != nil {
		return nil, 0, err
	}
	where, args := jobLogFilter(filter)
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_job_log"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("job_log.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var logs []row.TJobLog
	if err := session.SQL("SELECT * FROM t_job_log"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("job_log.list", err)
	}
	return row.FromJobLogs(logs), count, nil
}

func (r *MyJobLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "job_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&row.TJobLog{})
		return err
	})
}

func (r *MyJobLogRepo) Clean(ctx context.Context) error {
	return repoTx(r.engine, ctx, "job_log.clean", func(session *xorm.Session) error {
		_, err := session.Delete(&row.TJobLog{})
		return err
	})
}

func (r *MyJobLogRepo) ListGroups(ctx context.Context) (string, error) {
	session, err := repoSession(r.engine, ctx, "job_log.groups")
	if err != nil {
		return "", err
	}
	var group string
	if _, err := session.SQL(pgsql.ListJobLogGroups).Get(&group); err != nil {
		return "", apperrors.Unavailable("job_log.groups", err)
	}
	return group, nil
}

type MyJobRepository struct{ engine *xorm.Engine }

func NewJobRepository(engine *xorm.Engine) *MyJobRepository { return &MyJobRepository{engine: engine} }

func (r *MyJobRepository) Get(ctx context.Context, id int) (port.TJob, error) {
	session, err := repoSession(r.engine, ctx, "job.get")
	if err != nil {
		return port.TJob{}, err
	}
	var job row.TJob
	// The legacy schema declares a composite primary key consisting of id,
	// job_name and job_group.  The service contract addresses jobs by their
	// stable numeric id, so do not let xorm build a partial composite-key
	// predicate from ID(id).
	found, err := session.Where("id = ?", id).Get(&job)
	if err != nil {
		return port.TJob{}, apperrors.Unavailable("job.get", err)
	}
	if !found {
		return port.TJob{}, apperrors.NotFound("job.get")
	}
	return row.FromJob(job), nil
}

func (r *MyJobRepository) List(ctx context.Context, current, size int, filter port.JobFilter) ([]port.TJob, int, error) {
	session, err := repoSession(r.engine, ctx, "job.list")
	if err != nil {
		return nil, 0, err
	}
	where, args := jobFilter(filter)
	var count int
	if _, err := session.SQL("SELECT count(0) FROM t_job"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("job.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var jobs []row.TJob
	if err := session.SQL("SELECT * FROM t_job"+where+" ORDER BY status DESC LIMIT ? OFFSET ?", args...).Find(&jobs); err != nil {
		return nil, 0, apperrors.Unavailable("job.list", err)
	}
	return row.FromJobs(jobs), count, nil
}

func (r *MyJobRepository) ListGroups(ctx context.Context) ([]string, error) {
	session, err := repoSession(r.engine, ctx, "job.groups")
	if err != nil {
		return nil, err
	}
	var groups []string
	if err := session.SQL(pgsql.ListJobGroups).Find(&groups); err != nil {
		return nil, apperrors.Unavailable("job.groups", err)
	}
	return groups, nil
}

func (r *MyJobRepository) Save(ctx context.Context, job port.TJob) error {
	return repoTx(r.engine, ctx, "job.save", func(session *xorm.Session) error {
		jobRow := row.ToJob(job)
		if _, err := session.Insert(&jobRow); err != nil {
			return apperrors.Unavailable("job.save", err)
		}
		return nil
	})
}

func (r *MyJobRepository) Update(ctx context.Context, job port.TJob) error {
	return repoTx(r.engine, ctx, "job.update", func(session *xorm.Session) error {
		jobRow := row.ToJob(job)
		updated, err := session.Where("id = ?", job.Id).Cols(
			"job_name", "job_group", "invoke_target", "cron_expression",
			"misfire_policy", "concurrent", "status", "remark", "update_time",
		).Update(&jobRow)
		if err != nil {
			return apperrors.Unavailable("job.update", err)
		}
		if updated == 0 {
			return apperrors.NotFound("job.update")
		}
		return nil
	})
}

func (r *MyJobRepository) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "job.delete", func(session *xorm.Session) error {
		if _, err := session.In("id", ids).Delete(&row.TJob{}); err != nil {
			return apperrors.Unavailable("job.delete", err)
		}
		return nil
	})
}

func (r *MyJobRepository) UpdateStatus(ctx context.Context, id, status int) error {
	return repoTx(r.engine, ctx, "job.status", func(session *xorm.Session) error {
		updated, err := session.Where("id = ?", id).Cols("status", "update_time").Update(&row.TJob{
			Status:     status,
			UpdateTime: time.Now().UTC(),
		})
		if err != nil {
			return apperrors.Unavailable("job.status", err)
		}
		if updated == 0 {
			return apperrors.NotFound("job.status")
		}
		return nil
	})
}

func listExceptionLogs(engine *xorm.Engine, ctx context.Context, current, size int, keywords string) ([]port.TExceptionLog, int64, error) {
	session, err := repoSession(engine, ctx, "error_log.list")
	if err != nil {
		return nil, 0, err
	}
	where := ""
	args := []interface{}{}
	if strings.TrimSpace(keywords) != "" {
		where = " WHERE opt_desc LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_exception_log"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("error_log.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var logs []row.TExceptionLog
	if err := session.SQL("SELECT * FROM t_exception_log"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("error_log.list", err)
	}
	return row.FromExceptionLogs(logs), count, nil
}

func operationLogFilter(keywords string) (string, []interface{}) {
	if strings.TrimSpace(keywords) == "" {
		return "", nil
	}
	pattern := pgsql.ContainsPattern(keywords)
	return " WHERE (opt_module LIKE ? ESCAPE '\\' OR opt_desc LIKE ? ESCAPE '\\')", []interface{}{pattern, pattern}
}

func jobLogFilter(filter port.JobLogFilter) (string, []interface{}) {
	clauses := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	if filter.JobId != 0 {
		clauses = append(clauses, "job_id = ?")
		args = append(args, filter.JobId)
	}
	if filter.JobGroup != "" {
		clauses = append(clauses, "job_group LIKE ? ESCAPE '\\'")
		args = append(args, pgsql.ContainsPattern(filter.JobGroup))
	}
	if filter.JobName != "" {
		clauses = append(clauses, "job_name LIKE ? ESCAPE '\\'")
		args = append(args, pgsql.ContainsPattern(filter.JobName))
	}
	if filter.Status != nil {
		clauses = append(clauses, "status = ?")
		args = append(args, *filter.Status)
	}
	if filter.StartTime != "" && filter.EndTime != "" {
		clauses = append(clauses, "create_time BETWEEN ? AND ?")
		args = append(args, filter.StartTime, filter.EndTime)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func jobFilter(filter port.JobFilter) (string, []interface{}) {
	clauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)
	if filter.JobName != "" {
		clauses = append(clauses, "job_name LIKE ? ESCAPE '\\'")
		args = append(args, pgsql.ContainsPattern(filter.JobName))
	}
	if filter.JobGroup != "" {
		clauses = append(clauses, "job_group = ?")
		args = append(args, filter.JobGroup)
	}
	if filter.Status != 0 {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
