package configurer

import "runtime"

func getAppDir() string {
	if runtime.GOOS == "windows" {
		return "AppData\\Roaming"
	} else if runtime.GOOS == "darwin" {
		return "Applications"
	}

	return ""
}

func getAppDataDir() string {
	if runtime.GOOS == "windows" {
		return "AppData\\Roaming"
	} else if runtime.GOOS == "darwin" {
		return "Library/Application Support/"
	}

	return ""
}
