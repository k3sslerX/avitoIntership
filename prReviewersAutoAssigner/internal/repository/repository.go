package repository

import (
	"context"
	"database/sql"
	"errors"
	"prReviewersAutoAssigner/internal/consts"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrPRExists    = errors.New("PR already exists")
	ErrPRMerged    = errors.New("PR is merged")
	ErrNotAssigned = errors.New("reviewer not assigned")
	ErrNoCandidate = errors.New("no active candidate available")
	ErrNotFound    = errors.New("resource not found")
)

type Repository interface {
	CreateTeam(ctx context.Context, team *consts.Team) error
	GetTeam(ctx context.Context, teamName string) (*consts.Team, error)
	TeamExists(ctx context.Context, teamName string) (bool, error)

	CreateOrUpdateUser(ctx context.Context, user *consts.User) error
	GetUser(ctx context.Context, userID string) (*consts.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (*consts.User, error)
	GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]consts.User, error)
	GetUserTeam(ctx context.Context, userID string) (string, error)

	CreatePullRequest(ctx context.Context, pr *consts.PullRequest) error
	GetPullRequest(ctx context.Context, prID string) (*consts.PullRequest, error)
	PRExists(ctx context.Context, prID string) (bool, error)
	MergePullRequest(ctx context.Context, prID string) (*consts.PullRequest, error)
	GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]consts.PullRequestShort, error)
	AddReviewerToPR(ctx context.Context, prID, reviewerID string) error
	RemoveReviewerFromPR(ctx context.Context, prID, reviewerID string) error
	GetPRReviewers(ctx context.Context, prID string) ([]string, error)
	IsUserAssignedToPR(ctx context.Context, prID, userID string) (bool, error)

	GetReviewStats(ctx context.Context) ([]consts.ReviewStats, error)
	GetPRStats(ctx context.Context) (*consts.PRStats, error)
}

type PostgresRepository struct {
	Db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{Db: db}
}

func (r *PostgresRepository) CreateTeam(ctx context.Context, team *consts.Team) error {
	tx, err := r.Db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", team.TeamName)
	if err != nil {
		return err
	}
	if exists {
		return ErrTeamExists
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) 
			DO UPDATE SET username = $2, team_name = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP`,
			member.UserID, member.Username, team.TeamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetTeam(ctx context.Context, teamName string) (*consts.Team, error) {
	var team consts.Team
	team.TeamName = teamName

	query := `
		SELECT user_id, username, is_active 
		FROM users 
		WHERE team_name = $1 
		ORDER BY user_id`

	err := r.Db.SelectContext(ctx, &team.Members, query, teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if len(team.Members) == 0 {
		var exists bool
		err = r.Db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}

	return &team, nil
}

func (r *PostgresRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	err := r.Db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName)
	return exists, err
}

func (r *PostgresRepository) CreateOrUpdateUser(ctx context.Context, user *consts.User) error {
	_, err := r.Db.ExecContext(ctx, `
		INSERT INTO users (user_id, username, team_name, is_active) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET username = $2, team_name = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP`,
		user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (*consts.User, error) {
	var user consts.User
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`
	err := r.Db.GetContext(ctx, &user, query, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *PostgresRepository) SetUserActive(ctx context.Context, userID string, isActive bool) (*consts.User, error) {
	_, err := r.Db.ExecContext(ctx,
		"UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2",
		isActive, userID)
	if err != nil {
		return nil, err
	}

	return r.GetUser(ctx, userID)
}

func (r *PostgresRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]consts.User, error) {
	var users []consts.User
	query := `
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY user_id`

	err := r.Db.SelectContext(ctx, &users, query, teamName, excludeUserID)
	return users, err
}

