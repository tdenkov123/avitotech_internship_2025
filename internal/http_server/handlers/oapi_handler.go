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

}

func (h *APIHandler) PostPullRequestMerge(c *gin.Context) {

}

func (h *APIHandler) PostPullRequestReassign(c *gin.Context) {

}

func (h *APIHandler) PostTeamAdd(c *gin.Context) {

}

func (h *APIHandler) GetTeamGet(c *gin.Context, params openapi.GetTeamGetParams) {

}

func (h *APIHandler) GetUsersGetReview(c *gin.Context, params openapi.GetUsersGetReviewParams) {

}

func (h *APIHandler) PostUsersSetIsActive(c *gin.Context) {

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
