package api

import "github.com/eternallyzzz/stellar-beacon/internal/application/service"

var (
	articleService            service.ArticleService                = new(service.MyArticleService)
	platformService           service.PlatformService               = new(service.MyPlatformService)
	articleReactionService    service.ArticleReactionService        = new(service.MyArticleReactionService)
	seriesService             service.SeriesService                 = new(service.MySeriesService)
	stellarBeaconService      service.StellarBeaconInfoService      = new(service.MyStellarBeaconInfoService)
	categoryService           service.CategoryService               = new(service.MyCategoryService)
	commentService            service.CommentService                = new(service.MyCommentService)
	errorLogService           service.ErrorLogService               = new(service.MyErrorLogService)
	friendLinkService         service.FriendLinkService             = new(service.MyFriendLinkService)
	jobLogService             service.JobLogService                 = new(service.MyJobLogService)
	jobService                service.JobService                    = new(service.MyJobService)
	menuService               service.MenuService                   = new(service.MyMenuSService)
	mediaService              service.MediaService                  = new(service.MyMediaService)
	operationLogService       service.OperationLogService           = new(service.MyOperationLogService)
	photoAlbumService         service.PhotoAlbumService             = new(service.MyPhotoAlbumService)
	photoService              service.PhotoService                  = new(service.MyPhotoService)
	resourceService           service.ResourceService               = new(service.MyResourceService)
	roleService               service.RoleService                   = new(service.MyRoleService)
	tagService                service.TagService                    = new(service.MyTagService)
	talkService               service.TalkService                   = new(service.MyTalkService)
	userAuthService           service.UserAuthService               = new(service.MyUserAuthService)
	userInfoService           service.UserInfoService               = new(service.MyUserInfoService)
	seoService                service.SeoService                    = new(service.MySeoService)
	newsletterService         service.NewsletterService             = new(service.MyNewsletterService)
	growthService             service.GrowthService                 = new(service.MyGrowthService)
	contentAnalyticsService   service.ContentAnalyticsService       = new(service.MyContentAnalyticsService)
	contentAuditService       service.ContentAuditService           = new(service.MyContentAuditService)
	followService             service.FollowService                 = new(service.MyFollowService)
	topicSubscriptionService  service.TopicSubscriptionService      = new(service.MyTopicSubscriptionService)
	recommendationService     service.RecommendationService         = new(service.MyRecommendationService)
	collectionService         service.CollectionService             = new(service.MyCollectionService)
	collectionSubService      service.CollectionSubscriptionService = new(service.MyCollectionSubscriptionService)
	collectionReactionService service.CollectionReactionService     = new(service.MyCollectionReactionService)
)

type Services struct {
	Article            service.ArticleService
	Platform           service.PlatformService
	ArticleReaction    service.ArticleReactionService
	Series             service.SeriesService
	StellarBeacon      service.StellarBeaconInfoService
	Category           service.CategoryService
	Comment            service.CommentService
	ErrorLog           service.ErrorLogService
	FriendLink         service.FriendLinkService
	JobLog             service.JobLogService
	Job                service.JobService
	Menu               service.MenuService
	Media              service.MediaService
	OperationLog       service.OperationLogService
	PhotoAlbum         service.PhotoAlbumService
	Photo              service.PhotoService
	Resource           service.ResourceService
	Role               service.RoleService
	Tag                service.TagService
	Talk               service.TalkService
	UserAuth           service.UserAuthService
	UserInfo           service.UserInfoService
	Seo                service.SeoService
	Newsletter         service.NewsletterService
	Growth             service.GrowthService
	ContentAnalytics   service.ContentAnalyticsService
	ContentAudit       service.ContentAuditService
	Follow             service.FollowService
	TopicSubscription  service.TopicSubscriptionService
	Recommendation     service.RecommendationService
	Collection         service.CollectionService
	CollectionSub      service.CollectionSubscriptionService
	CollectionReaction service.CollectionReactionService
}

func ConfigureServices(s Services) {
	articleService = s.Article
	platformService = s.Platform
	articleReactionService = s.ArticleReaction
	seriesService = s.Series
	stellarBeaconService = s.StellarBeacon
	categoryService = s.Category
	commentService = s.Comment
	errorLogService = s.ErrorLog
	friendLinkService = s.FriendLink
	jobLogService = s.JobLog
	jobService = s.Job
	menuService = s.Menu
	mediaService = s.Media
	operationLogService = s.OperationLog
	photoAlbumService = s.PhotoAlbum
	photoService = s.Photo
	resourceService = s.Resource
	roleService = s.Role
	tagService = s.Tag
	talkService = s.Talk
	userAuthService = s.UserAuth
	userInfoService = s.UserInfo
	seoService = s.Seo
	newsletterService = s.Newsletter
	growthService = s.Growth
	contentAnalyticsService = s.ContentAnalytics
	contentAuditService = s.ContentAudit
	followService = s.Follow
	topicSubscriptionService = s.TopicSubscription
	recommendationService = s.Recommendation
	collectionService = s.Collection
	collectionSubService = s.CollectionSub
	collectionReactionService = s.CollectionReaction
}
