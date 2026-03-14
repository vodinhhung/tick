//
//  SettingsView.swift
//  tick
//

import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var authManager: AuthManager

    @State private var reminderDate: Date = {
        let hour = UserDefaults.standard.integer(forKey: "tick.reminderHour")
        let minute = UserDefaults.standard.integer(forKey: "tick.reminderMinute")
        var components = DateComponents()
        components.hour = hour == 0 ? 20 : hour
        components.minute = minute
        return Calendar.current.date(from: components) ?? Date()
    }()

    @State private var notificationsEnabled = false
    @State private var showingSignOutAlert = false

    var body: some View {
        NavigationStack {
            List {
                // Account section
                accountSection

                // Notifications section
                notificationsSection

                // About section
                aboutSection
            }
            .listStyle(.insetGrouped)
            .navigationTitle("Settings")
            .task {
                let status = await NotificationManager.shared.checkPermissionStatus()
                notificationsEnabled = (status == .authorized)
            }
        }
    }

    // MARK: - Account Section

    private var accountSection: some View {
        Section("Account") {
            switch authManager.appState {
            case .guest:
                VStack(alignment: .leading, spacing: 12) {
                    HStack(spacing: 12) {
                        Image(systemName: "person.circle.fill")
                            .font(.system(size: 40))
                            .foregroundColor(.secondary)
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Guest Mode")
                                .font(.headline)
                            Text("Sign in to sync your habits across devices")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    Button {
                        authManager.initiateGoogleSignIn()
                    } label: {
                        HStack {
                            Image(systemName: "person.badge.key.fill")
                            Text("Sign in with Google")
                                .fontWeight(.medium)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(Color.accentColor)
                        .foregroundColor(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                    }
                    .buttonStyle(.plain)

                    if authManager.isLoading {
                        HStack {
                            Spacer()
                            ProgressView()
                            Spacer()
                        }
                    }

                    if let error = authManager.errorMessage {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.red)
                    }
                }
                .padding(.vertical, 4)

            case .authenticated(let profile):
                HStack(spacing: 12) {
                    if let pictureURL = profile.picture, let url = URL(string: pictureURL) {
                        AsyncImage(url: url) { image in
                            image
                                .resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: 44, height: 44)
                                .clipShape(Circle())
                        } placeholder: {
                            Image(systemName: "person.circle.fill")
                                .font(.system(size: 40))
                                .foregroundColor(.accentColor)
                        }
                    } else {
                        Image(systemName: "person.circle.fill")
                            .font(.system(size: 40))
                            .foregroundColor(.accentColor)
                    }

                    VStack(alignment: .leading, spacing: 2) {
                        Text(profile.name)
                            .font(.headline)
                        Text(profile.email)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    Spacer()

                    Image(systemName: "checkmark.circle.fill")
                        .foregroundColor(.green)
                }

                Button(role: .destructive) {
                    showingSignOutAlert = true
                } label: {
                    HStack {
                        Image(systemName: "rectangle.portrait.and.arrow.right")
                        Text("Sign Out")
                    }
                }
                .alert("Sign Out", isPresented: $showingSignOutAlert) {
                    Button("Cancel", role: .cancel) {}
                    Button("Sign Out", role: .destructive) {
                        authManager.signOut()
                    }
                } message: {
                    Text("Your local habits will be kept on this device. You can sign back in anytime to sync.")
                }
            }
        }
    }

    // MARK: - Notifications Section

    private var notificationsSection: some View {
        Section("Reminders") {
            Toggle(isOn: $notificationsEnabled) {
                Label("Daily Reminder", systemImage: "bell.fill")
            }
            .onChange(of: notificationsEnabled) { _, enabled in
                if enabled {
                    Task {
                        let granted = await NotificationManager.shared.requestPermission()
                        if !granted {
                            notificationsEnabled = false
                        } else {
                            updateReminderSchedule()
                        }
                    }
                } else {
                    NotificationManager.shared.cancelReminder()
                }
            }

            if notificationsEnabled {
                DatePicker(
                    "Reminder Time",
                    selection: $reminderDate,
                    displayedComponents: .hourAndMinute
                )
                .onChange(of: reminderDate) { _, newDate in
                    let components = Calendar.current.dateComponents([.hour, .minute], from: newDate)
                    UserDefaults.standard.set(components.hour ?? 20, forKey: "tick.reminderHour")
                    UserDefaults.standard.set(components.minute ?? 0, forKey: "tick.reminderMinute")
                    updateReminderSchedule()
                }

                HStack {
                    Image(systemName: "info.circle")
                        .foregroundColor(.blue)
                        .font(.caption)
                    Text("You'll be reminded about pending habits at this time each day.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }
        }
    }

    // MARK: - About Section

    private var aboutSection: some View {
        Section("About") {
            HStack {
                Label("Version", systemImage: "info.circle")
                Spacer()
                Text(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0")
                    .foregroundColor(.secondary)
            }

            HStack {
                Label("Build", systemImage: "hammer")
                Spacer()
                Text(Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1")
                    .foregroundColor(.secondary)
            }

            Label("Made with SwiftUI & SwiftData", systemImage: "swift")
                .foregroundColor(.secondary)
                .font(.footnote)
        }
    }

    // MARK: - Helpers

    private func updateReminderSchedule() {
        let components = Calendar.current.dateComponents([.hour, .minute], from: reminderDate)
        // Use a placeholder pending count; the actual count will be set from TodayView
        var reminderComponents = DateComponents()
        reminderComponents.hour = components.hour
        reminderComponents.minute = components.minute
        NotificationManager.shared.rescheduleReminder(pendingCount: 1, reminderTime: reminderComponents)
    }
}
