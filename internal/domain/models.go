package domain

import "time"

type Team struct {
	Name    string
	Members []TeamMember
}

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	AssignedReviewers []string
	CreatedAt         time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	ID        string
	Name      string
	AuthorID  string
	Status    string
	CreatedAt time.Time
}
