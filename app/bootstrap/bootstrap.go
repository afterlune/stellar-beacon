package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	agentapp "benetnasch/app/application/agent"
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/api"
	"benetnasch/app/infra/ai"
	"benetnasch/app/infra/ai/writing"
	"benetnasch/app/infra/cache"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/mailer"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/observability"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/search"
	"benetnasch/app/infra/task"
	"benetnasch/app/infra/visitor"
	"github.com/meilisearch/meilisearch-go"
	"xorm.io/xorm"
)

// AIComponents is the optional AI composition root. Constructing it does not
// make network calls; callers can keep the existing disabled-by-default
// behavior while passing the router and bounded run observer into future AI
// application services.
type AIComponents struct {
	Router  *ai.ModelRouter
	Metrics *ai.RunMetricsObserver
}

func NewAIComponents() AIComponents {
	settings := new(config.AI).AI()
	metrics := ai.NewRunMetricsObserverWithConfig(ai.RunMetricsObserverConfig{
		Capacity:         128,
		DailyTokenBudget: settings.DailyTokenBudget,
		OnBudgetAlert: func(alert ai.BudgetAlert) {
			level := slog.LevelWarn
			message := "AI daily token budget threshold reached"
			if alert.Exceeded {
				level = slog.LevelError
				message = "AI daily token budget exceeded"
			}
			slog.LogAttrs(context.Background(), level, message,
				slog.String("date", alert.Date),
				slog.Int64("budget_tokens", alert.Budget),
				slog.Int64("used_tokens", alert.Used),
				slog.Int64("remaining_tokens", alert.Remaining),
			)
		},
	})
	client := &http.Client{Timeout: settings.RequestTimeout}
	return AIComponents{
		Router:  ai.NewModelRouter(*settings, config.ProviderSettings(), client, metrics),
		Metrics: metrics,
	}
}

var _ port.ModelRouter = (*ai.ModelRouter)(nil)

// Runtime contains optional long-running workers assembled alongside the
// application services. A nil worker means the corresponding capability is
// disabled and must not be registered with the supervisor.
type Runtime struct {
	JobRunner                  port.JobRunner
	ArticleIndexWorker         task.Worker
	ContentUnderstandingWorker task.Worker
	AgentBehaviorWorker        task.Worker
	ContentProjectionWorker    task.Worker
	DreamWorker                task.Worker
	DreamImageWorker           task.Worker
	TimeCapsuleWorker          task.Worker
}

// ArticleIndexBackfillRuntime is the explicit composition root used by the
// short-lived backfill commands. Constructing it only creates clients and
// adapters; it does not create an index, run a migration, or start a worker.
type ArticleIndexBackfillRuntime struct {
	Backfill *task.PersistentArticleIndexBackfill
	Spec     search.ArticleChunksIndexSpec
}

// ArticleIndexBackfillControlRuntime exposes only the durable control row.
// Pause, resume, and status must remain available even when an embedding
// provider or Meilisearch is temporarily unavailable.
type ArticleIndexBackfillControlRuntime struct {
	States port.ArticleIndexBackfillRepository
	RunID  string
}

// ArticleIndexPlanRuntime is the read-only data preflight for an article
// index rebuild. It deliberately contains neither a model router nor a
// Meilisearch adapter, so planning cannot call a provider or mutate an index.
type ArticleIndexPlanRuntime struct {
	Plan *task.ArticleIndexPlan
	Spec search.ArticleChunksIndexSpec
}

// ArticleChunksIndexProvisionRuntime is the explicit composition root for
// creating/configuring a versioned article chunk index. It does not contact
// Meilisearch while being assembled; callers must still pass their own
// write-authorization gate before invoking ProvisionArticleChunksIndex.
type ArticleChunksIndexProvisionRuntime struct {
	Client meilisearch.ServiceManager
	Spec   search.ArticleChunksIndexSpec
}

// Initialize keeps the historical bootstrap API for callers that do not need
// optional workers.
func Initialize() error {
	_, err := InitializeRuntime()
	return err
}

