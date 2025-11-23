package consts

import "time"

type Team struct {
	TeamName string       `json:"team_name" db:"team_name"`
	Members  []TeamMember `json:"members" db:"-"`
}

type TeamMember struct {
	UserID   string `json:"user_id" db:"user_id"`
	Username string `json:"username" db:"username"`
	IsActive bool   `json:"is_active" db:"is_active"`
}

type User struct {
	UserID   string `json:"user_id" db:"user_id"`
	Username string `json:"username" db:"username"`
	TeamName string `json:"team_name" db:"team_name"`
	IsActive bool   `json:"is_active" db:"is_active"`
}

type PullRequest struct {
	PullRequestID     string     `json:"pull_request_id" db:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name" db:"pull_request_name"`
	AuthorID          string     `json:"author_id" db:"author_id"`
	Status            string     `json:"status" db:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers" db:"-"`
	CreatedAt         *time.Time `json:"createdAt,omitempty" db:"created_at"`
	MergedAt          *time.Time `json:"mergedAt,omitempty" db:"merged_at"`
}

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type ReviewList struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type ReassignResponse struct {
	PR         PullRequest `json:"pr"`
	ReplacedBy string      `json:"replaced_by"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type ReviewStats struct {
	UserID      string `json:"user_id" db:"user_id"`
	Username    string `json:"username" db:"username"`
	TeamName    string `json:"team_name" db:"team_name"`
	ReviewCount int    `json:"review_count" db:"review_count"`
}

type PRStats struct {
	TotalPRs            int     `json:"total_prs"`
	OpenPRs             int     `json:"open_prs"`
	MergedPRs           int     `json:"merged_prs"`
	PRsWithoutReviewers int     `json:"prs_without_reviewers"`
	AvgReviewersPerPR   float64 `json:"avg_reviewers_per_pr"`
}

type StatsResponse struct {
	ReviewStats []ReviewStats `json:"review_stats"`
	PRStats     PRStats       `json:"pr_stats"`
}
