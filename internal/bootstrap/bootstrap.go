package bootstrap

import (
	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/cache"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/mail/smtp"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/repository"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/search/meilisearch"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/storage/object"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/visitor"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/handlers"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/middleware"
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

	service.ConfigureRepositories(category, job, jobLog, errorLog, operationLog, friendLink, menu, resource, role, tag)
	stellarBeacon, err := service.NewStellarBeaconInfoService(service.StellarBeaconInfoServiceDeps{
		Site:       site,
		Articles:   article,
		Categories: category,
		Tags:       tag,
		Cache:      redisCache,
		Visitor:    visitorResolver,
	})
	if err != nil {
		return errors.Unavailable("bootstrap.service.stellar_beacon_info", err)
	}
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
		Website: stellarBeacon,
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
		Website: stellarBeacon,
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
	mediaService := service.NewMediaService(ossStorage)

	api.ConfigureServices(api.Services{
		Article:       articleService,
		StellarBeacon: stellarBeacon,
		Category:      service.NewCategoryService(category),
		Comment:       commentService,
		ErrorLog:      service.NewErrorLogService(errorLog),
		FriendLink:    service.NewFriendLinkService(friendLink),
		JobLog:        service.NewJobLogService(jobLog),
		Job:           service.NewJobService(job),
		Menu:          service.NewMenuService(menu),
		Media:         mediaService,
		OperationLog:  service.NewOperationLogService(operationLog),
		PhotoAlbum:    photoAlbumService,
		Photo:         photoService,
		Resource:      service.NewResourceService(resource),
		Role:          service.NewRoleService(role),
		Tag:           service.NewTagService(tag),
		Talk:          talkService,
		UserAuth:      userAuthService,
		UserInfo:      userInfoService,
	})
	middlewares.ConfigureRoleRepository(role)
	middlewares.ConfigureUserAuthService(userAuthService)
	return nil
}
