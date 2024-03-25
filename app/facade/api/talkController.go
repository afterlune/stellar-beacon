package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListTalks
// @Summary		 说说模块
// @Description  查看说说列表
// @Success		 200	{object}	model.ResultVO
// @Router       /talks [GET]
func ListTalks(c *gin.Context) {
	res := service.ListTalks(c)
	c.JSON(http.StatusOK, res)
}

// GetTalkById
// @Summary		 说说模块
// @Description  根据id查看说说
// @Success		 200	{object}	model.ResultVO
// @Router       /talks/:talkId [GET]
func GetTalkById(c *gin.Context) {
	res := service.GetTalkById(c)
	c.JSON(http.StatusOK, res)
}

// SaveTalkImages
// @Summary		 说说模块
// @Description  上传说说图片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks/images [POST]
func SaveTalkImages(c *gin.Context) {
	res := service.SaveTalkImages(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateTalk
// @Summary		 说说模块
// @Description  保存或修改说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [POST]
func SaveOrUpdateTalk(c *gin.Context) {
	res := service.SaveOrUpdateTalk(c)
	c.JSON(http.StatusOK, res)
}

// DeleteTalks
// @Summary		 说说模块
// @Description  删除说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [DELETE]
func DeleteTalks(c *gin.Context) {
	res := service.DeleteTalks(c)
	c.JSON(http.StatusOK, res)
}

// ListBackTalks
// @Summary		 说说模块
// @Description  查看后台说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [GET]
func ListBackTalks(c *gin.Context) {
	res := service.ListBackTalks(c)
	c.JSON(http.StatusOK, res)
}

// GetBackTalkById
// @Summary		 说说模块
// @Description  根据id查看后台说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks/:talkId [GET]
func GetBackTalkById(c *gin.Context) {
	res := service.GetBackTalkById(c)
	c.JSON(http.StatusOK, res)
}
