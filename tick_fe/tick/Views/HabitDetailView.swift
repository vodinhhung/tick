//
//  HabitDetailView.swift
//  tick
//

import SwiftUI
import SwiftData

struct HabitDetailView: View {
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var authManager: AuthManager

    @Bindable var habit: Habit

    @State private var showingEditSheet = false
    @State private var showingDeleteAlert = false
    @State private var selectedMonth = Date()

    private var activeLogs: [HabitLog] {
        habit.activeLogs.sorted { $0.completedAt > $1.completedAt }
    }

    private var recentLogs: [HabitLog] {
        Array(activeLogs.prefix(20))
    }

    var body: some View {
        List {
            // Stats section
            statsSection

            // Heatmap section
            heatmapSection

            // Recent logs section
            recentLogsSection
        }
        .listStyle(.insetGrouped)
        .navigationTitle(habit.name)
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Menu {
                    Button {
                        showingEditSheet = true
                    } label: {
                        Label("Edit Habit", systemImage: "pencil")
                    }
                    Button(role: .destructive) {
                        showingDeleteAlert = true
                    } label: {
                        Label("Delete Habit", systemImage: "trash")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
            }
        }
        .sheet(isPresented: $showingEditSheet) {
            AddEditHabitView(habit: habit)
        }
        .alert("Delete Habit", isPresented: $showingDeleteAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Delete", role: .destructive) {
                deleteHabit()
            }
        } message: {
            Text("Are you sure you want to delete \"\(habit.name)\"? This will also delete all completion logs.")
        }
    }

    // MARK: - Stats Section

    private var statsSection: some View {
        Section("Statistics") {
            LazyVGrid(columns: [
                GridItem(.flexible()),
                GridItem(.flexible()),
                GridItem(.flexible())
            ], spacing: 16) {
                statCard(
                    title: "Current",
                    value: "\(StatsCalculator.currentStreak(for: habit))",
                    subtitle: "streak",
                    icon: "flame.fill",
                    color: .orange
                )
                statCard(
                    title: "Longest",
                    value: "\(StatsCalculator.longestStreak(for: habit))",
                    subtitle: "streak",
                    icon: "trophy.fill",
                    color: .yellow
                )
                statCard(
                    title: "Rate",
                    value: String(format: "%.0f%%", StatsCalculator.completionRate(for: habit)),
                    subtitle: "completion",
                    icon: "chart.line.uptrend.xyaxis",
                    color: .blue
                )
            }
            .padding(.vertical, 8)

            HStack {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundColor(.green)
                Text("Total completions: \(StatsCalculator.totalCompletions(for: habit))")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Spacer()
            }
        }
    }

    private func statCard(title: String, value: String, subtitle: String, icon: String, color: Color) -> some View {
        VStack(spacing: 6) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundColor(color)
            Text(value)
                .font(.title2)
                .fontWeight(.bold)
            Text(title)
                .font(.caption)
                .foregroundColor(.secondary)
            Text(subtitle)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(color.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Heatmap Section

    private var heatmapSection: some View {
        Section("Calendar") {
            CalendarHeatmapView(
                habit: habit,
                selectedMonth: $selectedMonth
            )
        }
    }

    // MARK: - Recent Logs Section

    private var recentLogsSection: some View {
        Section("Recent Activity") {
            if recentLogs.isEmpty {
                HStack {
                    Spacer()
                    VStack(spacing: 8) {
                        Image(systemName: "tray")
                            .font(.title2)
                            .foregroundColor(.secondary)
                        Text("No completions yet")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }
                    .padding(.vertical, 20)
                    Spacer()
                }
            } else {
                ForEach(recentLogs) { log in
                    logRow(log)
                }
                .onDelete(perform: deleteLogs)
            }
        }
    }

    private func logRow(_ log: HabitLog) -> some View {
        HStack(spacing: 12) {
            Image(systemName: log.isExtra ? "plus.circle.fill" : "checkmark.circle.fill")
                .foregroundColor(log.isExtra ? .blue : .green)
                .font(.title3)

            VStack(alignment: .leading, spacing: 2) {
                Text(log.completedAt, style: .date)
                    .font(.subheadline)
                    .fontWeight(.medium)
                Text(log.completedAt, style: .time)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()

            if let note = log.note, !note.isEmpty {
                Text(note)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
                    .frame(maxWidth: 120, alignment: .trailing)
            }

            if log.isExtra {
                Text("Extra")
                    .font(.caption2)
                    .fontWeight(.medium)
                    .foregroundColor(.blue)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.blue.opacity(0.1))
                    .clipShape(Capsule())
            }
        }
    }

    // MARK: - Actions

    private func deleteHabit() {
        habit.isDeleted = true
        habit.updatedAt = Date()
        try? modelContext.save()

        if authManager.appState.isAuthenticated {
            Task {
                await SyncManager.shared.scheduleDebouncedSync(modelContext: modelContext)
            }
        }

        dismiss()
    }

    private func deleteLogs(at offsets: IndexSet) {
        for index in offsets {
            let log = recentLogs[index]
            log.isDeleted = true
            log.updatedAt = Date()
        }
        try? modelContext.save()

        if authManager.appState.isAuthenticated {
            Task {
                await SyncManager.shared.scheduleDebouncedSync(modelContext: modelContext)
            }
        }
    }
}
