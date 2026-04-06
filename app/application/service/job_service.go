package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"strconv"
)

type JobService interface {
	SaveJob(c *gin.Context) model.ResultVO
	UpdateJob(c *gin.Context) model.ResultVO
	DeleteJobById(c *gin.Context) model.ResultVO
	GetJobById(c *gin.Context) model.ResultVO
	ListJobs(c *gin.Context) model.ResultVO
	UpdateJobStatus(c *gin.Context) model.ResultVO
	RunJob(c *gin.Context) model.ResultVO
	ListJobGroup() model.ResultVO
	checkCronIsValid(vo model.JobVO)
}

type MyJobService struct{}

func (j *MyJobService) SaveJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO checkJobCron
	return model.ResultOk()
}

func (j *MyJobService) UpdateJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO checkJobCron
	return model.ResultOk()
}

func (j *MyJobService) DeleteJobById(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO
	return model.ResultOk()
}

func (j *MyJobService) GetJobById(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("id"))
	var job entity.TJob
	_, err := ormInit.GetEngine().Prepare().ID(id).Get(&job)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var jobDTO model.JobDTO
	shared.StructCopy(job, &jobDTO)
	// TODO setNextValidTime
	return model.ResultOkWithData(jobDTO)
}

func (j *MyJobService) ListJobs(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		zlog.Error(err.Error())
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		zlog.Error(err.Error())
	}
	var vo model.JobSearchVO
	err = c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count := jobRepo.CountJobs(&vo)
	jobDTOs := jobRepo.ListJobs(current, size, &vo)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: jobDTOs, Count: int(count)})
}

func (j *MyJobService) UpdateJobStatus(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (j *MyJobService) RunJob(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (j *MyJobService) ListJobGroup() model.ResultVO {
	data := jobRepo.ListJobGroups()
	return model.ResultOkWithData(data)
}

func (j *MyJobService) checkCronIsValid(vo model.JobVO) {

}
