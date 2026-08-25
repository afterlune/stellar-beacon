package service

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/repository"
)

var (
	benetnaschService BenetnaschInfoService     = new(MyBenetnaschInfoService)
	articleRepo       port.ArticleRepository    = new(repository.MyArticleRepo)
	categoryRepo      repository.CategoryRepo   = new(repository.MyCategoryRepo)
	commentRepo       repository.CommentRepo    = new(repository.MyCommentRepo)
	jobRepo           repository.JobRepo        = new(repository.MyJobRepo)
	photoAlbumRepo    repository.PhotoAlbumRepo = new(repository.MyPhotoAlbumRepo)
	roleRepo          repository.RoleRepo       = new(repository.MyRoleRepo)
	tagRepo           repository.TagRepo        = new(repository.MyTagRepo)
	talkRepo          repository.TalkRepo       = new(repository.MyTalkRepo)
	userAuthRepo      repository.UserAuthRepo   = new(repository.MyUserAuthRepo)
)
