package model

import "time"

type Company struct {
	ID           string    `json:"id"`
	OwnerUserID  string    `json:"owner_user_id,omitempty"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	Industry     string    `json:"industry"`
	Website      string    `json:"website"`
	Address      string    `json:"address"`
	Contact      string    `json:"contact"`
	LogoURL      string    `json:"logo_url"`
	CoverURL     string    `json:"cover_url"`
	Status       string    `json:"status"`
	ReviewReason string    `json:"review_reason,omitempty"`
	Followers    int       `json:"followers"`
	Following    bool      `json:"following"`
	CreatedAt    time.Time `json:"created_at"`
}
type CompanyInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Industry    string `json:"industry"`
	Website     string `json:"website"`
	Address     string `json:"address"`
	Contact     string `json:"contact"`
	LogoURL     string `json:"logo_url"`
	CoverURL    string `json:"cover_url"`
}
type BusinessAccountInput struct {
	CompanyInput
	Identifier string `json:"identifier"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}
type PostInput struct {
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Topic      string   `json:"topic"`
	WorkshopID *string  `json:"workshop_id"`
	MediaIDs   []string `json:"media_ids"`
	Status     string   `json:"status"`
}
type Post struct {
	ID           string     `json:"id"`
	AuthorUserID string     `json:"author_user_id"`
	AuthorName   string     `json:"author_name"`
	CompanyID    *string    `json:"company_id"`
	CompanyName  string     `json:"company_name"`
	CompanySlug  string     `json:"company_slug"`
	WorkshopID   *string    `json:"workshop_id"`
	Workshop     *Workshop  `json:"workshop,omitempty"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Topic        string     `json:"topic"`
	MediaIDs     []string   `json:"media_ids"`
	Status       string     `json:"status"`
	Likes        int        `json:"likes"`
	Comments     int        `json:"comments"`
	Liked        bool       `json:"liked"`
	Bookmarked   bool       `json:"bookmarked"`
	CreatedAt    time.Time  `json:"created_at"`
	PublishedAt  *time.Time `json:"published_at"`
}
type FeedPage struct {
	Items      []Post `json:"items"`
	NextCursor string `json:"next_cursor"`
}
type Comment struct {
	ID           string    `json:"id"`
	PostID       string    `json:"post_id"`
	AuthorUserID string    `json:"author_user_id"`
	AuthorName   string    `json:"author_name"`
	ParentID     *string   `json:"parent_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}
type Revision struct {
	ID            string                `json:"id"`
	WorkshopID    string                `json:"workshop_id"`
	WorkshopTitle string                `json:"workshop_title"`
	CompanyName   string                `json:"company_name"`
	Payload       CreateWorkshopRequest `json:"payload"`
	Status        string                `json:"status"`
	Reason        string                `json:"reason"`
	Cancellation  bool                  `json:"cancellation"`
	CreatedAt     time.Time             `json:"created_at"`
}
type Report struct {
	ID         string    `json:"id"`
	PostID     *string   `json:"post_id"`
	CommentID  *string   `json:"comment_id"`
	Reason     string    `json:"reason"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	Resolution string    `json:"resolution"`
	CreatedAt  time.Time `json:"created_at"`
}
