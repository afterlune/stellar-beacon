package api

import "benetnasch/app/application/service"

var (
	articleService      service.ArticleService        = new(service.MyArticleService)
	benetnaschService   service.BenetnaschInfoService = new(service.MyBenetnaschInfoService)
	categoryService     service.CategoryService       = new(service.MyCategoryService)
	commentService      service.CommentService        = new(service.MyCommentService)
	errorLogService     service.ErrorLogService       = new(service.MyErrorLogService)
	friendLinkService   service.FriendLinkService     = new(service.MyFriendLinkService)
	jobLogService       service.JobLogService         = new(service.MyJobLogService)
	jobService          service.JobService            = new(service.MyJobService)
	menuService         service.MenuService           = new(service.MyMenuSService)
	operationLogService service.OperationLogService   = new(service.MyOperationLogService)
	photoAlbumService   service.PhotoAlbumService     = new(service.MyPhotoAlbumService)
	photoService        service.PhotoService          = new(service.MyPhotoService)
	resourceService     service.ResourceService       = new(service.MyResourceService)
	roleService         service.RoleService           = new(service.MyRoleService)
	tagService          service.TagService            = new(service.MyTagService)
	talkService         service.TalkService           = new(service.MyTalkService)
	userAuthService     service.UserAuthService       = new(service.MyUserAuthService)
	userInfoService     service.UserInfoService       = new(service.MyUserInfoService)
)

type Services struct {
	Article      service.ArticleService
	Benetnasch   service.BenetnaschInfoService
	Category     service.CategoryService
	Comment      service.CommentService
	ErrorLog     service.ErrorLogService
	FriendLink   service.FriendLinkService
	JobLog       service.JobLogService
	Job          service.JobService
	Menu         service.MenuService
	OperationLog service.OperationLogService
	PhotoAlbum   service.PhotoAlbumService
	Photo        service.PhotoService
	Resource     service.ResourceService
	Role         service.RoleService
	Tag          service.TagService
	Talk         service.TalkService
	UserAuth     service.UserAuthService
	UserInfo     service.UserInfoService
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
}
