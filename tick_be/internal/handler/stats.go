package handler

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tick/be/internal/model"
)

type StatsHandler struct {
	DB *gorm.DB
}

func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{DB: db}
}

type statsResponse struct {
	HabitID          uint    `json:"habit_id"`
	CurrentStreak    int     `json:"current_streak"`
	LongestStreak    int     `json:"longest_streak"`
	CompletionRate   float64 `json:"completion_rate"`
	TotalCompletions int64   `json:"total_completions"`
	ComputedAt       string  `json:"computed_at"`
}

func (h *StatsHandler) GetStats(c *gin.Context) {
	userID := getUserID(c)
	habitID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid habit ID")
		return
	}

	var habit model.Habit
	if err := h.DB.Where("id = ? AND user_id = ?", habitID, userID).First(&habit).Error; err != nil {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "Habit not found or not owned by user")
		return
	}

	// Fetch all non-deleted logs for this habit
	var logs []model.HabitLog
	h.DB.Where("habit_id = ? AND user_id = ? AND is_deleted = ?", habitID, userID, false).
		Order("completed_at ASC").
		Find(&logs)

	totalCompletions := int64(len(logs))
	now := time.Now().UTC()

	var currentStreak, longestStreak int
	var completionRate float64

	if habit.FrequencyType == "daily" {
		currentStreak, longestStreak, completionRate = computeDailyStats(logs, habit, now)
	} else if habit.FrequencyType == "weekly" {
		currentStreak, longestStreak, completionRate = computeWeeklyStats(logs, habit, now)
	}

	c.JSON(http.StatusOK, statsResponse{
		HabitID:          uint(habitID),
		CurrentStreak:    currentStreak,
		LongestStreak:    longestStreak,
		CompletionRate:   math.Round(completionRate*10) / 10,
		TotalCompletions: totalCompletions,
		ComputedAt:       now.Format(time.RFC3339),
	})
}

