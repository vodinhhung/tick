//
//  APIClient.swift
//  tick
//

import Foundation

// MARK: - API Error Types

enum APIError: LocalizedError {
    case invalidURL
    case unauthorized
    case badRequest(String)
    case notFound
    case conflict(String)
    case serverError(String)
    case networkError(Error)
    case decodingError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid URL"
        case .unauthorized: return "Unauthorized. Please sign in again."
        case .badRequest(let msg): return msg
        case .notFound: return "Resource not found"
        case .conflict(let msg): return msg
        case .serverError(let msg): return msg
        case .networkError(let err): return err.localizedDescription
        case .decodingError(let err): return "Failed to decode response: \(err.localizedDescription)"
        }
    }
}

// MARK: - Request/Response Models

struct AuthGoogleRequest: Codable {
    let idToken: String

    enum CodingKeys: String, CodingKey {
        case idToken = "id_token"
    }
}

struct AuthResponse: Codable {
    let token: String
    let expiresAt: String
    let user: UserProfile?

    enum CodingKeys: String, CodingKey {
        case token
        case expiresAt = "expires_at"
        case user
    }
}

struct UserProfile: Codable, Equatable {
    let id: Int
    let email: String
    let name: String
    let picture: String?
}

struct RefreshResponse: Codable {
    let token: String
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case token
        case expiresAt = "expires_at"
    }
}

struct CategoryResponse: Codable {
    let id: Int
    let name: String
    let isPreset: Bool
    let isDeleted: Bool
    let createdAt: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, name
        case isPreset = "is_preset"
        case isDeleted = "is_deleted"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct CategoriesResponse: Codable {
    let categories: [CategoryResponse]
}

struct CreateCategoryRequest: Codable {
    let name: String
}

struct HabitResponse: Codable {
    let id: Int
    let clientId: String
    let name: String
    let categoryId: Int?
    let frequencyType: String
    let frequencyValue: Int
    let isDeleted: Bool
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case clientId = "client_id"
        case name
        case categoryId = "category_id"
        case frequencyType = "frequency_type"
        case frequencyValue = "frequency_value"
        case isDeleted = "is_deleted"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct HabitsResponse: Codable {
    let habits: [HabitResponse]
    let serverTime: String

    enum CodingKeys: String, CodingKey {
        case habits
        case serverTime = "server_time"
    }
}

struct CreateHabitRequest: Codable {
    let clientId: String
    let name: String
    let categoryId: Int?
    let frequencyType: String
    let frequencyValue: Int

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case name
        case categoryId = "category_id"
        case frequencyType = "frequency_type"
        case frequencyValue = "frequency_value"
    }
}

struct UpdateHabitRequest: Codable {
    let name: String?
    let categoryId: Int?
    let frequencyType: String?
    let frequencyValue: Int?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case name
        case categoryId = "category_id"
        case frequencyType = "frequency_type"
        case frequencyValue = "frequency_value"
        case updatedAt = "updated_at"
    }
}

struct HabitLogResponse: Codable {
    let id: Int
    let clientId: String
    let habitId: Int
    let completedAt: String
    let note: String?
    let isExtra: Bool
    let isDeleted: Bool
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case clientId = "client_id"
        case habitId = "habit_id"
        case completedAt = "completed_at"
        case note
        case isExtra = "is_extra"
        case isDeleted = "is_deleted"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct HabitLogsResponse: Codable {
    let logs: [HabitLogResponse]
    let serverTime: String

    enum CodingKeys: String, CodingKey {
        case logs
        case serverTime = "server_time"
    }
}

struct CreateHabitLogRequest: Codable {
    let clientId: String
    let completedAt: String
    let note: String?

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case completedAt = "completed_at"
        case note
    }
}

// MARK: - Sync Models

struct SyncCategoryPayload: Codable {
    let clientId: String
    let name: String
    let isDeleted: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case name
        case isDeleted = "is_deleted"
        case updatedAt = "updated_at"
    }
}

struct SyncHabitPayload: Codable {
    let clientId: String
    let name: String
    let categoryClientId: String?
    let frequencyType: String
    let frequencyValue: Int
    let isDeleted: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case name
        case categoryClientId = "category_client_id"
        case frequencyType = "frequency_type"
        case frequencyValue = "frequency_value"
        case isDeleted = "is_deleted"
        case updatedAt = "updated_at"
    }
}

struct SyncLogPayload: Codable {
    let clientId: String
    let habitClientId: String
    let completedAt: String
    let note: String?
    let isDeleted: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case clientId = "client_id"
        case habitClientId = "habit_client_id"
        case completedAt = "completed_at"
        case note
        case isDeleted = "is_deleted"
        case updatedAt = "updated_at"
    }
}

