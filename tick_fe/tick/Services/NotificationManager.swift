//
//  NotificationManager.swift
//  tick
//

import Foundation
import UserNotifications

final class NotificationManager {
    static let shared = NotificationManager()
    private let notificationCenter = UNUserNotificationCenter.current()
    private let reminderIdentifier = "tick.daily.reminder"

    private init() {}

    // MARK: - Permission

    func requestPermission() async -> Bool {
        do {
            let granted = try await notificationCenter.requestAuthorization(options: [.alert, .sound, .badge])
            return granted
        } catch {
            print("Notification permission error: \(error.localizedDescription)")
            return false
        }
    }

    func checkPermissionStatus() async -> UNAuthorizationStatus {
        let settings = await notificationCenter.notificationSettings()
        return settings.authorizationStatus
    }

    // MARK: - Reminder Scheduling

    func rescheduleReminder(pendingCount: Int, reminderTime: DateComponents) {
        // Cancel existing reminder first
        cancelReminder()

        guard pendingCount > 0 else { return }

        let content = UNMutableNotificationContent()
        content.title = "Tick - Habit Tracker"
        if pendingCount == 1 {
            content.body = "You have 1 habit still pending today."
        } else {
            content.body = "You have \(pendingCount) habits still pending today."
        }
        content.sound = .default
        content.badge = NSNumber(value: pendingCount)

        var triggerComponents = DateComponents()
        triggerComponents.hour = reminderTime.hour
        triggerComponents.minute = reminderTime.minute

        let trigger = UNCalendarNotificationTrigger(dateMatching: triggerComponents, repeats: true)
        let request = UNNotificationRequest(
            identifier: reminderIdentifier,
            content: content,
            trigger: trigger
        )

        notificationCenter.add(request) { error in
            if let error {
                print("Failed to schedule reminder: \(error.localizedDescription)")
            }
        }
    }

    func cancelReminder() {
        notificationCenter.removePendingNotificationRequests(withIdentifiers: [reminderIdentifier])
    }

    func clearBadge() {
        notificationCenter.setBadgeCount(0) { error in
            if let error {
                print("Failed to clear badge: \(error.localizedDescription)")
            }
        }
    }
}
