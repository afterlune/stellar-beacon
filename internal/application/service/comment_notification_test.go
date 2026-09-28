package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type fakeCommentNotifier struct{ items []port.CommentNotification }

func (f *fakeCommentNotifier) EnqueueComment(item port.CommentNotification) error {
	f.items = append(f.items, item)
	return nil
}

// noticeWebsite embeds the shared fake and only overrides the site switch.
type noticeWebsite struct {
	fakeStellarBeaconInfoService
	config model.WebsiteConfigDTO
}

func (w noticeWebsite) GetWebsiteConfig(context.Context) model.ResultVO {
	return model.ResultOkWithData(w.config)
}

type notificationComments struct {
	fakeCommentRepository
	parent    entity.TComment
	parentErr error
	created   entity.TComment
}

func (f *notificationComments) GetByID(context.Context, int) (entity.TComment, error) {
	return f.parent, f.parentErr
}

func (f *notificationComments) Create(_ context.Context, comment entity.TComment) (int, error) {
	f.created = comment
	return 501, nil
}

type notificationArticles struct {
	fakeArticleRepository
	record entity.TArticle
}

func (f *notificationArticles) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.record, nil
}

type notificationUsers struct{ profile entity.TUserInfo }

func (f notificationUsers) UpdateProfile(context.Context, int, string, string, string) error {
	return nil
}
func (f notificationUsers) UpdateAvatar(context.Context, int, string) error { return nil }
func (f notificationUsers) GetByID(context.Context, int) (entity.TUserInfo, error) {
	return f.profile, nil
}
func (f notificationUsers) UpdateEmail(context.Context, int, string) error               { return nil }
func (f notificationUsers) UpdateSubscribe(context.Context, int, int) error              { return nil }
func (f notificationUsers) UpdateNotifyComment(context.Context, int, int) error          { return nil }
func (f notificationUsers) UpdateNotifyInteraction(context.Context, int, int) error      { return nil }
func (f notificationUsers) UpdateNotifyTopic(context.Context, int, int) error            { return nil }
func (f notificationUsers) UpdateNotifyCollection(context.Context, int, int) error       { return nil }
func (f notificationUsers) UpdateNotifyStudioActivation(context.Context, int, int) error { return nil }
func (f notificationUsers) UpdateRole(context.Context, int, string, []int) error         { return nil }
func (f notificationUsers) UpdateDisable(context.Context, int, int) error                { return nil }
func (f notificationUsers) IsEnabled(context.Context, int) (bool, error)                 { return true, nil }
func (f notificationUsers) FindAuthByUserInfoID(context.Context, int) (entity.TUserAuth, error) {
	return entity.TUserAuth{}, nil
}

// denyingNotifierLimiter denies every rate-limit check.
type denyingNotifierLimiter struct{}

func (denyingNotifierLimiter) Allow(context.Context, string, int64, time.Duration) (bool, error) {
	return false, nil
}

func newCommentNotificationService(t *testing.T, comments port.CommentRepository, articles port.ArticleRepository, users port.UserInfoRepository, notifier port.CommentNotifier, notice int, limiter port.RateLimiter) *MyCommentService {
	t.Helper()
	service, err := NewCommentService(CommentServiceDeps{
		Repo:          comments,
		Website:       noticeWebsite{config: model.WebsiteConfigDTO{IsCommentReview: 0, IsEmailNotice: notice}},
		Users:         users,
		Articles:      articles,
		Talks:         &fakeTalkRepository{},
		Notifications: notifier,
		Limiter:       limiter,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func postComment(t *testing.T, service *MyCommentService, body string, userInfoID int) model.ResultVO {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/comments", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if userInfoID > 0 {
		c.Set("userInfo", model.UserDetailsDTO{Id: userInfoID, UserInfoId: userInfoID, Nickname: "回复者"})
	}
	return service.SaveComment(c)
}

func TestCommentNotificationNotifiesReplyRecipient(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 7}}
	notifier := &fakeCommentNotifier{}
	articles := &notificationArticles{record: entity.TArticle{Id: 9, UserId: 3, ArticleTitle: "标题"}}
	users := notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "recipient@example.test", Nickname: "收件人", NotifyComment: 1}}
	service := newCommentNotificationService(t, comments, articles, users, notifier, 1, nil)

	result := postComment(t, service, `{"topicId":"9","commentContent":"回复内容","parentId":42,"replyUserId":7,"type":1}`, 3)
	if !result.Flag {
		t.Fatalf("comment write failed: %+v", result)
	}
	if len(notifier.items) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifier.items))
	}
	item := notifier.items[0]
	if item.Recipient != "recipient@example.test" || item.ReplyAuthor == "" {
		t.Fatalf("unexpected notification: %+v", item)
	}
	if item.ArticleID != 9 || !strings.Contains(item.ArticleURL, "/articles/9") {
		t.Fatalf("article context missing: %+v", item)
	}
}

