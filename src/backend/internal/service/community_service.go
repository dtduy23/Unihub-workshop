package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
)

type CommunityService struct {
	Repo         *repository.CommunityRepo
	Workshops    *WorkshopService
	WorkshopRepo *repository.WorkshopRepo
}

func NewCommunityService(repo *repository.CommunityRepo, workshops *WorkshopService, workshopRepo *repository.WorkshopRepo) *CommunityService {
	return &CommunityService{repo, workshops, workshopRepo}
}
func ValidatePost(in *model.PostInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Content = strings.TrimSpace(in.Content)
	if utf8.RuneCountInString(in.Title) < 2 || utf8.RuneCountInString(in.Title) > 200 || in.Content == "" || utf8.RuneCountInString(in.Content) > 12000 || len(in.Topic) > 80 {
		return fmt.Errorf("tiêu đề 2–200 ký tự, nội dung 1–12000 ký tự")
	}
	if in.Kind == "" {
		in.Kind = "COMMUNITY"
	}
	if in.Status == "" {
		in.Status = "DRAFT"
	}
	if in.Status != "DRAFT" && in.Status != "PUBLISHED" {
		return fmt.Errorf("invalid post status")
	}
	if in.Kind != "COMMUNITY" && in.Kind != "WORKSHOP_ANNOUNCEMENT" {
		return fmt.Errorf("invalid post kind")
	}
	if (in.Kind == "WORKSHOP_ANNOUNCEMENT") != (in.WorkshopID != nil) {
		return fmt.Errorf("bài giới thiệu phải liên kết với một workshop")
	}
	if in.WorkshopID != nil {
		if _, err := uuid.Parse(*in.WorkshopID); err != nil {
			return fmt.Errorf("invalid workshop ID")
		}
	}
	if len(in.MediaIDs) > 4 {
		return fmt.Errorf("tối đa 4 ảnh")
	}
	seen := map[string]bool{}
	for _, id := range in.MediaIDs {
		if _, err := uuid.Parse(id); err != nil || seen[id] {
			return fmt.Errorf("invalid image IDs")
		}
		seen[id] = true
	}
	if in.MediaIDs == nil {
		in.MediaIDs = []string{}
	}
	return nil
}
func ValidateCompany(in model.CompanyInput) error {
	if strings.TrimSpace(in.Name) == "" || utf8.RuneCountInString(in.Name) > 160 || len(in.Description) > 12000 || len(in.Industry) > 160 || len(in.Contact) > 500 || len(in.Address) > 500 {
		return fmt.Errorf("thông tin doanh nghiệp không hợp lệ")
	}
	if in.Website != "" {
		u, err := url.Parse(in.Website)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return fmt.Errorf("website phải là URL http/https")
		}
	}
	return nil
}
func (s *CommunityService) BusinessCompany(ctx context.Context, userID string, approved bool) (model.Company, error) {
	c, err := s.Repo.Company(ctx, userID, userID, true)
	if err != nil {
		return c, err
	}
	if approved && c.Status != "APPROVED" {
		return c, repository.ErrForbidden
	}
	return c, nil
}
func (s *CommunityService) MayInteract(ctx context.Context, user string, role model.Role) error {
	if role == model.RoleBusiness {
		_, err := s.BusinessCompany(ctx, user, true)
		return err
	}
	return nil
}
func (s *CommunityService) VisiblePost(ctx context.Context, user, id string) (model.Post, error) {
	p, err := s.Repo.Post(ctx, user, id, false)
	if err != nil {
		return p, err
	}
	if p.Status != "PUBLISHED" {
		return p, repository.ErrForbidden
	}
	if p.Workshop != nil && p.Workshop.Status != model.WorkshopPublished && p.Workshop.Status != model.WorkshopClosed && p.Workshop.Status != model.WorkshopCancelled {
		return p, repository.ErrForbidden
	}
	if p.CompanyID != nil {
		c, e := s.Repo.Company(ctx, user, *p.CompanyID, false)
		if e != nil || c.Status != "APPROVED" {
			return p, repository.ErrForbidden
		}
	}
	return p, nil
}
func (s *CommunityService) SavePost(ctx context.Context, user string, role model.Role, id string, in model.PostInput) (model.Post, error) {
	if role != model.RoleStudent && role != model.RoleBusiness && role != model.RoleAdmin {
		return model.Post{}, repository.ErrForbidden
	}
	if in.Kind == "WORKSHOP_ANNOUNCEMENT" && role != model.RoleBusiness && role != model.RoleAdmin {
		return model.Post{}, repository.ErrForbidden
	}
	if err := ValidatePost(&in); err != nil {
		return model.Post{}, err
	}
	id, err := s.Repo.SavePost(ctx, user, role, id, in)
	if err != nil {
		return model.Post{}, err
	}
	return s.Repo.Post(ctx, user, id, false)
}
func (s *CommunityService) CreateCompany(ctx context.Context, in model.BusinessAccountInput) (model.Company, error) {
	if err := ValidateCompany(in.CompanyInput); err != nil {
		return model.Company{}, err
	}
	in.Identifier = strings.TrimSpace(in.Identifier)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if _, err := mail.ParseAddress(in.Email); err != nil || len(in.Password) < 8 || len(in.Password) > 72 || in.Identifier == "" || len(in.Identifier) > 160 || !regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(in.Slug) || len(in.Slug) > 100 {
		return model.Company{}, fmt.Errorf("email, mã đăng nhập, slug hoặc mật khẩu (8–72 ký tự) không hợp lệ")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.Company{}, err
	}
	return s.Repo.CreateCompany(ctx, in, string(hash))
}
func (s *CommunityService) OwnedWorkshop(ctx context.Context, user, id string) (*model.Workshop, error) {
	c, err := s.BusinessCompany(ctx, user, true)
	if err != nil {
		return nil, err
	}
	w, err := s.WorkshopRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.CompanyID == nil || *w.CompanyID != c.ID {
		return nil, repository.ErrForbidden
	}
	return w, nil
}
func (s *CommunityService) SaveWorkshop(ctx context.Context, user, id string, in model.CreateWorkshopRequest) (*model.Workshop, error) {
	c, err := s.BusinessCompany(ctx, user, true)
	if err != nil {
		return nil, err
	}
	if err = ValidateWorkshop(&in); err != nil {
		return nil, err
	}
	if id == "" {
		return s.Workshops.CreateFor(ctx, &in, &c.ID, &user)
	}
	w, err := s.OwnedWorkshop(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if w.Status != model.WorkshopDraft && w.Status != model.WorkshopRejected {
		return nil, repository.ErrConflict
	}
	req := WorkshopUpdate(in)
	if err = s.Workshops.Update(ctx, id, &req); err != nil {
		return nil, err
	}
	return s.WorkshopRepo.FindByID(ctx, id)
}
func WorkshopUpdate(in model.CreateWorkshopRequest) model.UpdateWorkshopRequest {
	return model.UpdateWorkshopRequest{Title: &in.Title, Description: &in.Description, Speaker: &in.Speaker, Room: &in.Room, StartTime: &in.StartTime, EndTime: &in.EndTime, RegistrationStartTime: &in.RegistrationStartTime, RegistrationEndTime: &in.RegistrationEndTime, Capacity: &in.Capacity, Price: &in.Price, Summary: &in.Summary, RoomLayoutURL: &in.RoomLayoutURL, CoverURL: &in.CoverURL, Audience: &in.Audience, Benefits: &in.Benefits, Preparation: &in.Preparation, Agenda: &in.Agenda, Format: &in.Format}
}
func (s *CommunityService) SubmitWorkshop(ctx context.Context, user, id string) error {
	w, err := s.OwnedWorkshop(ctx, user, id)
	if err != nil {
		return err
	}
	in := WorkshopRequest(w)
	if err = ValidateWorkshop(&in); err != nil {
		return err
	}
	if !w.StartTime.After(time.Now()) {
		return fmt.Errorf("workshop phải diễn ra trong tương lai")
	}
	return s.Repo.SubmitWorkshop(ctx, user, id)
}
func (s *CommunityService) CreateRevision(ctx context.Context, user, id string, in model.CreateWorkshopRequest, cancel bool, reason string) error {
	if _, err := s.OwnedWorkshop(ctx, user, id); err != nil {
		return err
	}
	if !cancel {
		if err := ValidateWorkshop(&in); err != nil {
			return err
		}
	} else if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("vui lòng nhập lý do hủy")
	}
	return s.Repo.CreateRevision(ctx, user, id, in, cancel, reason)
}
