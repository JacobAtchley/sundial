// Package eventkit reads macOS calendars through EventKit via cgo.
package eventkit

// AuthStatus mirrors EKAuthorizationStatus.
type AuthStatus int

const (
	StatusNotDetermined AuthStatus = 0
	StatusRestricted    AuthStatus = 1
	StatusDenied        AuthStatus = 2
	StatusFullAccess    AuthStatus = 3
	StatusWriteOnly     AuthStatus = 4
)

// SettingsURL opens System Settings → Privacy & Security → Calendars.
const SettingsURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_Calendars"