// InitializeRuntime is the composition root for application services and
// feature-gated workers. The only layer that resolves the xorm engine and
// concrete repositories is this package; controllers and application services
// receive domain ports.
func InitializeRuntime() (Runtime, error) {
	var runtime Runtime
	manualJobHandlers := make(map[string]task.OneShotHandler)
	engine := ormInit.GetEngine()
	if engine == nil {
		return runtime, errors.Unavailable("bootstrap.database", nil)
	}
	redisConfig := new(config.Redis).Redis()
	redisCache := cache.NewRedisCache(redisConfig)
	ossStorage, err := oss.NewObjectStorage(new(config.Oss).Oss())
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.storage", err)
	}
	meiliConfig := new(config.MeiliSearch).MeiliSearch()
	searcher := search.NewMeiliSearcher(meiliConfig)
	searchMetrics := search.NewSearchMetricsObserverWithConfig(search.SearchMetricsObserverConfig{
		Capacity: 128,
		OnAlert: func(alert search.SearchAlert) {
			slog.Warn("search error rate threshold reached",
				"index", alert.Index,
				"mode", alert.Mode,
				"total", alert.Total,
				"failed", alert.Failed,
				"error_rate", alert.ErrorRate,
			)
		},
	})
	searcher.SetMetricsObserver(searchMetrics)
	smtpMailer := mailer.NewSMTPMailer(new(config.Email).Email())
	visitorResolver := visitor.NewResolver()
	aiSettings := new(config.AI).AI()
	spaceCompanionSettings := config.SpaceCompanion()
	aiComponents := NewAIComponents()
	operationalMetrics := observability.NewProvider(aiComponents.Router, searchMetrics)
	agentSafetySwitch, err := agentapp.NewEmergencySwitch(redisCache, aiSettings.EmergencyStop)
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.agent.safety_switch", err)
	}
	var aiStudio service.AIStudioService = new(service.MyAIStudioService)
	var aiVision service.AIVisionService = service.NewDisabledAIVisionService()
	var agentChat service.AgentChatService = new(service.MyAgentChatService)
	agentFeatures := port.AgentFeatureFlags{
		PublicChat: aiSettings.FeatureEnabled(config.AIFeaturePublicChat),
		Vitals:     aiSettings.FeatureEnabled(config.AIFeatureVitals),
		Galaxy:     aiSettings.FeatureEnabled(config.AIFeatureGalaxy),
		Dreams:     aiSettings.FeatureEnabled(config.AIFeatureDreams),
		Capsules:   aiSettings.FeatureEnabled(config.AIFeatureCapsules),
		Radio:      aiSettings.FeatureEnabled(config.AIFeatureRadio),
		Videos:     aiSettings.FeatureEnabled(config.AIFeatureVideos),
		TTSEnabled: aiSettings.FeatureEnabled(config.AIFeatureTTS),
	}
	var agentRhythm port.AgentRhythm
	var vitalsProvider port.AgentVitalsProvider
	var agentEvents port.AgentEventStore
	var agentActivity port.AgentActivityRepository
	var aiJobs port.AIJobRepository
	var agentProfileRepository port.AgentProfileRepository
	var agentReviewPolicyRepository port.AgentReviewPolicyRepository
	var agentMemoryService service.AgentMemoryService = service.NewDisabledAgentMemoryService()
	var aiReviewRepository port.AIReviewRepository
	var behaviorPolicy *agentapp.BehaviorPolicy
	var behaviorScheduler *agentapp.BehaviorScheduler
	var behaviorGenerator port.AgentBehaviorGenerator
	var agentReviewPublisher port.AgentReviewPublisher
	var dreamReviewPublisher port.AgentReviewPublisher
	var aiWriting port.WritingGateway
	var contentProjectionRepository port.ContentProjectionRepository
	var contentGalaxyService service.ContentGalaxyService = service.NewDisabledContentGalaxyService()
	var dreamRepository port.DreamRepository
	var dreamService service.DreamService = service.NewDisabledDreamService()
	var timeCapsuleRepository port.TimeCapsuleRepository
	var timeCapsuleService service.TimeCapsuleService = service.NewDisabledTimeCapsuleService()
	var radioService service.RadioService = service.NewDisabledRadioService()
	var videoRepository port.VideoRepository
	var videoService service.VideoService = service.NewDisabledVideoService()
	var spaceCompanionService service.SpaceCompanionService = service.NewDisabledSpaceCompanionService()
	if agentFeatures.Galaxy {
		contentProjectionRepository = repository.NewContentProjectionRepository(engine)
		contentGalaxyService, err = service.NewContentGalaxyService(contentProjectionRepository, true)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.content_galaxy", err)
		}
	}
	if aiSettings.FeatureEnabled(config.AIFeatureArticleIndexing) || aiSettings.FeatureEnabled(config.AIFeatureContentUnderstanding) || aiSettings.FeatureEnabled(config.AIFeatureBehavior) || aiSettings.FeatureEnabled(config.AIFeatureDreams) {
		// The queue table is created only by the explicit migrate command. Keep
		// this dependency nil by default so existing deployments are unchanged
		// until an operator enables an AI worker after applying migrations.
		aiJobs = repository.NewAIJobRepository(engine)
	}
	var articleIndexJobs port.AIJobRepository
	var articleChunkSpec search.ArticleChunksIndexSpec
	if aiSettings.FeatureEnabled(config.AIFeatureArticleIndexing) {
		articleIndexJobs = aiJobs
		articleChunkSpec, err = search.NewArticleChunksIndexSpec(aiSettings.Embedding, aiSettings.EmbeddingConfig)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.search.article_chunks.spec", err)
		}
		chunkSearcher, err := search.NewMeiliArticleChunkSearcherWithSemanticRatio(meiliConfig, articleChunkSpec, aiComponents.Router, aiSettings.SemanticRatio)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.search.article_chunks", err)
		}
		chunkSearcher.SetMetricsObserver(searchMetrics)
		searcher.SetArticleChunkSearcher(chunkSearcher)
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
	spacePrincipalRepository := repository.NewSpacePrincipalRepository(engine)
	spacePublicationRepository := repository.NewSpacePublicationRepository(engine)
	// Keep the profile repository available to the admin control plane even
	// while all public Agent features are disabled. Persistence is explicitly
	// opt-in because its table is installed only by the migrate command.
	agentProfileRepository, err = configureAgentProfileRepository(engine, aiSettings.AgentProfile)
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.agent_profile", err)
	}
	agentReviewPolicyRepository, err = configureAgentReviewPolicyRepository(engine, aiSettings.AgentReviewPolicy, aiSettings.AgentBehavior)
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.agent_review_policy", err)
	}
	if aiSettings.AgentMemory.PersistenceEnabled {
		memoryRepository := repository.NewAgentMemoryRepository(engine)
		agentMemoryService, err = service.NewAgentMemoryService(service.AgentMemoryServiceDeps{
			Assertions: memoryRepository,
			History:    memoryRepository,
			Conflicts:  memoryRepository,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_memory", err)
		}
	}
	if aiSettings.FeatureEnabled(config.AIFeatureWriting) || aiSettings.FeatureEnabled(config.AIFeatureVision) || aiSettings.FeatureEnabled(config.AIFeatureBehavior) || agentFeatures.Dreams {
		aiReviewRepository = repository.NewAIReviewRepository(engine)
	}
	if agentFeatures.Dreams || aiSettings.FeatureEnabled(config.AIFeatureDreamImages) {
		dreamRepository = repository.NewDreamRepository(engine)
	}
	if agentFeatures.Capsules {
		timeCapsuleRepository = repository.NewTimeCapsuleRepository(engine)
		timeCapsuleService, err = service.NewTimeCapsuleService(service.TimeCapsuleServiceDeps{
			Capsules: timeCapsuleRepository,
			Safety:   agentSafetySwitch,
			Enabled:  true,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.time_capsule.service", err)
		}
	}
	if agentFeatures.Videos {
		videoRepository = repository.NewVideoRepository(engine)
		videoService, err = service.NewVideoService(service.VideoServiceDeps{
			Repository:     videoRepository,
			Storage:        ossStorage,
			AllowedOrigins: aiSettings.VideoAllowedOrigins,
			Enabled:        true,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.video.service", err)
		}
	}
	if agentFeatures.Vitals || aiSettings.FeatureEnabled(config.AIFeatureBehavior) {
		// The activity table is created only by the explicit migrate command.
		// The adapter is optional for the public chat path and has no effect
		// while vitals/behavior are disabled.
		agentActivity = repository.NewAgentActivityRepository(engine)
	}
	if agentFeatures.PublicChat || agentFeatures.Vitals || agentFeatures.Radio {
		rhythmPolicy, rhythmErr := agentapp.NewRhythmPolicy(agentapp.RhythmSettings{
			Timezone:   aiSettings.AgentRhythm.Timezone,
			AwakeStart: aiSettings.AgentRhythm.AwakeStart,
			DuskStart:  aiSettings.AgentRhythm.DuskStart,
			NightStart: aiSettings.AgentRhythm.NightStart,
		})
		if rhythmErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent.rhythm", rhythmErr)
		}
		agentRhythm = rhythmPolicy
		vitalsProvider, err = agentapp.NewBasicVitalsProvider(agentapp.VitalsDeps{
			Site:     site,
			Cache:    redisCache,
			Rhythm:   agentRhythm,
			Activity: agentActivity,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent.vitals", err)
		}
	}
	if agentFeatures.Radio {
		radioService, err = service.NewRadioService(service.RadioServiceDeps{
			Articles:   article,
			Profiles:   agentProfileRepository,
			Rhythm:     agentRhythm,
			ProfileID:  aiSettings.AgentProfile.ID,
			Enabled:    true,
			TTSEnabled: agentFeatures.TTSEnabled,
			Now:        time.Now,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.radio.service", err)
		}
	}
	spaceCompanionService, err = service.NewSpaceCompanionService(service.SpaceCompanionServiceDeps{
		Articles:     article,
		Albums:       photoAlbum,
		Photos:       photo,
		Dreams:       dreamRepository,
		Videos:       videoRepository,
		Radio:        radioService,
		Publications: spacePublicationRepository,
		AgentID:      spaceCompanionSettings.AgentID,
		Enabled:      spaceCompanionSettings.Enabled,
		Publish:      spaceCompanionSettings.PublishEnabled,
		Now:          time.Now,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.space_companion.service", err)
	}

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
		return runtime, errors.Unavailable("bootstrap.service.benetnasch_info", err)
	}
	service.ConfigureBenetnaschService(benetnasch)

	articleService, err := service.NewArticleService(service.ArticleServiceDeps{
		Repo:                     article,
		Cache:                    redisCache,
		Storage:                  ossStorage,
		Search:                   searcher,
		AIJobs:                   articleIndexJobs,
		ContentUnderstandingJobs: aiJobsForFeature(aiSettings, aiJobs),
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.article", err)
	}
	commentService, err := service.NewCommentService(service.CommentServiceDeps{
		Repo:    comment,
		Website: benetnasch,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.comment", err)
	}
	photoAlbumService, err := service.NewPhotoAlbumService(service.PhotoAlbumServiceDeps{
		Repo:    photoAlbum,
		Photos:  photo,
		Storage: ossStorage,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.photo_album", err)
	}
	photoService, err := service.NewPhotoService(service.PhotoServiceDeps{
		Repo:    photo,
		Albums:  photoAlbum,
		Storage: ossStorage,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.photo", err)
	}
	talkService, err := service.NewTalkService(service.TalkServiceDeps{
		Repo:     talk,
		Comments: comment,
		Storage:  ossStorage,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.talk", err)
	}
	userAuthService, err := service.NewUserAuthService(service.UserAuthServiceDeps{
		Repo:    auth,
		Website: benetnasch,
		Cache:   redisCache,
		Mailer:  smtpMailer,
		Visitor: visitorResolver,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.user_auth", err)
	}
	userInfoService, err := service.NewUserInfoService(service.UserInfoServiceDeps{
		Repo:    userInfo,
		Cache:   redisCache,
		Storage: ossStorage,
	})
	if err != nil {
		return runtime, errors.Unavailable("bootstrap.service.user_info", err)
	}

	if aiSettings.FeatureEnabled(config.AIFeaturePublicChat) {
		chatGateway, _, err := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseChat)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_chat.chat", err)
		}
		sessionStore := cache.NewAgentSessionStore(redisCache)
		sessionCoordinator := cache.NewAgentSessionCoordinator(redisCache)
		agentEvents = cache.NewAgentEventStore(redisCache)
		turnQuota := cache.NewAgentTurnQuota(redisCache, aiSettings.AgentLimits.GuestDailyTurns, aiSettings.AgentLimits.AdminDailyTurns)
		toolRegistry, err := agentapp.NewPublicToolRegistry(agentapp.PublicToolDeps{
			Articles:   article,
			Searcher:   searcher,
			Categories: category,
			Tags:       tag,
			Vitals:     vitalsProvider,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_chat.tools", err)
		}
		publicAgent, err := service.NewAgentChatService(service.AgentChatServiceDeps{
			Chat:        chatGateway,
			Profiles:    agentProfileRepository,
			Sessions:    sessionStore,
			Coordinator: sessionCoordinator,
			Tools:       toolRegistry,
			Quota:       turnQuota,
			Safety:      agentSafetySwitch,
			Rhythm:      agentRhythm,
			Limits: service.AgentChatLimits{
				GuestDailyTurns:   aiSettings.AgentLimits.GuestDailyTurns,
				AdminDailyTurns:   aiSettings.AgentLimits.AdminDailyTurns,
				MaxConcurrent:     aiSettings.AgentLimits.MaxConcurrent,
				MaxInputRunes:     aiSettings.AgentLimits.MaxInputRunes,
				MaxAnswerRunes:    aiSettings.AgentLimits.MaxAnswerRunes,
				MaxToolCalls:      aiSettings.AgentLimits.MaxToolCalls,
				MaxOutputTokens:   aiSettings.MaxOutputTokens,
				SensitivePatterns: aiSettings.AgentLimits.SensitivePatterns,
			},
			ProfileID: aiSettings.AgentProfile.ID,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_chat", err)
		}
		agentChat = publicAgent
	}
	if aiSettings.FeatureEnabled(config.AIFeatureBehavior) {
		if len(aiSettings.AgentBehavior.AllowedActions) == 0 {
			return runtime, errors.Invalid("bootstrap.agent_behavior.actions", "allowed_actions must be explicitly configured before enabling behavior")
		}
		reviewPolicy, reviewPolicyErr := agentReviewPolicyRepository.Get(context.Background(), aiSettings.AgentReviewPolicy.ID)
		if reviewPolicyErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.review_policy", reviewPolicyErr)
		}
		policy, policyErr := agentapp.NewBehaviorPolicy(agentapp.BehaviorPolicyConfig{
			ActorID:             aiSettings.AgentBehavior.ActorID,
			Nickname:            aiSettings.AgentBehavior.Nickname,
			ProfileID:           aiSettings.AgentBehavior.ProfileID,
			PromptVersion:       aiSettings.AgentProfile.PromptVersion,
			AllowedActions:      reviewPolicy.ActionNames(),
			SensitivePatterns:   reviewPolicy.SensitivePatterns,
			MaxCandidateRunes:   reviewPolicy.MaxCandidateRunes,
			SimilarityThreshold: reviewPolicy.SimilarityThreshold,
			ReviewTTL:           reviewPolicy.ReviewTTL,
			DailyLimit:          reviewPolicy.DailyLimit,
			PerArticleLimit:     reviewPolicy.PerArticleLimit,
			PerActionLimit:      reviewPolicy.PerActionLimit,
		})
		if policyErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.policy", policyErr)
		}
		behaviorPolicy = policy
		if policy.ProfileID() != strings.TrimSpace(aiSettings.AgentProfile.ID) {
			return runtime, errors.Invalid("bootstrap.agent_behavior.profile", "behavior profile must match the configured agent profile")
		}
		actorUserID, actorErr := strconv.Atoi(policy.ActorID())
		if actorErr != nil || actorUserID <= 0 {
			return runtime, errors.Invalid("bootstrap.agent_behavior.actor", "actor_id must be a positive user id")
		}
		actorInfo, actorErr := userInfo.GetByID(context.Background(), actorUserID)
		if actorErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.actor", actorErr)
		}
		if actorInfo.IsDisable != 0 {
			return runtime, errors.New(errors.KindForbidden, "bootstrap.agent_behavior.actor", nil)
		}
		forgottenSelector, selectorErr := agentapp.NewForgottenArticleSelector(agentapp.ForgottenArticleSelectorConfig{
			StaleAfter:    aiSettings.AgentBehavior.ForgottenAfter,
			MaxCandidates: aiSettings.AgentBehavior.ForgottenMaxCandidates,
		})
		if selectorErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.forgotten", selectorErr)
		}
		agentReviewPublisher, err = service.NewAgentReviewPublisher(service.AgentReviewPublisherDeps{
			Articles: article,
			Comments: commentService,
			Talks:    talkService,
			Activity: agentActivity,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.publisher", err)
		}
		behaviorScheduler, err = agentapp.NewBehaviorScheduler(agentapp.BehaviorSchedulerDeps{
			Sources:           article,
			Jobs:              aiJobs,
			Policy:            behaviorPolicy,
			ForgottenSelector: forgottenSelector,
			BatchSize:         aiSettings.AgentBehavior.ScanBatchSize,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.scheduler", err)
		}
		behaviorChat, _, chatErr := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseBehavior)
		if chatErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.chat", chatErr)
		}
		behaviorGenerator, err = agentapp.NewBehaviorCandidateService(agentapp.BehaviorCandidateServiceDeps{
			Articles:        article,
			Profiles:        agentProfileRepository,
			Reviews:         aiReviewRepository,
			Chat:            behaviorChat,
			Policy:          behaviorPolicy,
			MaxOutputTokens: aiSettings.AgentBehavior.MaxOutputTokens,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.generator", err)
		}
		worker, workerErr := task.NewAgentBehaviorWorker(task.AgentBehaviorWorkerDeps{
			Jobs:         aiJobs,
			Scheduler:    behaviorScheduler,
			Generator:    behaviorGenerator,
			Safety:       agentSafetySwitch,
			ScanInterval: aiSettings.AgentBehavior.ScanInterval,
		})
		if workerErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_behavior.worker", workerErr)
		}
		runtime.AgentBehaviorWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetAgentBehavior] = worker.RunOnce
	}
	if agentFeatures.Dreams {
		actorID := strings.TrimSpace(aiSettings.AgentBehavior.ActorID)
		profileID := strings.TrimSpace(aiSettings.AgentBehavior.ProfileID)
		if profileID == "" {
			profileID = strings.TrimSpace(aiSettings.AgentProfile.ID)
		}
		if actorID == "" {
			return runtime, errors.Invalid("bootstrap.agent_dream.actor", "actor_id must be configured")
		}
		actorUserID, actorErr := strconv.Atoi(actorID)
		if actorErr != nil || actorUserID <= 0 {
			return runtime, errors.Invalid("bootstrap.agent_dream.actor", "actor_id must be a positive user id")
		}
		actorInfo, actorErr := userInfo.GetByID(context.Background(), actorUserID)
		if actorErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.actor", actorErr)
		}
		if actorInfo.IsDisable != 0 {
			return runtime, errors.New(errors.KindForbidden, "bootstrap.agent_dream.actor", nil)
		}
		dreamService, err = service.NewDreamService(dreamRepository, true)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.service", err)
		}
		dreamReviewPublisher, err = service.NewDreamReviewPublisher(service.DreamReviewPublisherDeps{Dreams: dreamRepository})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.publisher", err)
		}
		dreamScheduler, err := agentapp.NewDreamScheduler(agentapp.DreamSchedulerDeps{
			Sources:       article,
			Jobs:          aiJobs,
			ProfileID:     profileID,
			PromptVersion: aiSettings.AgentProfile.PromptVersion,
			BatchSize:     aiSettings.AgentBehavior.ScanBatchSize,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.scheduler", err)
		}
		dreamChat, _, chatErr := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseDream)
		if chatErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.chat", chatErr)
		}
		dreamGenerator, err := agentapp.NewDreamCandidateService(agentapp.DreamCandidateServiceDeps{
			Articles:          article,
			Profiles:          agentProfileRepository,
			Reviews:           aiReviewRepository,
			Dreams:            dreamRepository,
			Chat:              dreamChat,
			ActorID:           actorID,
			ProfileID:         profileID,
			PromptVersion:     aiSettings.AgentProfile.PromptVersion,
			ReviewTTL:         aiSettings.AgentBehavior.ReviewTTL,
			MaxOutputTokens:   aiSettings.AgentBehavior.MaxOutputTokens,
			MaxCandidateRunes: agentapp.DefaultDreamMaxCandidateRunes,
			SensitivePatterns: aiSettings.AgentBehavior.SensitivePatterns,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.generator", err)
		}
		worker, workerErr := task.NewDreamWorker(task.DreamWorkerDeps{
			Jobs:         aiJobs,
			Scheduler:    dreamScheduler,
			Generator:    dreamGenerator,
			Safety:       agentSafetySwitch,
			ScanInterval: aiSettings.AgentBehavior.ScanInterval,
		})
		if workerErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.worker", workerErr)
		}
		runtime.DreamWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetDream] = worker.RunOnce
	}
	if aiSettings.FeatureEnabled(config.AIFeatureDreamImages) {
		worker, workerErr := task.NewDreamImageWorker(task.DreamImageWorkerDeps{
			Dreams:         dreamRepository,
			Safety:         agentSafetySwitch,
			PlaceholderURL: "/dream-placeholder.svg",
		})
		if workerErr != nil {
			return runtime, errors.Unavailable("bootstrap.agent_dream.image_worker", workerErr)
		}
		runtime.DreamImageWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetDreamImage] = worker.RunOnce
	}
	if agentFeatures.Capsules {
		worker, workerErr := task.NewTimeCapsuleWorker(task.TimeCapsuleWorkerDeps{
			Capsules: timeCapsuleRepository,
			Safety:   agentSafetySwitch,
		})
		if workerErr != nil {
			return runtime, errors.Unavailable("bootstrap.time_capsule.worker", workerErr)
		}
		runtime.TimeCapsuleWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetTimeCapsule] = worker.RunOnce
	}

	if aiSettings.FeatureEnabled(config.AIFeatureArticleIndexing) {
		spec := articleChunkSpec
		chunker, err := search.NewMarkdownChunker(search.DefaultMarkdownChunkerConfig())
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.article_index.chunker", err)
		}
		chunkIndex, err := search.NewMeiliArticleChunkIndex(meiliConfig, spec)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.article_index.index", err)
		}
		worker, err := task.NewArticleIndexWorker(task.ArticleIndexWorkerDeps{
			Jobs:        articleIndexJobs,
			Articles:    article,
			Router:      aiComponents.Router,
			Index:       chunkIndex,
			Spec:        spec,
			Chunker:     chunker,
			Projections: contentProjectionRepository,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.article_index", err)
		}
		runtime.ArticleIndexWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetArticleIndex] = worker.RunOnce
	}
	if aiSettings.FeatureEnabled(config.AIFeatureContentUnderstanding) {
		chatGateway, route, err := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseWriting)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.content_understanding.chat", err)
		}
		analyzer, err := writing.NewContentAnalyzer(writing.Config{
			Chat:            chatGateway,
			Model:           route.Model,
			MaxOutputTokens: aiSettings.MaxOutputTokens,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.content_understanding.analyzer", err)
		}
		worker, err := task.NewContentUnderstandingWorker(task.ContentUnderstandingWorkerDeps{
			Jobs:     aiJobs,
			Articles: article,
			Analyzer: analyzer,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.worker.content_understanding", err)
		}
		runtime.ContentUnderstandingWorker = worker.Run
		manualJobHandlers[port.ManualJobTargetContentUnderstanding] = worker.RunOnce
		manualJobHandlers[port.ManualJobTargetContentUnderstandingAlias] = worker.RunOnce
	}
	if agentFeatures.Galaxy {
		processor, processorErr := agentapp.NewContentProjectionBatchProcessor(contentProjectionRepository)
		if processorErr != nil {
			return runtime, errors.Unavailable("bootstrap.worker.content_projection.processor", processorErr)
		}
		worker, workerErr := task.NewContentProjectionWorker(task.ContentProjectionWorkerDeps{
			Projections: contentProjectionRepository,
			Processor:   processor,
		})
		if workerErr != nil {
			return runtime, errors.Unavailable("bootstrap.worker.content_projection", workerErr)
		}
		runtime.ContentProjectionWorker = worker.Run
	}
	if aiSettings.FeatureEnabled(config.AIFeatureWriting) {
		chatGateway, route, err := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseWriting)
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.ai_studio.chat", err)
		}
		assistant, err := writing.NewAssistant(writing.Config{
			Chat:            chatGateway,
			Model:           route.Model,
			MaxOutputTokens: aiSettings.MaxOutputTokens,
		})
		if err != nil {
			return runtime, errors.Unavailable("bootstrap.ai_studio.writing", err)
		}
		aiWriting = assistant
	}
	if aiSettings.FeatureEnabled(config.AIFeatureVision) {
		chatGateway, route, visionErr := aiComponents.Router.ResolveChat(context.Background(), port.AIUseCaseVision)
		if visionErr != nil {
			return runtime, errors.Unavailable("bootstrap.ai_vision.chat", visionErr)
		}
		aiVision, visionErr = service.NewAIVisionService(service.AIVisionServiceDeps{
			Vision:          chatGateway,
			Reviews:         aiReviewRepository,
			ReviewPolicy:    agentReviewPolicyRepository,
			ReviewPolicyID:  aiSettings.AgentReviewPolicy.ID,
			Model:           route.Model,
			MaxOutputTokens: aiSettings.MaxOutputTokens,
			Timeout:         aiSettings.RequestTimeout,
			Enabled:         true,
		})
		if visionErr != nil {
			return runtime, errors.Unavailable("bootstrap.ai_vision", visionErr)
		}
	}
	if dreamReviewPublisher != nil {
		if agentReviewPublisher == nil {
			agentReviewPublisher = dreamReviewPublisher
		} else {
			agentReviewPublisher, err = service.NewCompositeAgentReviewPublisher(agentReviewPublisher, dreamReviewPublisher)
			if err != nil {
				return runtime, errors.Unavailable("bootstrap.agent_review_publisher", err)
			}
		}
	}
	if aiWriting != nil || aiSettings.FeatureEnabled(config.AIFeatureBehavior) || agentFeatures.Dreams {
		studio, studioErr := service.NewAIStudioService(service.AIStudioServiceDeps{
			Writing:        aiWriting,
			Reviews:        aiReviewRepository,
			ReviewPolicy:   agentReviewPolicyRepository,
			ReviewPolicyID: aiSettings.AgentReviewPolicy.ID,
			Publisher:      agentReviewPublisher,
			Dreams:         dreamRepository,
		})
		if studioErr != nil {
			return runtime, errors.Unavailable("bootstrap.ai_studio", studioErr)
		}
		aiStudio = studio
	}
	jobRunner, runnerErr := task.NewAllowlistedJobRunnerWithSafety(manualJobHandlers, agentSafetySwitch)
	if runnerErr != nil {
		return runtime, errors.Unavailable("bootstrap.job_runner", runnerErr)
	}
	runtime.JobRunner = jobRunner
	api.ConfigureServices(api.Services{
		Article:                  articleService,
		Benetnasch:               benetnasch,
		Category:                 service.NewCategoryService(category),
		Comment:                  commentService,
		ErrorLog:                 service.NewErrorLogService(errorLog),
		FriendLink:               service.NewFriendLinkService(friendLink),
		JobLog:                   service.NewJobLogService(jobLog),
		Job:                      service.NewJobService(job, runtime.JobRunner),
		Menu:                     service.NewMenuService(menu),
		OperationLog:             service.NewOperationLogService(operationLog),
		PhotoAlbum:               photoAlbumService,
		Photo:                    photoService,
		Resource:                 service.NewResourceService(resource),
		Role:                     service.NewRoleService(role),
		Tag:                      service.NewTagService(tag),
		Talk:                     talkService,
		UserAuth:                 userAuthService,
		UserInfo:                 userInfoService,
		AIStudio:                 aiStudio,
		AIVision:                 aiVision,
		AIProviderProbe:          service.NewAIProviderProbeService(aiComponents.Router, aiSettings.FeatureEnabled(config.AIFeatureProviderProbe)),
		OperationalObservability: service.NewOperationalObservabilityService(operationalMetrics, aiSettings.FeatureEnabled(config.AIFeatureObservability)),
		AgentProfile:             service.NewAgentProfileService(agentProfileRepository, aiSettings.AgentProfile.ID),
		AgentReviewPolicy:        service.NewAgentReviewPolicyService(agentReviewPolicyRepository, aiSettings.AgentReviewPolicy.ID),
		AgentMemory:              agentMemoryService,
		AgentChat:                agentChat,
		AgentGalaxy:              contentGalaxyService,
		AgentDreams:              dreamService,
		AgentCapsules:            timeCapsuleService,
		AgentRadio:               radioService,
		AgentVideos:              videoService,
		AgentVitals:              vitalsProvider,
		AgentFeatures:            agentFeatures,
		AgentEvents:              agentEvents,
		AgentSafety:              agentSafetySwitch,
		SpaceCompanion:           spaceCompanionService,
	})
	middlewares.ConfigureRoleRepository(role)
	middlewares.ConfigureRoleCache(redisCache)
	middlewares.ConfigureUserAuthService(userAuthService)
	middlewares.ConfigureSpacePrincipalRepository(spacePrincipalRepository)
	return runtime, nil
}

