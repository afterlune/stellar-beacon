package bootstrap

import (
	"context"
	"sync"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/cache"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/mail/smtp"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/notification"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/repository"
	appruntime "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/runtime"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/search/meilisearch"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/storage/object"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/task"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/visitor"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/handlers"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/middleware"
)

// Initialize is the composition root for application services.  The only
// layer that resolves the xorm engine and concrete repositories is this
// package; controllers and application services receive domain ports.
type Runtime struct {
	cancel         context.CancelFunc
	scheduler      *task.Scheduler
	newsletter     *service.MyNewsletterService
	newsletterDone chan struct{}
	commentQueue   *notification.CommentQueue

	stopOnce sync.Once
	stopErr  error
}

// Initialize is the composition root for application services. The only layer
// that resolves the xorm engine and concrete repositories is this package;
// controllers and application services receive domain ports.
func Initialize(parent context.Context) (*Runtime, error) {
	if parent == nil {
		parent = context.Background()
	}
	runCtx, cancel := context.WithCancel(parent)
	engine := ormInit.GetEngine()
	if engine == nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.database", nil)
	}
	redisConfig := new(config.Redis).Redis()
	redisCache := cache.NewRedisCache(redisConfig)
	ossStorage, err := oss.NewObjectStorage(new(config.Oss).Oss())
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.storage", err)
	}
	searcher := search.NewMeiliSearcher(new(config.MeiliSearch).MeiliSearch())
	smtpMailer := mailer.NewSMTPMailer(new(config.Email).Email())
	visitorResolver := visitor.NewResolver()
	commentQueue := notification.StartCommentQueue(context.Background(), notification.NewMailerSender(smtpMailer))
	appruntime.SetComponent("commentQueue", "ready")

	site := repository.NewSiteInfoRepo(engine)
	article := repository.NewArticleRepo(engine)
	articleReaction := repository.NewArticleReactionRepo(engine)
	series := repository.NewSeriesRepo(engine)
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
	newsletterRepo := repository.NewNewsletterRepo(engine)
	growthRepo := repository.NewGrowthRepo(engine)
	contentAnalyticsRepo := repository.NewContentAnalyticsRepo(engine)

	service.ConfigureRepositories(category, job, jobLog, errorLog, operationLog, friendLink, menu, resource, role, tag)
	service.ConfigureFriendLinkLimiter(redisCache)
	service.ConfigureFriendLinkVisitor(visitorResolver)
	stellarBeacon, err := service.NewStellarBeaconInfoService(service.StellarBeaconInfoServiceDeps{
		Site:       site,
		Articles:   article,
		Categories: category,
		Tags:       tag,
		Cache:      redisCache,
		Visitor:    visitorResolver,
		Newsletter: newsletterRepo,
		Growth:     growthRepo,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.stellar_beacon_info", err)
	}
	newsletterService, err := service.NewNewsletterService(service.NewsletterServiceDeps{
		Repo: newsletterRepo, Articles: article, Mailer: smtpMailer, Limiter: redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.newsletter", err)
	}
	growthService, err := service.NewGrowthService(service.GrowthServiceDeps{Repo: growthRepo, Limiter: redisCache})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.growth", err)
	}
	contentAnalyticsService, err := service.NewContentAnalyticsService(service.ContentAnalyticsServiceDeps{
		Repo: contentAnalyticsRepo, Articles: article, Cache: redisCache, Visitor: visitorResolver, Limiter: redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.content_analytics", err)
	}
	seoService, err := service.NewSeoService(article)
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.seo", err)
	}
	articleService, err := service.NewArticleService(service.ArticleServiceDeps{
		Repo:             article,
		Reactions:        articleReaction,
		ContentAnalytics: contentAnalyticsRepo,
		Cache:            redisCache,
		Storage:          ossStorage,
		Search:           searcher,
		Newsletter:       newsletterService,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.article", err)
	}
	commentService, err := service.NewCommentService(service.CommentServiceDeps{
		Repo:          comment,
		Website:       stellarBeacon,
		Users:         userInfo,
		Articles:      article,
		Notifications: notification.Notifier(),
		Limiter:       redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.comment", err)
	}
	articleReactionService, err := service.NewArticleReactionService(service.ArticleReactionServiceDeps{
		Repo:     articleReaction,
		Articles: article,
		Limiter:  redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.article_reaction", err)
	}
	photoAlbumService, err := service.NewPhotoAlbumService(service.PhotoAlbumServiceDeps{
		Repo:    photoAlbum,
		Photos:  photo,
		Storage: ossStorage,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.photo_album", err)
	}
	photoService, err := service.NewPhotoService(service.PhotoServiceDeps{
		Repo:    photo,
		Albums:  photoAlbum,
		Storage: ossStorage,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.photo", err)
	}
	talkService, err := service.NewTalkService(service.TalkServiceDeps{
		Repo:     talk,
		Comments: comment,
		Storage:  ossStorage,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.talk", err)
	}
	userAuthService, err := service.NewUserAuthService(service.UserAuthServiceDeps{
		Repo:    auth,
		Website: stellarBeacon,
		Cache:   redisCache,
		Mailer:  smtpMailer,
		Visitor: visitorResolver,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.user_auth", err)
	}
	userInfoService, err := service.NewUserInfoService(service.UserInfoServiceDeps{
		Repo:    userInfo,
		Cache:   redisCache,
		Storage: ossStorage,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.user_info", err)
	}
	seriesService, err := service.NewSeriesService(service.SeriesServiceDeps{
		Repo:     series,
		Articles: article,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.series", err)
	}
	mediaService := service.NewMediaService(ossStorage)

	scheduler := task.NewScheduler(job, jobLog, redisCache)
	if err := task.RegisterDefaultTargets(scheduler, task.DefaultTargetsDeps{
		Articles: article, Newsletter: newsletterService, Growth: growthRepo,
		JobLogs: jobLog, UserAreas: userAuthService,
	}); err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.scheduler.targets", err)
	}
	jobService := service.NewJobService(job, scheduler)

	api.ConfigureServices(api.Services{
		Article:          articleService,
		ArticleReaction:  articleReactionService,
		Series:           seriesService,
		StellarBeacon:    stellarBeacon,
		Category:         service.NewCategoryService(category),
		Comment:          commentService,
		ErrorLog:         service.NewErrorLogService(errorLog),
		FriendLink:       service.NewFriendLinkService(friendLink),
		JobLog:           service.NewJobLogService(jobLog),
		Job:              jobService,
		Menu:             service.NewMenuService(menu),
		Media:            mediaService,
		OperationLog:     service.NewOperationLogService(operationLog),
		PhotoAlbum:       photoAlbumService,
		Photo:            photoService,
		Resource:         service.NewResourceService(resource),
		Role:             service.NewRoleService(role),
		Tag:              service.NewTagService(tag),
		Talk:             talkService,
		UserAuth:         userAuthService,
		UserInfo:         userInfoService,
		Seo:              seoService,
		Newsletter:       newsletterService,
		Growth:           growthService,
		ContentAnalytics: contentAnalyticsService,
	})
	middlewares.ConfigureRoleRepository(role)
	middlewares.ConfigureUserAuthService(userAuthService)
	newsletterDone := make(chan struct{})
	go func() {
		defer close(newsletterDone)
		newsletterService.Run(runCtx)
	}()
	appruntime.SetComponent("newsletter", "ready")
	if err := scheduler.Start(runCtx); err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.scheduler.start", err)
	}
	appruntime.SetComponent("scheduler", "ready")
	return &Runtime{
		cancel: cancel, scheduler: scheduler, newsletter: newsletterService,
		newsletterDone: newsletterDone, commentQueue: commentQueue,
	}, nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.stopOnce.Do(func() {
		appruntime.SetReady(false)
		appruntime.SetComponent("scheduler", "stopping")
		if r.scheduler != nil {
			r.stopErr = r.scheduler.Stop(ctx)
		}
		appruntime.SetComponent("scheduler", "stopped")
		if r.cancel != nil {
			r.cancel()
		}
		if r.newsletterDone != nil {
			select {
			case <-r.newsletterDone:
			case <-ctx.Done():
				if r.stopErr == nil {
					r.stopErr = ctx.Err()
				}
			}
		}
		if r.commentQueue != nil {
			appruntime.SetComponent("commentQueue", "stopping")
			if err := r.commentQueue.Stop(ctx); err != nil && r.stopErr == nil {
				r.stopErr = err
			}
			appruntime.SetComponent("commentQueue", "stopped")
		}
		appruntime.SetComponent("newsletter", "stopped")
	})
	return r.stopErr
}
