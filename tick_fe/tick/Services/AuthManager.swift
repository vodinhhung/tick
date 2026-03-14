//
//  AuthManager.swift
//  tick
//

import Foundation
import SwiftUI

enum AppState: Equatable {
    case guest
    case authenticated(UserProfile)

    var isAuthenticated: Bool {
        if case .authenticated = self { return true }
        return false
    }

    var userProfile: UserProfile? {
        if case .authenticated(let profile) = self { return profile }
        return nil
    }
}

@MainActor
class AuthManager: ObservableObject {
    @Published var appState: AppState = .guest
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
        checkExistingToken()
    }

    private func checkExistingToken() {
        if KeychainManager.isTokenValid() {
            // We have a valid token but no user profile cached.
            // In a full implementation, we would decode the JWT or call a /me endpoint.
            // For now, we set authenticated with a minimal profile from UserDefaults cache.
            if let profileData = UserDefaults.standard.data(forKey: "tick.userProfile"),
               let profile = try? JSONDecoder().decode(UserProfile.self, from: profileData) {
                appState = .authenticated(profile)
            }
            // If token is expiring soon, refresh it
            if KeychainManager.isTokenExpiringSoon() {
                Task {
                    await refreshTokenIfNeeded()
                }
            }
        } else {
            appState = .guest
        }
    }

    // MARK: - Google Sign-In

    /// Sign in with Google.
    /// NOTE: Requires GoogleSignIn-iOS SPM dependency to be added to the project.
    /// The actual Google Sign-In flow would use GIDSignIn.sharedInstance.signIn(withPresenting:)
    /// to obtain a GIDGoogleUser, then extract user.idToken.tokenString.
    /// This method accepts the idToken directly for testability.
    func signIn(withGoogleIDToken idToken: String) async {
        isLoading = true
        errorMessage = nil

        do {
            let response = try await apiClient.authGoogle(idToken: idToken)

            // Parse expiry date
            let formatter = ISO8601DateFormatter()
            guard let expiresAt = formatter.date(from: response.expiresAt) else {
                errorMessage = "Invalid expiry date from server"
                isLoading = false
                return
            }

            // Save token to Keychain
            KeychainManager.saveToken(response.token, expiresAt: expiresAt)

            // Save user profile to UserDefaults for quick reload
            if let user = response.user {
                if let profileData = try? JSONEncoder().encode(user) {
                    UserDefaults.standard.set(profileData, forKey: "tick.userProfile")
                }
                appState = .authenticated(user)
            }

            isLoading = false
        } catch {
            errorMessage = error.localizedDescription
            isLoading = false
        }
    }

    /// Placeholder for initiating Google Sign-In flow from a SwiftUI view.
    /// In a full implementation, this would:
    /// 1. Get the root view controller
    /// 2. Call GIDSignIn.sharedInstance.signIn(withPresenting: rootVC)
    /// 3. Extract the idToken from the result
    /// 4. Call signIn(withGoogleIDToken:)
    ///
    /// Requires adding GoogleSignIn-iOS SPM package:
    /// https://github.com/google/GoogleSignIn-iOS
    func initiateGoogleSignIn() {
        // TODO: Implement when GoogleSignIn-iOS SPM dependency is added
        // Example implementation:
        //
        // guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
        //       let rootVC = windowScene.windows.first?.rootViewController else {
        //     errorMessage = "Cannot find root view controller"
        //     return
        // }
        //
        // Task {
        //     do {
        //         let result = try await GIDSignIn.sharedInstance.signIn(withPresenting: rootVC)
        //         guard let idToken = result.user.idToken?.tokenString else {
        //             errorMessage = "Failed to get ID token from Google"
        //             return
        //         }
        //         await signIn(withGoogleIDToken: idToken)
        //     } catch {
        //         errorMessage = "Google Sign-In failed: \(error.localizedDescription)"
        //     }
        // }

        errorMessage = "Google Sign-In requires the GoogleSignIn-iOS SPM dependency. Add it to enable this feature."
    }

    // MARK: - Sign Out

    func signOut() {
        KeychainManager.deleteToken()
        UserDefaults.standard.removeObject(forKey: "tick.userProfile")
        appState = .guest

        // TODO: Also clear GIDSignIn session when GoogleSignIn-iOS is added:
        // GIDSignIn.sharedInstance.signOut()
    }

    // MARK: - Token Refresh

    func refreshTokenIfNeeded() async {
        guard KeychainManager.isTokenExpiringSoon(),
              KeychainManager.loadToken() != nil else {
            return
        }

        do {
            let response = try await apiClient.refreshToken()

            let formatter = ISO8601DateFormatter()
            guard let expiresAt = formatter.date(from: response.expiresAt) else { return }

            KeychainManager.saveToken(response.token, expiresAt: expiresAt)
        } catch {
            if case APIError.unauthorized = error {
                signOut()
            }
        }
    }
}