func configureAgentProfileRepository(engine *xorm.Engine, settings config.AIAgentProfileSettings) (port.AgentProfileRepository, error) {
	configured := ai.NewConfiguredAgentProfileRepository(settings)
	if !settings.PersistenceEnabled {
		return configured, nil
	}
	persistent := repository.NewAgentProfileRepository(engine)
	profileID := strings.TrimSpace(settings.ID)
	if profileID == "" {
		profileID = port.DefaultAgentProfileID
	}
	_, err := persistent.Get(context.Background(), profileID)
	if err == nil {
		return persistent, nil
	}
	if !errors.IsKind(err, errors.KindNotFound) {
		return nil, err
	}
	seed, err := configured.Get(context.Background(), profileID)
	if err != nil {
		return nil, err
	}
	if err := persistent.Save(context.Background(), seed); err != nil {
		return nil, err
	}
	return persistent, nil
}

func configureAgentReviewPolicyRepository(engine *xorm.Engine, settings config.AIAgentReviewPolicySettings, behavior config.AIAgentBehaviorSettings) (port.AgentReviewPolicyRepository, error) {
	configured := ai.NewConfiguredAgentReviewPolicyRepository(settings, behavior)
	if !settings.PersistenceEnabled {
		return configured, nil
	}
	persistent := repository.NewAgentReviewPolicyRepository(engine)
	policyID := strings.TrimSpace(settings.ID)
	if policyID == "" {
		policyID = port.DefaultAgentReviewPolicyID
	}
	_, err := persistent.Get(context.Background(), policyID)
	if err == nil {
		return persistent, nil
	}
	if !errors.IsKind(err, errors.KindNotFound) {
		return nil, err
	}
	seed, err := configured.Get(context.Background(), policyID)
	if err != nil {
		return nil, err
	}
	if err := persistent.Save(context.Background(), seed); err != nil {
		return nil, err
	}
	return persistent, nil
}

