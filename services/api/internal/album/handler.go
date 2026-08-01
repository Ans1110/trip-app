package album

import (
	"errors"

	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(public, protected *gin.RouterGroup)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.With(zap.String("layer", "album.handler"))}
}

func (h *Handler) RegisterRoutes(public, protected *gin.RouterGroup) {
	trip := protected.Group("/trips/:id/album")
	{
		trip.POST("/init-upload", h.InitUpload)
		trip.POST("/photos", h.CompletePhoto)
		trip.GET("/photos", h.ListPhotos)
		trip.POST("/shares", h.CreateShareToken)
		trip.GET("/shares", h.ListShareTokens)
	}
	photo := protected.Group("/albums/photos/:photo_id")
	{
		photo.PATCH("", h.UpdatePhoto)
		photo.DELETE("", h.DeletePhoto)
		photo.GET("/download", h.DownloadOriginal)
	}
	share := protected.Group("/albums/shares/:token_id")
	{
		share.DELETE("", h.RevokeShareToken)
	}
	// Public — no auth middleware. The path segment IS the credential.
	public.GET("/public/albums/:token", h.PublicRead)
}

// ---- Handlers ----

// InitUpload godoc
// @Summary  Batch-init album uploads (returns presigned PUT slots)
// @Description Accepts up to 50 items per call. Each item validates against the `album` media purpose (mime allowlist + size ceiling) and reserves an upload session. The client PUTs bytes to each returned URL then finalises via CompletePhoto.
// @Tags     album
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string             true  "trip id"
// @Param    body  body      InitUploadPayload  true  "batch items"
// @Success  201   {object}  response.Response{data=InitUploadResponse}
// @Router   /trips/{id}/album/init-upload [post]
func (h *Handler) InitUpload(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p InitUploadPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.InitUpload(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// CompletePhoto godoc
// @Summary  Finalise one uploaded photo (runs EXIF + thumbnails inline)
// @Description Consumes an upload session id from InitUpload, verifies the object exists in storage, extracts EXIF (DateTimeOriginal + GPS) and generates 400px + 1024px JPEG thumbnails. Broadcasts ALBUM_PHOTO_UPLOADED to the trip.
// @Tags     album
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string                true  "trip id"
// @Param    body  body      CompletePhotoPayload  true  "upload session ref + optional caption"
// @Success  201   {object}  response.Response{data=PhotoWithURLsDTO}
// @Router   /trips/{id}/album/photos [post]
func (h *Handler) CompletePhoto(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CompletePhotoPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CompletePhoto(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// ListPhotos godoc
// @Summary  List all photos in a trip's album (ordered by taken_at desc)
// @Description Returns each photo with short-lived presigned URLs for original + both thumbnail sizes. URLs are TTL-bounded (media presign_get_ttl).
// @Tags     album
// @Security BearerAuth
// @Produce  json
// @Param    id  path      string  true  "trip id"
// @Success  200 {object}  response.Response{data=[]PhotoWithURLsDTO}
// @Router   /trips/{id}/album/photos [get]
func (h *Handler) ListPhotos(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListPhotos(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// UpdatePhoto godoc
// @Summary  Patch a photo's caption (uploader, trip owner, or admin only)
// @Tags     album
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    photo_id  path      string              true  "photo id"
// @Param    body      body      UpdatePhotoPayload  true  "patch"
// @Success  200       {object}  response.Response{data=PhotoWithURLsDTO}
// @Router   /albums/photos/{photo_id} [patch]
func (h *Handler) UpdatePhoto(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	photoID, ok := parseUUID(c, "photo_id", "invalid photo id")
	if !ok {
		return
	}
	var p UpdatePhotoPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdatePhoto(c.Request.Context(), uid, photoID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeletePhoto godoc
// @Summary  Soft-delete a photo (uploader, trip owner, or admin only)
// @Description Marks the photo row deleted; underlying media assets stay for GC. Broadcasts ALBUM_PHOTO_DELETED to the trip.
// @Tags     album
// @Security BearerAuth
// @Produce  json
// @Param    photo_id  path      string  true  "photo id"
// @Success  200       {object}  response.Response
// @Router   /albums/photos/{photo_id} [delete]
func (h *Handler) DeletePhoto(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	photoID, ok := parseUUID(c, "photo_id", "invalid photo id")
	if !ok {
		return
	}
	if err := h.svc.DeletePhoto(c.Request.Context(), uid, photoID); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// DownloadOriginal godoc
// @Summary  Issue a short-lived presigned URL for the full-resolution original
// @Tags     album
// @Security BearerAuth
// @Produce  json
// @Param    photo_id  path      string  true  "photo id"
// @Success  200       {object}  response.Response{data=DownloadOriginalResponse}
// @Router   /albums/photos/{photo_id}/download [get]
func (h *Handler) DownloadOriginal(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	photoID, ok := parseUUID(c, "photo_id", "invalid photo id")
	if !ok {
		return
	}
	out, err := h.svc.DownloadOriginal(c.Request.Context(), uid, photoID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// CreateShareToken godoc
// @Summary  Mint a public share token for a trip's album
// @Description Returns the plaintext token exactly once — only its SHA-256 hash is persisted. The token becomes the sole credential for the /public/albums/{token} read path. Broadcasts ALBUM_SHARE_CREATED.
// @Tags     album
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id    path      string                    true   "trip id"
// @Param    body  body      CreateShareTokenPayload   false  "optional expires_at"
// @Success  201   {object}  response.Response{data=CreateShareTokenResponse}
// @Router   /trips/{id}/album/shares [post]
func (h *Handler) CreateShareToken(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CreateShareTokenPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		// Empty body is fine — expires_at is optional.
		p = CreateShareTokenPayload{}
	}
	out, err := h.svc.CreateShareToken(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// ListShareTokens godoc
// @Summary  List active + revoked share tokens for a trip
// @Description Plaintext tokens are never returned — only metadata (creator, expiry, revocation, last access).
// @Tags     album
// @Security BearerAuth
// @Produce  json
// @Param    id  path      string  true  "trip id"
// @Success  200 {object}  response.Response{data=[]ShareTokenDTO}
// @Router   /trips/{id}/album/shares [get]
func (h *Handler) ListShareTokens(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListShareTokens(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// RevokeShareToken godoc
// @Summary  Revoke a share token (creator, trip owner, or admin only)
// @Description Idempotent — revoking an already-revoked token succeeds silently. Broadcasts ALBUM_SHARE_REVOKED.
// @Tags     album
// @Security BearerAuth
// @Produce  json
// @Param    token_id  path      string  true  "share token id"
// @Success  200       {object}  response.Response
// @Router   /albums/shares/{token_id} [delete]
func (h *Handler) RevokeShareToken(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	tokID, ok := parseUUID(c, "token_id", "invalid share token id")
	if !ok {
		return
	}
	if err := h.svc.RevokeShareToken(c.Request.Context(), uid, tokID); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// PublicRead godoc
// @Summary  Read a shared album by its plaintext token (no auth)
// @Description The token IS the credential — no session, no CSRF. Returns 404 if token is unknown, revoked, or expired. Access timestamp is fire-and-forget updated for audit.
// @Tags     album
// @Produce  json
// @Param    token  path      string  true  "plaintext share token"
// @Success  200    {object}  response.Response{data=object{photos=[]PhotoWithURLsDTO,share=ShareTokenDTO}}
// @Router   /public/albums/{token} [get]
func (h *Handler) PublicRead(c *gin.Context) {
	token := c.Param("token")
	photos, share, err := h.svc.ResolvePublicAlbum(c.Request.Context(), token)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, gin.H{"photos": photos, "share": share})
}

// ---- Helpers ----

func (h *Handler) bindUserAndTrip(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	tripID, ok := parseUUID(c, "id", "invalid trip id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return uid, tripID, true
}

func (h *Handler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPhotoNotFound),
		errors.Is(err, ErrShareTokenNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrInvalidPayload),
		errors.Is(err, ErrCaptionTooLong),
		errors.Is(err, ErrBadUploadPair),
		errors.Is(err, ErrShareExpired),
		errors.Is(err, ErrShareTokenInactive),
		errors.Is(err, media.ErrInvalidPurpose),
		errors.Is(err, media.ErrMimeNotAllowed),
		errors.Is(err, media.ErrTooLarge),
		errors.Is(err, media.ErrObjectMismatch),
		errors.Is(err, media.ErrSessionExpired),
		errors.Is(err, media.ErrSessionOwned),
		errors.Is(err, media.ErrSessionUsed):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("album handler: internal", zap.Error(err))
		response.InternalError(c, "internal error")
	}
}

func requireUser(c *gin.Context) (uuid.UUID, bool) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return uuid.Nil, false
	}
	return uid, true
}

func parseUUID(c *gin.Context, param, msg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, msg)
		return uuid.Nil, false
	}
	return id, true
}
