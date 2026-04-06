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