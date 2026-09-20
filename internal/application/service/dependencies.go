package service

import (
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

// ArticleServiceDeps contains every dependency required by the article use
// cases. Keeping the fields named prevents accidental argument reordering at
// the composition root.
type ArticleServiceDeps struct {
	Repo             port.ArticleRepository
	Reactions        port.ArticleReactionRepository
	ContentAnalytics port.ContentAnalyticsRepository
	Cache            port.Cache
	Storage          port.ObjectStorage
	Search           port.ArticleSearcher
	Newsletter       port.NewsletterEnqueuer
}

// StellarBeaconInfoServiceDeps contains the site-information use-case ports.
type StellarBeaconInfoServiceDeps struct {
	Site       port.SiteInfoRepository
	Articles   port.ArticleRepository
	Categories port.CategoryRepository
	Tags       port.TagRepository
	Cache      port.Cache
	Visitor    port.VisitorResolver
	Newsletter port.NewsletterRepository
	Growth     port.GrowthRepository
}

// ContentAnalyticsServiceDeps contains the aggregate store plus the
// collaborators needed to validate articles and deduplicate anonymous
// readers.
type ContentAnalyticsServiceDeps struct {
	Repo     port.ContentAnalyticsRepository
	Articles port.ArticleRepository
	Studio   port.StudioOperationsRepository
	Cache    port.Cache
	Visitor  port.VisitorResolver
	Limiter  port.RateLimiter
}

// UserInfoServiceDeps contains the user-profile use-case ports.
type UserInfoServiceDeps struct {
	Repo    port.UserInfoRepository
	Cache   port.Cache
	Storage port.ObjectStorage
}

// CommentServiceDeps contains the comment repository, site-information
// collaborator and the notification collaborators used after a comment lands.
type CommentServiceDeps struct {
	Repo          port.CommentRepository
	Website       StellarBeaconInfoService
	Users         port.UserInfoRepository
	Articles      port.ArticleRepository
	Notifications port.CommentNotifier
	Limiter       port.RateLimiter
}

// SeriesServiceDeps contains the collection repository plus the article
// repository used to render one collection's ordered articles.
type SeriesServiceDeps struct {
	Repo     port.SeriesRepository
	Articles port.ArticleRepository
}

func (d SeriesServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("series", "repository")
	}
	if d.Articles == nil {
		return missingServiceDependency("series", "article repository")
	}
	return nil
}

// ArticleReactionServiceDeps contains the reader-interaction ports plus the
// article repository used to validate targets and render favourites.
type ArticleReactionServiceDeps struct {
	Repo     port.ArticleReactionRepository
	Articles port.ArticleRepository
	Limiter  port.RateLimiter
}

// PhotoAlbumServiceDeps contains the album repository, photo repository and
// object storage port.
type PhotoAlbumServiceDeps struct {
	Repo    port.PhotoAlbumRepository
	Photos  port.PhotoRepository
	Storage port.ObjectStorage
}

// PhotoServiceDeps contains the photo repository, album repository and object
// storage port.
type PhotoServiceDeps struct {
	Repo    port.PhotoRepository
	Albums  port.PhotoAlbumRepository
	Storage port.ObjectStorage
}

// TalkServiceDeps contains the talk repository, comment repository and object
// storage port.
type TalkServiceDeps struct {
	Repo     port.TalkRepository
	Comments port.CommentRepository
	Storage  port.ObjectStorage
}

// UserAuthServiceDeps contains all collaborators used by authentication and
// account-protection flows.
type UserAuthServiceDeps struct {
	Repo    port.AuthRepository
	Website StellarBeaconInfoService
	Cache   port.Cache
	Mailer  port.Mailer
	Visitor port.VisitorResolver
}

func missingServiceDependency(serviceName, dependency string) error {
	return apperrors.Invalid("service."+serviceName+".dependencies", dependency+" is required")
}

func (d ArticleServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("article", "repository")
	}
	if d.Cache == nil {
		return missingServiceDependency("article", "cache")
	}
	if d.Storage == nil {
		return missingServiceDependency("article", "storage")
	}
	if d.Search == nil {
		return missingServiceDependency("article", "search")
	}
	if d.Reactions == nil {
		return missingServiceDependency("article", "reaction repository")
	}
	if d.ContentAnalytics == nil {
		return missingServiceDependency("article", "content analytics repository")
	}
	return nil
}

func (d StellarBeaconInfoServiceDeps) validate() error {
	if d.Site == nil {
		return missingServiceDependency("stellar_beacon_info", "site repository")
	}
	if d.Articles == nil {
		return missingServiceDependency("stellar_beacon_info", "article repository")
	}
	if d.Categories == nil {
		return missingServiceDependency("stellar_beacon_info", "category repository")
	}
	if d.Tags == nil {
		return missingServiceDependency("stellar_beacon_info", "tag repository")
	}
	if d.Cache == nil {
		return missingServiceDependency("stellar_beacon_info", "cache")
	}
	if d.Visitor == nil {
		return missingServiceDependency("stellar_beacon_info", "visitor")
	}
	return nil
}

func (d ContentAnalyticsServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("content_analytics", "repository")
	}
	if d.Articles == nil {
		return missingServiceDependency("content_analytics", "article repository")
	}
	if d.Cache == nil {
		return missingServiceDependency("content_analytics", "cache")
	}
	if d.Visitor == nil {
		return missingServiceDependency("content_analytics", "visitor resolver")
	}
	return nil
}

func (d UserInfoServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("user_info", "repository")
	}
	if d.Cache == nil {
		return missingServiceDependency("user_info", "cache")
	}
	if d.Storage == nil {
		return missingServiceDependency("user_info", "storage")
	}
	return nil
}

func (d CommentServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("comment", "repository")
	}
	if d.Website == nil {
		return missingServiceDependency("comment", "website")
	}
	if d.Users == nil {
		return missingServiceDependency("comment", "user info repository")
	}
	if d.Articles == nil {
		return missingServiceDependency("comment", "article repository")
	}
	if d.Notifications == nil {
		return missingServiceDependency("comment", "comment notifier")
	}
	return nil
}

func (d ArticleReactionServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("article_reaction", "repository")
	}
	if d.Articles == nil {
		return missingServiceDependency("article_reaction", "article repository")
	}
	return nil
}

func (d PhotoAlbumServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("photo_album", "repository")
	}
	if d.Photos == nil {
		return missingServiceDependency("photo_album", "photo repository")
	}
	if d.Storage == nil {
		return missingServiceDependency("photo_album", "storage")
	}
	return nil
}

func (d PhotoServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("photo", "repository")
	}
	if d.Albums == nil {
		return missingServiceDependency("photo", "album repository")
	}
	if d.Storage == nil {
		return missingServiceDependency("photo", "storage")
	}
	return nil
}

func (d TalkServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("talk", "repository")
	}
	if d.Comments == nil {
		return missingServiceDependency("talk", "comment repository")
	}
	if d.Storage == nil {
		return missingServiceDependency("talk", "storage")
	}
	return nil
}

func (d UserAuthServiceDeps) validate() error {
	if d.Repo == nil {
		return missingServiceDependency("user_auth", "repository")
	}
	if d.Website == nil {
		return missingServiceDependency("user_auth", "website")
	}
	if d.Cache == nil {
		return missingServiceDependency("user_auth", "cache")
	}
	if d.Mailer == nil {
		return missingServiceDependency("user_auth", "mailer")
	}
	if d.Visitor == nil {
		return missingServiceDependency("user_auth", "visitor")
	}
	return nil
}
