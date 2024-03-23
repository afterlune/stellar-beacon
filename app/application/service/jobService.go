package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"strconv"
)

func SaveJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO checkJobCron
	return model.ResultOk()
}

func UpdateJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO checkJobCron
	return model.ResultOk()
}

func DeleteJobById(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	// TODO
	return model.ResultOk()
}

func GetJobById(c *gin.Context) model.ResultVO {
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

func ListJobs(c *gin.Context) model.ResultVO {
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
	count := repository.CountJobs(&vo)
	jobDTOs := repository.ListJobs(current, size, &vo)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: jobDTOs, Count: int(count)})
}

func UpdateJobStatus(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func RunJob(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func ListJobGroup() model.ResultVO {
	data := repository.ListJobGroups()
	return model.ResultOkWithData(data)
}

func checkCronIsValid(vo model.JobVO) {

}
