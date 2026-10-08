package handler_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"unihub-workshop/internal/crypto"
	"unihub-workshop/internal/database"
	"unihub-workshop/internal/handler"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
	"unihub-workshop/internal/service"
)

const testSecret = "integration-only-unihub-secret"

type fixture struct {
	pool                  *pgxpool.Pool
	router                http.Handler
	users                 *repository.UserRepo
	companies             *repository.CommunityRepo
	workshops             *repository.WorkshopRepo
	community             *service.CommunityService
	ws                    *service.WorkshopService
	regs                  *service.RegistrationService
	auth                  *service.AuthService
	admin, student, staff model.User
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("Set TEST_DATABASE_URL to a dedicated PostgreSQL database ending in _test")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatal("TEST_DATABASE_URL must point to a dedicated database ending in _test")
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	t.Setenv("RUN_SEED", "false")
	t.Setenv("MEDIA_DIR", t.TempDir())
	if err = database.RunMigrations(pool); err != nil {
		t.Fatal(err)
	}
	// Migration replay must retain all data and constraints.
	if err = database.RunMigrations(pool); err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, users: repository.NewUserRepo(pool), companies: repository.NewCommunityRepo(pool), workshops: repository.NewWorkshopRepo(pool)}
	f.ws = service.NewWorkshopService(f.workshops, nil)
	f.community = service.NewCommunityService(f.companies, f.ws, f.workshops)
	f.auth = service.NewAuthService(f.users, testSecret, nil)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := crypto.NewRSAProvider(string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))
	if err != nil {
		t.Fatal(err)
	}
	f.regs = service.NewRegistrationService(repository.NewRegistrationRepo(pool), f.workshops, f.users, provider, nil, nil, nil, nil)
	f.admin = f.newUser(t, model.RoleAdmin)
	f.student = f.newUser(t, model.RoleStudent)
	f.staff = f.newUser(t, model.RoleStaff)
	authHandler := handler.NewAuthHandler(f.auth, provider)
	wsHandler := handler.NewWorkshopHandler(f.ws, nil)
	regHandler := handler.NewRegistrationHandler(f.regs)
	checkinHandler := handler.NewCheckinHandler(service.NewCheckinService(repository.NewRegistrationRepo(pool)))
	r := chi.NewRouter()
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/forgot-password", authHandler.ForgotPassword)
	r.Post("/api/v1/auth/reset-password", authHandler.ResetPassword)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(testSecret, f.users.FindByID))
		r.Get("/api/v1/auth/me", authHandler.GetMe)
		r.Get("/api/v1/auth/public-key", authHandler.GetPublicKey)
		r.Post("/api/v1/auth/change-password", authHandler.ChangePassword)
		r.Get("/api/v1/notifications", handler.NewNotificationHandler(service.NewNotificationService(repository.NewNotificationRepo(pool))).GetMyNotifications)
		r.Get("/api/v1/workshops", wsHandler.List)
		r.Get("/api/v1/workshops/{id}", wsHandler.GetByID)
		handler.NewCommunityHandler(f.community).Register(r)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleStudent, model.RoleAdmin))
			r.Post("/api/v1/registrations", regHandler.Register)
			r.Get("/api/v1/registrations/my", regHandler.MyRegistrations)
			r.Get("/api/v1/registrations/status/{correlationId}", regHandler.GetStatus)
			r.Post("/api/v1/registrations/{id}/cancel", regHandler.Cancel)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleStaff, model.RoleAdmin))
			r.Post("/api/v1/checkin/sync", checkinHandler.BulkSync)
		})
	})
	f.router = r
	return f
}

