//
//  Habit.swift
//  tick
//

import Foundation
import SwiftData

@Model
class Habit {
    @Attribute(.unique) var id: UUID
    var serverId: Int?
    var name: String
    var category: Category?
    var frequencyType: String
    var frequencyValue: Int
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?

    @Relationship(deleteRule: .cascade, inverse: \HabitLog.habit)
    var logs: [HabitLog] = []

    init(
        id: UUID = UUID(),
        serverId: Int? = nil,
        name: String,
        category: Category? = nil,
        frequencyType: String = "daily",
        frequencyValue: Int = 1,
        isDeleted: Bool = false,
        createdAt: Date = Date(),
        updatedAt: Date = Date(),
        syncedAt: Date? = nil
    ) {
        self.id = id
        self.serverId = serverId
        self.name = name
        self.category = category
        self.frequencyType = frequencyType
        self.frequencyValue = frequencyValue
        self.isDeleted = isDeleted
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.syncedAt = syncedAt
    }

    var isDirty: Bool {
        syncedAt == nil || updatedAt > (syncedAt ?? .distantPast)
    }

    var isDaily: Bool {
        frequencyType == "daily"
    }

    var isWeekly: Bool {
        frequencyType == "weekly"
    }

    var activeLogs: [HabitLog] {
        logs.filter { !$0.isDeleted }
    }

    func logsForDate(_ date: Date) -> [HabitLog] {
        let calendar = Calendar.current
        return activeLogs.filter { calendar.isDate($0.completedAt, inSameDayAs: date) }
    }

    func isCompletedForDate(_ date: Date) -> Bool {
        if isDaily {
            return !logsForDate(date).isEmpty
        } else {
            let calendar = Calendar.current
            guard let weekInterval = calendar.dateInterval(of: .weekOfYear, for: date) else {
                return false
            }
            let weekLogs = activeLogs.filter {
                $0.completedAt >= weekInterval.start && $0.completedAt < weekInterval.end
            }
            return weekLogs.count >= frequencyValue
        }
    }

    var completionCountToday: Int {
        logsForDate(Date()).count
    }
}
