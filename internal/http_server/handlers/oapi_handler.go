package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tdenkov123/avitotech_internship_2025/internal/domain"
	openapi "github.com/tdenkov123/avitotech_internship_2025/internal/http_server/api"
	"github.com/tdenkov123/avitotech_internship_2025/internal/service"
)

type APIHandler struct {
	logger  *zap.Logger
	service *service.Service
}

type reassignRequest struct {
	openapi.PostPullRequestReassignJSONRequestBody
	OldReviewerID string `json:"old_reviewer_id"`
}

func NewAPIHandler(logger *zap.Logger, svc *service.Service) *APIHandler {
	return &APIHandler{logger: logger, service: svc}
}

func (h *APIHandler) respondValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "invalid_request",
		"details": err.Error(),
	})
}

func (h *APIHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrTeamExists):
		c.JSON(http.StatusBadRequest, newErrorResponse(openapi.TEAMEXISTS, err.Error()))
	case errors.Is(err, domain.ErrTeamNotFound), errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrPullRequestNotFound):
		c.JSON(http.StatusNotFound, newErrorResponse(openapi.NOTFOUND, err.Error()))
	case errors.Is(err, domain.ErrPullRequestExists):
		c.JSON(http.StatusConflict, newErrorResponse(openapi.PREXISTS, err.Error()))
	case errors.Is(err, domain.ErrPullRequestMerged):
		c.JSON(http.StatusConflict, newErrorResponse(openapi.PRMERGED, err.Error()))
	case errors.Is(err, domain.ErrReviewerNotAssigned):
		c.JSON(http.StatusConflict, newErrorResponse(openapi.NOTASSIGNED, err.Error()))
	case errors.Is(err, domain.ErrNoCandidate):
		c.JSON(http.StatusConflict, newErrorResponse(openapi.NOCANDIDATE, err.Error()))
	default:
		h.logger.Error("unexpected error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func newErrorResponse(code openapi.ErrorResponseErrorCode, message string) openapi.ErrorResponse {
	var resp openapi.ErrorResponse
	resp.Error.Code = code
	resp.Error.Message = message
	return resp
}

func (h *APIHandler) PostPullRequestCreate(c *gin.Context) {
	var req openapi.PostPullRequestCreateJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondValidationError(c, err)
		return
	}

	pr, err := h.service.CreatePullRequest(c.Request.Context(), service.CreatePullRequestInput{
		ID:       req.PullRequestId,
		Name:     req.PullRequestName,
		AuthorID: req.AuthorId,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pr": toAPIPullRequest(pr)})
}

func (h *APIHandler) PostPullRequestMerge(c *gin.Context) {
	var req openapi.PostPullRequestMergeJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondValidationError(c, err)
		return
	}

	pr, err := h.service.MergePullRequest(c.Request.Context(), req.PullRequestId)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"pr": toAPIPullRequest(pr)})
}

func (h *APIHandler) PostPullRequestReassign(c *gin.Context) {
	var req reassignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondValidationError(c, err)
		return
	}

	oldReviewerID := req.OldUserId
	if oldReviewerID == "" {
		oldReviewerID = req.OldReviewerID
	}

	if req.PullRequestId == "" || oldReviewerID == "" {
		h.respondValidationError(c, errors.New("pull_request_id and old_user_id are required"))
		return
	}

	result, err := h.service.ReassignReviewer(c.Request.Context(), service.ReassignInput{
		PullRequestID: req.PullRequestId,
		OldReviewerID: oldReviewerID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr":          toAPIPullRequest(result.PullRequest),
		"replaced_by": result.ReplacedBy,
	})
}

func (h *APIHandler) PostTeamAdd(c *gin.Context) {
	var req openapi.PostTeamAddJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondValidationError(c, err)
		return
	}

	team := domain.Team{
		Name:    req.TeamName,
		Members: make([]domain.TeamMember, 0, len(req.Members)),
	}
	for _, member := range req.Members {
		team.Members = append(team.Members, domain.TeamMember{
			UserID:   member.UserId,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	created, err := h.service.CreateTeam(c.Request.Context(), team)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"team": toAPITeam(created)})
}

func (h *APIHandler) GetTeamGet(c *gin.Context, params openapi.GetTeamGetParams) {
	team, err := h.service.GetTeam(c.Request.Context(), params.TeamName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAPITeam(team))
}

func (h *APIHandler) GetUsersGetReview(c *gin.Context, params openapi.GetUsersGetReviewParams) {
	prs, err := h.service.GetUserReviews(c.Request.Context(), params.UserId)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       params.UserId,
		"pull_requests": toAPIPullRequestShort(prs),
	})
}

func (h *APIHandler) PostUsersSetIsActive(c *gin.Context) {
	var req openapi.PostUsersSetIsActiveJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondValidationError(c, err)
		return
	}

	user, err := h.service.SetUserActive(c.Request.Context(), req.UserId, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": toAPIUser(user)})
}

func toAPITeam(team domain.Team) openapi.Team {
	members := make([]openapi.TeamMember, 0, len(team.Members))
	for _, member := range team.Members {
		members = append(members, openapi.TeamMember{
			UserId:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}
	return openapi.Team{
		TeamName: team.Name,
		Members:  members,
	}
}

func toAPIUser(user domain.User) openapi.User {
	return openapi.User{
		UserId:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func toAPIPullRequest(pr domain.PullRequest) openapi.PullRequest {
	created := pr.CreatedAt
	var merged *time.Time
	if pr.MergedAt != nil {
		merged = pr.MergedAt
	}
	return openapi.PullRequest{
		PullRequestId:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorId:          pr.AuthorID,
		Status:            openapi.PullRequestStatus(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         &created,
		MergedAt:          merged,
	}
}

func toAPIPullRequestShort(items []domain.PullRequestShort) []openapi.PullRequestShort {
	result := make([]openapi.PullRequestShort, 0, len(items))
	for _, item := range items {
		result = append(result, openapi.PullRequestShort{
			PullRequestId:   item.ID,
			PullRequestName: item.Name,
			AuthorId:        item.AuthorID,
			Status:          openapi.PullRequestShortStatus(item.Status),
		})
	}
	return result
}