func (f *fixture) newUser(t *testing.T, role model.Role) model.User {
	t.Helper()
	id := uuid.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), "INSERT INTO users(id,user_id,full_name,email,password_hash,role) VALUES($1,$2,$3,$4,$5,$6)", id, "test-"+id, "Test "+string(role), id+"@example.test", string(hash), role)
	if err != nil {
		t.Fatal(err)
	}
	u, err := f.users.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return *u
}
func token(t *testing.T, u model.User) string {
	t.Helper()
	value, err := middleware.GenerateJWT(testSecret, u.ID, u.Role, u.AuthVersion)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func (f *fixture) request(t *testing.T, userToken, method, path string, body any, want int, target any) {
	t.Helper()
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if userToken != "" {
		req.Header.Set("Authorization", "Bearer "+userToken)
	}
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s: want %d got %d: %s", method, path, want, recorder.Code, recorder.Body.String())
	}
	if target != nil {
		var response struct {
			Data json.RawMessage `json:"data"`
		}
		if err = json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(response.Data, target); err != nil {
			t.Fatalf("decode %s: %v", recorder.Body.String(), err)
		}
	}
}
func (f *fixture) company(t *testing.T, approved bool) model.Company {
	t.Helper()
	var company model.Company
	id := uuid.NewString()
	f.request(t, token(t, f.admin), "POST", "/api/v1/admin/business-accounts", model.BusinessAccountInput{CompanyInput: model.CompanyInput{Name: "Doanh nghiệp " + id, Slug: "business-" + id, Industry: "Công nghệ"}, Identifier: "business-" + id, Email: id + "@business.test", Password: "password123"}, 200, &company)
	if approved {
		f.request(t, token(t, f.admin), "POST", "/api/v1/admin/companies/"+company.ID+"/review", map[string]string{"status": "APPROVED", "reason": "Đã kiểm tra hồ sơ"}, 200, nil)
		company.Status = "APPROVED"
	}
	return company
}
func (f *fixture) owner(t *testing.T, company model.Company) model.User {
	t.Helper()
	u, err := f.users.FindByID(context.Background(), company.OwnerUserID)
	if err != nil {
		t.Fatal(err)
	}
	return *u
}
func workshopInput() model.CreateWorkshopRequest {
	now := time.Now().UTC().Truncate(time.Second)
	return model.CreateWorkshopRequest{Title: "Workshop sắp tới", Description: "Thực hành Go và chia sẻ nghề nghiệp", Speaker: "Chuyên gia Go", Room: "Phòng A1", StartTime: now.Add(48 * time.Hour).Format(time.RFC3339), EndTime: now.Add(50 * time.Hour).Format(time.RFC3339), RegistrationStartTime: now.Add(-time.Hour).Format(time.RFC3339), RegistrationEndTime: now.Add(47 * time.Hour).Format(time.RFC3339), Capacity: 12, Format: "OFFLINE", Agenda: "09:00 Giới thiệu; 10:00 Thực hành", Audience: "Sinh viên", Benefits: "Kiến thức thực tế", Preparation: "Máy tính cá nhân"}
}
func (f *fixture) published(t *testing.T, company model.Company) *model.Workshop {
	t.Helper()
	owner := f.owner(t, company)
	var w model.Workshop
	f.request(t, token(t, owner), "POST", "/api/v1/business/workshops", workshopInput(), 200, &w)
	f.request(t, token(t, owner), "POST", "/api/v1/business/workshops/"+w.ID+"/submit", nil, 200, nil)
	f.request(t, token(t, f.admin), "POST", "/api/v1/admin/workshops/"+w.ID+"/review", map[string]string{"status": "PUBLISHED", "reason": "Đã duyệt nội dung"}, 200, nil)
	current, err := f.workshops.FindByID(context.Background(), w.ID)
	if err != nil {
		t.Fatal(err)
	}
	return current
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommunityBusinessWorkflow(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	admin := token(t, f.admin)
	student := token(t, f.student)
	staff := token(t, f.staff)
	for _, path := range []string{"/api/v1/feed", "/api/v1/companies", "/api/v1/workshops", "/api/v1/auth/public-key", "/api/v1/admin/companies", "/api/v1/business/profile"} {
		f.request(t, "", "GET", path, nil, 401, nil)
	}
	c := f.company(t, false)
	business := token(t, f.owner(t, c))
	for _, identity := range []string{student, staff, business} {
		f.request(t, identity, "GET", "/api/v1/admin/companies", nil, 403, nil)
	}
	for _, identity := range []string{student, staff, admin} {
		f.request(t, identity, "POST", "/api/v1/business/workshops", workshopInput(), 403, nil)
	}
	postInput := model.PostInput{Title: "Chia sẻ kiến thức", Content: "Một câu chuyện học tập", Kind: "COMMUNITY", Status: "PUBLISHED", Topic: uuid.NewString()}
	f.request(t, business, "POST", "/api/v1/posts", postInput, 403, nil)
	f.request(t, business, "POST", "/api/v1/business/workshops", workshopInput(), 403, nil)
	f.request(t, staff, "POST", "/api/v1/posts", postInput, 403, nil)
	f.request(t, admin, "POST", "/api/v1/admin/companies/"+c.ID+"/review", map[string]string{"status": "APPROVED", "reason": "Xác minh hồ sơ"}, 200, nil)
	c2 := f.company(t, true)
	otherBusiness := token(t, f.owner(t, c2))
	var w model.Workshop
	f.request(t, business, "POST", "/api/v1/business/workshops", workshopInput(), 200, &w)
	if w.Status != model.WorkshopDraft || w.CompanyID == nil || *w.CompanyID != c.ID || w.Description == nil {
		t.Fatalf("invalid draft: %+v", w)
	}
	f.request(t, student, "GET", "/api/v1/workshops/"+w.ID, nil, 404, nil)
	f.request(t, otherBusiness, "PATCH", "/api/v1/business/workshops/"+w.ID, workshopInput(), 403, nil)
	input := workshopInput()
	input.Title = "Workshop đã chỉnh sửa"
	input.Capacity = 15
	f.request(t, business, "PATCH", "/api/v1/business/workshops/"+w.ID, input, 200, &w)
	if w.Title != input.Title || w.AvailableSeats != 15 {
		t.Fatalf("draft update failed: %+v", w)
	}
	f.request(t, business, "POST", "/api/v1/business/workshops/"+w.ID+"/submit", nil, 200, nil)
	f.request(t, business, "PATCH", "/api/v1/business/workshops/"+w.ID, input, 409, nil)
	f.request(t, admin, "POST", "/api/v1/admin/workshops/"+w.ID+"/review", map[string]string{"status": "PUBLISHED", "reason": "Đã duyệt"}, 200, nil)
	f.request(t, student, "GET", "/api/v1/workshops/"+w.ID, nil, 200, &w)
	announcement := postInput
	announcement.Kind = "WORKSHOP_ANNOUNCEMENT"
	announcement.WorkshopID = &w.ID
	f.request(t, student, "POST", "/api/v1/posts", announcement, 403, nil)
	f.request(t, otherBusiness, "POST", "/api/v1/posts", announcement, 403, nil)
	var p model.Post
	f.request(t, business, "POST", "/api/v1/posts", announcement, 200, &p)
	if p.Workshop == nil || p.Workshop.Agenda != input.Agenda {
		t.Fatalf("announcement missing live workshop details: %+v", p)
	}
	f.request(t, otherBusiness, "PATCH", "/api/v1/posts/"+p.ID, announcement, 403, nil)
	for i := 0; i < 2; i++ {
		f.request(t, student, "PUT", "/api/v1/posts/"+p.ID+"/like", nil, 200, nil)
		f.request(t, student, "PUT", "/api/v1/posts/"+p.ID+"/bookmark", nil, 200, nil)
		f.request(t, student, "PUT", "/api/v1/companies/"+c.ID+"/follow", nil, 200, nil)
	}
	f.request(t, student, "GET", "/api/v1/posts/"+p.ID, nil, 200, &p)
	if p.Likes != 1 || !p.Liked || !p.Bookmarked {
		t.Fatalf("idempotency failed: %+v", p)
	}
	var follow model.Company
	f.request(t, student, "GET", "/api/v1/companies/"+c.ID, nil, 200, &follow)
	if follow.Followers != 1 || !follow.Following {
		t.Fatalf("invalid follow: %+v", follow)
	}
	var comment struct {
		ID string `json:"id"`
	}
	f.request(t, student, "POST", "/api/v1/posts/"+p.ID+"/comments", map[string]string{"content": "Workshop hữu ích!"}, 200, &comment)
	var reply struct {
		ID string `json:"id"`
	}
	f.request(t, business, "POST", "/api/v1/posts/"+p.ID+"/comments", map[string]string{"content": "Cảm ơn bạn", "parent_id": comment.ID}, 200, &reply)
	f.request(t, student, "POST", "/api/v1/posts/"+p.ID+"/comments", map[string]string{"content": "Trả lời cấp ba", "parent_id": reply.ID}, 409, nil)
	var otherPost model.Post
	f.request(t, student, "POST", "/api/v1/posts", postInput, 200, &otherPost)
	f.request(t, student, "POST", "/api/v1/posts/"+otherPost.ID+"/comments", map[string]string{"content": "Sai bài cha", "parent_id": comment.ID}, 409, nil)
	f.request(t, otherBusiness, "PATCH", "/api/v1/comments/"+comment.ID, map[string]string{"content": "Sửa trái phép"}, 403, nil)
	asset := uuid.NewString()
	_, err := f.pool.Exec(ctx, "INSERT INTO media_assets(id,owner_user_id,mime,size_bytes) VALUES($1,$2,'image/png',10)", asset, f.student.ID)
	must(t, err)
	stolen := announcement
	stolen.MediaIDs = []string{asset}
	f.request(t, business, "POST", "/api/v1/posts", stolen, 403, nil)
	var page model.FeedPage
	f.request(t, student, "GET", "/api/v1/feed?topic="+url.QueryEscape(postInput.Topic)+"&limit=1", nil, 200, &page)
	if len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("bad first page: %+v", page)
	}
	first := page.Items[0].ID
	f.request(t, student, "GET", "/api/v1/feed?topic="+url.QueryEscape(postInput.Topic)+"&limit=1&cursor="+url.QueryEscape(page.NextCursor), nil, 200, &page)
	if len(page.Items) != 1 || page.Items[0].ID == first || page.NextCursor != "" {
		t.Fatalf("unstable cursor: %+v", page)
	}
	f.request(t, student, "POST", "/api/v1/reports", map[string]string{"post_id": p.ID, "reason": "Nội dung cần kiểm tra"}, 200, nil)
	f.request(t, admin, "POST", "/api/v1/admin/posts/"+p.ID+"/moderate", map[string]string{"status": "HIDDEN", "reason": "Chờ chỉnh sửa"}, 200, nil)
	f.request(t, student, "GET", "/api/v1/posts/"+p.ID, nil, 404, nil)
	f.request(t, business, "PATCH", "/api/v1/posts/"+p.ID, announcement, 403, nil)
	f.request(t, student, "PUT", "/api/v1/posts/"+p.ID+"/like", nil, 404, nil)
	f.request(t, admin, "POST", "/api/v1/admin/companies/"+c.ID+"/review", map[string]string{"status": "SUSPENDED", "reason": "Tạm đình chỉ"}, 200, nil)
	f.request(t, business, "POST", "/api/v1/posts", postInput, 403, nil)
	f.request(t, student, "GET", "/api/v1/workshops/"+w.ID, nil, 404, nil)
}

