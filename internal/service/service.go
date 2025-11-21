package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tdenkov123/avitotech_internship_2025/internal/domain"
)

type Service struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type CreatePullRequestInput struct {
	ID       string
	Name     string
	AuthorID string
}

type ReassignInput struct {
	PullRequestID string
	OldReviewerID string
}

type ReassignResult struct {
	PullRequest domain.PullRequest
	ReplacedBy  string
}

type dbExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Service) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	return domain.Team{}, nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	return domain.Team{}, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, active bool) (domain.User, error) {
	return domain.User{}, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, input CreatePullRequestInput) (domain.PullRequest, error) {
	return domain.PullRequest{}, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	return domain.PullRequest{}, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, input ReassignInput) (ReassignResult, error) {
	return ReassignResult{}, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	return []domain.PullRequestShort{}, nil
}

func (s *Service) GetPullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	return domain.PullRequest{}, nil
}

func (s *Service) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return nil
}

func (s *Service) getUser(ctx context.Context, q dbExecutor, userID string) (domain.User, error) {
	return domain.User{}, nil
}

func (s *Service) getPullRequest(ctx context.Context, q dbExecutor, prID string) (domain.PullRequest, error) {
	return domain.PullRequest{}, nil
}

func (s *Service) listReviewers(ctx context.Context, q dbExecutor, prID string) ([]string, error) {
	return []string{}, nil
}

func (s *Service) pickReviewers(ctx context.Context, q dbExecutor, teamName, excludeUser string, limit int) ([]string, error) {
	return []string{}, nil
}

func (s *Service) pickReplacementCandidates(ctx context.Context, q dbExecutor, teamName string, assigned []string, oldReviewer string) ([]string, error) {
	return []string{}, nil
}