func aiJobsForFeature(settings *config.AI, jobs port.AIJobRepository) port.AIJobRepository {
	if settings == nil || !settings.FeatureEnabled(config.AIFeatureContentUnderstanding) {
		return nil
	}
	return jobs
}

// InitializeArticleIndexBackfill assembles the persistent full-rebuild path
// without starting the HTTP server or the normal lifecycle worker. The
// database migration and Meilisearch provisioning remain explicit operator
// actions, so a command invocation cannot silently alter either service just
// by constructing this runtime.
func InitializeArticleIndexBackfill(runID string, pageSize int) (ArticleIndexBackfillRuntime, error) {
	var runtime ArticleIndexBackfillRuntime
	engine := ormInit.GetEngine()
	if engine == nil {
		return runtime, errors.Unavailable("bootstrap.article_index_backfill.database", nil)
	}
	aiSettings := new(config.AI).AI()
	if !aiSettings.FeatureEnabled(config.AIFeatureArticleIndexing) {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.feature", "ai.article_indexing must be enabled for the backfill command")
	}
	aiComponents := NewAIComponents()
	spec, err := search.NewArticleChunksIndexSpec(aiSettings.Embedding, aiSettings.EmbeddingConfig)
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.spec", err.Error())
	}
	chunker, err := search.NewMarkdownChunker(search.DefaultMarkdownChunkerConfig())
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.chunker", err.Error())
	}
	chunkIndex, err := search.NewMeiliArticleChunkIndex(new(config.MeiliSearch).MeiliSearch(), spec)
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.index", err.Error())
	}
	projector, err := task.NewArticleIndexProjector(task.ArticleIndexProjectorDeps{
		Router:  aiComponents.Router,
		Index:   chunkIndex,
		Spec:    spec,
		Chunker: chunker,
	})
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.projector", err.Error())
	}
	articleRepo := repository.NewArticleRepo(engine)
	backfill, err := task.NewArticleIndexBackfill(task.ArticleIndexBackfillDeps{
		Sources:   articleRepo,
		Projector: projector,
		PageSize:  pageSize,
	})
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.runner", err.Error())
	}
	resolvedRunID, err := resolveArticleIndexBackfillRunID(runID, spec.IndexVersion)
	if err != nil {
		return runtime, err
	}
	state := port.ArticleIndexBackfillState{
		ID:                 resolvedRunID,
		IndexUID:           spec.UID,
		IndexVersion:       spec.IndexVersion,
		Provider:           spec.Provider,
		Model:              spec.Model,
		ModelVersion:       spec.ModelVersion,
		Dimension:          spec.Dimension,
		EmbeddingBatchSize: spec.EmbeddingBatchSize,
		PageSize:           backfill.PageSize(),
		Status:             port.ArticleIndexBackfillPending,
	}
	stateRepo := repository.NewArticleIndexBackfillRepository(engine)
	persistent, err := task.NewPersistentArticleIndexBackfill(task.PersistentArticleIndexBackfillDeps{
		Backfill: backfill,
		States:   stateRepo,
		Template: state,
	})
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_backfill.persistent", err.Error())
	}
	runtime.Backfill = persistent
	runtime.Spec = spec
	return runtime, nil
}

