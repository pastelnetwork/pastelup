//go:build darwin

package configurer

func getAppDir() string {
	return "Application"
}

func getAppDataDir() string {
	return "Library/Application Support/"
}
