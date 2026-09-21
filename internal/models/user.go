package models

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email,omitempty"`
	Name        string    `json:"name"`
	Color       string    `json:"color,omitempty"`
	FamilyID    string    `json:"familyId,omitempty"`
	Role        Role      `json:"role,omitempty"`
	HasPassword bool      `json:"hasPassword"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SetColorRequest struct {
	Color string `json:"color"`
}

// --- auth ---

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Invite   string `json:"invite,omitempty"` // код инвайта, опционально
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// --- invites ---

type CreateInviteRequest struct {
	ExpiresInDays int `json:"expiresInDays,omitempty"` // 0 = 7 по умолчанию
}

type Invite struct {
	ID         string     `json:"id"`
	Code       string     `json:"code"`
	FamilyID   string     `json:"familyId"`
	FamilyName string     `json:"familyName,omitempty"`
	CreatedBy  string     `json:"createdBy"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	UsedBy     string     `json:"usedBy,omitempty"`
	UsedAt     *time.Time `json:"usedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type InviteInfo struct {
	Code       string    `json:"code"`
	FamilyName string    `json:"familyName"`
	Valid      bool      `json:"valid"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// --- sessions (внутренний тип, наружу не отдаём) ---

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	UserAgent string
}
