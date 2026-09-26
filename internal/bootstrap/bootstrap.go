package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/cache"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/mail/smtp"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/monitoring"
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
	monitorDone    chan struct{}

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
	articleSearchService, err := service.NewArticleSearchService(article, searcher)
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.article_search", err)
	}
	platform := repository.NewPlatformRepo(engine)
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
	studioOperations := repository.NewStudioOperationsRepo(engine)
	contentAudit := repository.NewContentAuditRepo(engine)
	followRepo := repository.NewFollowRepo(engine)
	topicSubscriptionRepo := repository.NewTopicSubscriptionRepo(engine)
	recommendationRepo := repository.NewRecommendationRepo(engine)
	collectionRepo := repository.NewCollectionRepo(engine)
	collectionSubRepo := repository.NewCollectionSubscriptionRepo(engine)
	collectionReactionRepo := repository.NewCollectionReactionRepo(engine)
	commentReactionRepo := repository.NewCommentReactionRepo(engine)
	monitorRepo := repository.NewSystemMonitorRepository(engine)

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
		Repo: contentAnalyticsRepo, Articles: article, Studio: studioOperations, Cache: redisCache, Visitor: visitorResolver, Limiter: redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.content_analytics", err)
	}
	seoService, err := service.NewSeoService(article, platform)
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.seo", err)
	}
	platformService, err := service.NewPlatformService(service.PlatformServiceDeps{
		Repo: platform, Articles: article, Newsletter: newsletterService, Storage: ossStorage, Cache: redisCache,
		Reactions: articleReaction, Comments: comment, SearchIndex: articleSearchService,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.platform", err)
	}
	articleService, err := service.NewArticleService(service.ArticleServiceDeps{
		Repo:             article,
		Reactions:        articleReaction,
		ContentAnalytics: contentAnalyticsRepo,
		Cache:            redisCache,
		Storage:          ossStorage,
		Search:           searcher,
		SearchIndex:      articleSearchService,
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
		Talks:         talk,
		Collections:   collectionRepo,
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
	recommendationService, err := service.NewRecommendationService(recommendationRepo, articleReaction)
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.recommendation", err)
	}
	collectionService, err := service.NewCollectionService(collectionRepo, platform, article, articleReaction)
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.collection", err)
	}
	collectionSubService := service.NewCollectionSubscriptionService(collectionSubRepo)
	collectionReactionService, err := service.NewCollectionReactionService(service.CollectionReactionServiceDeps{
		Repo: collectionReactionRepo, Collections: collectionRepo, Limiter: redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.collection_reaction", err)
	}
	commentReactionService, err := service.NewCommentReactionService(service.CommentReactionServiceDeps{
		Repo: commentReactionRepo, Limiter: redisCache,
	})
	if err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.service.comment_reaction", err)
	}

	scheduler := task.NewScheduler(job, jobLog, redisCache)
	if err := task.RegisterDefaultTargets(scheduler, task.DefaultTargetsDeps{
		Publishes: article, Newsletter: newsletterService, Growth: growthRepo,
		JobLogs: jobLog, UserAreas: userAuthService, StudioActivation: platform, Search: articleSearchService,
	}); err != nil {
		cancel()
		return nil, errors.Unavailable("bootstrap.scheduler.targets", err)
	}
	jobService := service.NewJobService(job, scheduler)
	monitorProbes := []monitoring.Probe{
		{
			Name: "postgresql", Required: true,
			Check: func(ctx context.Context) (string, string) {
				if err := engine.PingContext(ctx); err != nil {
					return port.MonitorStatusUnhealthy, "postgresUnavailable"
				}
				return port.MonitorStatusHealthy, "connected"
			},
		},
		{
			Name: "redis",
			Check: func(ctx context.Context) (string, string) {
				if err := redisCache.Ping(ctx); err != nil {
					return port.MonitorStatusDegraded, "redisUnavailable"
				}
				return port.MonitorStatusHealthy, "connected"
			},
		},
		{
			Name: "meilisearch",
			Check: func(ctx context.Context) (string, string) {
				if err := searcher.CheckHealth(ctx); err != nil {
					return port.MonitorStatusDegraded, "meilisearchUnavailable"
				}
				return port.MonitorStatusHealthy, "connected"
			},
		},
		{
			Name: "objectStorage",
			Check: func(ctx context.Context) (string, string) {
				checker, ok := ossStorage.(port.ObjectStorageHealthChecker)
				if !ok {
					return port.MonitorStatusUnknown, "storageUnsupported"
				}
				if err := checker.CheckHealth(ctx); err != nil {
					return port.MonitorStatusDegraded, "storageUnavailable"
				}
				return port.MonitorStatusHealthy, "storageConnected"
			},
		},
		{
			Name: "smtp",
			Check: func(ctx context.Context) (string, string) {
				status := smtpMailer.Check(ctx)
				if !status.Configured {
					return port.MonitorStatusNotConfigured, "smtpNotConfigured"
				}
				if !status.Reachable {
					return port.MonitorStatusDegraded, "smtpUnavailable"
				}
				return port.MonitorStatusHealthy, "smtpConnected"
			},
		},
	}
	monitorCollector := monitoring.NewCollector(monitorRepo, monitoring.DefaultRequestMetrics, monitorProbes, func(ctx context.Context) []port.MonitorWorker {
		components := appruntime.Snapshot().Components
		workerStatus := func(name string) string {
			if components[name] == "ready" {
				return port.MonitorStatusHealthy
			}
			if components[name] == "stopped" || components[name] == "stopping" {
				return port.MonitorStatusUnhealthy
			}
			return port.MonitorStatusUnknown
		}
		workers := []port.MonitorWorker{
			{Name: "scheduler", Status: port.MonitorStatusUnhealthy, Running: scheduler.RunningCount(), UpdatedAt: time.Now()},
			{Name: "newsletter", Status: workerStatus("newsletter"), UpdatedAt: time.Now()},
			{Name: "databaseLogWriter", Status: workerStatus("logQueue"), UpdatedAt: time.Now()},
			commentQueue.MonitorWorker(),
		}
		if scheduler.Ready() {
			workers[0].Status = port.MonitorStatusHealthy
		}
		statsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		stats, err := newsletterRepo.Stats(statsCtx)
		cancel()
		if err == nil {
			workers[1].Queued = stats.Queued
			workers[1].Failed = stats.Failed
			workers[1].Running = int(stats.Sending)
		} else {
			workers[1].Status = port.MonitorStatusDegraded
		}
		return workers
	})

	api.ConfigureServices(api.Services{
		Article:            articleService,
		Platform:           platformService,
		ArticleReaction:    articleReactionService,
		Series:             seriesService,
		StellarBeacon:      stellarBeacon,
		Category:           service.NewCategoryService(category),
		Comment:            commentService,
		ErrorLog:           service.NewErrorLogService(errorLog),
		FriendLink:         service.NewFriendLinkService(friendLink),
		JobLog:             service.NewJobLogService(jobLog),
		Job:                jobService,
		Menu:               service.NewMenuService(menu),
		Media:              mediaService,
		OperationLog:       service.NewOperationLogService(operationLog),
		PhotoAlbum:         photoAlbumService,
		Photo:              photoService,
		Resource:           service.NewResourceService(resource),
		Role:               service.NewRoleService(role),
		Tag:                service.NewTagService(tag),
		Talk:               talkService,
		UserAuth:           userAuthService,
		UserInfo:           userInfoService,
		Seo:                seoService,
		Newsletter:         newsletterService,
		Growth:             growthService,
		ContentAnalytics:   contentAnalyticsService,
		ContentAudit:       service.NewContentAuditService(contentAudit),
		Follow:             service.NewFollowService(followRepo),
		TopicSubscription:  service.NewTopicSubscriptionService(topicSubscriptionRepo),
		Recommendation:     recommendationService,
		Collection:         collectionService,
		CollectionSub:      collectionSubService,
		CollectionReaction: collectionReactionService,
		CommentReaction:    commentReactionService,
		SystemMonitor:      service.NewSystemMonitorService(monitorCollector),
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
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		monitorCollector.Run(runCtx)
	}()
	return &Runtime{
		cancel: cancel, scheduler: scheduler, newsletter: newsletterService,
		newsletterDone: newsletterDone, commentQueue: commentQueue, monitorDone: monitorDone,
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
		if r.monitorDone != nil {
			select {
			case <-r.monitorDone:
			case <-ctx.Done():
				if r.stopErr == nil {
					r.stopErr = ctx.Err()
				}
			}
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
