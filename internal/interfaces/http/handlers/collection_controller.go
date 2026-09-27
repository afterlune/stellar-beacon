package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListPublicCollections(c *gin.Context)   { c.JSON(http.StatusOK, collectionService.ListPublic(c)) }
func GetPublicCollection(c *gin.Context)     { c.JSON(http.StatusOK, collectionService.GetPublic(c)) }
func ListAuthorCollections(c *gin.Context)   { c.JSON(http.StatusOK, collectionService.ListAuthor(c)) }
func ListStudioCollections(c *gin.Context)   { c.JSON(http.StatusOK, collectionService.ListOwned(c)) }
func GetStudioCollection(c *gin.Context)     { c.JSON(http.StatusOK, collectionService.GetOwned(c)) }
func CreateStudioCollection(c *gin.Context)  { c.JSON(http.StatusOK, collectionService.Create(c)) }
func UpdateStudioCollection(c *gin.Context)  { c.JSON(http.StatusOK, collectionService.Update(c)) }
func DeleteStudioCollection(c *gin.Context)  { c.JSON(http.StatusOK, collectionService.Delete(c)) }
func AddStudioCollectionItem(c *gin.Context) { c.JSON(http.StatusOK, collectionService.AddItem(c)) }
func RemoveStudioCollectionItem(c *gin.Context) {
	c.JSON(http.StatusOK, collectionService.RemoveItem(c))
}
func ReorderStudioCollection(c *gin.Context) { c.JSON(http.StatusOK, collectionService.Reorder(c)) }
func ListAdminCollections(c *gin.Context)    { c.JSON(http.StatusOK, collectionService.ListAdmin(c)) }
