//
//  HabitCardView.swift
//  tick
//

import SwiftUI

struct HabitCardView: View {
    let habit: Habit
    let onTick: () -> Void

    private var isCompletedToday: Bool {
        habit.isCompletedForDate(Date())
    }

    private var currentStreak: Int {
        StatsCalculator.currentStreak(for: habit)
    }

    private var todayCount: Int {
        habit.completionCountToday
    }

    var body: some View {
        HStack(spacing: 12) {
            // Tick button
            Button {
                onTick()
            } label: {
                ZStack {
                    Circle()
                        .stroke(isCompletedToday ? Color.green : Color(.systemGray4), lineWidth: 2.5)
                        .frame(width: 36, height: 36)
                    if isCompletedToday {
                        Circle()
                            .fill(Color.green)
                            .frame(width: 36, height: 36)
                        Image(systemName: "checkmark")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundColor(.white)
                    }
                }
            }
            .buttonStyle(.plain)

            // Habit info
            VStack(alignment: .leading, spacing: 4) {
                Text(habit.name)
                    .font(.body)
                    .fontWeight(.medium)
                    .foregroundColor(isCompletedToday ? .secondary : .primary)
                    .strikethrough(isCompletedToday)

                HStack(spacing: 8) {
                    if let category = habit.category {
                        Label(category.name, systemImage: "tag.fill")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    frequencyLabel
                }
            }

            Spacer()

            // Streak badge
            if currentStreak > 0 {
                VStack(spacing: 2) {
                    Image(systemName: "flame.fill")
                        .font(.system(size: 14))
                        .foregroundColor(.orange)
                    Text("\(currentStreak)")
                        .font(.caption2)
                        .fontWeight(.bold)
                        .foregroundColor(.orange)
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 4)
                .background(Color.orange.opacity(0.1))
                .clipShape(RoundedRectangle(cornerRadius: 8))
            }

            // Completion count for today (if multiple)
            if todayCount > 1 {
                Text("\(todayCount)x")
                    .font(.caption)
                    .fontWeight(.semibold)
                    .foregroundColor(.green)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.green.opacity(0.1))
                    .clipShape(Capsule())
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
    }

    private var frequencyLabel: some View {
        Group {
            if habit.isDaily {
                Label("Daily", systemImage: "calendar")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                Label("\(habit.frequencyValue)x/week", systemImage: "calendar")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
    }
}
