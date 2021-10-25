//go:build darwin

package configurer

func getAppDir() string {
	return "Applications"
}

func getAppDataDir() string {
	return "Library/Application Support/"
}
