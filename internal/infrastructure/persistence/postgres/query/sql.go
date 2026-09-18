package pgsql

// SQL statements in this file contain only static query text. Every value
// that originates from a request is passed as an argument to xorm.SQL.
const (
	// article
	ListTopAndFeaturedArticles = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id WHERE a.is_delete = 0 AND a.status IN (1, 2) ORDER BY is_top DESC, is_featured DESC"
	ArticleTags                = "SELECT t.tag_name FROM t_article_tag at JOIN t_tag t ON t.id = at.tag_id WHERE at.article_id = ?"
	ListArticles               = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE is_delete = 0 AND status IN (1, 2) ORDER BY id DESC LIMIT ? OFFSET ?) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	GetArticlesByCategoryId    = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE category_id = ? AND is_delete = 0 AND status IN (1, 2) ORDER BY id DESC LIMIT ? OFFSET ?) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	// ListArticlesByIds keeps the public card projection for an explicit id set
	// (reader favourites). %s is replaced with generated placeholders; the ids
	// themselves are always bound as arguments.
	ListArticlesByIds    = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE is_delete = 0 AND status IN (1, 2) AND id IN (%s) ORDER BY id DESC) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	ListArticlesBySeries = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time, series_order FROM t_article WHERE is_delete = 0 AND status IN (1, 2) AND series_id = ?) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id ORDER BY a.series_order, a.id"
	// ListRelatedArticles ranks public articles by shared tags, then category,
	// then recency. The current article and its series are excluded so the
	// dedicated series section remains the only place for serial navigation.
	ListRelatedArticles = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id WHERE a.id <> ? AND a.is_delete = 0 AND a.status = 1 AND COALESCE(NULLIF(a.series_id, 0), -1) <> ? ORDER BY (SELECT COUNT(DISTINCT related_tag.tag_id) FROM t_article_tag current_tag JOIN t_article_tag related_tag ON related_tag.tag_id = current_tag.tag_id WHERE current_tag.article_id = ? AND related_tag.article_id = a.id) DESC, CASE WHEN a.category_id = ? THEN 1 ELSE 0 END DESC, a.create_time DESC, a.id DESC LIMIT ?"
	// ListRelatedFallback fills sparse rule matches with recent public articles
	// while preserving the same current-article and series exclusions.
	ListRelatedFallback   = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id WHERE a.id <> ? AND a.is_delete = 0 AND a.status = 1 AND COALESCE(NULLIF(a.series_id, 0), -1) <> ? ORDER BY a.create_time DESC, a.id DESC LIMIT ?"
	GetArticleById        = "SELECT a.id AS id, article_cover, article_title, article_content, article_content_html, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website, type, original_url, series_id, series_order FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, article_content_html, is_top, is_featured, is_delete, status, type, original_url, series_id, series_order, create_time, update_time FROM t_article WHERE id = ? AND is_delete = 0 AND status IN (1, 2)) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	GetPreArticleById     = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE id < ? AND is_delete = 0 AND status IN (1, 2) ORDER BY id DESC LIMIT 1 OFFSET 0) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	GetNextArticleById    = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE id > ? AND is_delete = 0 AND status IN (1, 2) ORDER BY id LIMIT 1 OFFSET 0) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	GetFirstArticle       = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE is_delete = 0 AND status IN (1, 2) ORDER BY id LIMIT 1 OFFSET 0) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	GetLastArticle        = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM (SELECT id, user_id, category_id, article_cover, article_title, article_content, is_top, is_featured, is_delete, status, create_time, update_time FROM t_article WHERE is_delete = 0 AND status IN (1, 2) ORDER BY id DESC LIMIT 1 OFFSET 0) a LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id"
	ListArticlesByTagId   = "SELECT a.id AS id, article_cover, article_title, SUBSTR(article_content, 1, 500) AS article_content, is_top, is_featured,c.category_name AS category_Name, status, a.create_time, a.update_time, u.nickname AS nickname, u.avatar AS avatar, u.website AS website FROM t_article a LEFT JOIN t_article_tag at ON a.id = at.article_id LEFT JOIN t_category c ON a.category_id = c.id LEFT JOIN t_user_info u ON a.user_id = u.id WHERE at.tag_id = ? AND a.is_delete = 0 AND status IN (1, 2) LIMIT ? OFFSET ?"
	ListArchives          = "SELECT id, article_title, SUBSTR(article_content, 1, 500) AS article_content, create_time FROM t_article WHERE is_delete = 0 AND status = 1 ORDER BY create_time DESC LIMIT ? OFFSET ?"
	ArticlesAdminTags     = "SELECT t.id, t.tag_name FROM t_article_tag at JOIN t_tag t ON t.id = at.tag_id WHERE at.article_id = ?"
	ListArticleStatistics = "SELECT to_char(create_time, 'YYYY-MM-DD') AS date, COUNT(1) AS count FROM t_article GROUP BY date ORDER BY date DESC"

	// category
	ListCategories = "SELECT c.id, c.category_name, COUNT(a.id) AS article_count FROM t_category c LEFT JOIN (SELECT * FROM t_article WHERE is_delete = 0 AND status IN (1, 2)) a ON c.id = a.category_id GROUP BY c.id"

	// comment
	ListTopSixComments = "SELECT c.id, c.user_id, u.nickname, u.avatar, u.website, c.comment_content, c.create_time FROM t_comment c JOIN t_user_info u ON c.user_id = u.id WHERE c.is_review = 1 ORDER BY c.id DESC LIMIT 6 OFFSET 0"

	// job log
	ListJobLogGroups = "SELECT DISTINCT job_group FROM t_job_log"

	// job
	ListJobGroups = "SELECT DISTINCT job_group FROM t_job"

	// menu
	ListMenusByUserInfoId = "SELECT DISTINCT m.id, name, path, component, icon, order_num, parent_id, is_hidden FROM t_user_role ur JOIN t_role_menu rm ON ur.role_id = rm.role_id JOIN t_menu m ON rm.menu_id = m.id WHERE user_id = ?"

	// role
	ListResourceRoles     = "SELECT re.id AS id, url, request_method FROM t_resource re WHERE re.parent_id IS NOT NULL AND is_anonymous = 0 ORDER BY id ASC"
	ResourceRoles         = "SELECT role_name FROM t_resource re LEFT JOIN t_role_resource rr ON re.id = rr.resource_id LEFT JOIN t_role ro ON rr.role_id = ro.id WHERE re.parent_id IS NOT NULL AND is_anonymous = 0 AND re.id = ?"
	ListRolesByUserInfoId = "SELECT role_name FROM t_role r LEFT JOIN t_user_role ur ON r.id = ur.role_id WHERE ur.user_id = ?"

	// tag
	ListTags       = "SELECT t.id, tag_name, COUNT(aat.article_id) AS count FROM t_tag t LEFT JOIN (SELECT a.id AS article_id, at.tag_id AS tag_id FROM t_article_tag at LEFT JOIN t_article a ON at.article_id = a.id WHERE a.is_delete = 0 AND a.status IN (1, 2)) aat ON t.id = aat.tag_id GROUP BY t.id"
	ListTopTenTags = "SELECT t.id, tag_name, COUNT(aat.article_id) AS count FROM t_tag t LEFT JOIN (SELECT a.id AS article_id, at.tag_id AS tag_id FROM t_article_tag at LEFT JOIN t_article a ON at.article_id = a.id WHERE a.is_delete = 0 AND a.status IN (1, 2)) aat ON t.id = aat.tag_id GROUP BY t.id LIMIT 10 OFFSET 0"

	// talk
	ListTalks        = "SELECT t.id, nickname, avatar, content, images, t.is_top, t.create_time FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id WHERE t.status = 1 ORDER BY t.is_top DESC, t.id DESC LIMIT ? OFFSET ?"
	GetTalkById      = "SELECT t.id, nickname, avatar, content, images, t.create_time FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id WHERE t.id = ? AND t.status = 1"
	GetTalkByIdAdmin = "SELECT t.id, nickname, avatar, content, images, t.is_top, t.status, t.create_time FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id WHERE t.id = ?"

	// unique view
	ListUniqueViews = "SELECT to_char(create_time, 'YYYY-MM-DD') AS day, views_count FROM t_unique_view WHERE create_time > ? AND create_time <= ? ORDER BY create_time"
)