// InitializeArticleChunksIndexProvision assembles only the explicit
// Meilisearch provisioning path. It intentionally does not initialize the
// database, start workers, or call the index API. The article-indexing flag
// remains a prerequisite so a disabled deployment cannot accidentally create
// an index from a command with missing rollout intent.
func InitializeArticleChunksIndexProvision() (ArticleChunksIndexProvisionRuntime, error) {
	var runtime ArticleChunksIndexProvisionRuntime
	aiSettings := new(config.AI).AI()
	if !aiSettings.FeatureEnabled(config.AIFeatureArticleIndexing) {
		return runtime, errors.Invalid("bootstrap.article_chunks.provision.feature", "ai.article_indexing must be enabled for index provisioning")
	}
	if err := config.ValidateMeiliSearch(); err != nil {
		return runtime, errors.Invalid("bootstrap.article_chunks.provision.config", err.Error())
	}
	spec, err := search.NewArticleChunksIndexSpec(aiSettings.Embedding, aiSettings.EmbeddingConfig)
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_chunks.provision.spec", err.Error())
	}
	meiliConfig := new(config.MeiliSearch).MeiliSearch()
	runtime.Client = meilisearch.New(meiliConfig.URL, meilisearch.WithAPIKey(meiliConfig.ApiKey))
	runtime.Spec = spec
	return runtime, nil
}

