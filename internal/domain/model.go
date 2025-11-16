package domain

type Team struct {
	TeamName string
	Members  []TeamMember
}

type TeamMember struct {
	UserID   string
	UserName string
	IsActive bool
}

type User struct {
	UserID   string
	UserName string
	TeamName string
	IsActive bool
}
