//
//  AddEditHabitView.swift
//  tick
//

import SwiftUI
import SwiftData

struct AddEditHabitView: View {
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var authManager: AuthManager

    @Query(filter: #Predicate<Category> { !$0.isDeleted }, sort: \Category.name)
    private var categories: [Category]

    var habit: Habit?

    @State private var name: String = ""
    @State private var selectedCategory: Category?
    @State private var frequencyType: String = "daily"
    @State private var frequencyValue: Int = 1
    @State private var showingNewCategoryAlert = false
    @State private var newCategoryName = ""

    private var isEditing: Bool { habit != nil }
    private var isValid: Bool { !name.trimmingCharacters(in: .whitespaces).isEmpty }

    init(habit: Habit? = nil) {
        self.habit = habit
    }

    var body: some View {
        NavigationStack {
            Form {
                // Name section
                Section("Habit Name") {
                    TextField("e.g., Morning Run", text: $name)
                        .textInputAutocapitalization(.sentences)
                }

                // Category section
                Section("Category") {
                    Picker("Category", selection: $selectedCategory) {
                        Text("None").tag(nil as Category?)
                        ForEach(categories) { category in
                            Text(category.name).tag(category as Category?)
                        }
                    }

                    Button {
                        newCategoryName = ""
                        showingNewCategoryAlert = true
                    } label: {
                        Label("New Category", systemImage: "plus.circle")
                    }
                }

                // Frequency section
                Section("Frequency") {
                    Picker("Type", selection: $frequencyType) {
                        Text("Daily").tag("daily")
                        Text("Weekly").tag("weekly")
                    }
                    .pickerStyle(.segmented)

                    if frequencyType == "weekly" {
                        Stepper("Times per week: \(frequencyValue)", value: $frequencyValue, in: 1...7)
                    }
                }

                // Info section
                if !isEditing {
                    Section {
                        HStack {
                            Image(systemName: "info.circle")
                                .foregroundColor(.blue)
                            Text("Your habit will be saved locally and synced when you sign in.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                }
            }
            .navigationTitle(isEditing ? "Edit Habit" : "New Habit")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(isEditing ? "Save" : "Create") {
                        saveHabit()
                    }
                    .fontWeight(.semibold)
                    .disabled(!isValid)
                }
            }
            .alert("New Category", isPresented: $showingNewCategoryAlert) {
                TextField("Category name", text: $newCategoryName)
                Button("Cancel", role: .cancel) {}
                Button("Create") {
                    createCategory()
                }
            } message: {
                Text("Enter a name for the new category.")
            }
            .onAppear {
                if let habit {
                    name = habit.name
                    selectedCategory = habit.category
                    frequencyType = habit.frequencyType
                    frequencyValue = habit.frequencyValue
                }
            }
        }
    }

    // MARK: - Actions

    private func saveHabit() {
        let trimmedName = name.trimmingCharacters(in: .whitespaces)
        guard !trimmedName.isEmpty else { return }

        if let habit {
            // Editing existing habit
            habit.name = trimmedName
            habit.category = selectedCategory
            habit.frequencyType = frequencyType
            habit.frequencyValue = frequencyType == "daily" ? 1 : frequencyValue
            habit.updatedAt = Date()
        } else {
            // Creating new habit
            let newHabit = Habit(
                name: trimmedName,
                category: selectedCategory,
                frequencyType: frequencyType,
                frequencyValue: frequencyType == "daily" ? 1 : frequencyValue
            )
            modelContext.insert(newHabit)
        }

        try? modelContext.save()

        if authManager.appState.isAuthenticated {
            Task {
                await SyncManager.shared.scheduleDebouncedSync(modelContext: modelContext)
            }
        }

        dismiss()
    }

    private func createCategory() {
        let trimmedName = newCategoryName.trimmingCharacters(in: .whitespaces)
        guard !trimmedName.isEmpty else { return }

        let category = Category(name: trimmedName)
        modelContext.insert(category)
        try? modelContext.save()

        selectedCategory = category
    }
}
