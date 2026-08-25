package bootstrap

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/facade/api"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
)

// Initialize is the composition root for application services.  The only
// layer that resolves the xorm engine and concrete repositories is this
// package; controllers and application services receive domain ports.
func Initialize() error {
	engine := ormInit.GetEngine()
	if engine == nil {
		return errors.Unavailable("bootstrap.database", nil)
	}

	site := repository.NewSiteInfoRepo(engine)
	article := repository.NewArticleRepo(engine)
	category := repository.NewCategoryRepo(engine)
	comment := repository.NewCommentRepo(engine)
	job := repository.NewJobRepository(engine)
	jobLog := repository.NewJobLogRepo(engine)
	errorLog := repository.NewErrorLogRepo(engine)
	operationLog := repository.NewOperationLogRepo(engine)
	friendLink := repository.NewFriendLinkRepo(engine)
	menu := repository.NewMenuRepo(engine)
	resource := repository.NewResourceRepo(engine)
	photoAlbum := repository.NewPhotoAlbumRepository(engine)
	photo := repository.NewPhotoRepository(engine)
	role := repository.NewRoleRepository(engine)
	tag := repository.NewTagRepo(engine)
	talk := repository.NewTalkRepo(engine)
	auth := repository.NewUserAuthRepo(engine)
	userInfo := repository.NewUserInfoRepo(engine)

	service.ConfigureRepositories(site, article, category, comment, job, jobLog, errorLog, operationLog, friendLink, menu, resource, photoAlbum, photo, role, tag, talk, auth, userInfo)
	benetnasch := service.NewBenetnaschInfoService(site, article, category, tag)
	service.ConfigureBenetnaschService(benetnasch)

	api.ConfigureServices(api.Services{
		Article:      service.NewArticleService(article),
		Benetnasch:   benetnasch,
		Category:     service.NewCategoryService(category),
		Comment:      service.NewCommentService(comment, benetnasch),
		ErrorLog:     service.NewErrorLogService(errorLog),
		FriendLink:   service.NewFriendLinkService(friendLink),
		JobLog:       service.NewJobLogService(jobLog),
		Job:          service.NewJobService(job),
		Menu:         service.NewMenuService(menu),
		OperationLog: service.NewOperationLogService(operationLog),
		PhotoAlbum:   service.NewPhotoAlbumService(photoAlbum, photo),
		Photo:        service.NewPhotoService(photo, photoAlbum),
		Resource:     service.NewResourceService(resource),
		Role:         service.NewRoleService(role),
		Tag:          service.NewTagService(tag),
		Talk:         service.NewTalkService(talk, comment),
		UserAuth:     service.NewUserAuthService(auth, benetnasch),
		UserInfo:     service.NewUserInfoService(userInfo),
	})
	middlewares.ConfigureRoleRepository(role)
	return nil
}
