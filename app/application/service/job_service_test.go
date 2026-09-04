package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

type jobRepositoryFake struct {
	job         entity.TJob
	saved       entity.TJob
	updated     entity.TJob
	deleted     []int
	listCalls   int
	statusID    int
	status      int
	saveErr     error
	updateErr   error
	deleteErr   error
	statusErr   error
	statusCalls int
}

func (f *jobRepositoryFake) Get(context.Context, int) (entity.TJob, error) { return f.job, nil }

func (f *jobRepositoryFake) List(context.Context, int, int, port.JobFilter) ([]entity.TJob, int, error) {
	f.listCalls++
	return nil, 0, nil
}

func (f *jobRepositoryFake) ListGroups(context.Context) ([]string, error) { return nil, nil }

func (f *jobRepositoryFake) Save(_ context.Context, job entity.TJob) error {
	f.saved = job
	return f.saveErr
}

func (f *jobRepositoryFake) Update(_ context.Context, job entity.TJob) error {
	f.updated = job
	return f.updateErr
}

func (f *jobRepositoryFake) Delete(_ context.Context, ids []int) error {
	f.deleted = append([]int(nil), ids...)
	return f.deleteErr
}

func (f *jobRepositoryFake) UpdateStatus(_ context.Context, id, status int) error {
	f.statusID = id
	f.status = status
	f.statusCalls++
	return f.statusErr
}

func jobTestContext(method, path string, payload any) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return serviceTestRequest{ginContextForServiceTest: context}
}

func TestJobServicePersistsValidatedCommands(t *testing.T) {
	repo := &jobRepositoryFake{}
	service := NewJobService(repo, nil)
	input := model.JobVO{
		Id:             7,
		JobName:        " 内容理解 ",
		JobGroup:       " 默认 ",
		InvokeTarget:   "agent.content_understanding",
		CronExpression: "0 0 * * * ?",
		MisfirePolicy:  2,
		Concurrent:     1,
		Status:         1,
		Remark:         "  nightly ",
	}

	if result := service.SaveJob(jobTestContext(http.MethodPost, "/admin/jobs", input)); !result.Flag {
		t.Fatalf("SaveJob() failed: %+v", result)
	}
	want := entity.TJob{JobName: "内容理解", JobGroup: "默认", InvokeTarget: input.InvokeTarget, CronExpression: input.CronExpression, MisfirePolicy: 2, Concurrent: 1, Status: 1, Remark: "nightly"}
	if !reflect.DeepEqual(repo.saved, want) {
		t.Fatalf("saved job = %+v, want %+v", repo.saved, want)
	}

	if result := service.UpdateJob(jobTestContext(http.MethodPut, "/admin/jobs", input)); !result.Flag {
		t.Fatalf("UpdateJob() failed: %+v", result)
	}
	if repo.updated.Id != input.Id || repo.updated.JobName != want.JobName || repo.updated.Remark != want.Remark {
		t.Fatalf("updated job = %+v", repo.updated)
	}

	if result := service.DeleteJobById(jobTestContext(http.MethodDelete, "/admin/jobs", []int{7, 8})); !result.Flag {
		t.Fatalf("DeleteJobById() failed: %+v", result)
	}
	if !reflect.DeepEqual(repo.deleted, []int{7, 8}) {
		t.Fatalf("deleted IDs = %v", repo.deleted)
	}

	if result := service.UpdateJobStatus(jobTestContext(http.MethodPut, "/admin/jobs/status", model.JobStatusVO{Id: 7, Status: 0})); !result.Flag {
		t.Fatalf("UpdateJobStatus() failed: %+v", result)
	}
	if repo.statusID != 7 || repo.status != 0 || repo.statusCalls != 1 {
		t.Fatalf("status update = id:%d status:%d calls:%d", repo.statusID, repo.status, repo.statusCalls)
	}
}