func TestRegistrationConcurrencyAndCheckin(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	company := f.company(t, true)
	w := f.published(t, company)
	// Concurrent requests for one student must reserve exactly one place and reuse an ID.
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	failures := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := f.regs.EnqueueRegistration(ctx, f.student.ID, w.ID)
			if err != nil {
				failures <- err
			} else {
				ids <- id
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	var id string
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate active reservation")
		}
		id = got
	}
	current, err := f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 11 {
		t.Fatalf("expected one reserved seat, got %d", current.AvailableSeats)
	}
	if _, err = f.regs.GetStatus(ctx, f.staff.ID, id); err == nil {
		t.Fatal("another user can read reservation status")
	}
	msg := model.QueueMessage{CorrelationID: id, UserID: f.student.ID, WorkshopID: w.ID, Action: "REGISTER"}
	must(t, f.regs.ProcessRegistration(ctx, msg))
	must(t, f.regs.ProcessRegistration(ctx, msg))
	response, err := f.regs.GetStatus(ctx, f.student.ID, id)
	must(t, err)
	if response.Status != "SUCCESS" || response.Registration == nil || response.Registration.TicketSignature == nil {
		t.Fatalf("missing signed ticket: %+v", response)
	}
	var notices, outbox int
	must(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE event_id=$1", "REG_SUCCESS_"+id).Scan(&notices))
	must(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM notification_outbox WHERE event_id=$1", "REG_SUCCESS_"+id).Scan(&outbox))
	if notices != 1 || outbox != 1 {
		t.Fatalf("notice/outbox duplicated: %d %d", notices, outbox)
	}
	// Capacity changes preserve the reserved place; cancellation releases it only once.
	cap := 15
	must(t, f.ws.Update(ctx, w.ID, &model.UpdateWorkshopRequest{Capacity: &cap}))
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 14 {
		t.Fatal("capacity change lost reservation")
	}
	regID := response.Registration.ID
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := f.regs.CancelRegistration(ctx, f.student.ID, regID); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 15 {
		t.Fatalf("cancellation released wrong seat count: %d", current.AvailableSeats)
	}
	must(t, f.regs.ProcessRegistration(ctx, msg))
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 15 {
		t.Fatal("redelivery resurrected cancelled ticket")
	}
	id2, err := f.regs.EnqueueRegistration(ctx, f.student.ID, w.ID)
	must(t, err)
	msg.CorrelationID = id2
	must(t, f.regs.ProcessRegistration(ctx, msg))
	// Assigned staff only; Unix milliseconds are rejected and repeated sync preserves earliest time.
	now := time.Now().Unix()
	record := model.OfflineCheckinRecord{ID: uuid.NewString(), StudentID: f.student.StudentID, WorkshopID: w.ID, ScannedAt: now}
	staff := token(t, f.staff)
	admin := token(t, f.admin)
	f.request(t, staff, "POST", "/api/v1/checkin/sync", model.BulkCheckinRequest{Records: []model.OfflineCheckinRecord{record}}, 403, nil)
	f.request(t, admin, "PUT", "/api/v1/admin/workshops/"+w.ID+"/staff/"+f.staff.ID, nil, 200, nil)
	var syncResult struct{ Synced, Failed []string }
	f.request(t, staff, "POST", "/api/v1/checkin/sync", model.BulkCheckinRequest{Records: []model.OfflineCheckinRecord{record}}, 200, &syncResult)
	if len(syncResult.Synced) != 1 || len(syncResult.Failed) != 0 {
		t.Fatalf("sync failed: %+v", syncResult)
	}
	record.ID = uuid.NewString()
	record.ScannedAt = now - 60
	f.request(t, staff, "POST", "/api/v1/checkin/sync", model.BulkCheckinRequest{Records: []model.OfflineCheckinRecord{record}}, 200, &syncResult)
	response, err = f.regs.GetStatus(ctx, f.student.ID, id2)
	must(t, err)
	if response.Registration.CheckedInAt == nil || response.Registration.CheckedInAt.Unix() != now-60 {
		t.Fatal("earliest check-in not retained")
	}
	if err = f.regs.CancelRegistration(ctx, f.student.ID, response.Registration.ID); err == nil {
		t.Fatal("used ticket can be cancelled")
	}
	record.ID = uuid.NewString()
	record.ScannedAt = time.Now().UnixMilli()
	f.request(t, staff, "POST", "/api/v1/checkin/sync", model.BulkCheckinRequest{Records: []model.OfflineCheckinRecord{record}}, 200, &syncResult)
	if len(syncResult.Failed) != 1 {
		t.Fatal("milliseconds accepted as seconds")
	}
	// More students than available seats never oversell, even with concurrent enqueue.
	students := make([]model.User, 30)
	for i := range students {
		students[i] = f.newUser(t, model.RoleStudent)
	}
	successes := make(chan model.QueueMessage, len(students))
	for _, u := range students {
		wg.Add(1)
		go func(u model.User) {
			defer wg.Done()
			id, err := f.regs.EnqueueRegistration(ctx, u.ID, w.ID)
			if err == nil {
				successes <- model.QueueMessage{CorrelationID: id, UserID: u.ID, WorkshopID: w.ID, Action: "REGISTER"}
			}
		}(u)
	}
	wg.Wait()
	close(successes)
	var pending []model.QueueMessage
	for m := range successes {
		pending = append(pending, m)
	}
	if len(pending) != 14 {
		t.Fatalf("expected 14 seats, reserved %d", len(pending))
	}
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 0 {
		t.Fatal("oversold or lost seats")
	}
	tooSmall := 14
	if err = f.ws.Update(ctx, w.ID, &model.UpdateWorkshopRequest{Capacity: &tooSmall}); err == nil {
		t.Fatal("capacity lowered below held seats")
	}
	// A failed reservation restores exactly one seat; retries remain bounded across redelivery.
	failed := pending[0]
	_, err = f.pool.Exec(ctx, "UPDATE registration_requests SET attempts=3 WHERE id=$1", failed.CorrelationID)
	must(t, err)
	if !f.regs.FailAfterRetries(ctx, failed) || !f.regs.FailAfterRetries(ctx, failed) {
		t.Fatal("retry exhaustion failed")
	}
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 1 {
		t.Fatal("failure double-released seat")
	}
	must(t, f.regs.ProcessRegistration(ctx, failed))
	current, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if current.AvailableSeats != 1 {
		t.Fatal("failed reservation was processed again")
	}
}

func TestWorkshopRevisionAndCancellation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	company := f.company(t, true)
	w := f.published(t, company)
	owner := token(t, f.owner(t, company))
	admin := token(t, f.admin)
	id, err := f.regs.EnqueueRegistration(ctx, f.student.ID, w.ID)
	must(t, err)
	must(t, f.regs.ProcessRegistration(ctx, model.QueueMessage{CorrelationID: id, UserID: f.student.ID, WorkshopID: w.ID}))
	input := workshopInput()
	input.Title = "Địa điểm và chương trình mới"
	input.Capacity = 20
	input.Room = "Phòng B2"
	f.request(t, owner, "POST", "/api/v1/business/workshops/"+w.ID+"/revisions", map[string]any{"payload": input}, 200, nil)
	var revisions []model.Revision
	f.request(t, admin, "GET", "/api/v1/admin/workshop-revisions", nil, 200, &revisions)
	revision := revisions[0]
	before, err := f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if before.Title == input.Title {
		t.Fatal("unreviewed edit changed live workshop")
	}
	f.request(t, admin, "POST", "/api/v1/admin/workshop-revisions/"+revision.ID+"/review", map[string]string{"status": "APPROVED", "reason": "Đã kiểm tra lịch mới"}, 200, nil)
	after, err := f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if after.Title != input.Title || after.AvailableSeats != 19 {
		t.Fatalf("bad revision: %+v", after)
	}
	f.request(t, admin, "POST", "/api/v1/admin/workshop-revisions/"+revision.ID+"/review", map[string]string{"status": "APPROVED", "reason": "Gửi lặp"}, 409, nil)
	another := f.newUser(t, model.RoleStudent)
	pending, err := f.regs.EnqueueRegistration(ctx, another.ID, w.ID)
	must(t, err)
	f.request(t, owner, "POST", "/api/v1/business/workshops/"+w.ID+"/cancellation-requests", map[string]string{"reason": "Không thể tổ chức"}, 200, nil)
	f.request(t, admin, "GET", "/api/v1/admin/workshop-revisions", nil, 200, &revisions)
	revision = revisions[0]
	f.request(t, admin, "POST", "/api/v1/admin/workshop-revisions/"+revision.ID+"/review", map[string]string{"status": "APPROVED", "reason": "Đồng ý hủy, đã thông báo sinh viên"}, 200, nil)
	after, err = f.workshops.FindByID(ctx, w.ID)
	must(t, err)
	if after.Status != model.WorkshopCancelled || after.AvailableSeats != after.Capacity {
		t.Fatal("workshop not cancelled atomically")
	}
	result, err := f.regs.GetStatus(ctx, another.ID, pending)
	must(t, err)
	if result.Status != "FAILED" {
		t.Fatal("pending reservation not cancelled")
	}
	must(t, f.regs.ProcessRegistration(ctx, model.QueueMessage{CorrelationID: pending, UserID: another.ID, WorkshopID: w.ID}))
	regs, err := f.regs.GetUserRegistrations(ctx, f.student.ID)
	must(t, err)
	if regs[0].Status != model.RegCancelled {
		t.Fatal("ticket still valid after cancellation")
	}
}

func TestPasswordResetAndSessionRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	original := token(t, f.student)
	f.request(t, original, "POST", "/api/v1/auth/change-password", map[string]string{"old_password": "password123", "new_password": "password456"}, 200, nil)
	f.request(t, original, "GET", "/api/v1/feed", nil, 401, nil)
	login, err := f.auth.Login(ctx, &model.LoginRequest{StudentID: f.student.StudentID, Password: "password456"})
	must(t, err)
	f.request(t, login.Token, "GET", "/api/v1/feed", nil, 200, nil)
	reset := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	sum := sha256.Sum256([]byte(reset))
	must(t, f.users.StoreReset(ctx, f.student.ID, hex.EncodeToString(sum[:])))
	must(t, f.auth.ResetPassword(ctx, reset, "password789"))
	if err = f.auth.ResetPassword(ctx, reset, "anotherpassword"); err == nil {
		t.Fatal("reset token reused")
	}
	f.request(t, login.Token, "GET", "/api/v1/feed", nil, 401, nil)
	if _, err = f.auth.Login(ctx, &model.LoginRequest{StudentID: f.student.StudentID, Password: "password789"}); err != nil {
		t.Fatal(err)
	}
	// Requesting recovery does not change the password and has the same response for missing users.
	f.request(t, "", "POST", "/api/v1/auth/forgot-password", map[string]string{"identifier": f.student.StudentID}, 200, nil)
	f.request(t, "", "POST", "/api/v1/auth/forgot-password", map[string]string{"identifier": "missing-" + uuid.NewString()}, 200, nil)
	if _, err = f.auth.Login(ctx, &model.LoginRequest{StudentID: f.student.StudentID, Password: "password789"}); err != nil {
		t.Fatal("forgot-password changed password before confirmation")
	}
	var secretCount int
	must(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE user_id=$1 AND (content LIKE '%password789%' OR content LIKE '%reset-password?token=%')", f.student.ID).Scan(&secretCount))
	if secretCount != 0 {
		t.Fatal("password reset secret leaked into notifications")
	}
}

