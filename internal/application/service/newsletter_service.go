package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

const newsletterTemplate = "template/newsletter.html"

type NewsletterService interface {
	Subscribe(*gin.Context) model.ResultVO
	Confirm(*gin.Context) model.ResultVO
	Unsubscribe(*gin.Context) model.ResultVO
	ListSubscribers(*gin.Context) model.ResultVO
	UpdateSubscriberStatus(*gin.Context) model.ResultVO
	ResendConfirmation(*gin.Context) model.ResultVO
	ListDeliveries(*gin.Context) model.ResultVO
	RetryDelivery(*gin.Context) model.ResultVO
	RetryFailed(*gin.Context) model.ResultVO
	Health(*gin.Context) model.ResultVO
	EnqueueArticle(context.Context, int) error
	Run(context.Context)
}

type NewsletterServiceDeps struct {
	Repo     port.NewsletterRepository
	Articles port.ArticleRepository
	Mailer   port.Mailer
	Limiter  port.RateLimiter
}

type MyNewsletterService struct {
	repo     port.NewsletterRepository
	articles port.ArticleRepository
	mailer   port.Mailer
	limiter  port.RateLimiter
	baseURL  string
}

func NewNewsletterService(deps NewsletterServiceDeps) (*MyNewsletterService, error) {
	if deps.Repo == nil {
		return nil, missingServiceDependency("newsletter", "repository")
	}
	if deps.Articles == nil {
		return nil, missingServiceDependency("newsletter", "article repository")
	}
	if deps.Mailer == nil {
		return nil, missingServiceDependency("newsletter", "mailer")
	}
	return &MyNewsletterService{
		repo: deps.Repo, articles: deps.Articles, mailer: deps.Mailer, limiter: deps.Limiter,
		baseURL: strings.TrimRight(config.PublicSiteURL, "/"),
	}, nil
}

func (s *MyNewsletterService) Subscribe(c *gin.Context) model.ResultVO {
	var request model.NewsletterSubscribeVO
	if err := c.ShouldBindJSON(&request); err != nil {
		return model.ResultFailWithMessage("请输入有效邮箱")
	}
	email, err := normalizeEmail(request.Email)
	if err != nil {
		return model.ResultFailWithMessage("请输入有效邮箱")
	}
	if result := s.allowSubscribe(c, email); result != nil {
		return *result
	}
	token, err := randomToken()
	if err != nil {
		return model.ResultFromError(err)
	}
	unsubToken, err := randomToken()
	if err != nil {
		return model.ResultFromError(err)
	}
	subscriber, findErr := s.repo.FindSubscriberByEmail(c.Request.Context(), email)
	if findErr != nil && !apperrors.IsKind(findErr, apperrors.KindNotFound) {
		return model.ResultFromError(findErr)
	}
	if findErr == nil && subscriber.Status == port.NewsletterActive {
		return model.ResultOkWithMessage(nil, "这个邮箱已经订阅，无需重复操作")
	}
	subscriber.Email = email
	subscriber.Status = port.NewsletterPending
	subscriber.ConfirmTokenHash = hashToken(token)
	subscriber.ConfirmTokenExpiresAt = time.Now().Add(48 * time.Hour)
	subscriber.UnsubscribeTokenHash = hashToken(unsubToken)
	if err := s.repo.SaveSubscriber(c.Request.Context(), subscriber); err != nil {
		return model.ResultFromError(err)
	}
	if err := s.sendConfirmation(c.Request.Context(), email, token); err != nil {
		// The subscription is retained so a later admin resend can recover from
		// a temporary SMTP failure without exposing provider details publicly.
		slog.ErrorContext(c.Request.Context(), "send newsletter confirmation failed", "error", err)
	}
	return model.ResultOkWithMessage(nil, "请查收确认邮件，完成订阅")
}

func (s *MyNewsletterService) Confirm(c *gin.Context) model.ResultVO {
	var request model.NewsletterTokenVO
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Token) == "" {
		return model.ResultFailWithMessage("确认链接无效或已过期")
	}
	subscriber, err := s.repo.FindSubscriberByToken(c.Request.Context(), "confirm_token_hash", hashToken(request.Token))
	if err != nil {
		return model.ResultFailWithMessage("确认链接无效或已过期")
	}
	if subscriber.Status == port.NewsletterActive {
		return model.ResultOkWithMessage(nil, "订阅已确认")
	}
	if subscriber.ConfirmTokenExpiresAt.Before(time.Now()) {
		return model.ResultFailWithMessage("确认链接无效或已过期")
	}
	if err := s.repo.ActivateSubscriber(c.Request.Context(), subscriber.Id, time.Now()); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithMessage(nil, "订阅已确认，之后会收到新文章通知")
}

