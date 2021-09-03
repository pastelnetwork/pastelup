package utils

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/term"

	"github.com/dustin/go-humanize"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pkg/errors"
)

// CreateFolder creates the folder in the specified `path`
// Print success info log on successfully ran command, return error if fail
func CreateFolder(ctx context.Context, path string, force bool) error {
	create := func(path string) error {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating directory")
			return errors.Errorf("Failed to create directory(%s): %v", path, err)
		}
		log.WithContext(ctx).Infof("Directory created on %s", path)
		return nil
	}
	if force {
		return create(path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return create(path)
	}

	return fs.ErrExist
}

// CreateFile creates pastel.conf file
// Print success info log on successfully ran command, return error if fail
func CreateFile(ctx context.Context, filePath string, force bool) error {
	create := func(filePath string) error {
		file, err := os.Create(filePath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating file")
			return errors.Errorf("failed to create file: %v - err: %v", filePath, err)
		}
		defer file.Close()

		log.WithContext(ctx).Infof("File created: %s \n", filePath)
		return nil
	}

	if force {
		return create(filePath)
	}

	// create if not already exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return create(filePath)
	}

	return fs.ErrExist
}

// GenerateRandomString is a helper func for generating
// random string of the given input length
// returns the generated string
func GenerateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()

	return str
}

// DeleteFile deletes specified file
func DeleteFile(filePath string) error {
	e := os.Remove(filePath)
	if e != nil {
		return e
	}
	return nil
}

// WriteFile writes a file as data
func WriteFile(fileName string, data string) (err error) {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(data)

	return err
}

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total   uint64
	Context context.Context
}

// Write wites the number of bytes written to it.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()

	return n, nil
}

// PrintProgress make logs of the downloading file.
func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)

	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(ctx context.Context, filepath string, url string) error {
	log.WithContext(ctx).Infof("Download url: %s \n", url)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return errors.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("file not found")
	}

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	counter.Context = ctx
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return errors.Errorf("write file failed: %v", err)
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")
	// Close the file without defer so it can happen before Rename()
	out.Close()

	return os.Rename(filepath+".tmp", filepath)
}

// GetOS gets current OS.
func GetOS() constants.OSType {
	os := runtime.GOOS
	switch os {
	case "windows":
		return constants.Windows
	case "darwin":
		return constants.Mac
	case "linux":
		return constants.Linux
	default:
		return constants.Unknown
	}
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader, filenames ...string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		if !Contains(filenames, target) {
			continue
		}

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}

	defer r.Close()
	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			return filenames, fmt.Errorf("copy %s", err)
		}

		// Close the file without defer to close before next iteration of loop
		if err := outFile.Close(); err != nil {
			return filenames, fmt.Errorf("outfile close %s", err)
		}
		if err := rc.Close(); err != nil {
			return filenames, fmt.Errorf("rc close %s", err)
		}
	}

	return filenames, nil
}

// CheckFileExist check the file exist
func CheckFileExist(filepath string) bool {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false
	}
	return true
}

// CopyFile copies the file.
func CopyFile(ctx context.Context, src string, dstFolder string, dstFileName string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		log.WithContext(ctx).Errorf("%s file does not exist!!!", src)
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		log.WithContext(ctx).Errorf("%s is not a regular file", src)
		return errors.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		log.WithContext(ctx).Errorf("%s file cannot be opened!!!", src)
		return err
	}
	defer source.Close()

	if _, err := os.Stat(dstFolder); os.IsNotExist(err) {
		if err = CreateFolder(ctx, dstFolder, true); err != nil {
			log.WithContext(ctx).Errorf("Could not create folder on this %s", dstFolder)
			return CreateFolder(ctx, dstFolder, true)
		}
	}

	destination, err := os.Create(filepath.Join(dstFolder, dstFileName))
	if err != nil {
		log.WithContext(ctx).Errorf("Could not copy file to %s", dstFolder)
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	return err
}

// Contains check the slice contains the special string
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// GetChecksum gets the checksum of file
func GetChecksum(_ context.Context, fileName string) (checksum string, err error) {
	if _, err := os.Stat(fileName); err != nil {
		return "", errors.Errorf("file missing: %v", err)
	}

	f, err := os.Open(fileName)
	if err != nil {
		return "", errors.Errorf("open file failed: %v", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("copy file failed: %s", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func readCheckSum(path string) (string, error) {
	b, err := ioutil.ReadFile(path)

	if err != nil {
		return "", errors.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(b), "\n")

	if len(lines) == 0 {
		return "", errors.Errorf("empty checksum file")
	}

	tokens := strings.Split(lines[0], " ")
	if len(tokens) == 0 {
		return "", errors.Errorf("empty checksum line")
	}

	return tokens[0], nil
}

// GetChecksum gets the checksum of file
func VerifyChecksum(ctx context.Context, fileName, checkSumFile string) (err error) {
	calculatedHash, err := GetChecksum(ctx, fileName)
	if err != nil {
		return errors.Errorf("calculate hash of file failed: %v", err)
	}

	expectedHash, err := readCheckSum(checkSumFile)
	if err != nil {
		return errors.Errorf("read checksum file failed: %v", err)
	}

	if calculatedHash != expectedHash {
		return errors.New("invalid checksum")
	}

	return nil
}

// GetInstalledPackages returns a map which contains install packages
func GetInstalledPackages(ctx context.Context) map[string]bool {
	m := make(map[string]bool)
	switch GetOS() {
	case constants.Linux:
		cmd := exec.Command("dpkg-query", "-f", "${binary:Package} ", "-W")
		stdout, err := cmd.Output()
		if err != nil {
			log.WithContext(ctx).Errorf("failed to execute cmd: %v", err)
			return m
		}

		packages := strings.Split(string(stdout), " ")
		for _, p := range packages {
			tokens := strings.Split(p, ":")
			if tokens[0] != "" {
				m[tokens[0]] = true
			}
		}
	case constants.Mac:
		paths := os.Getenv("PATH")
		for _, path := range strings.Split(paths, ":") {
			files, err := ioutil.ReadDir(path)
			if err != nil {
				log.WithContext(ctx).Errorf("failed to read dir: %v", err)
				continue
			}

			for _, file := range files {
				m[file.Name()] = true
			}
		}
	}
	return m
}

// GetServiceConfig returns service configuration
func GetServiceConfig(toolType constants.ToolType, format string, value interface{}) (string, error) {
	temp, err := template.New(string(toolType)).Parse(format)
	if err != nil {
		return "", errors.Errorf("failed to parse service config template: %v", err)
	}

	var buf bytes.Buffer
	if err = temp.Execute(&buf, value); err != nil {
		return "", errors.Errorf("failed to execute service config template: %v", err)
	}

	return buf.String(), nil
}

// ReadStrings reads string???
func ReadStrings(comment string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s-> ", comment)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

// Credentials reads user credentials from standard input
func Credentials(userName string, needPassword bool) (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	var err error
	username := userName
	if len(userName) == 0 {
		fmt.Print("Enter Username: ")
		username, err = reader.ReadString('\n')
		if err != nil {
			return "", "", err
		}
	}

	password := ""
	if needPassword {
		fmt.Print("Enter Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", "", err
		}
		fmt.Print("\n")
		password = string(bytePassword)
	}

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

// IsValidNetworkOpt does a lookup in constants.NetworkModes
func IsValidNetworkOpt(val string) bool {
	val = strings.TrimSpace(val)
	for _, n := range constants.NetworkModes {
		if strings.EqualFold(n, val) {
			return true
		}
	}

	return false
}
