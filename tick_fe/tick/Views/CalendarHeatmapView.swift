//
//  CalendarHeatmapView.swift
//  tick
//

import SwiftUI

struct CalendarHeatmapView: View {
    let habit: Habit
    @Binding var selectedMonth: Date

    @State private var selectedDay: Date?
    @State private var showingDayLogs = false

    private let calendar = Calendar.current
    private let columns = Array(repeating: GridItem(.flexible(), spacing: 4), count: 7)
    private let weekdaySymbols = Calendar.current.veryShortWeekdaySymbols

    private var monthTitle: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "MMMM yyyy"
        return formatter.string(from: selectedMonth)
    }

    private var daysInMonth: [Date?] {
        guard let monthInterval = calendar.dateInterval(of: .month, for: selectedMonth),
              let monthRange = calendar.range(of: .day, in: .month, for: selectedMonth) else {
            return []
        }

        let firstDayOfMonth = monthInterval.start
        let firstWeekday = calendar.component(.weekday, from: firstDayOfMonth)
        let leadingEmptyDays = firstWeekday - calendar.firstWeekday
        let adjustedLeadingDays = leadingEmptyDays < 0 ? leadingEmptyDays + 7 : leadingEmptyDays

        var days: [Date?] = Array(repeating: nil, count: adjustedLeadingDays)

        for day in monthRange {
            if let date = calendar.date(bySetting: .day, value: day, of: firstDayOfMonth) {
                days.append(date)
            }
        }

        return days
    }

    var body: some View {
        VStack(spacing: 12) {
            // Month navigation
            HStack {
                Button {
                    navigateMonth(by: -1)
                } label: {
                    Image(systemName: "chevron.left")
                        .font(.body)
                        .foregroundColor(.accentColor)
                        .frame(width: 36, height: 36)
                }

                Spacer()

                Text(monthTitle)
                    .font(.headline)

                Spacer()

                Button {
                    navigateMonth(by: 1)
                } label: {
                    Image(systemName: "chevron.right")
                        .font(.body)
                        .foregroundColor(canNavigateForward ? .accentColor : .secondary)
                        .frame(width: 36, height: 36)
                }
                .disabled(!canNavigateForward)
            }

            // Weekday headers
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(weekdaySymbols, id: \.self) { symbol in
                    Text(symbol)
                        .font(.caption2)
                        .fontWeight(.medium)
                        .foregroundColor(.secondary)
                        .frame(height: 20)
                }
            }

            // Calendar grid
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(Array(daysInMonth.enumerated()), id: \.offset) { _, day in
                    if let day {
                        dayCell(for: day)
                    } else {
                        Color.clear
                            .frame(height: 36)
                    }
                }
            }

            // Legend
            HStack(spacing: 12) {
                Spacer()
                legendItem(color: Color(.systemGray5), label: "None")
                legendItem(color: .green.opacity(0.3), label: "Partial")
                legendItem(color: .green, label: "Complete")
                Spacer()
            }
            .padding(.top, 4)
        }
        .padding(.vertical, 4)
        .sheet(isPresented: $showingDayLogs) {
            if let selectedDay {
                dayLogsSheet(for: selectedDay)
            }
        }
    }

    // MARK: - Day Cell

    private func dayCell(for date: Date) -> some View {
        let completionLevel = completionLevel(for: date)
        let isToday = calendar.isDateInToday(date)
        let isFuture = date > Date()

        return Button {
            if !isFuture {
                selectedDay = date
                showingDayLogs = true
            }
        } label: {
            Text("\(calendar.component(.day, from: date))")
                .font(.caption)
                .fontWeight(isToday ? .bold : .regular)
                .foregroundColor(isFuture ? .secondary.opacity(0.4) : .primary)
                .frame(width: 36, height: 36)
                .background(cellColor(level: completionLevel, isFuture: isFuture))
                .clipShape(RoundedRectangle(cornerRadius: 6))
                .overlay(
                    RoundedRectangle(cornerRadius: 6)
                        .stroke(isToday ? Color.accentColor : Color.clear, lineWidth: 2)
                )
        }
        .buttonStyle(.plain)
        .disabled(isFuture)
    }

    private func completionLevel(for date: Date) -> Int {
        let dayLogs = habit.logsForDate(date)
        if dayLogs.isEmpty { return 0 }

        if habit.isDaily {
            return dayLogs.count >= 1 ? 2 : 0
        } else {
            // For weekly, show partial if any logs exist for this day
            if dayLogs.count >= 1 {
                // Check if the week is fully completed
                if habit.isCompletedForDate(date) {
                    return 2
                }
                return 1
            }
            return 0
        }
    }

    private func cellColor(level: Int, isFuture: Bool) -> Color {
        if isFuture { return Color(.systemGray6).opacity(0.3) }
        switch level {
        case 0: return Color(.systemGray5)
        case 1: return .green.opacity(0.3)
        default: return .green.opacity(0.7)
        }
    }

    // MARK: - Legend

    private func legendItem(color: Color, label: String) -> some View {
        HStack(spacing: 4) {
            RoundedRectangle(cornerRadius: 3)
                .fill(color)
                .frame(width: 12, height: 12)
            Text(label)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
    }

    // MARK: - Navigation

    private var canNavigateForward: Bool {
        let nextMonth = calendar.date(byAdding: .month, value: 1, to: selectedMonth)!
        return nextMonth <= Date()
    }

    private func navigateMonth(by value: Int) {
        if let newMonth = calendar.date(byAdding: .month, value: value, to: selectedMonth) {
            selectedMonth = newMonth
        }
    }

    // MARK: - Day Logs Sheet

    private func dayLogsSheet(for date: Date) -> some View {
        NavigationStack {
            let logs = habit.logsForDate(date)
            List {
                if logs.isEmpty {
                    HStack {
                        Spacer()
                        VStack(spacing: 8) {
                            Image(systemName: "tray")
                                .font(.title2)
                                .foregroundColor(.secondary)
                            Text("No completions on this day")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                        }
                        .padding(.vertical, 20)
                        Spacer()
                    }
                } else {
                    ForEach(logs) { log in
                        HStack {
                            Image(systemName: log.isExtra ? "plus.circle.fill" : "checkmark.circle.fill")
                                .foregroundColor(log.isExtra ? .blue : .green)

                            VStack(alignment: .leading, spacing: 2) {
                                Text(log.completedAt, style: .time)
                                    .font(.subheadline)
                                if let note = log.note, !note.isEmpty {
                                    Text(note)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                            }

                            Spacer()

                            if log.isExtra {
                                Text("Extra")
                                    .font(.caption2)
                                    .foregroundColor(.blue)
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(Color.blue.opacity(0.1))
                                    .clipShape(Capsule())
                            }
                        }
                    }
                }
            }
            .navigationTitle(date.formatted(date: .abbreviated, time: .omitted))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") {
                        showingDayLogs = false
                    }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