// Optional local API for checking the web UI against the real repositories.
// Never enabled in a normal test run or in the application server.
func TestBrowserPreview(t *testing.T) {
	addr := os.Getenv("PREVIEW_ADDR")
	if addr == "" {
		t.Skip("PREVIEW_ADDR is unset")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("preview must bind to loopback")
	}
	f := setup(t)
	company := f.company(t, true)
	owner := f.owner(t, company)
	w := f.published(t, company)
	_, err := f.community.SavePost(context.Background(), owner.ID, model.RoleBusiness, "", model.PostInput{Title: "Workshop Go: thực hành và định hướng nghề nghiệp", Content: "Cùng doanh nghiệp tìm hiểu Go, thực hành xây dựng API và trao đổi về công việc sau tốt nghiệp.", Kind: "WORKSHOP_ANNOUNCEMENT", Status: "PUBLISHED", Topic: "Công nghệ", WorkshopID: &w.ID})
	must(t, err)
	mux := http.NewServeMux()
	mux.Handle("/", f.router)
	mux.HandleFunc("GET /__fixtures", func(rw http.ResponseWriter, r *http.Request) {
		json.NewEncoder(rw).Encode(map[string]string{"student": f.student.StudentID, "staff": f.staff.StudentID, "business": owner.StudentID, "admin": f.admin.StudentID, "company": company.Slug, "password": "password123"})
	})
	listener, err := net.Listen("tcp", addr)
	must(t, err)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	t.Cleanup(func() { server.Close() })
	go server.Serve(listener)
	t.Log("UI preview API:", addr)
	time.Sleep(10 * time.Minute)
}

