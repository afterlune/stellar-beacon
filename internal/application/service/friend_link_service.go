package service

import (
	"container/list"
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FriendLinkService interface {
	ListFriendLinks() model.ResultVO
	ListFriendLinkDTO(c *gin.Context) model.ResultVO
	SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO
	DeleteFriendLink(c *gin.Context) model.ResultVO
	ApplyFriendLink(c *gin.Context) model.ResultVO
	ReviewFriendLinks(c *gin.Context) model.ResultVO
}

// friendLinkLimiter is injected by the composition root; a nil limiter simply
// disables application throttling (the shared helper treats nil as allowed).
var friendLinkLimiter port.RateLimiter

// friendLinkApplyDailyLimit caps reader submissions per origin per day. It is
// deliberately generous: shared NATs (offices, campuses) must still be able to
// apply, while a scripted flood is stopped.
const friendLinkApplyDailyLimit = 20

// friendLinkVisitor resolves the applicant identity for throttling. It is a
// domain port so this package keeps depending on ports only.
var friendLinkVisitor port.VisitorResolver

func ConfigureFriendLinkLimiter(limiter port.RateLimiter) {
	friendLinkLimiter = limiter
}

func ConfigureFriendLinkVisitor(resolver port.VisitorResolver) {
	friendLinkVisitor = resolver
}

type MyFriendLinkService struct{ repo port.FriendLinkRepository }

func NewFriendLinkService(repo port.FriendLinkRepository) *MyFriendLinkService {
	return &MyFriendLinkService{repo: repo}
}

func (f *MyFriendLinkService) friendLinkRepository() port.FriendLinkRepository {
	if f.repo != nil {
		return f.repo
	}
	return friendLinkRepo
}

func (f *MyFriendLinkService) ListFriendLinks() model.ResultVO {
	links, err := f.friendLinkRepository().ListPublic(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.FriendLinkDTO
	StructCopy(links, &dtos)
	return model.ResultOkWithData(dtos)
}

func (f *MyFriendLinkService) ListFriendLinkDTO(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	links, count, err := f.friendLinkRepository().ListAdmin(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.FriendLinkAdminDTO
	StructCopy(links, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (f *MyFriendLinkService) SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO {
	var vo model.FriendLinkVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	link := entity.TFriendLink{Id: vo.Id, LinkName: vo.LinkName, LinkAvatar: vo.LinkAvatar, LinkAddress: vo.LinkAddress, LinkIntro: vo.LinkIntro}
	if err := f.friendLinkRepository().SaveOrUpdate(c.Request.Context(), link); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (f *MyFriendLinkService) DeleteFriendLink(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := f.friendLinkRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

// ApplyFriendLink accepts a reader submission. The response never reveals
// whether the address was already known, so the endpoint cannot be used to
// probe the link list.
func (f *MyFriendLinkService) ApplyFriendLink(c *gin.Context) model.ResultVO {
	var vo model.FriendLinkApplyVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	// A filled honeypot means a bot: accept silently and store nothing.
	if strings.TrimSpace(vo.Honeypot) != "" {
		return model.ResultOk()
	}
	name := strings.TrimSpace(vo.LinkName)
	address := strings.TrimSpace(vo.LinkAddress)
	avatar := strings.TrimSpace(vo.LinkAvatar)
	intro := strings.TrimSpace(vo.LinkIntro)
	email := strings.ToLower(strings.TrimSpace(vo.Email))
	if name == "" || len([]rune(name)) > 20 {
		return model.ResultFromError(apperrors.Invalid("friend_link.apply", "link name is required"))
	}
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return model.ResultFromError(apperrors.Invalid("friend_link.apply", "link address must be an absolute URL"))
	}
	if len(address) > 255 || len(avatar) > 255 || len([]rune(intro)) > 100 {
		return model.ResultFromError(apperrors.Invalid("friend_link.apply", "submission is too long"))
	}
	if email != "" {
		if parsed, err := mail.ParseAddress(email); err != nil || parsed.Address != email {
			return model.ResultFromError(apperrors.Invalid("friend_link.apply", "email is invalid"))
		}
	}
	applicant := ""
	if friendLinkVisitor != nil {
		if identity, identityErr := friendLinkVisitor.Resolve(c.Request.Context(), c.Request); identityErr == nil {
			applicant = identity.IP
		}
	}
	allowed, err := allowRateLimit(c.Request.Context(), friendLinkLimiter, "friend-link-apply:", applicant, friendLinkApplyDailyLimit, 24*time.Hour)
	if err != nil {
		return model.ResultFromError(apperrors.Unavailable("friend_link.apply", err))
	}
	if !allowed {
		return model.ResultFromError(apperrors.Invalid("friend_link.apply", "too many applications"))
	}
	exists, err := f.friendLinkRepository().AddressExists(c.Request.Context(), address)
	if err != nil {
		return model.ResultFromError(err)
	}
	if exists {
		return model.ResultOk()
	}
	if _, err := f.friendLinkRepository().CreateApplication(c.Request.Context(), entity.TFriendLink{
		LinkName: name, LinkAvatar: avatar, LinkAddress: address, LinkIntro: intro, ApplicantEmail: email,
	}); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (f *MyFriendLinkService) ReviewFriendLinks(c *gin.Context) model.ResultVO {
	var vo model.FriendLinkReviewVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if len(vo.Ids) == 0 {
		return model.ResultFromError(apperrors.Invalid("friend_link.review", "ids are required"))
	}
	if vo.Status != port.FriendLinkStatusApproved && vo.Status != port.FriendLinkStatusRejected {
		return model.ResultFromError(apperrors.Invalid("friend_link.review", "unsupported review status"))
	}
	if err := f.friendLinkRepository().Review(c.Request.Context(), vo.Ids, vo.Status); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
