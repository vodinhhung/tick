//
//  tickApp.swift
//  tick
//
//  Created by Vo Hung on 7/10/25.
//

import SwiftUI
import SwiftData

@main
struct tickApp: App {
    @StateObject private var authManager = AuthManager()

    var sharedModelContainer: ModelContainer = {
        let schema = Schema([
            Category.self,
            Habit.self,
            HabitLog.self,
        ])
        let modelConfiguration = ModelConfiguration(schema: schema, isStoredInMemoryOnly: false)

        do {
            return try ModelContainer(for: schema, configurations: [modelConfiguration])
        } catch {
            fatalError("Could not create ModelContainer: \(error)")
        }
    }()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(authManager)
                .onAppear {
                    seedPresetCategories()
                    requestNotificationPermission()
                }
                .task {
                    // Sync on launch if authenticated
                    if authManager.appState.isAuthenticated {
                        let context = sharedModelContainer.mainContext
                        await SyncManager.shared.syncIfNeeded(modelContext: context)
                    }
                }
                .onChange(of: scenePhase) { _, newPhase in
                    handleScenePhaseChange(newPhase)
                }
        }
        .modelContainer(sharedModelContainer)
    }

    @Environment(\.scenePhase) private var scenePhase

    // MARK: - Seed Preset Categories

    private func seedPresetCategories() {
        let context = sharedModelContainer.mainContext
        let hasSeeded = UserDefaults.standard.bool(forKey: "tick.presetCategoriesSeeded")

        guard !hasSeeded else { return }

        for name in Category.presetNames {
            let category = Category(
                name: name,
                isPreset: true,
                syncedAt: Date() // Presets are not synced; mark as synced to avoid pushing
            )
            context.insert(category)
        }

        try? context.save()
        UserDefaults.standard.set(true, forKey: "tick.presetCategoriesSeeded")
    }

    // MARK: - Notification Permission

    private func requestNotificationPermission() {
        Task {
            _ = await NotificationManager.shared.requestPermission()
        }
    }

    // MARK: - Scene Phase Handling

    private func handleScenePhaseChange(_ phase: ScenePhase) {
        switch phase {
        case .active:
            NotificationManager.shared.clearBadge()
            if authManager.appState.isAuthenticated {
                Task {
                    let context = sharedModelContainer.mainContext
                    await authManager.refreshTokenIfNeeded()
                    await SyncManager.shared.syncIfNeeded(modelContext: context)
                }
            }
        case .background:
            if authManager.appState.isAuthenticated {
                let context = sharedModelContainer.mainContext
                Task {
                    await SyncManager.shared.cancelDebouncedSync(modelContext: context)
                }
            }
        default:
            break
        }
    }
}