func TestCommentNotificationSkipsSelfReply(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 5}}
	notifier := &fakeCommentNotifier{}
	service := newCommentNotificationService(t, comments, &notificationArticles{}, notificationUsers{profile: entity.TUserInfo{Id: 5, Email: "self@example.test", NotifyComment: 1}}, notifier, 1, nil)

	postComment(t, service, `{"topicId":"9","commentContent":"自回复","parentId":42,"replyUserId":5,"type":1}`, 5)
	if len(notifier.items) != 0 {
		t.Fatalf("self replies must not notify: %+v", notifier.items)
	}
}

func TestCommentNotificationRespectsRecipientOptOut(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 7}}
	notifier := &fakeCommentNotifier{}
	service := newCommentNotificationService(t, comments, &notificationArticles{}, notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "off@example.test", NotifyComment: 0}}, notifier, 1, nil)

	postComment(t, service, `{"topicId":"9","commentContent":"回复内容","parentId":42,"replyUserId":7,"type":1}`, 3)
	if len(notifier.items) != 0 {
		t.Fatalf("opted-out recipients must not be notified: %+v", notifier.items)
	}
}

func TestCommentNotificationRespectsSiteSwitch(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 7}}
	notifier := &fakeCommentNotifier{}
	service := newCommentNotificationService(t, comments, &notificationArticles{}, notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "on@example.test", NotifyComment: 1}}, notifier, 0, nil)

	postComment(t, service, `{"topicId":"9","commentContent":"回复内容","parentId":42,"replyUserId":7,"type":1}`, 3)
	if len(notifier.items) != 0 {
		t.Fatalf("site switch off must disable notifications: %+v", notifier.items)
	}
}

func TestCommentNotificationNotifiesArticleAuthor(t *testing.T) {
	comments := &notificationComments{}
	notifier := &fakeCommentNotifier{}
	articles := &notificationArticles{record: entity.TArticle{Id: 9, UserId: 4, ArticleTitle: "文章"}}
	service := newCommentNotificationService(t, comments, articles, notificationUsers{profile: entity.TUserInfo{Id: 4, Email: "author@example.test", Nickname: "作者", NotifyComment: 1}}, notifier, 1, nil)

	postComment(t, service, `{"topicId":"9","commentContent":"沙发","type":1}`, 3)
	if len(notifier.items) != 1 {
		t.Fatalf("expected the article author notification, got %+v", notifier.items)
	}
	if notifier.items[0].ReplyAuthor != "" || notifier.items[0].Recipient != "author@example.test" {
		t.Fatalf("unexpected article notification: %+v", notifier.items[0])
	}
}

func TestCommentNotificationNotifiesTalkAuthor(t *testing.T) {
	notifier := &fakeCommentNotifier{}
	talks := &fakeTalkRepository{talk: port.Talk{Id: 9, UserId: 4, Content: "一条新的随想内容"}}
	service, err := NewCommentService(CommentServiceDeps{
		Repo:          &notificationComments{},
		Website:       noticeWebsite{config: model.WebsiteConfigDTO{IsCommentReview: 0, IsEmailNotice: 1}},
		Users:         notificationUsers{profile: entity.TUserInfo{Id: 4, Email: "talk@example.test", Nickname: "说说作者", NotifyComment: 1}},
		Articles:      &notificationArticles{},
		Talks:         talks,
		Notifications: notifier,
	})
	if err != nil {
		t.Fatal(err)
	}

	postComment(t, service, `{"topicId":"9","commentContent":"说说评论","type":5}`, 3)
	if len(notifier.items) != 1 {
		t.Fatalf("expected the talk author notification, got %+v", notifier.items)
	}
	item := notifier.items[0]
	if item.Recipient != "talk@example.test" || item.ArticleID != 9 || !strings.Contains(item.ArticleURL, "/talks/9") {
		t.Fatalf("unexpected talk notification: %+v", item)
	}
}
func TestCommentNotificationIgnoresNonArticleTargets(t *testing.T) {
	comments := &notificationComments{}
	notifier := &fakeCommentNotifier{}
	service := newCommentNotificationService(t, comments, &notificationArticles{}, notificationUsers{profile: entity.TUserInfo{Id: 4, Email: "author@example.test", NotifyComment: 1}}, notifier, 1, nil)

	postComment(t, service, `{"topicId":"9","commentContent":"留言","type":2}`, 3)
	if len(notifier.items) != 0 {
		t.Fatalf("message board comments have no recipient: %+v", notifier.items)
	}
}

