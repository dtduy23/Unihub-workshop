package handler

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
	"unihub-workshop/internal/service"
)

type CommunityHandler struct {
	s        *service.CommunityService
	mediaDir string
}

func NewCommunityHandler(s *service.CommunityService) *CommunityHandler {
	dir := os.Getenv("MEDIA_DIR")
	if dir == "" {
		dir = "./data/media"
	}
	return &CommunityHandler{s, dir}
}
func (h *CommunityHandler) Register(r chi.Router) {
	r.Get("/api/v1/feed", h.Feed)
	r.Get("/api/v1/me/bookmarks", h.Feed)
	r.Get("/api/v1/me/posts", h.Feed)
	r.Get("/api/v1/posts/{id}", h.Post)
	r.Post("/api/v1/posts", h.SavePost)
	r.Patch("/api/v1/posts/{id}", h.SavePost)
	r.Delete("/api/v1/posts/{id}", h.DeletePost)
	r.Put("/api/v1/posts/{id}/like", h.Interact)
	r.Delete("/api/v1/posts/{id}/like", h.Interact)
	r.Put("/api/v1/posts/{id}/bookmark", h.Interact)
	r.Delete("/api/v1/posts/{id}/bookmark", h.Interact)
	r.Get("/api/v1/posts/{id}/comments", h.Comments)
	r.Post("/api/v1/posts/{id}/comments", h.SaveComment)
	r.Patch("/api/v1/comments/{id}", h.SaveComment)
	r.Delete("/api/v1/comments/{id}", h.DeleteComment)
	r.Get("/api/v1/companies", h.Companies)
	r.Get("/api/v1/companies/{id}", h.Company)
	r.Get("/api/v1/companies/{id}/posts", h.Feed)
	r.Get("/api/v1/companies/{id}/workshops", h.CompanyWorkshops)
	r.Put("/api/v1/companies/{id}/follow", h.Follow)
	r.Delete("/api/v1/companies/{id}/follow", h.Follow)
	r.Post("/api/v1/reports", h.Report)
	r.Post("/api/v1/media", h.UploadMedia)
	r.Get("/api/v1/media/{id}", h.Media)
	r.With(middleware.RequireRole(model.RoleStaff, model.RoleAdmin)).Get("/api/v1/staff/workshops", h.AssignedWorkshops)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole(model.RoleBusiness))
		r.Get("/api/v1/business/profile", h.BusinessProfile)
		r.Patch("/api/v1/business/profile", h.BusinessProfile)
		r.Get("/api/v1/business/stats", h.BusinessStats)
		r.Get("/api/v1/business/workshops", h.BusinessWorkshops)
		r.Post("/api/v1/business/workshops", h.SaveWorkshop)
		r.Patch("/api/v1/business/workshops/{id}", h.SaveWorkshop)
		r.Get("/api/v1/business/workshops/{id}", h.BusinessWorkshop)
		r.Post("/api/v1/business/workshops/{id}/submit", h.SubmitWorkshop)
		r.Post("/api/v1/business/workshops/{id}/revisions", h.CreateRevision)
		r.Post("/api/v1/business/workshops/{id}/cancellation-requests", h.CreateRevision)
		r.Get("/api/v1/business/revisions", h.Revisions)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole(model.RoleAdmin))
		r.Get("/api/v1/admin/companies", h.Companies)
		r.Post("/api/v1/admin/business-accounts", h.CreateCompany)
		r.Post("/api/v1/admin/companies/{id}/review", h.ReviewCompany)
		r.Get("/api/v1/admin/workshop-reviews", h.WorkshopReviews)
		r.Post("/api/v1/admin/workshops/{id}/review", h.ReviewWorkshop)
		r.Get("/api/v1/admin/workshop-revisions", h.Revisions)
		r.Post("/api/v1/admin/workshop-revisions/{id}/review", h.ReviewRevision)
		r.Get("/api/v1/admin/reports", h.Reports)
		r.Post("/api/v1/admin/reports/{id}/resolve", h.ResolveReport)
		r.Post("/api/v1/admin/posts/{id}/moderate", h.Moderate)
		r.Post("/api/v1/admin/comments/{id}/moderate", h.Moderate)
		r.Get("/api/v1/admin/workshops/{id}/staff", h.Staff)
		r.Put("/api/v1/admin/workshops/{id}/staff/{userID}", h.AssignStaff)
		r.Delete("/api/v1/admin/workshops/{id}/staff/{userID}", h.AssignStaff)
	})
}
func (h *CommunityHandler) respond(w http.ResponseWriter, data any, err error) {
	if err != nil {
		status := 400
		message := err.Error()
		switch {
		case errors.Is(err, repository.ErrForbidden):
			status = 403
		case errors.Is(err, repository.ErrConflict):
			status = 409
		case errors.Is(err, pgx.ErrNoRows):
			status = 404
			message = "Không tìm thấy dữ liệu"
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			status = 503
			message = "Không thể xử lý dữ liệu"
			if pg.Code == "23505" {
				status = 409
				message = "Dữ liệu đã tồn tại hoặc đang có yêu cầu chờ duyệt"
			}
			if pg.Code == "22P02" || pg.Code == "23503" || pg.Code == "23514" {
				status = 400
				message = "Dữ liệu không hợp lệ"
			}
			log.Printf("[COMMUNITY] %v", err)
		}
		errorResponse(w, status, message)
		return
	}
	writeJSON(w, 200, model.APIResponse{Success: true, Data: data})
}
func (h *CommunityHandler) Feed(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if strings.Contains(r.URL.Path, "/me/bookmarks") {
		mode = "saved"
	}
	if strings.Contains(r.URL.Path, "/me/posts") {
		mode = "mine"
	}
	if mode != "" && mode != "following" && mode != "saved" && mode != "mine" {
		h.respond(w, nil, fmt.Errorf("invalid feed mode"))
		return
	}
	company := ""
	if id := getURLParam(r, "id"); id != "" {
		c, err := h.s.Repo.Company(r.Context(), getUserID(r), id, false)
		if err != nil || c.Status != "APPROVED" {
			h.respond(w, nil, pgx.ErrNoRows)
			return
		}
		company = c.ID
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	page, err := h.s.Repo.Posts(r.Context(), getUserID(r), mode, company, r.URL.Query().Get("cursor"), r.URL.Query().Get("topic"), limit)
	h.respond(w, page, err)
}
func (h *CommunityHandler) Post(w http.ResponseWriter, r *http.Request) {
	p, err := h.s.Repo.Post(r.Context(), getUserID(r), getURLParam(r, "id"), middleware.GetUserRole(r.Context()) == model.RoleAdmin)
	h.respond(w, p, err)
}
func (h *CommunityHandler) SavePost(w http.ResponseWriter, r *http.Request) {
	var in model.PostInput
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	p, err := h.s.SavePost(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()), getURLParam(r, "id"), in)
	h.respond(w, p, err)
}
func (h *CommunityHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	err := h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()))
	if err == nil {
		err = h.s.Repo.DeletePost(r.Context(), getUserID(r), getURLParam(r, "id"))
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Interact(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	_, err := h.s.VisiblePost(r.Context(), getUserID(r), id)
	if err == nil {
		err = h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()))
	}
	if err == nil {
		kind := "like"
		if strings.HasSuffix(r.URL.Path, "bookmark") {
			kind = "bookmark"
		}
		err = h.s.Repo.Interact(r.Context(), getUserID(r), id, kind, r.Method == "DELETE")
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Comments(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if _, err := h.s.VisiblePost(r.Context(), getUserID(r), id); err != nil {
		h.respond(w, nil, err)
		return
	}
	items, err := h.s.Repo.Comments(r.Context(), id)
	h.respond(w, items, err)
}
func (h *CommunityHandler) SaveComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Content  string  `json:"content"`
		ParentID *string `json:"parent_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	in.Content = strings.TrimSpace(in.Content)
	if len(in.Content) == 0 || len(in.Content) > 4000 {
		h.respond(w, nil, fmt.Errorf("bình luận 1–4000 ký tự"))
		return
	}
	if err := h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context())); err != nil {
		h.respond(w, nil, err)
		return
	}
	postID := getURLParam(r, "id")
	id := ""
	if strings.Contains(r.URL.Path, "/api/v1/comments/") {
		id = postID
		cm, err := h.s.Repo.Comment(r.Context(), id)
		if err != nil {
			h.respond(w, nil, err)
			return
		}
		postID = cm.PostID
	}
	if _, err := h.s.VisiblePost(r.Context(), getUserID(r), postID); err != nil {
		h.respond(w, nil, err)
		return
	}
	saved, err := h.s.Repo.SaveComment(r.Context(), getUserID(r), postID, id, in.Content, in.ParentID)
	h.respond(w, map[string]string{"id": saved}, err)
}
func (h *CommunityHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	err := h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()))
	if err == nil {
		err = h.s.Repo.DeleteComment(r.Context(), getUserID(r), getURLParam(r, "id"))
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Companies(w http.ResponseWriter, r *http.Request) {
	all := strings.Contains(r.URL.Path, "/admin/")
	items, err := h.s.Repo.Companies(r.Context(), getUserID(r), all)
	if !all {
		for i := range items {
			items[i].OwnerUserID = ""
			items[i].ReviewReason = ""
		}
	}
	h.respond(w, items, err)
}
func (h *CommunityHandler) Company(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.Repo.Company(r.Context(), getUserID(r), getURLParam(r, "id"), false)
	if err == nil && c.Status != "APPROVED" && middleware.GetUserRole(r.Context()) != model.RoleAdmin {
		err = pgx.ErrNoRows
	}
	if middleware.GetUserRole(r.Context()) != model.RoleAdmin {
		c.OwnerUserID = ""
		c.ReviewReason = ""
	}
	h.respond(w, c, err)
}
func (h *CommunityHandler) Follow(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.Repo.Company(r.Context(), getUserID(r), getURLParam(r, "id"), false)
	if err == nil && c.Status != "APPROVED" {
		err = repository.ErrForbidden
	}
	if err == nil {
		err = h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()))
	}
	if err == nil {
		err = h.s.Repo.Follow(r.Context(), getUserID(r), c.ID, r.Method == "DELETE")
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) CompanyWorkshops(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.Repo.Company(r.Context(), getUserID(r), getURLParam(r, "id"), false)
	if err != nil || c.Status != "APPROVED" {
		h.respond(w, nil, pgx.ErrNoRows)
		return
	}
	items, err := h.s.WorkshopRepo.List(r.Context(), "", false, c.ID)
	visible := []model.Workshop{}
	for _, v := range items {
		if v.Status == model.WorkshopPublished || v.Status == model.WorkshopClosed || v.Status == model.WorkshopCancelled {
			visible = append(visible, v)
		}
	}
	h.respond(w, visible, err)
}
func (h *CommunityHandler) BusinessProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PATCH" {
		var in model.CompanyInput
		if err := decodeJSON(r, &in); err != nil {
			h.respond(w, nil, err)
			return
		}
		if err := service.ValidateCompany(in); err != nil {
			h.respond(w, nil, err)
			return
		}
		if err := h.s.Repo.UpdateCompany(r.Context(), getUserID(r), in); err != nil {
			h.respond(w, nil, err)
			return
		}
	}
	c, err := h.s.BusinessCompany(r.Context(), getUserID(r), false)
	h.respond(w, c, err)
}
func (h *CommunityHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var in model.BusinessAccountInput
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	c, err := h.s.CreateCompany(r.Context(), in)
	h.respond(w, c, err)
}
func reviewBody(r *http.Request) (string, string, error) {
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	err := decodeJSON(r, &in)
	in.Reason = strings.TrimSpace(in.Reason)
	if err == nil && (in.Reason == "" || len(in.Reason) > 2000) {
		err = fmt.Errorf("vui lòng nhập lý do hoặc ghi chú duyệt")
	}
	return in.Status, in.Reason, err
}
func (h *CommunityHandler) ReviewCompany(w http.ResponseWriter, r *http.Request) {
	status, reason, err := reviewBody(r)
	if err == nil && status != "APPROVED" && status != "REJECTED" && status != "SUSPENDED" {
		err = fmt.Errorf("invalid status")
	}
	if err == nil {
		err = h.s.Repo.ReviewCompany(r.Context(), getUserID(r), getURLParam(r, "id"), status, reason)
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) BusinessStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.s.Repo.BusinessStats(r.Context(), getUserID(r))
	h.respond(w, stats, err)
}

func (h *CommunityHandler) AssignedWorkshops(w http.ResponseWriter, r *http.Request) {
	items, err := h.s.WorkshopRepo.Assigned(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()) == model.RoleAdmin)
	h.respond(w, items, err)
}
func (h *CommunityHandler) BusinessWorkshops(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.BusinessCompany(r.Context(), getUserID(r), false)
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	items, err := h.s.WorkshopRepo.List(r.Context(), "", true, c.ID)
	h.respond(w, items, err)
}
func (h *CommunityHandler) BusinessWorkshop(w http.ResponseWriter, r *http.Request) {
	item, err := h.s.OwnedWorkshop(r.Context(), getUserID(r), getURLParam(r, "id"))
	h.respond(w, item, err)
}
func (h *CommunityHandler) SaveWorkshop(w http.ResponseWriter, r *http.Request) {
	var in model.CreateWorkshopRequest
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	item, err := h.s.SaveWorkshop(r.Context(), getUserID(r), getURLParam(r, "id"), in)
	h.respond(w, item, err)
}
func (h *CommunityHandler) SubmitWorkshop(w http.ResponseWriter, r *http.Request) {
	h.respond(w, nil, h.s.SubmitWorkshop(r.Context(), getUserID(r), getURLParam(r, "id")))
}
func (h *CommunityHandler) WorkshopReviews(w http.ResponseWriter, r *http.Request) {
	items, err := h.s.Repo.WorkshopReviews(r.Context())
	h.respond(w, items, err)
}
func (h *CommunityHandler) ReviewWorkshop(w http.ResponseWriter, r *http.Request) {
	status, reason, err := reviewBody(r)
	if err == nil && status != "PUBLISHED" && status != "REJECTED" {
		err = fmt.Errorf("invalid status")
	}
	if err == nil {
		w, e := h.s.WorkshopRepo.FindByID(r.Context(), getURLParam(r, "id"))
		err = e
		if e == nil {
			in := service.WorkshopRequest(w)
			err = service.ValidateWorkshop(&in)
		}
	}
	if err == nil {
		err = h.s.Repo.ReviewWorkshop(r.Context(), getUserID(r), getURLParam(r, "id"), status, reason)
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) CreateRevision(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Payload model.CreateWorkshopRequest `json:"payload"`
		Reason  string                      `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	cancel := strings.HasSuffix(r.URL.Path, "cancellation-requests")
	h.respond(w, nil, h.s.CreateRevision(r.Context(), getUserID(r), getURLParam(r, "id"), in.Payload, cancel, in.Reason))
}
func (h *CommunityHandler) Revisions(w http.ResponseWriter, r *http.Request) {
	items, err := h.s.Repo.Revisions(r.Context(), getUserID(r), strings.Contains(r.URL.Path, "/admin/"))
	h.respond(w, items, err)
}
func (h *CommunityHandler) ReviewRevision(w http.ResponseWriter, r *http.Request) {
	status, reason, err := reviewBody(r)
	if err == nil && status != "APPROVED" && status != "REJECTED" {
		err = fmt.Errorf("invalid status")
	}
	if err == nil {
		err = h.s.Repo.ReviewRevision(r.Context(), getUserID(r), getURLParam(r, "id"), status, reason)
	}
	if err == nil {
		items, _ := h.s.Repo.Revisions(r.Context(), getUserID(r), true)
		for _, rev := range items {
			if rev.ID == getURLParam(r, "id") {
				h.s.Workshops.DeleteCache(r.Context(), rev.WorkshopID)
			}
		}
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Report(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PostID    *string `json:"post_id"`
		CommentID *string `json:"comment_id"`
		Reason    string  `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		h.respond(w, nil, err)
		return
	}
	if (in.PostID == nil) == (in.CommentID == nil) || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 2000 {
		h.respond(w, nil, fmt.Errorf("chọn một bài hoặc bình luận và nhập lý do"))
		return
	}
	postID := ""
	if in.PostID != nil {
		postID = *in.PostID
	} else {
		c, err := h.s.Repo.Comment(r.Context(), *in.CommentID)
		if err != nil {
			h.respond(w, nil, err)
			return
		}
		postID = c.PostID
	}
	if _, err := h.s.VisiblePost(r.Context(), getUserID(r), postID); err != nil {
		h.respond(w, nil, err)
		return
	}
	err := h.s.MayInteract(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()))
	if err == nil {
		err = h.s.Repo.Report(r.Context(), getUserID(r), in.PostID, in.CommentID, in.Reason)
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Reports(w http.ResponseWriter, r *http.Request) {
	items, err := h.s.Repo.Reports(r.Context())
	h.respond(w, items, err)
}
func (h *CommunityHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	_, reason, err := reviewBody(r)
	if err == nil {
		err = h.s.Repo.ResolveReport(r.Context(), getUserID(r), getURLParam(r, "id"), reason)
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	_, reason, err := reviewBody(r)
	kind := "post"
	if strings.Contains(r.URL.Path, "/comments/") {
		kind = "comment"
	}
	if err == nil {
		err = h.s.Repo.Moderate(r.Context(), getUserID(r), getURLParam(r, "id"), kind, reason)
	}
	h.respond(w, nil, err)
}
func (h *CommunityHandler) Staff(w http.ResponseWriter, r *http.Request) {
	items, err := h.s.Repo.Staff(r.Context(), getURLParam(r, "id"))
	h.respond(w, items, err)
}
func (h *CommunityHandler) AssignStaff(w http.ResponseWriter, r *http.Request) {
	h.respond(w, nil, h.s.Repo.AssignStaff(r.Context(), getURLParam(r, "id"), getURLParam(r, "userID"), r.Method == "DELETE"))
}
func (h *CommunityHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	if middleware.GetUserRole(r.Context()) == model.RoleBusiness {
		company, err := h.s.BusinessCompany(r.Context(), getUserID(r), false)
		if err != nil || company.Status == "SUSPENDED" {
			h.respond(w, nil, repository.ErrForbidden)
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024+64*1024)
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		h.respond(w, nil, fmt.Errorf("ảnh tối đa 5 MB"))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
	if err != nil || len(data) == 0 || len(data) > 5*1024*1024 {
		h.respond(w, nil, fmt.Errorf("ảnh tối đa 5 MB"))
		return
	}
	mime := http.DetectContentType(data)
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		h.respond(w, nil, fmt.Errorf("chỉ hỗ trợ PNG, JPG, WebP"))
		return
	}
	id := uuid.NewString()
	if err = os.MkdirAll(h.mediaDir, 0750); err == nil {
		err = os.WriteFile(filepath.Join(h.mediaDir, id), data, 0640)
	}
	if err == nil {
		err = h.s.Repo.StoreMedia(r.Context(), id, getUserID(r), mime, int64(len(data)))
	}
	if err != nil {
		_ = os.Remove(filepath.Join(h.mediaDir, id))
		h.respond(w, nil, err)
		return
	}
	h.respond(w, map[string]string{"id": id, "url": "/api/backend/api/v1/media/" + id}, nil)
}
func (h *CommunityHandler) Media(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		h.respond(w, nil, pgx.ErrNoRows)
		return
	}
	mime, err := h.s.Repo.Media(r.Context(), id)
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, filepath.Join(h.mediaDir, id))
}