func TestJobServiceRejectsInvalidCommandsAndDoesNotRunTargets(t *testing.T) {
	repo := &jobRepositoryFake{}
	service := NewJobService(repo, nil)

	invalid := model.JobVO{JobName: "job", JobGroup: "default", InvokeTarget: "target", CronExpression: "cron", Status: 3}
	result := service.SaveJob(jobTestContext(http.MethodPost, "/admin/jobs", invalid))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("invalid SaveJob() result = %+v", result)
	}
	if repo.saved.JobName != "" {
		t.Fatal("invalid job must not be persisted")
	}

	result = service.DeleteJobById(jobTestContext(http.MethodDelete, "/admin/jobs", []int{0}))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("invalid DeleteJobById() result = %+v", result)
	}
	if len(repo.deleted) != 0 {
		t.Fatal("invalid IDs must not be deleted")
	}

	result = service.UpdateJobStatus(jobTestContext(http.MethodPut, "/admin/jobs/status", model.JobStatusVO{Id: 7, Status: 2}))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("invalid UpdateJobStatus() result = %+v", result)
	}
	if repo.statusCalls != 0 {
		t.Fatal("invalid status must not reach repository")
	}

	result = service.RunJob(jobTestContext(http.MethodPut, "/admin/jobs/run", map[string]any{"id": 7}))
	if result.Flag || result.Message != "手动执行任务暂未启用" {
		t.Fatalf("RunJob() result = %+v", result)
	}
}

func TestJobServiceRejectsMalformedListPagination(t *testing.T) {
	repo := &jobRepositoryFake{}
	service := NewJobService(repo, nil)

	result := service.ListJobs(jobTestContext(http.MethodGet, "/admin/jobs?current=bad&size=10", nil))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("malformed list pagination result = %+v", result)
	}
	if repo.listCalls != 0 {
		t.Fatal("malformed pagination must not reach repository")
	}
}

func TestJobServiceSurfacesRepositoryErrors(t *testing.T) {
	repo := &jobRepositoryFake{saveErr: apperrors.Unavailable("job.save", errors.New("database unavailable"))}
	service := NewJobService(repo, nil)
	input := model.JobVO{JobName: "job", JobGroup: "default", InvokeTarget: "target", CronExpression: "cron", Status: 1}
	result := service.SaveJob(jobTestContext(http.MethodPost, "/admin/jobs", input))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("repository error result = %+v", result)
	}
}

type jobRunnerFake struct {
	canRun  bool
	called  int
	request port.JobRunRequest
	outcome port.JobRunOutcome
	err     error
}

func (f *jobRunnerFake) CanRun(string) bool { return f.canRun }

func (f *jobRunnerFake) Run(_ context.Context, request port.JobRunRequest) (port.JobRunOutcome, error) {
	f.called++
	f.request = request
	return f.outcome, f.err
}

func TestJobServiceRunsARegisteredTargetAndDoesNotAcceptGroupSpoofing(t *testing.T) {
	repo := &jobRepositoryFake{job: entity.TJob{
		Id:           7,
		JobName:      "内容理解",
		JobGroup:     "默认",
		InvokeTarget: port.ManualJobTargetContentUnderstanding,
		Status:       1,
	}}
	runner := &jobRunnerFake{canRun: true, outcome: port.JobRunOutcome{
		JobID:     7,
		Target:    port.ManualJobTargetContentUnderstanding,
		Processed: true,
	}}
	service := NewJobService(repo, runner)
	result := service.RunJob(jobTestContext(http.MethodPut, "/admin/jobs/run", model.JobRunVO{Id: 7, JobGroup: "默认"}))
	if !result.Flag || result.Message != "任务已执行一次" || runner.called != 1 {
		t.Fatalf("RunJob() result=%+v called=%d", result, runner.called)
	}
	if runner.request.ID != 7 || runner.request.InvokeTarget != port.ManualJobTargetContentUnderstanding {
		t.Fatalf("runner request=%+v", runner.request)
	}

	result = service.RunJob(jobTestContext(http.MethodPut, "/admin/jobs/run", model.JobRunVO{Id: 7, JobGroup: "伪造分组"}))
	if result.Flag || result.Message != "参数格式不正确" || runner.called != 1 {
		t.Fatalf("group spoof result=%+v called=%d", result, runner.called)
	}
}

func TestJobServiceReportsUnsupportedManualTarget(t *testing.T) {
	repo := &jobRepositoryFake{job: entity.TJob{Id: 8, JobGroup: "默认", InvokeTarget: "legacy.reflection.target"}}
	runner := &jobRunnerFake{}
	service := NewJobService(repo, runner)
	result := service.RunJob(jobTestContext(http.MethodPut, "/admin/jobs/run", model.JobRunVO{Id: 8, JobGroup: "默认"}))
	if result.Flag || result.Message != "该任务目标暂不支持手动执行" || runner.called != 0 {
		t.Fatalf("unsupported target result=%+v called=%d", result, runner.called)
	}
}
