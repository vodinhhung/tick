package dep

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	// GORM Standard Fields
	ID        uint      `gorm:"primaryKey"` // Internal Auto-Incrementing ID
	CreatedAt time.Time `gorm:"createTime"`
	UpdatedAt time.Time `gorm:"updateTime"`

	// Google OAuth Fields (and their persistence properties)
	GoogleID      string `gorm:"uniqueIndex:idx_google_id;not null" json:"id"`
	Email         string `gorm:"uniqueIndex:idx_email;not null" json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `gorm:"index:idx_name" json:"name"` // Indexed for name lookups
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// TableName overrides the table name to 'user_tab' as requested.
func (User) TableName() string {
	return "user_tab"
}

// ConnectDatabase initializes the MySQL connection and runs migrations.
func ConnectDatabase() *gorm.DB {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal("DSN environment variable not set. Please set your MySQL connection string.")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// AutoMigrate creates the table if it doesn't exist.
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	fmt.Println("Database connection established and migration complete.")
	return db
}

// AddOrUpdateUser checks if a user exists by their GoogleID or Email.
// If the user exists, it updates their data; otherwise, it creates a new record.
func AddOrUpdateUser(db *gorm.DB, googleUser *User) (*User, error) {
	existingUser := &User{}

	// Attempt to find the user by their unique GoogleID
	result := db.Where("google_id = ?", googleUser.GoogleID).First(existingUser)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query error: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		// User found: Update the existing record
		// Note: We exclude ID, CreatedAt, and GoogleID from the update
		db.Model(existingUser).
			Select("Email", "VerifiedEmail", "Name", "GivenName", "FamilyName", "Picture", "Locale").
			Updates(googleUser)

		fmt.Printf("Updated existing user: %s\n", existingUser.Email)
		return existingUser, nil
	}

	// User not found: Create a new record
	if err := db.Create(googleUser).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("Created new user: %s\n", googleUser.Email)
	return googleUser, nil
}

// GetUserByEmail finds a single user record based on their unique email address.
func GetUserByEmail(db *gorm.DB, email string) (*User, error) {
	user := &User{}
	result := db.Where("email = ?", email).First(user)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil // User not found
	}
	if result.Error != nil {
		return nil, fmt.Errorf("database query error: %w", result.Error)
	}
	return user, nil
}

// GetUsersByName finds all user records matching the given name (as Name is not unique).
func GetUsersByName(db *gorm.DB, name string) ([]User, error) {
	var users []User
	// Using LIKE for partial matching, often better for name searches
	result := db.Where("name LIKE ?", "%"+name+"%").Find(&users)

	if result.Error != nil {
		return nil, fmt.Errorf("database query error: %w", result.Error)
	}
	return users, nil
}
