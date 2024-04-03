package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListTalks
// @Summary		 说说模块
// @Description  查看说说列表
// @Success		 200	{object}	model.ResultVO
// @Router       /talks [GET]
func ListTalks(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.ListTalks(c))
}

// GetTalkById
// @Summary		 说说模块
// @Description  根据id查看说说
// @Success		 200	{object}	model.ResultVO
// @Router       /talks/:talkId [GET]
func GetTalkById(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.GetTalkById(c))
}

// SaveTalkImages
// @Summary		 说说模块
// @Description  上传说说图片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks/images [POST]
func SaveTalkImages(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.SaveTalkImages(c))
}

// SaveOrUpdateTalk
// @Summary		 说说模块
// @Description  保存或修改说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [POST]
func SaveOrUpdateTalk(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.SaveOrUpdateTalk(c))
}

// DeleteTalks
// @Summary		 说说模块
// @Description  删除说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [DELETE]
func DeleteTalks(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.DeleteTalks(c))
}

// ListBackTalks
// @Summary		 说说模块
// @Description  查看后台说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks [GET]
func ListBackTalks(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.ListBackTalks(c))
}

// GetBackTalkById
// @Summary		 说说模块
// @Description  根据id查看后台说说
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/talks/:talkId [GET]
func GetBackTalkById(c *gin.Context) {
	c.JSON(http.StatusOK, talkService.GetBackTalkById(c))
}
