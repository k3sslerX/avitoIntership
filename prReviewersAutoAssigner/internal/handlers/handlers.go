package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"prReviewersAutoAssigner/internal/consts"
	"prReviewersAutoAssigner/internal/repository"
	"prReviewersAutoAssigner/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
	service *service.Service
}

func NewHandlers(service *service.Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", h.CreateTeam)
		r.Get("/get", h.GetTeam)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", h.SetUserActive)
		r.Get("/getReview", h.GetPullRequestsByReviewer)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", h.CreatePullRequest)
		r.Post("/merge", h.MergePullRequest)
		r.Post("/reassign", h.ReassignReviewer)
	})

	r.Get("/stats", h.GetStats)

	return r
}

func (h *Handlers) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team consts.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		h.sendError(w, "Invalid request body", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateTeam(r.Context(), &team); err != nil {
		switch {
		case errors.Is(err, repository.ErrTeamExists):
			h.sendError(w, "team_name already exists", "TEAM_EXISTS", http.StatusBadRequest)
		default:
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"team": team,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.sendError(w, "team_name is required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(w, "Team not found", "NOT_FOUND", http.StatusNotFound)
		} else {
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(team)
}

func (h *Handlers) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	user, err := h.service.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(w, "User not found", "NOT_FOUND", http.StatusNotFound)
		} else {
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"user": user,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	pr, err := h.service.CreatePullRequest(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			h.sendError(w, "Author/team not found", "NOT_FOUND", http.StatusNotFound)
		case errors.Is(err, repository.ErrPRExists):
			h.sendError(w, "PR id already exists", "PR_EXISTS", http.StatusConflict)
		case errors.Is(err, repository.ErrNoCandidate):
			h.sendError(w, "No active reviewers available", "NO_CANDIDATE", http.StatusNotFound)
		default:
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"pr": pr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	pr, err := h.service.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(w, "PR not found", "NOT_FOUND", http.StatusNotFound)
		} else {
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"pr": pr,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_reviewer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	result, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			h.sendError(w, "PR or user not found", "NOT_FOUND", http.StatusNotFound)
		case errors.Is(err, repository.ErrPRMerged):
			h.sendError(w, "cannot reassign on merged PR", "PR_MERGED", http.StatusConflict)
		case errors.Is(err, repository.ErrNotAssigned):
			h.sendError(w, "reviewer is not assigned to this PR", "NOT_ASSIGNED", http.StatusConflict)
		case errors.Is(err, repository.ErrNoCandidate):
			h.sendError(w, "no active replacement candidate in team", "NO_CANDIDATE", http.StatusConflict)
		default:
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handlers) GetPullRequestsByReviewer(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.sendError(w, "user_id is required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetPullRequestsByReviewer(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(w, "User not found", "NOT_FOUND", http.StatusNotFound)
		} else {
			h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handlers) sendError(w http.ResponseWriter, message, code string, status int) {
	response := consts.ErrorResponse{}
	response.Error.Code = code
	response.Error.Message = message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		h.sendError(w, err.Error(), "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
