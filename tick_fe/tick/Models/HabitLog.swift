//
//  HabitLog.swift
//  tick
//

import Foundation
import SwiftData

@Model
class HabitLog {
    @Attribute(.unique) var id: UUID
    var serverId: Int?
    var habit: Habit?
    var completedAt: Date
    var note: String?
    var isExtra: Bool
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?

    init(
        id: UUID = UUID(),
        serverId: Int? = nil,
        habit: Habit? = nil,
        completedAt: Date = Date(),
        note: String? = nil,
        isExtra: Bool = false,
        isDeleted: Bool = false,
        createdAt: Date = Date(),
        updatedAt: Date = Date(),
        syncedAt: Date? = nil
    ) {
        self.id = id
        self.serverId = serverId
        self.habit = habit
        self.completedAt = completedAt
        self.note = note
        self.isExtra = isExtra
        self.isDeleted = isDeleted
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.syncedAt = syncedAt
    }

    var isDirty: Bool {
        syncedAt == nil || updatedAt > (syncedAt ?? .distantPast)
    }
}