func (r *PostgresRepository) GetUserTeam(ctx context.Context, userID string) (string, error) {
	var teamName string
	query := `SELECT team_name FROM users WHERE user_id = $1`
	err := r.Db.GetContext(ctx, &teamName, query, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return teamName, err
}

func (r *PostgresRepository) CreatePullRequest(ctx context.Context, pr *consts.PullRequest) error {
	tx, err := r.Db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.PullRequestID)
	if err != nil {
		return err
	}
	if exists {
		return ErrPRExists
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
		VALUES ($1, $2, $3, $4)`,
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
			pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetPullRequest(ctx context.Context, prID string) (*consts.PullRequest, error) {
	var pr consts.PullRequest
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at 
		FROM pull_requests 
		WHERE pull_request_id = $1`

	err := r.Db.GetContext(ctx, &pr, query, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	reviewers, err := r.GetPRReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *PostgresRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	err := r.Db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", prID)
	return exists, err
}

func (r *PostgresRepository) MergePullRequest(ctx context.Context, prID string) (*consts.PullRequest, error) {
	pr, err := r.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}

	if pr.Status == "MERGED" {
		return pr, nil
	}

	now := time.Now()
	_, err = r.Db.ExecContext(ctx,
		"UPDATE pull_requests SET status = 'MERGED', merged_at = $1 WHERE pull_request_id = $2",
		now, prID)
	if err != nil {
		return nil, err
	}

	return r.GetPullRequest(ctx, prID)
}

func (r *PostgresRepository) GetPullRequestsByReviewer(ctx context.Context, reviewerID string) ([]consts.PullRequestShort, error) {
	var prs []consts.PullRequestShort
	query := `
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_requests p
		INNER JOIN pr_reviewers pr ON p.pull_request_id = pr.pr_id
		WHERE pr.user_id = $1
		ORDER BY p.created_at DESC`

	err := r.Db.SelectContext(ctx, &prs, query, reviewerID)
	return prs, err
}

func (r *PostgresRepository) AddReviewerToPR(ctx context.Context, prID, reviewerID string) error {
	_, err := r.Db.ExecContext(ctx,
		"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
		prID, reviewerID)
	return err
}

func (r *PostgresRepository) RemoveReviewerFromPR(ctx context.Context, prID, reviewerID string) error {
	_, err := r.Db.ExecContext(ctx,
		"DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2",
		prID, reviewerID)
	return err
}

func (r *PostgresRepository) GetPRReviewers(ctx context.Context, prID string) ([]string, error) {
	var reviewers []string
	query := `SELECT user_id FROM pr_reviewers WHERE pr_id = $1`
	err := r.Db.SelectContext(ctx, &reviewers, query, prID)
	return reviewers, err
}

func (r *PostgresRepository) IsUserAssignedToPR(ctx context.Context, prID, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2)`
	err := r.Db.GetContext(ctx, &exists, query, prID, userID)
	return exists, err
}

func (r *PostgresRepository) GetReviewStats(ctx context.Context) ([]consts.ReviewStats, error) {
	var stats []consts.ReviewStats
	query := `
		SELECT 
			u.user_id, 
			u.username, 
			u.team_name,
			COUNT(pr.user_id) as review_count
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.user_id = pr.user_id
		WHERE u.is_active = true
		GROUP BY u.user_id, u.username, u.team_name
		ORDER BY review_count DESC, u.username`

	err := r.Db.SelectContext(ctx, &stats, query)
	return stats, err
}

func (r *PostgresRepository) GetPRStats(ctx context.Context) (*consts.PRStats, error) {
	stats := &consts.PRStats{}

	// Общее количество PR
	err := r.Db.GetContext(ctx, &stats.TotalPRs, "SELECT COUNT(*) FROM pull_requests")
	if err != nil {
		return nil, err
	}

	// Открытые PR
	err = r.Db.GetContext(ctx, &stats.OpenPRs, "SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'")
	if err != nil {
		return nil, err
	}

	// Мерженые PR
	err = r.Db.GetContext(ctx, &stats.MergedPRs, "SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'")
	if err != nil {
		return nil, err
	}

	// PR без ревьюверов
	err = r.Db.GetContext(ctx, &stats.PRsWithoutReviewers, `
		SELECT COUNT(*) 
		FROM pull_requests 
		WHERE NOT EXISTS (
			SELECT 1 FROM pr_reviewers WHERE pr_id = pull_requests.pull_request_id
		)`)
	if err != nil {
		return nil, err
	}

	// Среднее количество ревьюверов
	err = r.Db.GetContext(ctx, &stats.AvgReviewers, `
		SELECT COALESCE(AVG(reviewer_count), 0)
		FROM (
			SELECT pr_id, COUNT(*) as reviewer_count
			FROM pr_reviewers
			GROUP BY pr_id
		) pr_counts`)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
