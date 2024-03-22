package route

import (
	"benetnasch/app/facade/api"
	"github.com/gin-gonic/gin"
)

func WebAdapter(router *gin.Engine) {
	article := router.Group("/articles")
	{
		// article
		article.GET("/topAndFeatured", api.ListTopAndFeaturedArticles)
		article.GET("/all", api.ListArticles)
		article.GET("/categoryId", api.GetArticlesByCategoryId)
		article.GET("/:articleId", api.GetArticleById)
		article.POST("/access", api.AccessArticle)
		article.GET("/tagId", api.ListArticlesByTagId)
		article.GET("/search", api.ListArticlesBySearch)
	}
	router.GET("/archives/all", api.ListArchives)

	admin := router.Group("/admin")
	{
		// article
		admin.GET("/articles", api.ListArticlesAdmin)
		admin.POST("/articles", api.SaveOrUpdateArticle)
		admin.PUT("/articles/topAndFeatured", api.UpdateArticleTopAndFeatured)
		admin.PUT("/articles", api.UpdateArticleDelete)
		admin.DELETE("/articles/delete", api.DeleteArticles)
		admin.POST("/articles/images", api.SaveArticleImages)
		admin.GET("/articles/:articleId", api.GetArticleBackById)
		admin.POST("/articles/import", api.ImportArticles)
		admin.POST("/articles/export", api.ExportArticles)
		// BenetnaschInfo
		admin.GET("", api.GetBlogBackInfo)
		admin.PUT("/website/config", api.UpdateWebsiteConfig)
		admin.GET("/website/config", api.GetWebsiteConfig)
		admin.PUT("/about", api.UpdateAbout)
		admin.POST("/config/images", api.SaveBlogPhotoAlbumCover)
		// category
		admin.GET("/categories", api.ListCategoriesAdmin)
		admin.GET("/categories/search", api.ListCategoriesAdminBySearch)
		admin.DELETE("/categories", api.DeleteCategories)
		admin.POST("/categories", api.SaveOrUpdateCategory)
		// comment
		admin.GET("/comments", api.ListCommentBackDTO)
		admin.PUT("/comments/review", api.UpdateCommentsReview)
		admin.DELETE("/comments", api.DeleteComments)
		// exceptionLog
		admin.GET("/exception/logs", api.ListExceptionLogs)
		admin.DELETE("/exception/logs", api.DeleteExceptionLogs)
		// FriendLink
		admin.GET("/links", api.ListFriendLinkDTO)
		admin.POST("/links", api.SaveOrUpdateFriendLink)
		admin.DELETE("/links", api.DeleteFriendLink)
		// job
		admin.POST("/jobs", api.SaveJob)
		admin.PUT("/jobs", api.UpdateJob)
		admin.DELETE("/jobs", api.DeleteJobById)
		admin.GET("/jobs/:id", api.GetJobById)
		admin.GET("/jobs", api.ListJobs)
		admin.PUT("/jobs/status", api.UpdateJobStatus)
		admin.PUT("/jobs/run", api.RunJob)
		admin.GET("/jobs/jobGroups", api.ListJobGroup)
		// jobLog
		admin.GET("/jobLogs", api.ListJobLogs)
		admin.DELETE("/jobLogs", api.DeleteJobLogs)
		admin.DELETE("/jobLogs/clean", api.CleanJobLogs)
		admin.GET("/jobLogs/jobGroups", api.ListJobLogGroups)
		// menu
		admin.GET("/menus", api.ListMenus)
		admin.POST("/menus", api.SaveOrUpdateMenu)
		admin.PUT("menus/isHidden", api.UpdateMenuIsHidden)
		admin.DELETE("/menus/:menuId", api.DeleteMenu)
		admin.GET("/role/menus", api.ListMenuOptions)
		admin.GET("/user/menus", api.ListUserMenus)
		// operationLog
		admin.GET("/operation/logs", api.ListOperationLogs)
		admin.DELETE("/operation/logs", api.DeleteOperationLogs)
		// photoAlbum
		admin.POST("/photos/albums/upload", api.SavePhotoAlbumCover)
		admin.POST("/photos/albums", api.SaveOrUpdatePhotoAlbum)
		admin.GET("/photos/albums", api.ListPhotoAlbumBacks)
		admin.GET("/photos/albums/info", api.ListPhotoAlbumBackInfos)
		admin.GET("/photos/albums/:albumId/info", api.GetPhotoAlbumBackById)
		admin.DELETE("photos/albums/:albumId", api.DeletePhotoAlbumById)
		// photo
		admin.POST("/photos/upload", api.SavePhotosAlbumCover)
		admin.GET("/photos", api.ListPhotos)
		admin.PUT("/photos", api.UpdatePhoto)
		admin.POST("/photos", api.SavePhotos)
		admin.PUT("/photos/album", api.UpdatePhotosAlbum)
		admin.PUT("/photos/delete", api.UpdatePhotoDelete)
		admin.DELETE("/photos", api.DeletePhotos)
		// resource
		admin.GET("/resources", api.ListResources)
		admin.DELETE("/resources/:resourceId", api.DeleteResource)
		admin.POST("/resources", api.SaveOrUpdateResource)
		admin.GET("/role/resources", api.ListResourceOption)
		// role
		admin.GET("/users/role", api.ListUserRoles)
		admin.GET("/roles", api.ListRoles)
		admin.POST("/role", api.SaveOrUpdateRole)
		admin.DELETE("/roles", api.DeleteRoles)
		// tag
		admin.GET("/tags", api.ListTagsAdmin)
		admin.GET("/tags/search", api.ListTagsAdminBySearch)
		admin.POST("/tags", api.SaveOrUpdateTag)
		admin.DELETE("/tags", api.DeleteTag)
		// talk
		admin.POST("/talks/images", api.SaveTalkImages)
		admin.POST("/talks", api.SaveOrUpdateTalk)
		admin.DELETE("/talks", api.DeleteTalks)
		admin.GET("/talks", api.ListBackTalks)
		admin.GET("/talks/:talkId", api.GetBackTalkById)
		// user
		admin.GET("/users/area", api.ListUserAreas)
		admin.GET("/users", api.ListUsers)
		admin.PUT("/users/password", api.UpdateAdminPassword)
		// userinfo
		admin.PUT("/users/role", api.UpdateUserRole)
		admin.PUT("/users/disable", api.UpdateUserDisable)
		admin.GET("/users/online", api.ListOnlineUsers)
		admin.DELETE("/users/:userInfoId/online", api.RemoveOnlineUser)
	}
	// BenetnaschInfo
	router.POST("/report", api.Report)
	router.GET("/", api.GetBlogHomeInfo)
	router.GET("/about", api.GetAbout)
	// bizException
	router.GET("/bizException", api.HandleBizException)
	// category
	router.GET("/categories/all", api.ListCategories)
	// FriendLink
	router.GET("/links", api.ListFriendLinks)
	// photoAlbum
	router.GET("/photos/albums", api.ListPhotoAlbums)
	// photo
	router.GET("/albums/:albumId/photos", api.ListPhotosByAlbumId)
	// tag
	router.GET("/tags/all", api.GetAllTags)
	router.GET("/tags/topTen", api.GetTopTenTags)
	// talk
	router.GET("/talks", api.ListTalks)
	router.GET("/talks/:talkId", api.GetTalkById)

	comments := router.Group("/comments")
	{
		// comment
		comments.POST("/save", api.SaveComment)
		comments.GET("", api.GetComments)
		comments.GET("/:commentId/replies", api.ListRepliesByCommentId)
		comments.GET("/topSix", api.ListTopSixComments)
	}

	users := router.Group("/users")
	{
		// user
		users.GET("/code", api.SendCode)
		users.POST("/register", api.Register)
		users.PUT("/password", api.UpdatePassword)
		users.POST("/logout", api.Logout)
		users.POST("/oauth/qq", api.QQLogin)
		// userinfo
		users.PUT("/info", api.UpdateUserInfo)
		users.POST("/avatar", api.UpdateUserAvatar)
		users.PUT("/email", api.SaveUserEmail)
		users.PUT("/subscribe", api.UpdateUserSubscribe)
		users.GET("/info/:userInfoId", api.GetUserInfoById)
	}
}
