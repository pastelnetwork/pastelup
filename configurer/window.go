//go:build windows

package configurer

func getAppDir() string {
	return "AppData\\Roaming"
}

func getAppDataDir() string {
	return "AppData\\Roaming"
}
