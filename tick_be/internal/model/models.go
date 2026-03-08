package model

import (
	"time"
)

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	GoogleID      string    `gorm:"column:google_id;uniqueIndex:idx_google_id;not null" json:"google_id"`
	Email         string    `gorm:"column:email;uniqueIndex:idx_email;not null" json:"email"`
	VerifiedEmail bool      `gorm:"column:verified_email;default:0" json:"verified_email"`
	Name          string    `gorm:"column:name;index:idx_name" json:"name"`
	GivenName     string    `gorm:"column:given_name" json:"given_name"`
	FamilyName    string    `gorm:"column:family_name" json:"family_name"`
	Picture       string    `gorm:"column:picture" json:"picture"`
	Locale        string    `gorm:"column:locale" json:"locale"`
	CreatedAt     time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (User) TableName() string {
	return "user_tab"
}

type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `gorm:"column:user_id;index:idx_category_user;index:idx_category_updated" json:"user_id,omitempty"`
	Name      string    `gorm:"column:name;size:100;not null" json:"name"`
	IsPreset  bool      `gorm:"column:is_preset;not null;default:0" json:"is_preset"`
	IsDeleted bool      `gorm:"column:is_deleted;not null;default:0" json:"is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (Category) TableName() string {
	return "category_tab"
}

type Habit struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"column:user_id;not null;index:idx_habit_user;index:idx_habit_sync" json:"user_id"`
	ClientID       string    `gorm:"column:client_id;size:36;not null;uniqueIndex:idx_habit_client" json:"client_id"`
	Name           string    `gorm:"column:name;size:255;not null" json:"name"`
	CategoryID     *uint     `gorm:"column:category_id" json:"category_id,omitempty"`
	FrequencyType  string    `gorm:"column:frequency_type;size:32;not null" json:"frequency_type"`
	FrequencyValue int       `gorm:"column:frequency_value;not null;default:1" json:"frequency_value"`
	IsDeleted      bool      `gorm:"column:is_deleted;not null;default:0" json:"is_deleted"`
	CreatedAt      time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (Habit) TableName() string {
	return "habit_tab"
}

type HabitLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"column:user_id;not null;index:idx_log_user_sync" json:"user_id"`
	HabitID     uint      `gorm:"column:habit_id;not null;index:idx_log_habit_date" json:"habit_id"`
	ClientID    string    `gorm:"column:client_id;size:36;not null;uniqueIndex:idx_log_client" json:"client_id"`
	CompletedAt time.Time `gorm:"column:completed_at;not null" json:"completed_at"`
	Note        *string   `gorm:"column:note;size:280" json:"note,omitempty"`
	IsExtra     bool      `gorm:"column:is_extra;not null;default:0" json:"is_extra"`
	IsDeleted   bool      `gorm:"column:is_deleted;not null;default:0" json:"is_deleted"`
	CreatedAt   time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (HabitLog) TableName() string {
	return "habit_log_tab"
}