func computeDailyStats(logs []model.HabitLog, habit model.Habit, now time.Time) (currentStreak, longestStreak int, completionRate float64) {
	if len(logs) == 0 {
		return 0, 0, 0
	}

	// Build a set of dates with sufficient completions
	dayCounts := make(map[string]int)
	for _, l := range logs {
		day := l.CompletedAt.Format("2006-01-02")
		dayCounts[day]++
	}

	fulfilledDays := make(map[string]bool)
	for day, count := range dayCounts {
		if count >= habit.FrequencyValue {
			fulfilledDays[day] = true
		}
	}

	// Sort fulfilled days
	sortedDays := make([]string, 0, len(fulfilledDays))
	for day := range fulfilledDays {
		sortedDays = append(sortedDays, day)
	}
	sort.Strings(sortedDays)

	// Compute streaks
	longestStreak = 0
	currentStreak = 0

	if len(sortedDays) > 0 {
		streak := 1
		for i := 1; i < len(sortedDays); i++ {
			prev, _ := time.Parse("2006-01-02", sortedDays[i-1])
			curr, _ := time.Parse("2006-01-02", sortedDays[i])
			if curr.Sub(prev) == 24*time.Hour {
				streak++
			} else {
				if streak > longestStreak {
					longestStreak = streak
				}
				streak = 1
			}
		}
		if streak > longestStreak {
			longestStreak = streak
		}

		// Current streak: check if the last fulfilled day is today or yesterday
		lastDay, _ := time.Parse("2006-01-02", sortedDays[len(sortedDays)-1])
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		if lastDay.Equal(today) || lastDay.Equal(today.Add(-24*time.Hour)) {
			currentStreak = 1
			for i := len(sortedDays) - 2; i >= 0; i-- {
				curr, _ := time.Parse("2006-01-02", sortedDays[i+1])
				prev, _ := time.Parse("2006-01-02", sortedDays[i])
				if curr.Sub(prev) == 24*time.Hour {
					currentStreak++
				} else {
					break
				}
			}
		}
	}

	// Completion rate: fulfilled days / total days since habit creation
	createdDay := time.Date(habit.CreatedAt.Year(), habit.CreatedAt.Month(), habit.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	totalDays := int(today.Sub(createdDay).Hours()/24) + 1
	if totalDays > 0 {
		completionRate = float64(len(fulfilledDays)) / float64(totalDays) * 100
	}

	return
}

func computeWeeklyStats(logs []model.HabitLog, habit model.Habit, now time.Time) (currentStreak, longestStreak int, completionRate float64) {
	if len(logs) == 0 {
		return 0, 0, 0
	}

	// Group logs by ISO week
	weekCounts := make(map[string]int)
	for _, l := range logs {
		year, week := l.CompletedAt.ISOWeek()
		key := strconv.Itoa(year) + "-W" + strconv.Itoa(week)
		weekCounts[key]++
	}

	fulfilledWeeks := make(map[string]bool)
	for week, count := range weekCounts {
		if count >= habit.FrequencyValue {
			fulfilledWeeks[week] = true
		}
	}

	// Build a sorted list of fulfilled weeks as time values (start of each ISO week)
	type weekEntry struct {
		key  string
		date time.Time
	}
	var sortedWeeks []weekEntry
	for key := range fulfilledWeeks {
		// Parse year and week number
		var year, week int
		n, _ := strconv.Atoi(key[:4])
		year = n
		wStr := key[6:] // after "-W"
		w, _ := strconv.Atoi(wStr)
		week = w

		// Get the Monday of this ISO week
		jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
		_, jan4Week := jan4.ISOWeek()
		mondayOfWeek1 := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday))
		if jan4.Weekday() == time.Sunday {
			mondayOfWeek1 = jan4.AddDate(0, 0, -6)
		}
		weekStart := mondayOfWeek1.AddDate(0, 0, (week-jan4Week)*7)
		sortedWeeks = append(sortedWeeks, weekEntry{key: key, date: weekStart})
	}

	sort.Slice(sortedWeeks, func(i, j int) bool {
		return sortedWeeks[i].date.Before(sortedWeeks[j].date)
	})

	// Compute streaks
	if len(sortedWeeks) > 0 {
		streak := 1
		for i := 1; i < len(sortedWeeks); i++ {
			diff := sortedWeeks[i].date.Sub(sortedWeeks[i-1].date)
			if diff >= 6*24*time.Hour && diff <= 8*24*time.Hour {
				streak++
			} else {
				if streak > longestStreak {
					longestStreak = streak
				}
				streak = 1
			}
		}
		if streak > longestStreak {
			longestStreak = streak
		}

		// Current streak
		currentYear, currentWeek := now.ISOWeek()
		lastWeek := sortedWeeks[len(sortedWeeks)-1]
		lastYear, lastWeekNum := lastWeek.date.ISOWeek()

		isCurrentOrLastWeek := (lastYear == currentYear && (lastWeekNum == currentWeek || lastWeekNum == currentWeek-1)) ||
			(lastYear == currentYear-1 && currentWeek == 1 && lastWeekNum >= 52)

		if isCurrentOrLastWeek {
			currentStreak = 1
			for i := len(sortedWeeks) - 2; i >= 0; i-- {
				diff := sortedWeeks[i+1].date.Sub(sortedWeeks[i].date)
				if diff >= 6*24*time.Hour && diff <= 8*24*time.Hour {
					currentStreak++
				} else {
					break
				}
			}
		}
	}

	// Completion rate
	createdYear, createdWeek := habit.CreatedAt.ISOWeek()
	currentYear, currentWeek := now.ISOWeek()
	totalWeeks := (currentYear-createdYear)*52 + (currentWeek - createdWeek) + 1
	if totalWeeks > 0 {
		completionRate = float64(len(fulfilledWeeks)) / float64(totalWeeks) * 100
	}

	return
}
