package service

import (
	"context"
	"math/rand"
	"prReviewersAutoAssigner/internal/consts"
	"prReviewersAutoAssigner/internal/repository"
	"time"
)

type Service struct {
	repo repository.Repository
	rand *rand.Rand
}

func NewService(repo repository.Repository) *Service {
	source := rand.NewSource(time.Now().UnixNano())
	return &Service{
		repo: repo,
		rand: rand.New(source),
	}
}

func (s *Service) CreateTeam(ctx context.Context, team *consts.Team) error {
	return s.repo.CreateTeam(ctx, team)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*consts.Team, error) {
	return s.repo.GetTeam(ctx, teamName)
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) (*consts.User, error) {
	return s.repo.SetUserActive(ctx, userID, isActive)
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*consts.PullRequest, error) {
	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return nil, err
	}

	if !author.IsActive {
		return nil, repository.ErrNoCandidate
	}

	teamName := author.TeamName

	potentialReviewers, err := s.repo.GetActiveTeamMembers(ctx, teamName, authorID)
	if err != nil {
		return nil, err
	}

	var reviewers []string
	if len(potentialReviewers) > 0 {
		s.rand.Shuffle(len(potentialReviewers), func(i, j int) {
			potentialReviewers[i], potentialReviewers[j] = potentialReviewers[j], potentialReviewers[i]
		})

		count := min(2, len(potentialReviewers))
		for i := 0; i < count; i++ {
			reviewers = append(reviewers, potentialReviewers[i].UserID)
		}
	}

	pr := &consts.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
	}

	if err := s.repo.CreatePullRequest(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) GetPullRequest(ctx context.Context, prID string) (*consts.PullRequest, error) {
	return s.repo.GetPullRequest(ctx, prID)
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*consts.PullRequest, error) {
	return s.repo.MergePullRequest(ctx, prID)
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*consts.ReassignResponse, error) {
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}

	if pr.Status == "MERGED" {
		return nil, repository.ErrPRMerged
	}

	isAssigned, err := s.repo.IsUserAssignedToPR(ctx, prID, oldUserID)
	if err != nil {
		return nil, err
	}
	if !isAssigned {
		return nil, repository.ErrNotAssigned
	}

	oldUserTeam, err := s.repo.GetUserTeam(ctx, oldUserID)
	if err != nil {
		return nil, err
	}

	potentialReplacements, err := s.repo.GetActiveTeamMembers(ctx, oldUserTeam, pr.AuthorID)
	if err != nil {
		return nil, err
	}

	var availableReplacements []consts.User
	for _, user := range potentialReplacements {
		isReviewer := false
		for _, reviewer := range pr.AssignedReviewers {
			if user.UserID == reviewer {
				isReviewer = true
				break
			}
		}
		if !isReviewer && user.UserID != oldUserID {
			availableReplacements = append(availableReplacements, user)
		}
	}

	if len(availableReplacements) == 0 {
		return nil, repository.ErrNoCandidate
	}

	newUser := availableReplacements[s.rand.Intn(len(availableReplacements))]

	tx, err := s.repo.(*repository.PostgresRepository).Db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx,
		"DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2",
		prID, oldUserID)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)",
		prID, newUser.UserID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	updatedPR, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}

	return &consts.ReassignResponse{
		PR:         *updatedPR,
		ReplacedBy: newUser.UserID,
	}, nil
}

func (s *Service) GetPullRequestsByReviewer(ctx context.Context, reviewerID string) (*consts.ReviewList, error) {
	prs, err := s.repo.GetPullRequestsByReviewer(ctx, reviewerID)
	if err != nil {
		return nil, err
	}

	return &consts.ReviewList{
		UserID:       reviewerID,
		PullRequests: prs,
	}, nil
}

func (s *Service) GetStats(ctx context.Context) (*consts.StatsResponse, error) {
	reviewStats, err := s.repo.GetReviewStats(ctx)
	if err != nil {
		return nil, err
	}

	prStats, err := s.repo.GetPRStats(ctx)
	if err != nil {
		return nil, err
	}

	return &consts.StatsResponse{
		ReviewStats: reviewStats,
		PRStats:     *prStats,
	}, nil
}
