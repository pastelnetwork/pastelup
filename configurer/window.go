//go:build windows

package configurer

func getAppDir() string {
	return ""
}

func getAppDataDir() string {
	return "AppData\\Roaming"
}
