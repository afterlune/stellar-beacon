package api

import "benetnasch/app/application/service"
import "benetnasch/app/domain/port"

var (
	articleService           service.ArticleService                  = new(service.MyArticleService)
	benetnaschService        service.BenetnaschInfoService           = new(service.MyBenetnaschInfoService)
	categoryService          service.CategoryService                 = new(service.MyCategoryService)
	commentService           service.CommentService                  = new(service.MyCommentService)
	errorLogService          service.ErrorLogService                 = new(service.MyErrorLogService)
	friendLinkService        service.FriendLinkService               = new(service.MyFriendLinkService)
	jobLogService            service.JobLogService                   = new(service.MyJobLogService)
	jobService               service.JobService                      = new(service.MyJobService)
	menuService              service.MenuService                     = new(service.MyMenuSService)
	operationLogService      service.OperationLogService             = new(service.MyOperationLogService)
	photoAlbumService        service.PhotoAlbumService               = new(service.MyPhotoAlbumService)
	photoService             service.PhotoService                    = new(service.MyPhotoService)
	resourceService          service.ResourceService                 = new(service.MyResourceService)
	roleService              service.RoleService                     = new(service.MyRoleService)
	tagService               service.TagService                      = new(service.MyTagService)
	talkService              service.TalkService                     = new(service.MyTalkService)
	userAuthService          service.UserAuthService                 = new(service.MyUserAuthService)
	userInfoService          service.UserInfoService                 = new(service.MyUserInfoService)
	aiStudioService          service.AIStudioService                 = new(service.MyAIStudioService)
	aiVisionService          service.AIVisionService                 = service.NewDisabledAIVisionService()
	aiProviderProbeService   service.AIProviderProbeService          = service.NewDisabledAIProviderProbeService()
	operationalObservability service.OperationalObservabilityService = service.NewDisabledOperationalObservabilityService()
	agentProfileService      service.AgentProfileService             = new(service.MyAgentProfileService)
	agentReviewPolicyService service.AgentReviewPolicyService        = new(service.MyAgentReviewPolicyService)
	agentMemoryService       service.AgentMemoryService              = service.NewDisabledAgentMemoryService()
	agentChatService         service.AgentChatService                = new(service.MyAgentChatService)
	contentGalaxyService     service.ContentGalaxyService            = service.NewDisabledContentGalaxyService()
	dreamService             service.DreamService                    = service.NewDisabledDreamService()
	capsuleService           service.TimeCapsuleService              = service.NewDisabledTimeCapsuleService()
	radioService             service.RadioService                    = service.NewDisabledRadioService()
	videoService             service.VideoService                    = service.NewDisabledVideoService()
	spaceCompanionService    service.SpaceCompanionService           = service.NewDisabledSpaceCompanionService()
	agentVitalsProvider      port.AgentVitalsProvider
	agentFeatureFlags        port.AgentFeatureFlags
	agentEventStore          port.AgentEventStore
	agentSafetySwitch        port.AgentSafetySwitch
)

type Services struct {
	Article                  service.ArticleService
	Benetnasch               service.BenetnaschInfoService
	Category                 service.CategoryService
	Comment                  service.CommentService
	ErrorLog                 service.ErrorLogService
	FriendLink               service.FriendLinkService
	JobLog                   service.JobLogService
	Job                      service.JobService
	Menu                     service.MenuService
	OperationLog             service.OperationLogService
	PhotoAlbum               service.PhotoAlbumService
	Photo                    service.PhotoService
	Resource                 service.ResourceService
	Role                     service.RoleService
	Tag                      service.TagService
	Talk                     service.TalkService
	UserAuth                 service.UserAuthService
	UserInfo                 service.UserInfoService
	AIStudio                 service.AIStudioService
	AIVision                 service.AIVisionService
	AIProviderProbe          service.AIProviderProbeService
	OperationalObservability service.OperationalObservabilityService
	AgentProfile             service.AgentProfileService
	AgentReviewPolicy        service.AgentReviewPolicyService
	AgentMemory              service.AgentMemoryService
	AgentChat                service.AgentChatService
	AgentGalaxy              service.ContentGalaxyService
	AgentDreams              service.DreamService
	AgentCapsules            service.TimeCapsuleService
	AgentRadio               service.RadioService
	AgentVideos              service.VideoService
	AgentVitals              port.AgentVitalsProvider
	AgentFeatures            port.AgentFeatureFlags
	AgentEvents              port.AgentEventStore
	AgentSafety              port.AgentSafetySwitch
	SpaceCompanion           service.SpaceCompanionService
}

func ConfigureServices(s Services) {
	articleService = s.Article
	benetnaschService = s.Benetnasch
	categoryService = s.Category
	commentService = s.Comment
	errorLogService = s.ErrorLog
	friendLinkService = s.FriendLink
	jobLogService = s.JobLog
	jobService = s.Job
	menuService = s.Menu
	operationLogService = s.OperationLog
	photoAlbumService = s.PhotoAlbum
	photoService = s.Photo
	resourceService = s.Resource
	roleService = s.Role
	tagService = s.Tag
	talkService = s.Talk
	userAuthService = s.UserAuth
	userInfoService = s.UserInfo
	if s.AIStudio != nil {
		aiStudioService = s.AIStudio
	}
	if s.AIVision != nil {
		aiVisionService = s.AIVision
	}
	if s.AIProviderProbe != nil {
		aiProviderProbeService = s.AIProviderProbe
	}
	if s.OperationalObservability != nil {
		operationalObservability = s.OperationalObservability
	}
	if s.AgentProfile != nil {
		agentProfileService = s.AgentProfile
	}
	if s.AgentReviewPolicy != nil {
		agentReviewPolicyService = s.AgentReviewPolicy
	}
	if s.AgentMemory != nil {
		agentMemoryService = s.AgentMemory
	}
	if s.AgentChat != nil {
		agentChatService = s.AgentChat
	}
	if s.AgentGalaxy != nil {
		contentGalaxyService = s.AgentGalaxy
	}
	if s.AgentDreams != nil {
		dreamService = s.AgentDreams
	}
	if s.AgentCapsules != nil {
		capsuleService = s.AgentCapsules
	}
	if s.AgentRadio != nil {
		radioService = s.AgentRadio
	}
	if s.AgentVideos != nil {
		videoService = s.AgentVideos
	}
	if s.SpaceCompanion != nil {
		spaceCompanionService = s.SpaceCompanion
	}
	agentVitalsProvider = s.AgentVitals
	agentFeatureFlags = s.AgentFeatures
	agentEventStore = s.AgentEvents
	agentSafetySwitch = s.AgentSafety
}