// InitializeArticleIndexBackfillControl assembles only the database-backed
// control plane. It intentionally does not require the AI feature flag and
// does not construct a model client or Meilisearch adapter.
func InitializeArticleIndexBackfillControl(runID string) (ArticleIndexBackfillControlRuntime, error) {
	var runtime ArticleIndexBackfillControlRuntime
	engine := ormInit.GetEngine()
	if engine == nil {
		return runtime, errors.Unavailable("bootstrap.article_index_backfill_control.database", nil)
	}
	settings := new(config.AI).AI()
	resolvedRunID, err := resolveArticleIndexBackfillRunID(runID, settings.EmbeddingConfig.IndexVersion)
	if err != nil {
		return runtime, err
	}
	runtime.States = repository.NewArticleIndexBackfillRepository(engine)
	runtime.RunID = resolvedRunID
	return runtime, nil
}

// InitializeArticleIndexPlan assembles only the database-read and local
// Markdown-chunking path. It does not require AI rollout flags or provider
// credentials, and it never constructs a Meilisearch client.
func InitializeArticleIndexPlan(pageSize int) (ArticleIndexPlanRuntime, error) {
	var runtime ArticleIndexPlanRuntime
	if err := config.ValidateDatabase(); err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_plan.config", err.Error())
	}
	engine := ormInit.GetEngine()
	if engine == nil {
		return runtime, errors.Unavailable("bootstrap.article_index_plan.database", nil)
	}
	settings := new(config.AI).AI()
	spec, err := search.NewArticleChunksIndexSpec(settings.Embedding, settings.EmbeddingConfig)
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_plan.spec", err.Error())
	}
	chunker, err := search.NewMarkdownChunker(search.DefaultMarkdownChunkerConfig())
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_plan.chunker", err.Error())
	}
	planner, err := task.NewArticleIndexPlan(task.ArticleIndexPlanDeps{
		Sources:  repository.NewArticleRepo(engine),
		Chunker:  chunker,
		PageSize: pageSize,
		Index:    spec.UID,
	})
	if err != nil {
		return runtime, errors.Invalid("bootstrap.article_index_plan.runner", err.Error())
	}
	runtime.Plan = planner
	runtime.Spec = spec
	return runtime, nil
}

func resolveArticleIndexBackfillRunID(runID, indexVersion string) (string, error) {
	runID = strings.TrimSpace(runID)
	if runID != "" {
		if len(runID) > 128 {
			return "", errors.Invalid("bootstrap.article_index_backfill.run_id", "backfill run ID must not exceed 128 bytes")
		}
		return runID, nil
	}
	indexVersion = strings.ToLower(strings.TrimSpace(indexVersion))
	if _, err := search.ArticleChunksIndexUID(indexVersion); err != nil {
		return "", err
	}
	return "article-index-backfill-" + indexVersion, nil
}