func (s *MyNewsletterService) Unsubscribe(c *gin.Context) model.ResultVO {
	var request model.NewsletterTokenVO
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Token) == "" {
		return model.ResultFailWithMessage("退订链接无效")
	}
	subscriber, err := s.repo.FindSubscriberByToken(c.Request.Context(), "unsubscribe_token_hash", hashToken(request.Token))
	if err != nil {
		// Article emails carry the stored hash as a bearer token because the
		// raw unsubscribe token is intentionally never persisted.
		subscriber, err = s.repo.FindSubscriberByToken(c.Request.Context(), "unsubscribe_token_hash", strings.TrimSpace(request.Token))
		if err != nil {
			return model.ResultFailWithMessage("退订链接无效")
		}
	}
	if err := s.repo.UnsubscribeSubscriber(c.Request.Context(), subscriber.Id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithMessage(nil, "已停止接收文章通知")
}

func (s *MyNewsletterService) ListSubscribers(c *gin.Context) model.ResultVO {
	var request model.NewsletterFilterVO
	if err := c.ShouldBindQuery(&request); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, total, err := s.repo.ListSubscribers(c.Request.Context(), port.NewsletterFilter{
		Current: request.Current, Size: request.Size, Status: request.Status, Keyword: request.Keyword,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: request.Current, PageSize: request.Size})
}

func (s *MyNewsletterService) UpdateSubscriberStatus(c *gin.Context) model.ResultVO {
	var request model.NewsletterStatusVO
	if err := c.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.SetSubscriberStatus(c.Request.Context(), request.Id, request.Status); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyNewsletterService) ResendConfirmation(c *gin.Context) model.ResultVO {
	id, err := parsePositiveID(c.Param("subscriberId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	subscriber, err := s.repo.FindSubscriberByID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if subscriber.Status == port.NewsletterActive {
		return model.ResultOkWithMessage(nil, "该订阅已经确认")
	}
	token, err := randomToken()
	if err != nil {
		return model.ResultFromError(err)
	}
	subscriber.Status = port.NewsletterPending
	subscriber.ConfirmTokenHash = hashToken(token)
	subscriber.ConfirmTokenExpiresAt = time.Now().Add(48 * time.Hour)
	if err := s.repo.SaveSubscriber(c.Request.Context(), subscriber); err != nil {
		return model.ResultFromError(err)
	}
	if err := s.sendConfirmation(c.Request.Context(), subscriber.Email, token); err != nil {
		return model.ResultFailWithMessage("邮件发送失败，请检查 SMTP 配置")
	}
	return model.ResultOkWithMessage(nil, "确认邮件已重新发送")
}

func (s *MyNewsletterService) ListDeliveries(c *gin.Context) model.ResultVO {
	var request model.NewsletterDeliveryFilterVO
	if err := c.ShouldBindQuery(&request); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, total, err := s.repo.ListDeliveries(c.Request.Context(), port.DeliveryFilter{Current: request.Current, Size: request.Size, Status: request.Status})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: request.Current, PageSize: request.Size})
}

