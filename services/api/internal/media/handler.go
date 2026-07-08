package media

import (
	"errors"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		svc:    svc,
		logger: logger.With(zap.String("layer", "media.handler")),
	}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/media")
	{
		g.POST("/upload/init", h.initUpload)
		g.POST("/upload/complete", h.completeUpload)
		g.GET("/:id", h.getAsset)
		g.GET("/:id/url", h.presignRead)
		g.DELETE("/:id", h.softDelete)
	}
}

// initUpload godoc
// @Summary  Start a two-phase upload — returns a presigned PUT URL
// @Tags     media
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      InitUploadRequest  true  "purpose + advisory mime/bytes"
// @Success  200   {object}  response.Response{data=InitUploadResponse}
// @Router   /media/upload/init [post]
func (h *Handler) initUpload(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.InitUpload(c.Request.Context(), uid, req)
	if err != nil {
		h.mapInitErr(c, err)
		return
	}
	response.OK(c, res)
}

// completeUpload godoc
// @Summary  Confirm the client-side PUT succeeded and materialize an asset row
// @Description The server HEAD's the object and uses THOSE values (size/mime/etag) on the asset. Client-supplied metadata is not trusted.
// @Tags     media
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      CompleteUploadRequest  true  "upload_id"
// @Success  200   {object}  response.Response{data=AssetDTO}
// @Router   /media/upload/complete [post]
func (h *Handler) completeUpload(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	sessionID, err := uuid.Parse(req.UploadID)
	if err != nil {
		response.BadRequest(c, "invalid upload_id")
		return
	}
	asset, err := h.svc.CompleteUpload(c.Request.Context(), uid, sessionID)
	if err != nil {
		h.mapCompleteErr(c, err)
		return
	}
	response.OK(c, asset)
}

// getAsset godoc
// @Summary  Fetch an asset's metadata (no URL — call /:id/url for that)
// @Tags     media
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "asset id"
// @Success  200  {object}  response.Response{data=AssetDTO}
// @Router   /media/{id} [get]
func (h *Handler) getAsset(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid asset id")
		return
	}
	dto, err := h.svc.GetAsset(c.Request.Context(), uid, assetID)
	if err != nil {
		h.mapAssetErr(c, err)
		return
	}
	response.OK(c, dto)
}

// presignRead godoc
// @Summary  Get a short-lived presigned GET URL for the underlying object
// @Tags     media
// @Security BearerAuth
// @Produce  json
// @Param    id   path      string  true  "asset id"
// @Success  200  {object}  response.Response{data=PresignedGetResponse}
// @Router   /media/{id}/url [get]
func (h *Handler) presignRead(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid asset id")
		return
	}
	res, err := h.svc.PresignRead(c.Request.Context(), uid, assetID)
	if err != nil {
		h.mapAssetErr(c, err)
		return
	}
	response.OK(c, res)
}

// softDelete godoc
// @Summary  Soft-delete an asset (owner only; S3 object survives until janitor sweep)
// @Tags     media
// @Security BearerAuth
// @Param    id   path  string  true  "asset id"
// @Success  204  "no content"
// @Router   /media/{id} [delete]
func (h *Handler) softDelete(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid asset id")
		return
	}
	if err := h.svc.SoftDelete(c.Request.Context(), uid, assetID); err != nil {
		if errors.Is(err, ErrAssetNotFound) {
			response.NotFound(c, "asset not found")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Forbidden(c)
			return
		}
		h.logger.Error("media: soft delete", zap.Error(err))
		response.InternalError(c, "delete failed")
		return
	}
	c.Status(204)
}

func (h *Handler) mapInitErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidPurpose):
		response.BadRequest(c, "invalid purpose")
	case errors.Is(err, ErrMimeNotAllowed):
		response.BadRequest(c, "mime not allowed for purpose")
	case errors.Is(err, ErrTooLarge):
		response.BadRequest(c, "declared size exceeds purpose limit")
	default:
		h.logger.Error("media: init upload", zap.Error(err))
		response.InternalError(c, "init upload failed")
	}
}

func (h *Handler) mapCompleteErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		response.NotFound(c, "upload session not found")
	case errors.Is(err, ErrSessionOwned):
		response.Forbidden(c)
	case errors.Is(err, ErrSessionUsed):
		response.Conflict(c, "upload session already completed")
	case errors.Is(err, ErrSessionExpired):
		response.BadRequest(c, "upload session expired")
	case errors.Is(err, ErrObjectMismatch):
		response.BadRequest(c, "uploaded object violates purpose rules")
	case errors.Is(err, ErrObjectNotFound):
		response.BadRequest(c, "object not uploaded")
	default:
		h.logger.Error("media: complete upload", zap.Error(err))
		response.InternalError(c, "complete upload failed")
	}
}

func (h *Handler) mapAssetErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAssetNotFound):
		response.NotFound(c, "asset not found")
	case errors.Is(err, ErrObjectNotFound):
		response.NotFound(c, "object not found")
	default:
		h.logger.Error("media: asset op", zap.Error(err))
		response.InternalError(c, "asset op failed")
	}
}
