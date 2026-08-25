package bootstrap

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/facade/api"
	"benetnasch/app/infra/cache"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/mailer"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/search"
	"benetnasch/app/infra/visitor"
)

// Initialize is the composition root for application services.  The only
// layer that resolves the xorm engine and concrete repositories is this
// package; controllers and application services receive domain ports.
func Initialize() error {
	engine := ormInit.GetEngine()
	if engine == nil {
		return errors.Unavailable("bootstrap.database", nil)
	}
	redisConfig := new(config.Redis).Redis()
	redisCache := cache.NewRedisCache(redisConfig)
	ossStorage, err := oss.NewObjectStorage(new(config.Oss).Oss())
	if err != nil {
		return errors.Unavailable("bootstrap.storage", err)
	}
	searcher := search.NewMeiliSearcher(new(config.MeiliSearch).MeiliSearch())
	smtpMailer := mailer.NewSMTPMailer(new(config.Email).Email())
	visitorResolver := visitor.NewResolver()

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
	benetnasch, err := service.NewBenetnaschInfoService(service.BenetnaschInfoServiceDeps{
		Site:       site,
		Articles:   article,
		Categories: category,
		Tags:       tag,
		Cache:      redisCache,
		Visitor:    visitorResolver,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.benetnasch_info", err)
	}
	service.ConfigureBenetnaschService(benetnasch)

	articleService, err := service.NewArticleService(service.ArticleServiceDeps{
		Repo:    article,
		Cache:   redisCache,
		Storage: ossStorage,
		Search:  searcher,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.article", err)
	}
	commentService, err := service.NewCommentService(service.CommentServiceDeps{
		Repo:    comment,
		Website: benetnasch,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.comment", err)
	}
	photoAlbumService, err := service.NewPhotoAlbumService(service.PhotoAlbumServiceDeps{
		Repo:    photoAlbum,
		Photos:  photo,
		Storage: ossStorage,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.photo_album", err)
	}
	photoService, err := service.NewPhotoService(service.PhotoServiceDeps{
		Repo:    photo,
		Albums:  photoAlbum,
		Storage: ossStorage,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.photo", err)
	}
	talkService, err := service.NewTalkService(service.TalkServiceDeps{
		Repo:     talk,
		Comments: comment,
		Storage:  ossStorage,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.talk", err)
	}
	userAuthService, err := service.NewUserAuthService(service.UserAuthServiceDeps{
		Repo:    auth,
		Website: benetnasch,
		Cache:   redisCache,
		Mailer:  smtpMailer,
		Visitor: visitorResolver,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.user_auth", err)
	}
	userInfoService, err := service.NewUserInfoService(service.UserInfoServiceDeps{
		Repo:    userInfo,
		Cache:   redisCache,
		Storage: ossStorage,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.user_info", err)
	}

	api.ConfigureServices(api.Services{
		Article:      articleService,
		Benetnasch:   benetnasch,
		Category:     service.NewCategoryService(category),
		Comment:      commentService,
		ErrorLog:     service.NewErrorLogService(errorLog),
		FriendLink:   service.NewFriendLinkService(friendLink),
		JobLog:       service.NewJobLogService(jobLog),
		Job:          service.NewJobService(job),
		Menu:         service.NewMenuService(menu),
		OperationLog: service.NewOperationLogService(operationLog),
		PhotoAlbum:   photoAlbumService,
		Photo:        photoService,
		Resource:     service.NewResourceService(resource),
		Role:         service.NewRoleService(role),
		Tag:          service.NewTagService(tag),
		Talk:         talkService,
		UserAuth:     userAuthService,
		UserInfo:     userInfoService,
	})
	middlewares.ConfigureRoleRepository(role)
	middlewares.ConfigureUserAuthService(userAuthService)
	return nil
}
