package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unihub-workshop/internal/model"
)

func (r *CommunityRepo) SubmitWorkshop(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE workshops w SET status='PENDING_REVIEW',review_reason='',updated_at=now() FROM companies c WHERE w.id=$1 AND w.company_id=c.id AND c.owner_user_id=$2 AND c.status='APPROVED' AND w.status IN ('DRAFT','REJECTED')`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return err
}
func (r *CommunityRepo) WorkshopReviews(ctx context.Context) ([]model.Workshop, error) {
	rows, err := r.pool.Query(ctx, workshopSelect+"WHERE w.status='PENDING_REVIEW' ORDER BY w.created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Workshop{}
	for rows.Next() {
		v, e := scanWorkshop(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
func (r *CommunityRepo) ReviewWorkshop(ctx context.Context, actor, id, status, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owner, title string
	err = tx.QueryRow(ctx, `UPDATE workshops w SET status=$1,review_reason=$2,updated_at=now() FROM companies c WHERE w.id=$3 AND w.company_id=c.id AND c.status='APPROVED' AND w.status='PENDING_REVIEW' RETURNING c.owner_user_id,w.title`, status, reason, id).Scan(&owner, &title)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, actor, id, "WORKSHOP_"+status, reason); err != nil {
		return err
	}
	if err = notify(ctx, tx, owner, "Workshop "+title+": "+status, reason, "/business/workshops"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *CommunityRepo) CreateRevision(ctx context.Context, userID, id string, payload model.CreateWorkshopRequest, cancel bool, reason string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO workshop_revisions(workshop_id,requested_by,payload,cancellation,reason)
 SELECT w.id,$1,$2,$3,$4 FROM workshops w JOIN companies c ON c.id=w.company_id WHERE w.id=$5 AND c.owner_user_id=$1 AND c.status='APPROVED' AND w.status IN ('PUBLISHED','CLOSED')`, userID, data, cancel, reason, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return err
}
func (r *CommunityRepo) Revisions(ctx context.Context, userID string, admin bool) ([]model.Revision, error) {
	q := `SELECT to_jsonb(r)||jsonb_build_object('workshop_title',w.title,'company_name',c.name) FROM workshop_revisions r JOIN workshops w ON w.id=r.workshop_id JOIN companies c ON c.id=w.company_id`
	args := []any{}
	if !admin {
		q += " WHERE c.owner_user_id=$1"
		args = append(args, userID)
	}
	q += " ORDER BY r.created_at DESC LIMIT 100"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Revision{}
	for rows.Next() {
		v, e := decodeRow[model.Revision](rows)
		if e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *CommunityRepo) ReviewRevision(ctx context.Context, actor, id, status, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var workshopID string
	if err = tx.QueryRow(ctx, "SELECT workshop_id FROM workshop_revisions WHERE id=$1", id).Scan(&workshopID); err != nil {
		return err
	}
	var capacity, available int
	var workshopState, title, owner, companyState string
	err = tx.QueryRow(ctx, `SELECT w.capacity,w.available_seats,w.status,w.title,c.owner_user_id,c.status FROM workshops w JOIN companies c ON c.id=w.company_id WHERE w.id=$1 FOR UPDATE OF w`, workshopID).Scan(&capacity, &available, &workshopState, &title, &owner, &companyState)
	if err != nil {
		return err
	}
	var data []byte
	var state string
	var cancel bool
	if err = tx.QueryRow(ctx, "SELECT payload,status,cancellation FROM workshop_revisions WHERE id=$1 FOR UPDATE", id).Scan(&data, &state, &cancel); err != nil {
		return err
	}
	if state != "PENDING" {
		return ErrConflict
	}
	if status == "APPROVED" {
		if companyState != "APPROVED" || (workshopState != "PUBLISHED" && workshopState != "CLOSED") {
			return ErrConflict
		}
		if cancel {
			if _, err = tx.Exec(ctx, "UPDATE workshops SET status='CANCELLED',updated_at=now() WHERE id=$1", workshopID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "UPDATE registrations SET status='CANCELLED',updated_at=now() WHERE workshop_id=$1 AND status='SUCCESS'", workshopID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "UPDATE registration_requests SET status='FAILED',message='Workshop đã hủy',updated_at=now() WHERE workshop_id=$1 AND status='PROCESSING'", workshopID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "UPDATE workshops SET available_seats=capacity WHERE id=$1", workshopID); err != nil {
				return err
			}
		} else {
			var in model.CreateWorkshopRequest
			if err = json.Unmarshal(data, &in); err != nil {
				return err
			}
			if in.Capacity < capacity-available {
				return fmt.Errorf("sức chứa nhỏ hơn số ghế đã giữ")
			}
			times := make([]time.Time, 4)
			for i, value := range []string{in.StartTime, in.EndTime, in.RegistrationStartTime, in.RegistrationEndTime} {
				times[i], err = time.Parse(time.RFC3339, value)
				if err != nil {
					return err
				}
			}
			_, err = tx.Exec(ctx, `UPDATE workshops SET title=$1,description=$2,speaker=$3,room=$4,start_time=$5,end_time=$6,registration_start_time=$7,registration_end_time=$8,capacity=$9,available_seats=$10,price=$11,summary=$12,cover_url=$13,audience=$14,benefits=$15,preparation=$16,agenda=$17,format=$18,updated_at=now() WHERE id=$19`, in.Title, in.Description, in.Speaker, in.Room, times[0], times[1], times[2], times[3], in.Capacity, available+in.Capacity-capacity, in.Price, in.Summary, in.CoverURL, in.Audience, in.Benefits, in.Preparation, in.Agenda, in.Format, workshopID)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,channel,title,content,status,event_id,link) SELECT DISTINCT user_id,'WEB',$1,$2,'SENT',$3||user_id::text,'/' FROM registrations WHERE workshop_id=$4 ON CONFLICT(event_id,channel) DO NOTHING`, "Workshop cập nhật: "+title, reason, "REVISION_"+id, workshopID)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE workshop_revisions SET status=$1,reason=$2 WHERE id=$3", status, reason, id); err != nil {
		return err
	}
	if err = audit(ctx, tx, actor, id, "REVISION_"+status, reason); err != nil {
		return err
	}
	if err = notify(ctx, tx, owner, "Yêu cầu thay đổi workshop: "+status, reason, "/business/workshops"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *CommunityRepo) BusinessStats(ctx context.Context, userID string) (map[string]int, error) {
	var workshops, posts, followers, registrations, checkins int
	err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM workshops w WHERE w.company_id=c.id AND w.status<>'DELETED'),(SELECT count(*) FROM posts p WHERE p.company_id=c.id AND p.status<>'DELETED'),(SELECT count(*) FROM company_follows f WHERE f.company_id=c.id),(SELECT count(*) FROM registrations r JOIN workshops w ON w.id=r.workshop_id WHERE w.company_id=c.id AND r.status='SUCCESS'),(SELECT count(*) FROM registrations r JOIN workshops w ON w.id=r.workshop_id WHERE w.company_id=c.id AND r.status='SUCCESS' AND r.is_checked_in) FROM companies c WHERE c.owner_user_id=$1`, userID).Scan(&workshops, &posts, &followers, &registrations, &checkins)
	return map[string]int{"workshops": workshops, "posts": posts, "followers": followers, "registrations": registrations, "checkins": checkins}, err
}
func (r *CommunityRepo) Staff(ctx context.Context, id string) ([]map[string]string, error) {
	rows, err := r.pool.Query(ctx, "SELECT u.id,u.full_name,EXISTS(SELECT 1 FROM workshop_staff ws WHERE ws.user_id=u.id AND ws.workshop_id=$1) FROM users u WHERE role='STAFF' ORDER BY full_name", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []map[string]string{}
	for rows.Next() {
		var uid, name string
		var assigned bool
		if err = rows.Scan(&uid, &name, &assigned); err != nil {
			return nil, err
		}
		value := "false"
		if assigned {
			value = "true"
		}
		result = append(result, map[string]string{"id": uid, "full_name": name, "assigned": value})
	}
	return result, rows.Err()
}
func (r *CommunityRepo) AssignStaff(ctx context.Context, id, user string, remove bool) error {
	if remove {
		_, err := r.pool.Exec(ctx, "DELETE FROM workshop_staff WHERE workshop_id=$1 AND user_id=$2", id, user)
		return err
	}
	tag, err := r.pool.Exec(ctx, "INSERT INTO workshop_staff(workshop_id,user_id) SELECT $1,id FROM users WHERE id=$2 AND role='STAFF' ON CONFLICT DO NOTHING", id, user)
	if err == nil && tag.RowsAffected() == 0 {
		var ok bool
		_ = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workshop_staff WHERE workshop_id=$1 AND user_id=$2)", id, user).Scan(&ok)
		if !ok {
			return ErrForbidden
		}
	}
	return err
}