func (s *MyNewsletterService) RetryDelivery(c *gin.Context) model.ResultVO {
	var request model.NewsletterDeliveryRetryVO
	if err := c.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.RetryDelivery(c.Request.Context(), request.Id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyNewsletterService) RetryFailed(c *gin.Context) model.ResultVO {
	count, err := s.repo.RetryFailedDeliveries(c.Request.Context())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithMessage(map[string]int{"count": count}, fmt.Sprintf("已重新排队 %d 条投递", count))
}

func (s *MyNewsletterService) Health(c *gin.Context) model.ResultVO {
	stats, err := s.repo.Stats(c.Request.Context())
	if err != nil {
		return model.ResultFromError(err)
	}
	status := port.MailerHealth{Message: "邮件服务未提供健康检查"}
	if checker, ok := s.mailer.(port.MailerHealthChecker); ok {
		status = checker.Check(c.Request.Context())
	}
	metrics := dashboardGrowthDTO(stats, nil, nil, port.StudioActivationFunnel{}, time.Now(), time.Now(), "day")
	return model.ResultOkWithData(model.NewsletterHealthDTO{
		Subscribers: metrics.Subscribers,
		Deliveries:  metrics.Deliveries,
		SMTP: model.SMTPHealthDTO{
			Configured: status.Configured, Reachable: status.Reachable, Host: status.Host,
			Port: status.Port, TLS: status.TLS, Auth: status.Auth,
			CheckedAt: status.CheckedAt, Message: status.Message,
		},
		GeneratedAt: time.Now(),
	})
}

func (s *MyNewsletterService) EnqueueArticle(ctx context.Context, articleID int) error {
	if articleID <= 0 {
		return apperrors.Invalid("newsletter.enqueue", "article id must be positive")
	}
	return s.repo.CreateDeliveriesForArticle(ctx, articleID)
}

func (s *MyNewsletterService) Run(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		s.processOne(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *MyNewsletterService) processOne(ctx context.Context) {
	delivery, found, err := s.repo.ClaimNextDelivery(ctx)
	if err != nil || !found {
		if err != nil {
			slog.ErrorContext(ctx, "claim newsletter delivery failed", "error", err)
		}
		return
	}
	subscriber, err := s.repo.FindSubscriberByID(ctx, delivery.SubscriberId)
	if err != nil {
		_ = s.repo.MarkDeliveryFailed(ctx, delivery.Id, "subscriber not found", false)
		return
	}
	article, err := s.articles.GetArticleByID(ctx, delivery.ArticleId)
	if err != nil || article.Status != 1 {
		_ = s.repo.MarkDeliveryFailed(ctx, delivery.Id, "article is no longer public", false)
		return
	}
	message := port.EmailMessage{
		To: subscriber.Email, Subject: "Stellar Beacon · " + article.ArticleTitle,
		Template: templatePath(),
		CommentMap: map[string]any{
			"title": article.ArticleTitle, "description": articleDescription(article.ArticleContent),
			"actionURL":      s.baseURL + "/articles/" + fmt.Sprint(article.Id),
			"actionLabel":    "打开文章",
			"unsubscribeURL": s.baseURL + "/subscribe/unsubscribe?token=" + url.QueryEscape(subscriber.UnsubscribeTokenHash),
		},
	}
	if err := s.mailer.SendHTML(ctx, message); err != nil {
		retry := delivery.Attempts < 5
		if markErr := s.repo.MarkDeliveryFailed(ctx, delivery.Id, err.Error(), retry); markErr != nil {
			slog.ErrorContext(ctx, "mark newsletter delivery failed", "error", markErr)
		}
		return
	}
	if err := s.repo.MarkDeliverySent(ctx, delivery.Id, time.Now()); err != nil {
		slog.ErrorContext(ctx, "mark newsletter delivery sent failed", "error", err)
	}
}

func (s *MyNewsletterService) sendConfirmation(ctx context.Context, email, token string) error {
	return s.mailer.SendHTML(ctx, port.EmailMessage{
		To: email, Subject: "确认订阅 Stellar Beacon",
		Template: templatePath(),
		CommentMap: map[string]any{
			"title": "确认你的订阅", "description": "点击按钮确认接收 Stellar Beacon 的新文章通知。",
			"actionURL":      s.baseURL + "/subscribe/confirm?token=" + url.QueryEscape(token),
			"actionLabel":    "确认订阅",
			"unsubscribeURL": s.baseURL,
		},
	})
}

func (s *MyNewsletterService) allowSubscribe(c *gin.Context, email string) *model.ResultVO {
	if s.limiter == nil {
		return nil
	}
	for _, item := range []struct {
		name   string
		value  string
		limit  int64
		window time.Duration
	}{
		{name: "newsletter:subscribe:ip:", value: c.ClientIP(), limit: 20, window: 15 * time.Minute},
		{name: "newsletter:subscribe:email:", value: email, limit: 3, window: 15 * time.Minute},
	} {
		allowed, err := allowRateLimit(c.Request.Context(), s.limiter, item.name, item.value, item.limit, item.window)
		if err != nil {
			result := model.ResultFromError(err)
			return &result
		}
		if !allowed {
			result := model.ResultFailWithCodeAndMessage(42900, "请求过于频繁，请稍后再试")
			return &result
		}
	}
	return nil
}

func templatePath() string {
	path, err := config.ResourcePath(newsletterTemplate)
	if err != nil {
		return newsletterTemplate
	}
	return path
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return "", fmt.Errorf("invalid email")
	}
	return value, nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

var _ NewsletterService = (*MyNewsletterService)(nil)
