//
//  ContentView.swift
//  tick
//
//  Created by Vo Hung on 7/10/25.
//

import SwiftUI
import SwiftData

struct ContentView: View {
    @EnvironmentObject private var authManager: AuthManager

    var body: some View {
        TabView {
            TodayView()
                .tabItem {
                    Label("Today", systemImage: "checkmark.circle.fill")
                }

            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
        }
        .tint(.green)
    }
}

#Preview {
    ContentView()
        .modelContainer(for: [Category.self, Habit.self, HabitLog.self], inMemory: true)
        .environmentObject(AuthManager())
}
