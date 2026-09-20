package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func PlatformFeed(c *gin.Context)        { c.JSON(http.StatusOK, platformService.Feed(c)) }
func ListPlatformAuthors(c *gin.Context) { c.JSON(http.StatusOK, platformService.Authors(c)) }
func GetAuthor(c *gin.Context)           { c.JSON(http.StatusOK, platformService.Author(c)) }
func ListAuthorArticles(c *gin.Context)  { c.JSON(http.StatusOK, platformService.AuthorArticles(c)) }
func ListAuthorTalks(c *gin.Context)     { c.JSON(http.StatusOK, platformService.AuthorTalks(c)) }
func ListAuthorSeries(c *gin.Context)    { c.JSON(http.StatusOK, platformService.AuthorSeries(c)) }
func ListTopicArticles(c *gin.Context)   { c.JSON(http.StatusOK, platformService.TopicArticles(c)) }

func StudioDashboard(c *gin.Context)     { c.JSON(http.StatusOK, platformService.Dashboard(c)) }
func GetStudioProfile(c *gin.Context)    { c.JSON(http.StatusOK, platformService.GetProfile(c)) }
func UpdateStudioProfile(c *gin.Context) { c.JSON(http.StatusOK, platformService.UpdateProfile(c)) }
func ListStudioArticles(c *gin.Context)  { c.JSON(http.StatusOK, platformService.ListOwnedArticles(c)) }
func GetStudioArticle(c *gin.Context)    { c.JSON(http.StatusOK, platformService.GetOwnedArticle(c)) }
func SaveStudioArticle(c *gin.Context)   { c.JSON(http.StatusOK, platformService.SaveOwnedArticle(c)) }
func DeleteStudioArticles(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.DeleteOwnedArticles(c))
}

func ListStudioTalks(c *gin.Context)   { c.JSON(http.StatusOK, platformService.ListOwnedTalks(c)) }
func GetStudioTalk(c *gin.Context)     { c.JSON(http.StatusOK, platformService.GetOwnedTalk(c)) }
func SaveStudioTalk(c *gin.Context)    { c.JSON(http.StatusOK, platformService.SaveOwnedTalk(c)) }
func DeleteStudioTalks(c *gin.Context) { c.JSON(http.StatusOK, platformService.DeleteOwnedTalks(c)) }

func ListStudioSeries(c *gin.Context)   { c.JSON(http.StatusOK, platformService.ListOwnedSeries(c)) }
func GetStudioSeries(c *gin.Context)    { c.JSON(http.StatusOK, platformService.GetOwnedSeries(c)) }
func SaveStudioSeries(c *gin.Context)   { c.JSON(http.StatusOK, platformService.SaveOwnedSeries(c)) }
func DeleteStudioSeries(c *gin.Context) { c.JSON(http.StatusOK, platformService.DeleteOwnedSeries(c)) }
func PreviewStudioContent(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.BatchPreviewContent(c))
}
func BatchUpdateStudioContentStatus(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.BatchUpdateContentStatus(c))
}
func BatchDeleteStudioContent(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.BatchDeleteContent(c))
}
func RetryStudioArticlePublication(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.RetryScheduledPublication(c))
}

func ListStudioCategories(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.ListOwnedCategories(c))
}
func SaveStudioCategory(c *gin.Context) { c.JSON(http.StatusOK, platformService.SaveOwnedCategory(c)) }
func DeleteStudioCategory(c *gin.Context) {
	c.JSON(http.StatusOK, platformService.DeleteOwnedCategory(c))
}
func ListStudioTags(c *gin.Context)  { c.JSON(http.StatusOK, platformService.ListOwnedTags(c)) }
func SaveStudioTag(c *gin.Context)   { c.JSON(http.StatusOK, platformService.SaveOwnedTag(c)) }
func DeleteStudioTag(c *gin.Context) { c.JSON(http.StatusOK, platformService.DeleteOwnedTag(c)) }

func ModerateContent(c *gin.Context)   { c.JSON(http.StatusOK, platformService.Moderate(c)) }
func DistributeArticle(c *gin.Context) { c.JSON(http.StatusOK, platformService.DistributeArticle(c)) }
func UploadStudioAsset(c *gin.Context) { c.JSON(http.StatusOK, platformService.Upload(c)) }
