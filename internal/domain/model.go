package domain

import "time"

type Team struct {
	TeamName string
	Members  []TeamMember
}

type TeamMember struct {
	UserID   string
	UserName string
	IsActive bool
}

type User struct {
	UserID   string
	UserName string
	TeamName string
	IsActive bool
}

const (
	PRStatusOpen   = "OPEN"
	PRStatusMerged = "MERGED"
)

type PullRequest struct {
	PullRequestId     string
	PullRequestName   string
	AuthorId          string
	Status            string
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}
