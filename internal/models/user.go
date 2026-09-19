package models

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color,omitempty"`
	FamilyID  string    `json:"familyId,omitempty"`
	Role      Role      `json:"role,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type SetMeRequest struct {
	Name string `json:"name"`
}

type SetColorRequest struct {
	Color string `json:"color"`
}