// Comments held for moderation must not be emailed before approval.
func TestCommentNotificationSkipsPendingModeration(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 7}}
	notifier := &fakeCommentNotifier{}
	service, err := NewCommentService(CommentServiceDeps{
		Repo:          comments,
		Website:       noticeWebsite{config: model.WebsiteConfigDTO{IsCommentReview: 1, IsEmailNotice: 1}},
		Users:         notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "pending@example.test", NotifyComment: 1}},
		Articles:      &notificationArticles{},
		Talks:         &fakeTalkRepository{},
		Notifications: notifier,
	})
	if err != nil {
		t.Fatal(err)
	}

	postComment(t, service, `{"topicId":"9","commentContent":"待审","parentId":42,"replyUserId":7,"type":1}`, 3)
	if len(notifier.items) != 0 {
		t.Fatalf("unapproved comments must not notify: %+v", notifier.items)
	}
}

type reviewedCommentRepository struct {
	fakeCommentRepository
	comment entity.TComment
	marked  []int
}

func (f *reviewedCommentRepository) GetByID(context.Context, int) (entity.TComment, error) {
	return f.comment, nil
}

func (f *reviewedCommentRepository) Review(_ context.Context, _ []int, review int) error {
	f.comment.IsReview = review
	return nil
}

func (f *reviewedCommentRepository) MarkNotificationDispatched(_ context.Context, commentID int) error {
	f.marked = append(f.marked, commentID)
	return nil
}

func TestCommentNotificationDispatchesAfterManualApproval(t *testing.T) {
	repo := &reviewedCommentRepository{comment: entity.TComment{Id: 42, UserId: 3, ReplyUserId: 7, TopicId: 9, Type: 1, IsReview: 0}}
	notifier := &fakeCommentNotifier{}
	service, err := NewCommentService(CommentServiceDeps{
		Repo:          repo,
		Website:       noticeWebsite{config: model.WebsiteConfigDTO{IsCommentReview: 1, IsEmailNotice: 1}},
		Users:         notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "reviewed@example.test", Nickname: "被回复者", NotifyComment: 1}},
		Articles:      &notificationArticles{record: entity.TArticle{Id: 9, UserId: 4, ArticleTitle: "审核文章"}},
		Talks:         &fakeTalkRepository{},
		Notifications: notifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/admin/comments/review", strings.NewReader(`{"ids":[42],"isReview":1}`))
	c.Request.Header.Set("Content-Type", "application/json")
	result := service.UpdateCommentsReview(c)
	if !result.Flag || len(notifier.items) != 1 || len(repo.marked) != 1 || repo.marked[0] != 42 {
		t.Fatalf("unexpected approval notification: result=%+v mail=%+v marked=%v", result, notifier.items, repo.marked)
	}
}
func TestCommentNotificationHonoursDailyCap(t *testing.T) {
	comments := &notificationComments{parent: entity.TComment{Id: 42, UserId: 7}}
	notifier := &fakeCommentNotifier{}
	service := newCommentNotificationService(t, comments, &notificationArticles{}, notificationUsers{profile: entity.TUserInfo{Id: 7, Email: "capped@example.test", NotifyComment: 1}}, notifier, 1, denyingNotifierLimiter{})

	postComment(t, service, `{"topicId":"9","commentContent":"回复内容","parentId":42,"replyUserId":7,"type":1}`, 3)
	if len(notifier.items) != 0 {
		t.Fatalf("rate-limited notifications must be skipped: %+v", notifier.items)
	}
}
