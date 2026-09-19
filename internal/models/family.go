package models

import "time"

type Family struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"inviteCode"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

type FamilyMember struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
	Color string `json:"color"`
}

type FamilyView struct {
	Family  Family         `json:"family"`
	Members []FamilyMember `json:"members"`
}

type CreateFamilyRequest struct {
	Name string `json:"name"`
}

type JoinFamilyRequest struct {
	Code string `json:"code"`
}
