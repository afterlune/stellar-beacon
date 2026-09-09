package repository

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
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

func (r *MyErrorLogRepo) List(ctx context.Context, current, size int, keywords string) ([]entity.TExceptionLog, int64, error) {
	return listExceptionLogs(r.engine, ctx, current, size, keywords)
}

func (r *MyErrorLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "error_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&entity.TExceptionLog{})
		return err
	})
}

type MyOperationLogRepo struct{ engine *xorm.Engine }

func NewOperationLogRepo(engine *xorm.Engine) *MyOperationLogRepo {
	return &MyOperationLogRepo{engine: engine}
}

func (r *MyOperationLogRepo) List(ctx context.Context, current, size int, keywords string) ([]entity.TOperationLog, int64, error) {
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
	var logs []entity.TOperationLog
	args = append(args, limit, offset)
	query := "SELECT * FROM t_operation_log" + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	if err := session.SQL(query, args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("operation_log.list", err)
	}
	return logs, count, nil
}

func (r *MyOperationLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "operation_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&entity.TOperationLog{})
		return err
	})
}

type MyJobLogRepo struct{ engine *xorm.Engine }

func NewJobLogRepo(engine *xorm.Engine) *MyJobLogRepo { return &MyJobLogRepo{engine: engine} }

func (r *MyJobLogRepo) List(ctx context.Context, current, size int, filter port.JobLogFilter) ([]entity.TJobLog, int64, error) {
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
	var logs []entity.TJobLog
	if err := session.SQL("SELECT * FROM t_job_log"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("job_log.list", err)
	}
	return logs, count, nil
}

func (r *MyJobLogRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "job_log.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&entity.TJobLog{})
		return err
	})
}

func (r *MyJobLogRepo) Clean(ctx context.Context) error {
	return repoTx(r.engine, ctx, "job_log.clean", func(session *xorm.Session) error {
		_, err := session.Delete(&entity.TJobLog{})
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

func (r *MyJobRepository) Get(ctx context.Context, id int) (entity.TJob, error) {
	session, err := repoSession(r.engine, ctx, "job.get")
	if err != nil {
		return entity.TJob{}, err
	}
	var job entity.TJob
	found, err := session.ID(id).Get(&job)
	if err != nil {
		return entity.TJob{}, apperrors.Unavailable("job.get", err)
	}
	if !found {
		return entity.TJob{}, apperrors.NotFound("job.get")
	}
	return job, nil
}

func (r *MyJobRepository) List(ctx context.Context, current, size int, filter port.JobFilter) ([]entity.TJob, int, error) {
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
	var jobs []entity.TJob
	if err := session.SQL("SELECT * FROM t_job"+where+" ORDER BY status DESC LIMIT ? OFFSET ?", args...).Find(&jobs); err != nil {
		return nil, 0, apperrors.Unavailable("job.list", err)
	}
	return jobs, count, nil
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

func (r *MyJobRepository) SaveOrUpdate(ctx context.Context, job entity.TJob) error {
	return repoTx(r.engine, ctx, "job.save", func(session *xorm.Session) error {
		var existing entity.TJob
		found, err := session.Select("id").Where("job_name = ? AND job_group = ?", job.JobName, job.JobGroup).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("job.check_name", err)
		}
		if found && existing.Id != job.Id {
			return apperrors.Conflict("job.save", "job name already exists in this group")
		}

		if job.Id == 0 {
			if job.CreateTime.IsZero() {
				job.CreateTime = time.Now()
			}
			if _, err := session.Insert(&job); err != nil {
				return apperrors.Unavailable("job.insert", err)
			}
			return nil
		}

		result, err := session.Exec(
			"UPDATE t_job SET job_name = ?, job_group = ?, invoke_target = ?, cron_expression = ?, misfire_policy = ?, concurrent = ?, status = ?, update_time = CURRENT_TIMESTAMP, remark = ? WHERE id = ?",
			job.JobName,
			job.JobGroup,
			job.InvokeTarget,
			job.CronExpression,
			job.MisfirePolicy,
			job.Concurrent,
			job.Status,
			job.Remark,
			job.Id,
		)
		if err != nil {
			return apperrors.Unavailable("job.update", err)
		}
		updated, err := result.RowsAffected()
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
		if _, err := session.In("id", ids).Delete(&entity.TJob{}); err != nil {
			return apperrors.Unavailable("job.delete", err)
		}
		return nil
	})
}

func (r *MyJobRepository) UpdateStatus(ctx context.Context, id, status int) error {
	return repoTx(r.engine, ctx, "job.status", func(session *xorm.Session) error {
		result, err := session.Exec(
			"UPDATE t_job SET status = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?",
			status,
			id,
		)
		if err != nil {
			return apperrors.Unavailable("job.status", err)
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("job.status", err)
		}
		if updated == 0 {
			return apperrors.NotFound("job.status")
		}
		return nil
	})
}

func listExceptionLogs(engine *xorm.Engine, ctx context.Context, current, size int, keywords string) ([]entity.TExceptionLog, int64, error) {
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
	var logs []entity.TExceptionLog
	if err := session.SQL("SELECT * FROM t_exception_log"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...).Find(&logs); err != nil {
		return nil, 0, apperrors.Unavailable("error_log.list", err)
	}
	return logs, count, nil
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