type fakeEmail struct {
	calls int
	fail  bool
}

func (e *fakeEmail) Channel() model.NotificationChannel { return model.ChannelEmail }
func (e *fakeEmail) Send(ctx context.Context, n *model.Notification) error {
	e.calls++
	if e.fail {
		return fmt.Errorf("temporary SMTP error")
	}
	return nil
}
func TestNotificationDeliveryRetries(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	email := &fakeEmail{fail: true}
	notifications := service.NewNotificationService(repository.NewNotificationRepo(f.pool), email)
	event := model.NotificationEvent{EventID: uuid.NewString(), UserID: f.student.ID, WorkshopTitle: "Workshop <script>unsafe</script>", Type: "REGISTRATION_SUCCESS", Metadata: map[string]string{"email_only": "true"}}
	if err := notifications.Dispatch(ctx, event); err == nil {
		t.Fatal("SMTP error was swallowed")
	}
	email.fail = false
	must(t, notifications.Dispatch(ctx, event))
	must(t, notifications.Dispatch(ctx, event))
	if email.calls != 2 {
		t.Fatal("already-sent email delivered again")
	}
	var rows int
	must(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE event_id=$1 AND channel='EMAIL' AND status='SENT'", event.EventID).Scan(&rows))
	if rows != 1 {
		t.Fatal("email not recorded idempotently")
	}
	reset := model.NotificationEvent{EventID: uuid.NewString(), UserID: f.student.ID, Type: "PASSWORD_RESET", Metadata: map[string]string{"reset_url": "http://localhost/reset-password?token=secret"}}
	must(t, notifications.Dispatch(ctx, reset))
	must(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE event_id=$1", reset.EventID).Scan(&rows))
	if rows != 0 {
		t.Fatal("reset secret was persisted in notifications")
	}
}
