package domain

import "errors"

var (
	ErrTeamExists          = errors.New("team already exists")
	ErrTeamNotFound        = errors.New("team not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrPullRequestNotFound = errors.New("pull request not found")
	ErrPullRequestExists   = errors.New("pull request already exists")
	ErrPullRequestMerged   = errors.New("pull request already merged")
	ErrReviewerNotAssigned = errors.New("reviewer not assigned to pull request")
	ErrNoCandidate         = errors.New("no replacement candidate")
)
