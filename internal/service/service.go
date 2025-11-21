package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

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

func (s *Service) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
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

func (s *Service) getUser(ctx context.Context, q dbExecutor, userID string) (domain.User, error) {
	return domain.User{}, nil
}

func (s *Service) GetPullRequest(ctx context.Context, q dbExecutor, prID string) (domain.PullRequest, error) {
	var pr domain.PullRequest
	err := q.QueryRow(ctx, `
        SELECT id, name, author_id, status, created_at, merged_at
        FROM pull_requests
        WHERE id = $1
    `, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrPullRequestNotFound
		}
		return domain.PullRequest{}, err
	}

	reviewers, err := s.listReviewers(ctx, q, prID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.AssignedReviewers = reviewers

	return pr, nil
}

func (s *Service) listReviewers(ctx context.Context, q dbExecutor, prID string) ([]string, error) {
	rows, err := q.Query(ctx, `
        SELECT reviewer_id
        FROM pull_request_reviewers
        WHERE pull_request_id = $1
        ORDER BY reviewer_id
    `, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewer string
		if err := rows.Scan(&reviewer); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewer)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return reviewers, nil
}

func (s *Service) pickReviewers(ctx context.Context, q dbExecutor, teamName, excludeUser string, limit int) ([]string, error) {
	return []string{}, nil
}

func (s *Service) pickReplacementCandidates(ctx context.Context, q dbExecutor, teamName string, assigned []string, oldReviewer string) ([]string, error) {
	return []string{}, nil
}
