//
//  TodayView.swift
//  tick
//

import SwiftUI
import SwiftData

struct TodayView: View {
    @Environment(\.modelContext) private var modelContext
    @EnvironmentObject private var authManager: AuthManager
    @Query(filter: #Predicate<Habit> { !$0.isDeleted }, sort: \Habit.createdAt)
    private var habits: [Habit]
    @Query(filter: #Predicate<Category> { !$0.isDeleted }, sort: \Category.name)
    private var categories: [Category]

    @State private var selectedCategory: Category?
    @State private var showingAddHabit = false
    @State private var showingNoteAlert = false
    @State private var pendingNoteHabit: Habit?
    @State private var noteText = ""
    @State private var isRefreshing = false

    private var filteredHabits: [Habit] {
        if let selectedCategory {
            return habits.filter { $0.category?.id == selectedCategory.id }
        }
        return habits
    }

    private var completedCount: Int {
        habits.filter { $0.isCompletedForDate(Date()) }.count
    }

    var body: some View {
        NavigationStack {
            ZStack(alignment: .bottomTrailing) {
                VStack(spacing: 0) {
                    // Category filter
                    categoryFilterBar

                    if filteredHabits.isEmpty {
                        emptyStateView
                    } else {
                        habitList
                    }
                }

                // Floating add button
                addButton
            }
            .navigationTitle("Today")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    progressIndicator
                }
            }
            .sheet(isPresented: $showingAddHabit) {
                AddEditHabitView()
            }
            .alert("Add a Note", isPresented: $showingNoteAlert) {
                TextField("How did it go?", text: $noteText)
                Button("Skip") {
                    completeHabit(pendingNoteHabit, note: nil)
                }
                Button("Save") {
                    completeHabit(pendingNoteHabit, note: noteText.isEmpty ? nil : noteText)
                }
            } message: {
                Text("Optionally add a note for this completion.")
            }
        }
    }

    // MARK: - Subviews

    private var categoryFilterBar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                filterChip(label: "All", isSelected: selectedCategory == nil) {
                    selectedCategory = nil
                }
                ForEach(categories) { category in
                    filterChip(label: category.name, isSelected: selectedCategory?.id == category.id) {
                        selectedCategory = category
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
        .background(Color(.systemBackground))
    }

    private func filterChip(label: String, isSelected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(label)
                .font(.subheadline)
                .fontWeight(isSelected ? .semibold : .regular)
                .padding(.horizontal, 14)
                .padding(.vertical, 7)
                .background(isSelected ? Color.accentColor : Color(.systemGray6))
                .foregroundColor(isSelected ? .white : .primary)
                .clipShape(Capsule())
        }
    }

    private var emptyStateView: some View {
        VStack(spacing: 16) {
            Spacer()
            Image(systemName: "leaf.fill")
                .font(.system(size: 56))
                .foregroundColor(.green.opacity(0.5))
            Text("No habits yet")
                .font(.title2)
                .fontWeight(.semibold)
            Text("Tap the + button to create your first habit")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding()
    }

    private var habitList: some View {
        List {
            // Progress header
            Section {
                HStack {
                    Image(systemName: "chart.bar.fill")
                        .foregroundColor(.accentColor)
                    Text("\(completedCount)/\(habits.count) completed today")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .listRowBackground(Color(.systemGroupedBackground))
            }

            // Habits
            Section {
                ForEach(filteredHabits) { habit in
                    NavigationLink(destination: HabitDetailView(habit: habit)) {
                        HabitCardView(habit: habit) {
                            onTickHabit(habit)
                        }
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .refreshable {
            await performSync()
        }
    }

    private var addButton: some View {
        Button {
            showingAddHabit = true
        } label: {
            Image(systemName: "plus")
                .font(.title2)
                .fontWeight(.semibold)
                .foregroundColor(.white)
                .frame(width: 56, height: 56)
                .background(Color.accentColor)
                .clipShape(Circle())
                .shadow(color: .accentColor.opacity(0.3), radius: 8, x: 0, y: 4)
        }
        .padding(.trailing, 20)
        .padding(.bottom, 20)
    }

    private var progressIndicator: some View {
        let progress = habits.isEmpty ? 0.0 : Double(completedCount) / Double(habits.count)
        return CircularProgressView(progress: progress)
            .frame(width: 28, height: 28)
    }

    // MARK: - Actions

    private func onTickHabit(_ habit: Habit) {
        pendingNoteHabit = habit
        noteText = ""
        showingNoteAlert = true
    }

    private func completeHabit(_ habit: Habit?, note: String?) {
        guard let habit else { return }

        let isExtra = habit.isCompletedForDate(Date())
        let log = HabitLog(
            habit: habit,
            completedAt: Date(),
            note: note,
            isExtra: isExtra
        )
        modelContext.insert(log)

        // Update habit's updatedAt to trigger sync
        habit.updatedAt = Date()

        try? modelContext.save()

        // Schedule sync if authenticated
        if authManager.appState.isAuthenticated {
            Task {
                await SyncManager.shared.scheduleDebouncedSync(modelContext: modelContext)
            }
        }

        // Update notification
        updateReminder()

        pendingNoteHabit = nil
    }

    private func performSync() async {
        guard authManager.appState.isAuthenticated else { return }
        await SyncManager.shared.syncNow(modelContext: modelContext)
    }

    private func updateReminder() {
        let pendingCount = habits.filter { !$0.isCompletedForDate(Date()) }.count
        let hour = UserDefaults.standard.integer(forKey: "tick.reminderHour")
        let minute = UserDefaults.standard.integer(forKey: "tick.reminderMinute")

        if hour == 0 && minute == 0 {
            // Default reminder at 8:00 PM
            var components = DateComponents()
            components.hour = 20
            components.minute = 0
            NotificationManager.shared.rescheduleReminder(pendingCount: pendingCount, reminderTime: components)
        } else {
            var components = DateComponents()
            components.hour = hour
            components.minute = minute
            NotificationManager.shared.rescheduleReminder(pendingCount: pendingCount, reminderTime: components)
        }
    }
}

// MARK: - Circular Progress View

struct CircularProgressView: View {
    let progress: Double

    var body: some View {
        ZStack {
            Circle()
                .stroke(Color(.systemGray5), lineWidth: 3)
            Circle()
                .trim(from: 0, to: progress)
                .stroke(Color.green, style: StrokeStyle(lineWidth: 3, lineCap: .round))
                .rotationEffect(.degrees(-90))
            if progress >= 1.0 {
                Image(systemName: "checkmark")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundColor(.green)
            }
        }
    }
}
