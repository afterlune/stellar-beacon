package api

import "github.com/eternallyzzz/stellar-beacon/internal/application/service"

var (
	articleService       service.ArticleService           = new(service.MyArticleService)
	stellarBeaconService service.StellarBeaconInfoService = new(service.MyStellarBeaconInfoService)
	categoryService      service.CategoryService          = new(service.MyCategoryService)
	commentService       service.CommentService           = new(service.MyCommentService)
	errorLogService      service.ErrorLogService          = new(service.MyErrorLogService)
	friendLinkService    service.FriendLinkService        = new(service.MyFriendLinkService)
	jobLogService        service.JobLogService            = new(service.MyJobLogService)
	jobService           service.JobService               = new(service.MyJobService)
	menuService          service.MenuService              = new(service.MyMenuSService)
	mediaService         service.MediaService             = new(service.MyMediaService)
	operationLogService  service.OperationLogService      = new(service.MyOperationLogService)
	photoAlbumService    service.PhotoAlbumService        = new(service.MyPhotoAlbumService)
	photoService         service.PhotoService             = new(service.MyPhotoService)
	resourceService      service.ResourceService          = new(service.MyResourceService)
	roleService          service.RoleService              = new(service.MyRoleService)
	tagService           service.TagService               = new(service.MyTagService)
	talkService          service.TalkService              = new(service.MyTalkService)
	userAuthService      service.UserAuthService          = new(service.MyUserAuthService)
	userInfoService      service.UserInfoService          = new(service.MyUserInfoService)
)

type Services struct {
	Article       service.ArticleService
	StellarBeacon service.StellarBeaconInfoService
	Category      service.CategoryService
	Comment       service.CommentService
	ErrorLog      service.ErrorLogService
	FriendLink    service.FriendLinkService
	JobLog        service.JobLogService
	Job           service.JobService
	Menu          service.MenuService
	Media         service.MediaService
	OperationLog  service.OperationLogService
	PhotoAlbum    service.PhotoAlbumService
	Photo         service.PhotoService
	Resource      service.ResourceService
	Role          service.RoleService
	Tag           service.TagService
	Talk          service.TalkService
	UserAuth      service.UserAuthService
	UserInfo      service.UserInfoService
}

func ConfigureServices(s Services) {
	articleService = s.Article
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
}
