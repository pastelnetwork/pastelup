//go:build windows

package configurer

func getAppDir() string {
	return ""
}

func getAppDataDir() {
	return "AppData\\Local"
}
