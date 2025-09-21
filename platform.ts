export enum Platform {
	Windows = "windows",
	MacOS = "macos",
	Linux = "linux",
	Android = "android",
	iOS = "ios",
}

// Internal global variables (computed once at module load)
let _isWindows = false
let _isMacOS = false
let _isLinux = false
let _isAndroid = false
let _isIOS = false
let _isMobile = false
let _userAgent: string | undefined
let _platform: Platform = Platform.Linux // Default fallback

interface INavigator {
	userAgent: string
	maxTouchPoints?: number
}
declare const navigator: INavigator

// Platform detection logic (runs once at module load)
if (typeof navigator === "object") {
	_userAgent = navigator.userAgent

	// iOS detection (must come before macOS since iOS contains "Mac")
	if (
		/iPad|iPhone|iPod/.test(_userAgent) ||
		(/Macintosh/.test(_userAgent) &&
			navigator.maxTouchPoints &&
			navigator.maxTouchPoints > 0)
	) {
		_isIOS = true
		_platform = Platform.iOS
		_isMobile = /iPhone|iPod/.test(_userAgent) || /Mobi/.test(_userAgent)
	}
	// Android detection
	else if (/Android/.test(_userAgent)) {
		_isAndroid = true
		_platform = Platform.Android
		_isMobile = true
	}
	// Windows detection
	else if (/Windows/.test(_userAgent)) {
		_isWindows = true
		_platform = Platform.Windows
		_isMobile = /Mobi/.test(_userAgent)
	}
	// macOS detection
	else if (/Macintosh|Mac OS X/.test(_userAgent)) {
		_isMacOS = true
		_platform = Platform.MacOS
		_isMobile = false
	}
	// Linux detection
	else if (/Linux/.test(_userAgent)) {
		_isLinux = true
		_platform = Platform.Linux
		_isMobile = /Mobi/.test(_userAgent)
	}
	// Fallback - check for mobile
	else {
		_isMobile = /Mobi/.test(_userAgent)
	}
}

// Exported constants (computed once)
export const isWindows = _isWindows
export const isMacOS = _isMacOS
export const isLinux = _isLinux
export const isAndroid = _isAndroid
export const isIOS = _isIOS
export const isMobile = _isMobile
export const platform = _platform
export const userAgent = _userAgent
