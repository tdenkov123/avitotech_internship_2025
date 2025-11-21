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

type ReassignmentChange struct {
	PullRequestID string
	OldReviewerID string
	NewReviewerID *string
}

type BulkDeactivateResult struct {
	Team             domain.Team
	DeactivatedUsers []string
	Reassignments    []ReassignmentChange
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
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO teams (name) VALUES ($1)`, team.Name); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrTeamExists
			}
			return err
		}

		for _, member := range team.Members {
			if member.UserID == "" {
				continue
			}
			_, err := tx.Exec(ctx, `
                INSERT INTO users (id, username, team_name, is_active)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (id) DO UPDATE
                SET username = EXCLUDED.username,
                    team_name = EXCLUDED.team_name,
                    is_active = EXCLUDED.is_active
            `, member.UserID, member.Username, team.Name, member.IsActive)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Team{}, err
	}
	return s.GetTeam(ctx, team.Name)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM teams WHERE name = $1`, teamName).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Team{}, domain.ErrTeamNotFound
		}
		return domain.Team{}, err
	}

	rows, err := s.db.Query(ctx, `
        SELECT id, username, is_active
        FROM users
        WHERE team_name = $1
        ORDER BY username
    `, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	defer rows.Close()

	members := make([]domain.TeamMember, 0)
	for rows.Next() {
		var member domain.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return domain.Team{}, err
		}
		members = append(members, member)
	}
	if rows.Err() != nil {
		return domain.Team{}, rows.Err()
	}

	return domain.Team{Name: name, Members: members}, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, active bool) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRow(ctx, `
        UPDATE users
        SET is_active = $2
        WHERE id = $1
        RETURNING id, username, team_name, is_active
    `, userID, active).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (BulkDeactivateResult, error) {
	result := BulkDeactivateResult{}
	if teamName == "" || len(userIDs) == 0 {
		return result, domain.ErrInvalidInput
	}

	unique := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return result, domain.ErrInvalidInput
	}
	result.DeactivatedUsers = unique

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1)`, teamName).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrTeamNotFound
		}

		for _, id := range unique {
			user, err := s.getUser(ctx, tx, id)
			if err != nil {
				return err
			}
			if user.TeamName != teamName {
				return domain.ErrUserNotFound
			}
		}

		for _, id := range unique {
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET is_active = false
				WHERE id = $1
			`, id); err != nil {
				return err
			}
		}

		for _, id := range unique {
			rows, err := tx.Query(ctx, `
				SELECT pr.id
				FROM pull_requests pr
				JOIN pull_request_reviewers r ON r.pull_request_id = pr.id
				WHERE r.reviewer_id = $1 AND pr.status = 'OPEN'
			`, id)
			if err != nil {
				return err
			}
			var prIDs []string
			for rows.Next() {
				var prID string
				if err := rows.Scan(&prID); err != nil {
					rows.Close()
					return err
				}
				prIDs = append(prIDs, prID)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()

			for _, prID := range prIDs {
				assigned, err := s.listReviewers(ctx, tx, prID)
				if err != nil {
					return err
				}
				candidates, err := s.pickReplacementCandidates(ctx, tx, teamName, assigned, id)
				if err != nil {
					return err
				}
				var newReviewer *string
				if len(candidates) > 0 {
					choice := candidates[r.Intn(len(candidates))]
					newReviewer = &choice
					if _, err := tx.Exec(ctx, `
						DELETE FROM pull_request_reviewers
						WHERE pull_request_id = $1 AND reviewer_id = $2
					`, prID, id); err != nil {
						return err
					}
					if _, err := tx.Exec(ctx, `
						INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
						VALUES ($1, $2)
					`, prID, choice); err != nil {
						return err
					}
				} else {
					if _, err := tx.Exec(ctx, `
						DELETE FROM pull_request_reviewers
						WHERE pull_request_id = $1 AND reviewer_id = $2
					`, prID, id); err != nil {
						return err
					}
				}
				result.Reassignments = append(result.Reassignments, ReassignmentChange{
					PullRequestID: prID,
					OldReviewerID: id,
					NewReviewerID: newReviewer,
				})
			}
		}
		return nil
	})
	if err != nil {
		return BulkDeactivateResult{}, err
	}

	team, err := s.GetTeam(ctx, teamName)
	if err != nil {
		return BulkDeactivateResult{}, err
	}
	result.Team = team
	return result, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, input CreatePullRequestInput) (domain.PullRequest, error) {
	var result domain.PullRequest
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		author, err := s.getUser(ctx, tx, input.AuthorID)
		if err != nil {
			return err
		}

		row := tx.QueryRow(ctx, `
            INSERT INTO pull_requests (id, name, author_id)
            VALUES ($1, $2, $3)
            RETURNING id, name, author_id, status, created_at, merged_at
        `, input.ID, input.Name, input.AuthorID)
		if err := row.Scan(&result.ID, &result.Name, &result.AuthorID, &result.Status, &result.CreatedAt, &result.MergedAt); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrPullRequestExists
			}
			return err
		}

		reviewers, err := s.pickReviewers(ctx, tx, author.TeamName, input.AuthorID, 2)
		if err != nil {
			return err
		}
		result.AssignedReviewers = reviewers

		for _, reviewer := range reviewers {
			if _, err := tx.Exec(ctx, `
                INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
                VALUES ($1, $2)
            `, result.ID, reviewer); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.PullRequest{}, err
	}

	fullPR, err := s.GetPullRequest(ctx, s.db, result.ID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	return fullPR, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	ct, err := s.db.Exec(ctx, `
		UPDATE pull_requests
		SET status = 'MERGED',
			merged_at = COALESCE(merged_at, NOW())
		WHERE id = $1
	`, prID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if ct.RowsAffected() == 0 {
		return domain.PullRequest{}, domain.ErrPullRequestNotFound
	}

	pr, err := s.GetPullRequest(ctx, s.db, prID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, input ReassignInput) (ReassignResult, error) {
	var result ReassignResult
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		pr, err := s.GetPullRequest(ctx, tx, input.PullRequestID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrPullRequestNotFound
			}
			return err
		}
		if pr.Status == "MERGED" {
			return domain.ErrPullRequestMerged
		}

		assigned, err := s.listReviewers(ctx, tx, input.PullRequestID)
		if err != nil {
			return err
		}

		hasOld := false
		for _, reviewer := range assigned {
			if reviewer == input.OldReviewerID {
				hasOld = true
				break
			}
		}
		if !hasOld {
			return domain.ErrReviewerNotAssigned
		}

		oldUser, err := s.getUser(ctx, tx, input.OldReviewerID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return err
		}

		candidates, err := s.pickReplacementCandidates(ctx, tx, oldUser.TeamName, assigned, input.OldReviewerID)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return domain.ErrNoCandidate
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		newReviewer := candidates[r.Intn(len(candidates))]

		if _, err := tx.Exec(ctx, `
            DELETE FROM pull_request_reviewers
            WHERE pull_request_id = $1 AND reviewer_id = $2
        `, input.PullRequestID, input.OldReviewerID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
            INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
            VALUES ($1, $2)
        `, input.PullRequestID, newReviewer); err != nil {
			return err
		}

		updated, err := s.GetPullRequest(ctx, tx, input.PullRequestID)
		if err != nil {
			return err
		}
		result.PullRequest = updated
		result.ReplacedBy = newReviewer
		return nil
	})
	return result, err
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	if _, err := s.getUser(ctx, s.db, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
        SELECT pr.id, pr.name, pr.author_id, pr.status, pr.created_at
        FROM pull_requests pr
        JOIN pull_request_reviewers r ON r.pull_request_id = pr.id
        WHERE r.reviewer_id = $1
        ORDER BY pr.created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return prs, nil
}

func (s *Service) getUser(ctx context.Context, q dbExecutor, userID string) (domain.User, error) {
	var user domain.User
	err := q.QueryRow(ctx, `
        SELECT id, username, team_name, is_active
        FROM users
        WHERE id = $1
    `, userID).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return user, nil
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
	rows, err := q.Query(ctx, `
        SELECT id
        FROM users
        WHERE team_name = $1 AND is_active = true AND id <> $2
    `, teamName, excludeUser)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		candidates = append(candidates, id)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(candidates) <= limit {
		return candidates, nil
	}

	shuffled := make([]string, len(candidates))
	copy(shuffled, candidates)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:limit], nil
}

func (s *Service) pickReplacementCandidates(ctx context.Context, q dbExecutor, teamName string, assigned []string, oldReviewer string) ([]string, error) {
	rows, err := q.Query(ctx, `
        SELECT id
        FROM users
        WHERE team_name = $1 AND is_active = true
    `, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignedSet := make(map[string]struct{}, len(assigned))
	for _, id := range assigned {
		assignedSet[id] = struct{}{}
	}

	var candidates []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id == oldReviewer {
			continue
		}
		if _, exists := assignedSet[id]; exists {
			continue
		}
		candidates = append(candidates, id)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return candidates, nil
}
