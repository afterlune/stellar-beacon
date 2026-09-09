package service

import (
	"benetnasch/internal/domain/port"
)

var (
	benetnaschService BenetnaschInfoService
	articleRepo       port.ArticleRepository
	categoryRepo      port.CategoryRepository
	commentRepo       port.CommentRepository
	jobRepo           port.JobRepository
	jobLogRepo        port.JobLogRepository
	errorLogRepo      port.ErrorLogRepository
	operationLogRepo  port.OperationLogRepository
	friendLinkRepo    port.FriendLinkRepository
	menuRepo          port.MenuRepository
	resourceRepo      port.ResourceRepository
	photoAlbumRepo    port.PhotoAlbumRepository
	photoRepo         port.PhotoRepository
	roleRepo          port.RoleRepository
	tagRepo           port.TagRepository
	talkRepo          port.TalkRepository
	userAuthRepo      port.AuthRepository
	userInfoRepo      port.UserInfoRepository
	siteInfoRepo      port.SiteInfoRepository
)

// ConfigureRepositories is called by the composition root during startup.
// Keeping the compatibility registry here lets existing facade handler
// functions retain their signatures while every dependency remains a domain
// port rather than a concrete persistence implementation.
func ConfigureRepositories(
	site port.SiteInfoRepository,
	article port.ArticleRepository,
	category port.CategoryRepository,
	comment port.CommentRepository,
	job port.JobRepository,
	jobLog port.JobLogRepository,
	errorLog port.ErrorLogRepository,
	operationLog port.OperationLogRepository,
	friendLink port.FriendLinkRepository,
	menu port.MenuRepository,
	resource port.ResourceRepository,
	photoAlbum port.PhotoAlbumRepository,
	photo port.PhotoRepository,
	role port.RoleRepository,
	tag port.TagRepository,
	talk port.TalkRepository,
	auth port.AuthRepository,
	userInfo port.UserInfoRepository,
) {
	siteInfoRepo = site
	articleRepo = article
	categoryRepo = category
	commentRepo = comment
	jobRepo = job
	jobLogRepo = jobLog
	errorLogRepo = errorLog
	operationLogRepo = operationLog
	friendLinkRepo = friendLink
	menuRepo = menu
	resourceRepo = resource
	photoAlbumRepo = photoAlbum
	photoRepo = photo
	roleRepo = role
	tagRepo = tag
	talkRepo = talk
	userAuthRepo = auth
	userInfoRepo = userInfo
}

func ConfigureBenetnaschService(s BenetnaschInfoService) {
	benetnaschService = s
}