struct SyncRequest: Codable {
    let lastSyncedAt: String?
    let categories: [SyncCategoryPayload]
    let habits: [SyncHabitPayload]
    let logs: [SyncLogPayload]

    enum CodingKeys: String, CodingKey {
        case lastSyncedAt = "last_synced_at"
        case categories, habits, logs
    }
}

struct SyncIDMap: Codable {
    let categories: [String: Int]?
    let habits: [String: Int]?
    let logs: [String: Int]?
}

struct SyncResponse: Codable {
    let serverTime: String
    let categories: [CategoryResponse]?
    let habits: [HabitResponse]?
    let logs: [HabitLogResponse]?
    let idMap: SyncIDMap?

    enum CodingKeys: String, CodingKey {
        case serverTime = "server_time"
        case categories, habits, logs
        case idMap = "id_map"
    }
}

struct HabitStatsResponse: Codable {
    let habitId: Int
    let currentStreak: Int
    let longestStreak: Int
    let completionRate: Double
    let totalCompletions: Int
    let computedAt: String

    enum CodingKeys: String, CodingKey {
        case habitId = "habit_id"
        case currentStreak = "current_streak"
        case longestStreak = "longest_streak"
        case completionRate = "completion_rate"
        case totalCompletions = "total_completions"
        case computedAt = "computed_at"
    }
}

struct APIErrorResponse: Codable {
    let code: String
    let message: String
}

// MARK: - API Client

final class APIClient {
    static let shared = APIClient()

    private let baseURL: String
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private let isoFormatter: ISO8601DateFormatter

    init(baseURL: String = "http://localhost:8080/api/v1") {
        self.baseURL = baseURL
        self.session = URLSession.shared
        self.decoder = JSONDecoder()
        self.encoder = JSONEncoder()
        self.isoFormatter = ISO8601DateFormatter()
    }

    // MARK: - Auth

    func authGoogle(idToken: String) async throws -> AuthResponse {
        let body = AuthGoogleRequest(idToken: idToken)
        return try await post(path: "/auth/google", body: body, authenticated: false)
    }

    func refreshToken() async throws -> RefreshResponse {
        return try await post(path: "/auth/refresh", body: Empty?.none, authenticated: true)
    }

    // MARK: - Categories

    func getCategories() async throws -> CategoriesResponse {
        return try await get(path: "/categories")
    }

    func createCategory(name: String) async throws -> CategoryResponse {
        let body = CreateCategoryRequest(name: name)
        return try await post(path: "/categories", body: body)
    }

    func updateCategory(id: Int, name: String) async throws -> CategoryResponse {
        let body = CreateCategoryRequest(name: name)
        return try await put(path: "/categories/\(id)", body: body)
    }

    func deleteCategory(id: Int) async throws {
        try await delete(path: "/categories/\(id)")
    }

    // MARK: - Habits

    func getHabits(updatedAfter: Date? = nil, includeDeleted: Bool = false) async throws -> HabitsResponse {
        var queryItems: [URLQueryItem] = []
        if let updatedAfter {
            queryItems.append(URLQueryItem(name: "updated_after", value: isoFormatter.string(from: updatedAfter)))
        }
        if includeDeleted {
            queryItems.append(URLQueryItem(name: "include_deleted", value: "true"))
        }
        return try await get(path: "/habits", queryItems: queryItems)
    }

    func createHabit(clientId: String, name: String, categoryId: Int?, frequencyType: String, frequencyValue: Int) async throws -> HabitResponse {
        let body = CreateHabitRequest(
            clientId: clientId,
            name: name,
            categoryId: categoryId,
            frequencyType: frequencyType,
            frequencyValue: frequencyValue
        )
        return try await post(path: "/habits", body: body)
    }

    func updateHabit(id: Int, name: String?, categoryId: Int?, frequencyType: String?, frequencyValue: Int?, updatedAt: Date) async throws -> HabitResponse {
        let body = UpdateHabitRequest(
            name: name,
            categoryId: categoryId,
            frequencyType: frequencyType,
            frequencyValue: frequencyValue,
            updatedAt: isoFormatter.string(from: updatedAt)
        )
        return try await put(path: "/habits/\(id)", body: body)
    }

    func deleteHabit(id: Int) async throws {
        try await delete(path: "/habits/\(id)")
    }

    // MARK: - Habit Logs

    func getHabitLogs(habitId: Int, from: Date? = nil, to: Date? = nil, updatedAfter: Date? = nil, includeDeleted: Bool = false) async throws -> HabitLogsResponse {
        var queryItems: [URLQueryItem] = []
        if let from {
            queryItems.append(URLQueryItem(name: "from", value: isoFormatter.string(from: from)))
        }
        if let to {
            queryItems.append(URLQueryItem(name: "to", value: isoFormatter.string(from: to)))
        }
        if let updatedAfter {
            queryItems.append(URLQueryItem(name: "updated_after", value: isoFormatter.string(from: updatedAfter)))
        }
        if includeDeleted {
            queryItems.append(URLQueryItem(name: "include_deleted", value: "true"))
        }
        return try await get(path: "/habits/\(habitId)/logs", queryItems: queryItems)
    }

