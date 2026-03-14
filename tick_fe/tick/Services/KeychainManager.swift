//
//  KeychainManager.swift
//  tick
//

import Foundation
import Security

enum KeychainManager {
    private static let jwtKey = "tick.jwt"
    private static let jwtExpiryKey = "tick.jwt.expiry"
    private static let serviceName = "com.tick.app"

    // MARK: - JWT Token

    static func saveToken(_ token: String, expiresAt: Date) {
        save(key: jwtKey, data: Data(token.utf8))
        let expiryString = ISO8601DateFormatter().string(from: expiresAt)
        save(key: jwtExpiryKey, data: Data(expiryString.utf8))
    }

    static func loadToken() -> String? {
        guard let data = load(key: jwtKey) else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func loadTokenExpiry() -> Date? {
        guard let data = load(key: jwtExpiryKey),
              let string = String(data: data, encoding: .utf8) else {
            return nil
        }
        return ISO8601DateFormatter().date(from: string)
    }

    static func isTokenValid() -> Bool {
        guard loadToken() != nil,
              let expiry = loadTokenExpiry() else {
            return false
        }
        return expiry > Date()
    }

    static func isTokenExpiringSoon() -> Bool {
        guard let expiry = loadTokenExpiry() else { return true }
        let oneHourFromNow = Date().addingTimeInterval(3600)
        return expiry < oneHourFromNow
    }

    static func deleteToken() {
        delete(key: jwtKey)
        delete(key: jwtExpiryKey)
    }

    // MARK: - Generic Keychain Operations

    private static func save(key: String, data: Data) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)

        let addQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]
        SecItemAdd(addQuery as CFDictionary, nil)
    }

    private static func load(key: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else { return nil }
        return result as? Data
    }

    private static func delete(key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: serviceName,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}
