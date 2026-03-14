//
//  StatsCalculator.swift
//  tick
//

import Foundation

enum StatsCalculator {

    // MARK: - Current Streak

    static func currentStreak(for habit: Habit) -> Int {
        let calendar = Calendar.current
        let logs = habit.activeLogs.sorted { $0.completedAt > $1.completedAt }

        guard !logs.isEmpty else { return 0 }

        if habit.isDaily {
            return dailyStreak(logs: logs, calendar: calendar, fromDate: Date())
        } else {
            return weeklyStreak(logs: logs, calendar: calendar, fromDate: Date(), targetCount: habit.frequencyValue)
        }
    }

    // MARK: - Longest Streak

    static func longestStreak(for habit: Habit) -> Int {
        let calendar = Calendar.current
        let logs = habit.activeLogs.sorted { $0.completedAt < $1.completedAt }

        guard !logs.isEmpty else { return 0 }

        if habit.isDaily {
            return longestDailyStreak(logs: logs, calendar: calendar)
        } else {
            return longestWeeklyStreak(logs: logs, calendar: calendar, targetCount: habit.frequencyValue)
        }
    }

    // MARK: - Completion Rate

    static func completionRate(for habit: Habit) -> Double {
        let calendar = Calendar.current
        let logs = habit.activeLogs

        guard !logs.isEmpty else { return 0.0 }

        let startDate = habit.createdAt
        let today = Date()

        if habit.isDaily {
            let totalDays = max(1, calendar.dateComponents([.day], from: calendar.startOfDay(for: startDate), to: calendar.startOfDay(for: today)).day! + 1)
            let completedDays = Set(logs.map { calendar.startOfDay(for: $0.completedAt) }).count
            return Double(completedDays) / Double(totalDays) * 100.0
        } else {
            let startWeek = calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: startDate)
            let currentWeek = calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: today)

            guard let startWeekDate = calendar.date(from: startWeek),
                  let currentWeekDate = calendar.date(from: currentWeek) else {
                return 0.0
            }

            let totalWeeks = max(1, calendar.dateComponents([.weekOfYear], from: startWeekDate, to: currentWeekDate).weekOfYear! + 1)

            // Count weeks that met the target
            var weekLogCounts: [Date: Int] = [:]
            for log in logs {
                let weekStart = calendar.date(from: calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: log.completedAt))!
                weekLogCounts[weekStart, default: 0] += 1
            }
            let fulfilledWeeks = weekLogCounts.values.filter { $0 >= habit.frequencyValue }.count

            return Double(fulfilledWeeks) / Double(totalWeeks) * 100.0
        }
    }

    // MARK: - Total Completions

    static func totalCompletions(for habit: Habit) -> Int {
        habit.activeLogs.count
    }

    // MARK: - Private Helpers

    private static func dailyStreak(logs: [HabitLog], calendar: Calendar, fromDate: Date) -> Int {
        // Create set of unique completion dates
        let completionDays = Set(logs.map { calendar.startOfDay(for: $0.completedAt) })

        var streak = 0
        var checkDate = calendar.startOfDay(for: fromDate)

        // If today is not completed, start checking from yesterday
        if !completionDays.contains(checkDate) {
            guard let yesterday = calendar.date(byAdding: .day, value: -1, to: checkDate) else {
                return 0
            }
            checkDate = yesterday
        }

        while completionDays.contains(checkDate) {
            streak += 1
            guard let previousDay = calendar.date(byAdding: .day, value: -1, to: checkDate) else {
                break
            }
            checkDate = previousDay
        }

        return streak
    }

    private static func weeklyStreak(logs: [HabitLog], calendar: Calendar, fromDate: Date, targetCount: Int) -> Int {
        // Group logs by week
        var weekLogCounts: [Date: Int] = [:]
        for log in logs {
            let weekStart = calendar.date(from: calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: log.completedAt))!
            weekLogCounts[weekStart, default: 0] += 1
        }

        var streak = 0
        var currentWeekStart = calendar.date(from: calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: fromDate))!

        // If current week doesn't meet target, start from previous week
        if (weekLogCounts[currentWeekStart] ?? 0) < targetCount {
            guard let prevWeek = calendar.date(byAdding: .weekOfYear, value: -1, to: currentWeekStart) else {
                return 0
            }
            currentWeekStart = prevWeek
        }

        while (weekLogCounts[currentWeekStart] ?? 0) >= targetCount {
            streak += 1
            guard let prevWeek = calendar.date(byAdding: .weekOfYear, value: -1, to: currentWeekStart) else {
                break
            }
            currentWeekStart = prevWeek
        }

        return streak
    }

    private static func longestDailyStreak(logs: [HabitLog], calendar: Calendar) -> Int {
        let completionDays = Set(logs.map { calendar.startOfDay(for: $0.completedAt) }).sorted()

        guard !completionDays.isEmpty else { return 0 }

        var maxStreak = 1
        var currentStreak = 1

        for i in 1..<completionDays.count {
            let daysBetween = calendar.dateComponents([.day], from: completionDays[i - 1], to: completionDays[i]).day ?? 0
            if daysBetween == 1 {
                currentStreak += 1
                maxStreak = max(maxStreak, currentStreak)
            } else {
                currentStreak = 1
            }
        }

        return maxStreak
    }

    private static func longestWeeklyStreak(logs: [HabitLog], calendar: Calendar, targetCount: Int) -> Int {
        var weekLogCounts: [Date: Int] = [:]
        for log in logs {
            let weekStart = calendar.date(from: calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: log.completedAt))!
            weekLogCounts[weekStart, default: 0] += 1
        }

        let qualifiedWeeks = weekLogCounts.filter { $0.value >= targetCount }.keys.sorted()

        guard !qualifiedWeeks.isEmpty else { return 0 }

        var maxStreak = 1
        var currentStreak = 1

        for i in 1..<qualifiedWeeks.count {
            let weeksBetween = calendar.dateComponents([.weekOfYear], from: qualifiedWeeks[i - 1], to: qualifiedWeeks[i]).weekOfYear ?? 0
            if weeksBetween == 1 {
                currentStreak += 1
                maxStreak = max(maxStreak, currentStreak)
            } else {
                currentStreak = 1
            }
        }

        return maxStreak
    }
}
