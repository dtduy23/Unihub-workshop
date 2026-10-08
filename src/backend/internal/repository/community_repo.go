package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
	"unihub-workshop/internal/model"
)

var ErrForbidden = errors.New("không có quyền thực hiện thao tác")
var ErrConflict = errors.New("nội dung đang được duyệt hoặc không thể thay đổi")

type CommunityRepo struct{ pool *pgxpool.Pool }

func NewCommunityRepo(pool *pgxpool.Pool) *CommunityRepo { return &CommunityRepo{pool: pool} }
func decodeRow[T any](row pgx.Row) (T, error) {
	var value T
	var data []byte
	if err := row.Scan(&data); err != nil {
		return value, err
	}
	err := json.Unmarshal(data, &value)
	return value, err
}

const companySelect = `SELECT to_jsonb(c)||jsonb_build_object('followers',(SELECT count(*) FROM company_follows f WHERE f.company_id=c.id),'following',EXISTS(SELECT 1 FROM company_follows f WHERE f.company_id=c.id AND f.user_id=$1)) FROM companies c `

func (r *CommunityRepo) Company(ctx context.Context, userID, id string, owner bool) (model.Company, error) {
	where := "c.id::text=$2 OR c.slug=$2"
	if owner {
		where = "c.owner_user_id=$2"
	}
	return decodeRow[model.Company](r.pool.QueryRow(ctx, companySelect+"WHERE "+where, userID, id))
}
func (r *CommunityRepo) Companies(ctx context.Context, userID string, admin bool) ([]model.Company, error) {
	q := companySelect
	if !admin {
		q += "WHERE c.status='APPROVED' "
	}
	rows, err := r.pool.Query(ctx, q+"ORDER BY c.created_at DESC LIMIT 100", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Company{}
	for rows.Next() {
		v, e := decodeRow[model.Company](rows)
		if e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *CommunityRepo) CreateCompany(ctx context.Context, in model.BusinessAccountInput, passwordHash string) (model.Company, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Company{}, err
	}
	defer tx.Rollback(ctx)
	var userID, id string
	err = tx.QueryRow(ctx, "INSERT INTO users(user_id,password_hash,full_name,email,role) VALUES($1,$2,$3,$4,'BUSINESS') RETURNING id", in.Identifier, passwordHash, in.Name, in.Email).Scan(&userID)
	if err != nil {
		return model.Company{}, err
	}
	err = tx.QueryRow(ctx, "INSERT INTO companies(owner_user_id,name,slug,description,industry,website,address,contact) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id", userID, in.Name, in.Slug, in.Description, in.Industry, in.Website, in.Address, in.Contact).Scan(&id)
	if err != nil {
		return model.Company{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Company{}, err
	}
	return r.Company(ctx, userID, id, false)
}
func (r *CommunityRepo) UpdateCompany(ctx context.Context, userID string, in model.CompanyInput) error {
	tag, err := r.pool.Exec(ctx, "UPDATE companies SET name=$1,description=$2,industry=$3,website=$4,address=$5,contact=$6,logo_url=$7,cover_url=$8,updated_at=now(),status=CASE WHEN status='REJECTED' THEN 'PENDING' ELSE status END WHERE owner_user_id=$9 AND status<>'SUSPENDED'", in.Name, in.Description, in.Industry, in.Website, in.Address, in.Contact, in.LogoURL, in.CoverURL, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return err
}
func (r *CommunityRepo) ReviewCompany(ctx context.Context, actorID, id, status, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owner string
	err = tx.QueryRow(ctx, "UPDATE companies SET status=$1,review_reason=$2,updated_at=now() WHERE id=$3 RETURNING owner_user_id", status, reason, id).Scan(&owner)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, actorID, id, "COMPANY_"+status, reason); err != nil {
		return err
	}
	if err = notify(ctx, tx, owner, "Hồ sơ doanh nghiệp: "+status, reason, "/business"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func audit(ctx context.Context, tx pgx.Tx, actor, id, action, reason string) error {
	_, err := tx.Exec(ctx, "INSERT INTO moderation_actions(actor_id,target_id,action,reason) VALUES($1,$2,$3,$4)", actor, id, action, reason)
	return err
}
func notify(ctx context.Context, tx pgx.Tx, user, title, content, link string) error {
	_, err := tx.Exec(ctx, "INSERT INTO notifications(user_id,channel,title,content,status,event_id,link) VALUES($1,'WEB',$2,$3,'SENT',$4,$5)", user, title, content, uuid.NewString(), link)
	return err
}

const postSelect = `SELECT to_jsonb(p)||jsonb_build_object('author_name',u.full_name,'company_name',COALESCE(c.name,''),'company_slug',COALESCE(c.slug,''),'workshop',to_jsonb(w),
 'likes',(SELECT count(*) FROM post_likes l WHERE l.post_id=p.id),'comments',(SELECT count(*) FROM comments cm WHERE cm.post_id=p.id AND cm.status='PUBLISHED'),
 'liked',EXISTS(SELECT 1 FROM post_likes l WHERE l.post_id=p.id AND l.user_id=$1),'bookmarked',EXISTS(SELECT 1 FROM post_bookmarks b WHERE b.post_id=p.id AND b.user_id=$1))
 FROM posts p JOIN users u ON u.id=p.author_user_id LEFT JOIN companies c ON c.id=p.company_id LEFT JOIN workshops w ON w.id=p.workshop_id `
const postVisible = `p.status='PUBLISHED' AND (p.company_id IS NULL OR c.status='APPROVED') AND (p.workshop_id IS NULL OR w.status IN ('PUBLISHED','CLOSED','CANCELLED'))`

func (r *CommunityRepo) Post(ctx context.Context, userID, id string, admin bool) (model.Post, error) {
	q := postSelect + "WHERE p.id=$2 AND (" + postVisible + " OR p.author_user_id=$1"
	if admin {
		q += " OR true"
	}
	q += ")"
	return decodeRow[model.Post](r.pool.QueryRow(ctx, q, userID, id))
}

type feedCursor struct {
	Time time.Time `json:"t"`
	ID   string    `json:"i"`
}

func (r *CommunityRepo) Posts(ctx context.Context, userID, mode, company, cursor, topic string, limit int) (model.FeedPage, error) {
	q := postSelect + "WHERE "
	args := []any{userID}
	if mode == "mine" {
		q += "p.author_user_id=$1 AND p.status<>'DELETED'"
	} else {
		q += postVisible
	}
	if mode == "following" {
		q += " AND EXISTS(SELECT 1 FROM company_follows f WHERE f.company_id=p.company_id AND f.user_id=$1)"
	}
	if mode == "saved" {
		q += " AND EXISTS(SELECT 1 FROM post_bookmarks b WHERE b.post_id=p.id AND b.user_id=$1)"
	}
	if company != "" {
		args = append(args, company)
		q += fmt.Sprintf(" AND p.company_id=$%d", len(args))
	}
	if topic != "" {
		args = append(args, topic)
		q += fmt.Sprintf(" AND p.topic=$%d", len(args))
	}
	if cursor != "" {
		var c feedCursor
		data, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil || json.Unmarshal(data, &c) != nil || c.Time.IsZero() {
			return model.FeedPage{}, fmt.Errorf("invalid cursor")
		}
		if _, e = uuid.Parse(c.ID); e != nil {
			return model.FeedPage{}, fmt.Errorf("invalid cursor")
		}
		args = append(args, c.Time, c.ID)
		q += fmt.Sprintf(" AND (COALESCE(p.published_at,p.created_at),p.id)<($%d,$%d)", len(args)-1, len(args))
	}
	args = append(args, limit+1)
	q += fmt.Sprintf(" ORDER BY COALESCE(p.published_at,p.created_at) DESC,p.id DESC LIMIT $%d", len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return model.FeedPage{}, err
	}
	defer rows.Close()
	page := model.FeedPage{Items: []model.Post{}}
	for rows.Next() {
		v, e := decodeRow[model.Post](rows)
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, v)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		t := last.CreatedAt
		if last.PublishedAt != nil {
			t = *last.PublishedAt
		}
		data, _ := json.Marshal(feedCursor{t, last.ID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}
func (r *CommunityRepo) SavePost(ctx context.Context, userID string, role model.Role, id string, in model.PostInput) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var companyID *string
	if role == model.RoleBusiness {
		var cid, status string
		if err = tx.QueryRow(ctx, "SELECT id,status FROM companies WHERE owner_user_id=$1 FOR SHARE", userID).Scan(&cid, &status); err != nil {
			return "", err
		}
		if status != "APPROVED" {
			return "", ErrForbidden
		}
		companyID = &cid
	}
	var oldStatus string
	if id != "" {
		var author string
		if err = tx.QueryRow(ctx, "SELECT author_user_id,status FROM posts WHERE id=$1 FOR UPDATE", id).Scan(&author, &oldStatus); err != nil {
			return "", err
		}
		if author != userID {
			return "", ErrForbidden
		}
		if oldStatus == "HIDDEN" || oldStatus == "DELETED" {
			return "", ErrForbidden
		}
	}
	if len(in.MediaIDs) > 0 {
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM media_assets WHERE id=ANY($1::uuid[]) AND owner_user_id=$2", in.MediaIDs, userID).Scan(&count); err != nil {
			return "", err
		}
		if count != len(in.MediaIDs) {
			return "", ErrForbidden
		}
	}
	if in.WorkshopID != nil {
		var owner *string
		var state, companyState string
		err = tx.QueryRow(ctx, "SELECT w.company_id,w.status,COALESCE(c.status,'APPROVED') FROM workshops w LEFT JOIN companies c ON c.id=w.company_id WHERE w.id=$1 FOR SHARE OF w", *in.WorkshopID).Scan(&owner, &state, &companyState)
		if err != nil {
			return "", err
		}
		if state != "PUBLISHED" || companyState != "APPROVED" {
			return "", ErrConflict
		}
		if role != model.RoleAdmin && (companyID == nil || owner == nil || *companyID != *owner) {
			return "", ErrForbidden
		}
		if role == model.RoleAdmin {
			companyID = owner
		}
	}
	if id == "" {
		id = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO posts(id,author_user_id,company_id,workshop_id,kind,title,content,topic,media_ids,status,published_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,CASE WHEN $10='PUBLISHED' THEN now() END)`, id, userID, companyID, in.WorkshopID, in.Kind, in.Title, in.Content, in.Topic, in.MediaIDs, in.Status)
	} else {
		_, err = tx.Exec(ctx, `UPDATE posts SET company_id=$1,workshop_id=$2,kind=$3,title=$4,content=$5,topic=$6,media_ids=$7,status=$8,updated_at=now(),published_at=CASE WHEN $8='PUBLISHED' THEN COALESCE(published_at,now()) ELSE published_at END WHERE id=$9`, companyID, in.WorkshopID, in.Kind, in.Title, in.Content, in.Topic, in.MediaIDs, in.Status, id)
	}
	if err != nil {
		return "", err
	}
	if in.Status == "PUBLISHED" && oldStatus != "PUBLISHED" && companyID != nil {
		_, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,channel,title,content,status,event_id,link) SELECT user_id,'WEB',$1,$2,'SENT',$3||':'||user_id::text,$4 FROM company_follows WHERE company_id=$5 AND user_id<>$6 ON CONFLICT(event_id,channel) DO NOTHING`, in.Title, "Doanh nghiệp bạn theo dõi có bài viết mới.", "POST_"+id, "/posts/"+id, *companyID, userID)
		if err != nil {
			return "", err
		}
	}
	return id, tx.Commit(ctx)
}
func (r *CommunityRepo) DeletePost(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE posts SET status='DELETED',updated_at=now() WHERE id=$1 AND author_user_id=$2", id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return err
}
func (r *CommunityRepo) Interact(ctx context.Context, userID, id, kind string, remove bool) error {
	table := "post_likes"
	if kind == "bookmark" {
		table = "post_bookmarks"
	}
	if remove {
		_, err := r.pool.Exec(ctx, "DELETE FROM "+table+" WHERE post_id=$1 AND user_id=$2", id, userID)
		return err
	}
	_, err := r.pool.Exec(ctx, "INSERT INTO "+table+"(post_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING", id, userID)
	return err
}
func (r *CommunityRepo) Follow(ctx context.Context, userID, id string, remove bool) error {
	if remove {
		_, err := r.pool.Exec(ctx, "DELETE FROM company_follows WHERE company_id=$1 AND user_id=$2", id, userID)
		return err
	}
	_, err := r.pool.Exec(ctx, "INSERT INTO company_follows(company_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING", id, userID)
	return err
}
func (r *CommunityRepo) Comments(ctx context.Context, id string) ([]model.Comment, error) {
	rows, err := r.pool.Query(ctx, "SELECT to_jsonb(c)||jsonb_build_object('author_name',u.full_name) FROM comments c JOIN users u ON u.id=c.author_user_id WHERE c.post_id=$1 AND c.status='PUBLISHED' ORDER BY c.created_at,c.id LIMIT 500", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Comment{}
	for rows.Next() {
		v, e := decodeRow[model.Comment](rows)
		if e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *CommunityRepo) SaveComment(ctx context.Context, userID, postID, id, content string, parentID *string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if id != "" {
		tag, e := tx.Exec(ctx, "UPDATE comments SET content=$1,updated_at=now() WHERE id=$2 AND author_user_id=$3 AND status='PUBLISHED'", content, id, userID)
		if e != nil {
			return "", e
		}
		if tag.RowsAffected() == 0 {
			return "", ErrForbidden
		}
		return id, tx.Commit(ctx)
	}
	if parentID != nil {
		var valid bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM comments WHERE id=$1 AND post_id=$2 AND parent_id IS NULL AND status='PUBLISHED')", *parentID, postID).Scan(&valid)
		if err != nil {
			return "", err
		}
		if !valid {
			return "", ErrConflict
		}
	}
	id = uuid.NewString()
	if _, err = tx.Exec(ctx, "INSERT INTO comments(id,post_id,author_user_id,parent_id,content) VALUES($1,$2,$3,$4,$5)", id, postID, userID, parentID, content); err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,channel,title,content,status,event_id,link) SELECT recipient,'WEB','Bình luận mới',$1,'SENT',$2||recipient::text,$3 FROM (SELECT author_user_id AS recipient FROM posts WHERE id=$4 UNION SELECT author_user_id FROM comments WHERE id=$5) x WHERE recipient<>$6`, content, "COMMENT_"+id, "/posts/"+postID, postID, parentID, userID)
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}
func (r *CommunityRepo) Comment(ctx context.Context, id string) (model.Comment, error) {
	return decodeRow[model.Comment](r.pool.QueryRow(ctx, "SELECT to_jsonb(c) FROM comments c WHERE id=$1 AND status='PUBLISHED'", id))
}
func (r *CommunityRepo) DeleteComment(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE comments SET status='DELETED',updated_at=now() WHERE id=$1 AND author_user_id=$2", id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return err
}
func (r *CommunityRepo) Report(ctx context.Context, userID string, postID, commentID *string, reason string) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO content_reports(reporter_id,post_id,comment_id,reason) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", userID, postID, commentID, reason)
	return err
}
func (r *CommunityRepo) Reports(ctx context.Context) ([]model.Report, error) {
	rows, err := r.pool.Query(ctx, "SELECT to_jsonb(r)||jsonb_build_object('content',COALESCE(cm.content,p.content,'')) FROM content_reports r LEFT JOIN posts p ON p.id=r.post_id LEFT JOIN comments cm ON cm.id=r.comment_id ORDER BY r.created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Report{}
	for rows.Next() {
		v, e := decodeRow[model.Report](rows)
		if e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *CommunityRepo) Moderate(ctx context.Context, actor, id, kind, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	table := "posts"
	if kind == "comment" {
		table = "comments"
	}
	tag, err := tx.Exec(ctx, "UPDATE "+table+" SET status='HIDDEN',updated_at=now() WHERE id=$1 AND status='PUBLISHED'", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err = audit(ctx, tx, actor, id, "HIDE_"+strings.ToUpper(kind), reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *CommunityRepo) ResolveReport(ctx context.Context, actor, id, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE content_reports SET status='RESOLVED',resolution=$1 WHERE id=$2 AND status='PENDING'", reason, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err = audit(ctx, tx, actor, id, "RESOLVE_REPORT", reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *CommunityRepo) StoreMedia(ctx context.Context, id, user, mime string, size int64) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO media_assets(id,owner_user_id,mime,size_bytes) VALUES($1,$2,$3,$4)", id, user, mime, size)
	return err
}
func (r *CommunityRepo) Media(ctx context.Context, id string) (string, error) {
	var mime string
	err := r.pool.QueryRow(ctx, "SELECT mime FROM media_assets WHERE id=$1", id).Scan(&mime)
	return mime, err
}
func (r *CommunityRepo) Pool() *pgxpool.Pool { return r.pool }
