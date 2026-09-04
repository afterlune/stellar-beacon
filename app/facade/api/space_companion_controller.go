package api

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type spaceSearchRequest struct {
	Query string   `json:"query"`
	Types []string `json:"types"`
	Limit int      `json:"limit"`
}

type spacePublicationRequest struct {
	Type            string `json:"type"`
	Title           string `json:"title"`
	Body            string `json:"body"`
	MediaURL        string `json:"mediaUrl"`
	SourceSessionID string `json:"sourceSessionId"`
	SourceRunID     string `json:"sourceRunId"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

func GetSpaceCapabilities(c *gin.Context) {
	if spaceCompanionService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, spaceCompanionService.Capabilities())
}

func SearchSpaceContent(c *gin.Context) {
	if spaceCompanionService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	var request spaceSearchRequest
	if err := bindSpaceJSON(c, &request); err != nil {
		writeSpaceError(c, errors.Invalid("space.search.request", "request body is invalid"))
		return
	}
	result, err := spaceCompanionService.Search(c.Request.Context(), port.SpaceSearchQuery{
		Query: request.Query,
		Types: request.Types,
		Limit: request.Limit,
	})
	if err != nil {
		writeSpaceError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}

func GetSpaceContent(c *gin.Context) {
	if spaceCompanionService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	contentType, err := port.NormalizeSpaceContentType(c.Param("type"))
	if err != nil {
		writeSpaceError(c, errors.Invalid("space.content.type", err.Error()))
		return
	}
	content, err := spaceCompanionService.Get(c.Request.Context(), contentType, c.Param("id"))
	if err != nil {
		writeSpaceError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, content)
}

func PublishSpaceContent(c *gin.Context) {
	if spaceCompanionService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	var request spacePublicationRequest
	if err := bindSpaceJSON(c, &request); err != nil {
		writeSpaceError(c, errors.Invalid("space.publish.request", "request body is invalid"))
		return
	}
	principal, ok := spacePrincipalFromContext(c)
	if !ok {
		writeSpaceError(c, errors.Unauthorized("space.publish.identity"))
		return
	}
	result, err := spaceCompanionService.Publish(c.Request.Context(), principal, port.SpacePublicationInput{
		Type:            request.Type,
		Title:           request.Title,
		Body:            request.Body,
		MediaURL:        request.MediaURL,
		SourceSessionID: request.SourceSessionID,
		SourceRunID:     request.SourceRunID,
		IdempotencyKey:  request.IdempotencyKey,
	})
	if err != nil {
		writeSpaceError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}

func bindSpaceJSON(c *gin.Context, destination interface{}) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return errors.Invalid("space.request", "request body is required")
	}
	maxBytes := int64(256 * 1024)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err == nil {
		return errors.Invalid("space.request", "request body must contain one JSON object")
	} else if err != io.EOF {
		return err
	}
	return nil
}

func spacePrincipalFromContext(c *gin.Context) (port.SpacePrincipal, bool) {
	if c == nil {
		return port.SpacePrincipal{}, false
	}
	value, ok := c.Get(port.SpacePrincipalContextKey)
	if !ok {
		return port.SpacePrincipal{}, false
	}
	principal, ok := value.(port.SpacePrincipal)
	return principal, ok
}

func writeSpaceError(c *gin.Context, err error) {
	status := http.StatusServiceUnavailable
	message := "space service unavailable"
	switch errors.KindOf(err) {
	case errors.KindValidation:
		status, message = http.StatusBadRequest, "invalid space request"
	case errors.KindNotFound:
		status, message = http.StatusNotFound, "space content not found"
	case errors.KindUnauthorized:
		status, message = http.StatusUnauthorized, "space authentication required"
	case errors.KindForbidden:
		status, message = http.StatusForbidden, "space operation forbidden"
	case errors.KindConflict:
		status, message = http.StatusConflict, "space publication conflicts with an existing record"
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": message})
}
