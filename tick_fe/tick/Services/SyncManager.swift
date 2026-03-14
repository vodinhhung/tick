//
//  SyncManager.swift
//  tick
//

import Foundation
import SwiftData

actor SyncManager {
    static let shared = SyncManager()

    private var isSyncing = false
    private var debouncedTask: Task<Void, Never>?
    private let apiClient: APIClient
    private let minimumSyncInterval: TimeInterval = 60
    private let debounceInterval: TimeInterval = 5

    private var lastSyncedAt: Date? {
        get { UserDefaults.standard.object(forKey: "tick.lastSyncedAt") as? Date }
        set { UserDefaults.standard.set(newValue, forKey: "tick.lastSyncedAt") }
    }

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    // MARK: - Public Interface

    func syncIfNeeded(modelContext: ModelContext) async {
        guard KeychainManager.isTokenValid() else { return }

        if let lastSync = lastSyncedAt,
           Date().timeIntervalSince(lastSync) < minimumSyncInterval {
            return
        }

        await syncNow(modelContext: modelContext)
    }

    func syncNow(modelContext: ModelContext) async {
        guard !isSyncing else { return }
        guard KeychainManager.isTokenValid() else { return }

        isSyncing = true
        defer { isSyncing = false }

        do {
            let isoFormatter = ISO8601DateFormatter()

            // Collect dirty categories
            let allCategories = try modelContext.fetch(FetchDescriptor<Category>())
            let dirtyCategories = allCategories.filter { $0.isDirty && !$0.isPreset }
            let categoryPayloads = dirtyCategories.map { cat in
                SyncCategoryPayload(
                    clientId: cat.id.uuidString,
                    name: cat.name,
                    isDeleted: cat.isDeleted,
                    updatedAt: isoFormatter.string(from: cat.updatedAt)
                )
            }

            // Collect dirty habits
            let allHabits = try modelContext.fetch(FetchDescriptor<Habit>())
            let dirtyHabits = allHabits.filter { $0.isDirty }
            let habitPayloads = dirtyHabits.map { habit in
                SyncHabitPayload(
                    clientId: habit.id.uuidString,
                    name: habit.name,
                    categoryClientId: habit.category?.id.uuidString,
                    frequencyType: habit.frequencyType,
                    frequencyValue: habit.frequencyValue,
                    isDeleted: habit.isDeleted,
                    updatedAt: isoFormatter.string(from: habit.updatedAt)
                )
            }

            // Collect dirty logs
            let allLogs = try modelContext.fetch(FetchDescriptor<HabitLog>())
            let dirtyLogs = allLogs.filter { $0.isDirty }
            let logPayloads = dirtyLogs.compactMap { log -> SyncLogPayload? in
                guard let habit = log.habit else { return nil }
                return SyncLogPayload(
                    clientId: log.id.uuidString,
                    habitClientId: habit.id.uuidString,
                    completedAt: isoFormatter.string(from: log.completedAt),
                    note: log.note,
                    isDeleted: log.isDeleted,
                    updatedAt: isoFormatter.string(from: log.updatedAt)
                )
            }

            let syncRequest = SyncRequest(
                lastSyncedAt: lastSyncedAt.map { isoFormatter.string(from: $0) },
                categories: categoryPayloads,
                habits: habitPayloads,
                logs: logPayloads
            )

            let response = try await apiClient.sync(request: syncRequest)

            guard let serverTime = isoFormatter.date(from: response.serverTime) else { return }

            // Apply ID map for categories
            if let categoryMap = response.idMap?.categories {
                for (clientIdString, serverId) in categoryMap {
                    guard let clientUUID = UUID(uuidString: clientIdString) else { continue }
                    if let cat = allCategories.first(where: { $0.id == clientUUID }) {
                        cat.serverId = serverId
                        cat.syncedAt = serverTime
                    }
                }
            }

            // Apply ID map for habits
            if let habitMap = response.idMap?.habits {
                for (clientIdString, serverId) in habitMap {
                    guard let clientUUID = UUID(uuidString: clientIdString) else { continue }
                    if let habit = allHabits.first(where: { $0.id == clientUUID }) {
                        habit.serverId = serverId
                        habit.syncedAt = serverTime
                    }
                }
            }

            // Apply ID map for logs
            if let logMap = response.idMap?.logs {
                for (clientIdString, serverId) in logMap {
                    guard let clientUUID = UUID(uuidString: clientIdString) else { continue }
                    if let log = allLogs.first(where: { $0.id == clientUUID }) {
                        log.serverId = serverId
                        log.syncedAt = serverTime
                    }
                }
            }

            // Mark pushed records as synced
            for cat in dirtyCategories {
                cat.syncedAt = serverTime
            }
            for habit in dirtyHabits {
                habit.syncedAt = serverTime
            }
            for log in dirtyLogs {
                log.syncedAt = serverTime
            }

            // Apply server-side category changes
            if let serverCategories = response.categories {
                for serverCat in serverCategories {
                    let existing = allCategories.first(where: { $0.serverId == serverCat.id })
                    if let existing {
                        if let serverUpdated = isoFormatter.date(from: serverCat.updatedAt),
                           serverUpdated > existing.updatedAt {
                            existing.name = serverCat.name
                            existing.isDeleted = serverCat.isDeleted
                            existing.updatedAt = serverUpdated
                            existing.syncedAt = serverTime
                        }
                    } else {
                        let newCat = Category(
                            name: serverCat.name,
                            isPreset: serverCat.isPreset,
                            isDeleted: serverCat.isDeleted,
                            createdAt: isoFormatter.date(from: serverCat.createdAt ?? serverCat.updatedAt) ?? Date(),
                            updatedAt: isoFormatter.date(from: serverCat.updatedAt) ?? Date(),
                            syncedAt: serverTime
                        )
                        newCat.serverId = serverCat.id
                        modelContext.insert(newCat)
                    }
                }
            }

            // Apply server-side habit changes
            if let serverHabits = response.habits {
                let refreshedCategories = try modelContext.fetch(FetchDescriptor<Category>())
                for serverHabit in serverHabits {
                    let existing = allHabits.first(where: { $0.serverId == serverHabit.id })
                        ?? allHabits.first(where: { $0.id.uuidString == serverHabit.clientId })
                    if let existing {
                        if let serverUpdated = isoFormatter.date(from: serverHabit.updatedAt),
                           serverUpdated > existing.updatedAt {
                            existing.name = serverHabit.name
                            existing.frequencyType = serverHabit.frequencyType
                            existing.frequencyValue = serverHabit.frequencyValue
                            existing.isDeleted = serverHabit.isDeleted
                            if let catId = serverHabit.categoryId {
                                existing.category = refreshedCategories.first(where: { $0.serverId == catId })
                            } else {
                                existing.category = nil
                            }
                            existing.updatedAt = serverUpdated
                            existing.syncedAt = serverTime
                        }
                    } else {
                        let newHabit = Habit(
                            name: serverHabit.name,
                            frequencyType: serverHabit.frequencyType,
                            frequencyValue: serverHabit.frequencyValue,
                            isDeleted: serverHabit.isDeleted,
                            createdAt: isoFormatter.date(from: serverHabit.createdAt) ?? Date(),
                            updatedAt: isoFormatter.date(from: serverHabit.updatedAt) ?? Date(),
                            syncedAt: serverTime
                        )
                        newHabit.serverId = serverHabit.id
                        if let catId = serverHabit.categoryId {
                            newHabit.category = refreshedCategories.first(where: { $0.serverId == catId })
                        }
                        if let uuid = UUID(uuidString: serverHabit.clientId) {
                            newHabit.id = uuid
                        }
                        modelContext.insert(newHabit)
                    }
                }
            }

            // Apply server-side log changes
            if let serverLogs = response.logs {
                let refreshedHabits = try modelContext.fetch(FetchDescriptor<Habit>())
                for serverLog in serverLogs {
                    let existing = allLogs.first(where: { $0.serverId == serverLog.id })
                        ?? allLogs.first(where: { $0.id.uuidString == serverLog.clientId })
                    if let existing {
                        if let serverUpdated = isoFormatter.date(from: serverLog.updatedAt),
                           serverUpdated > existing.updatedAt {
                            existing.completedAt = isoFormatter.date(from: serverLog.completedAt) ?? existing.completedAt
                            existing.note = serverLog.note
                            existing.isExtra = serverLog.isExtra
                            existing.isDeleted = serverLog.isDeleted
                            existing.updatedAt = serverUpdated
                            existing.syncedAt = serverTime
                        }
                    } else {
                        let newLog = HabitLog(
                            completedAt: isoFormatter.date(from: serverLog.completedAt) ?? Date(),
                            note: serverLog.note,
                            isExtra: serverLog.isExtra,
                            isDeleted: serverLog.isDeleted,
                            createdAt: isoFormatter.date(from: serverLog.createdAt) ?? Date(),
                            updatedAt: isoFormatter.date(from: serverLog.updatedAt) ?? Date(),
                            syncedAt: serverTime
                        )
                        newLog.serverId = serverLog.id
                        newLog.habit = refreshedHabits.first(where: { $0.serverId == serverLog.habitId })
                        if let uuid = UUID(uuidString: serverLog.clientId) {
                            newLog.id = uuid
                        }
                        modelContext.insert(newLog)
                    }
                }
            }

            try modelContext.save()
            lastSyncedAt = serverTime

        } catch {
            // Sync failed; dirty records remain for next attempt
            print("Sync failed: \(error.localizedDescription)")
        }
    }

    func scheduleDebouncedSync(modelContext: ModelContext) {
        debouncedTask?.cancel()
        debouncedTask = Task {
            try? await Task.sleep(nanoseconds: UInt64(debounceInterval * 1_000_000_000))
            guard !Task.isCancelled else { return }
            await syncNow(modelContext: modelContext)
        }
    }

    func cancelDebouncedSync(modelContext: ModelContext) {
        debouncedTask?.cancel()
        debouncedTask = nil
        Task {
            await syncNow(modelContext: modelContext)
        }
    }
}