    func createHabitLog(habitId: Int, clientId: String, completedAt: Date, note: String?) async throws -> HabitLogResponse {
        let body = CreateHabitLogRequest(
            clientId: clientId,
            completedAt: isoFormatter.string(from: completedAt),
            note: note
        )
        return try await post(path: "/habits/\(habitId)/logs", body: body)
    }

    func deleteHabitLog(habitId: Int, logId: Int) async throws {
        try await delete(path: "/habits/\(habitId)/logs/\(logId)")
    }

    // MARK: - Sync

    func sync(request: SyncRequest) async throws -> SyncResponse {
        return try await post(path: "/sync", body: request)
    }

    // MARK: - Stats

    func getHabitStats(habitId: Int) async throws -> HabitStatsResponse {
        return try await get(path: "/habits/\(habitId)/stats")
    }

    // MARK: - HTTP Helpers

    private struct Empty: Codable {}

    private func buildURL(path: String, queryItems: [URLQueryItem] = []) throws -> URL {
        guard var components = URLComponents(string: baseURL + path) else {
            throw APIError.invalidURL
        }
        if !queryItems.isEmpty {
            components.queryItems = queryItems
        }
        guard let url = components.url else {
            throw APIError.invalidURL
        }
        return url
    }

    private func buildRequest(url: URL, method: String, body: Data? = nil, authenticated: Bool = true) -> URLRequest {
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if authenticated, let token = KeychainManager.loadToken() {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        request.httpBody = body
        return request
    }

    private func handleResponse<T: Decodable>(_ data: Data, _ response: URLResponse) throws -> T {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.serverError("Invalid response")
        }

        switch httpResponse.statusCode {
        case 200, 201:
            do {
                return try decoder.decode(T.self, from: data)
            } catch {
                throw APIError.decodingError(error)
            }
        case 204:
            if let empty = Empty() as? T {
                return empty
            }
            throw APIError.serverError("Unexpected 204 response")
        case 400:
            let errorResp = try? decoder.decode(APIErrorResponse.self, from: data)
            throw APIError.badRequest(errorResp?.message ?? "Bad request")
        case 401:
            throw APIError.unauthorized
        case 404:
            throw APIError.notFound
        case 409:
            let errorResp = try? decoder.decode(APIErrorResponse.self, from: data)
            throw APIError.conflict(errorResp?.message ?? "Conflict")
        default:
            let errorResp = try? decoder.decode(APIErrorResponse.self, from: data)
            throw APIError.serverError(errorResp?.message ?? "Server error (\(httpResponse.statusCode))")
        }
    }

    private func handleVoidResponse(_ data: Data, _ response: URLResponse) throws {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.serverError("Invalid response")
        }

        switch httpResponse.statusCode {
        case 200, 201, 204:
            return
        case 401:
            throw APIError.unauthorized
        case 404:
            throw APIError.notFound
        default:
            let errorResp = try? decoder.decode(APIErrorResponse.self, from: data)
            throw APIError.serverError(errorResp?.message ?? "Server error (\(httpResponse.statusCode))")
        }
    }

    private func get<T: Decodable>(path: String, queryItems: [URLQueryItem] = []) async throws -> T {
        let url = try buildURL(path: path, queryItems: queryItems)
        let request = buildRequest(url: url, method: "GET")
        let (data, response) = try await session.data(for: request)
        return try handleResponse(data, response)
    }

    private func post<T: Decodable, B: Encodable>(path: String, body: B?, authenticated: Bool = true) async throws -> T {
        let url = try buildURL(path: path)
        var bodyData: Data?
        if let body {
            bodyData = try encoder.encode(body)
        }
        let request = buildRequest(url: url, method: "POST", body: bodyData, authenticated: authenticated)
        let (data, response) = try await session.data(for: request)
        return try handleResponse(data, response)
    }

    private func put<T: Decodable, B: Encodable>(path: String, body: B) async throws -> T {
        let url = try buildURL(path: path)
        let bodyData = try encoder.encode(body)
        let request = buildRequest(url: url, method: "PUT", body: bodyData)
        let (data, response) = try await session.data(for: request)
        return try handleResponse(data, response)
    }

    private func delete(path: String) async throws {
        let url = try buildURL(path: path)
        let request = buildRequest(url: url, method: "DELETE")
        let (data, response) = try await session.data(for: request)
        try handleVoidResponse(data, response)
    }
}
