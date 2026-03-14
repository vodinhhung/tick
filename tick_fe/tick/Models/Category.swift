//
//  Category.swift
//  tick
//

import Foundation
import SwiftData

@Model
class Category {
    @Attribute(.unique) var id: UUID
    var serverId: Int?
    var name: String
    var isPreset: Bool
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?

    @Relationship(deleteRule: .nullify, inverse: \Habit.category)
    var habits: [Habit] = []

    init(
        id: UUID = UUID(),
        serverId: Int? = nil,
        name: String,
        isPreset: Bool = false,
        isDeleted: Bool = false,
        createdAt: Date = Date(),
        updatedAt: Date = Date(),
        syncedAt: Date? = nil
    ) {
        self.id = id
        self.serverId = serverId
        self.name = name
        self.isPreset = isPreset
        self.isDeleted = isDeleted
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.syncedAt = syncedAt
    }

    var isDirty: Bool {
        syncedAt == nil || updatedAt > (syncedAt ?? .distantPast)
    }

    static let presetNames = [
        "Health",
        "Fitness",
        "Mindfulness",
        "Productivity",
        "Learning",
        "Social",
        "Finance",
        "Creativity"
    ]
}
