package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestCreateFolderWithForce(t *testing.T) {
	var cxt context.Context
	var path string
	var force bool
	path = "$HOME/test"
	force = true
	var err = CreateFolder(cxt, path, force)
	if err != nil {
		t.Fatalf(`CreateFolder Function Failed`)
	} else {
		t.Logf(fmt.Sprintf("CreateFolder Function OK : %s", path))
	}
	err = CreateFolder(cxt, path, force)
	if err != nil {
		t.Fatalf(`Create Folder Failed`)
	} else {
		t.Logf(fmt.Sprintf("CreateFolder Function OK : %s", path))
	}

}

func TestCreateFolderWithoutForce(t *testing.T) {
	var cxt context.Context
	var path string
	var force bool
	path = "$HOME/test"
	force = true
	var err = CreateFolder(cxt, path, force)
	if err != nil {
		t.Fatalf(fmt.Sprintf("%s %s", "CreateFolder Function Failed", err.Error()))
	} else {
		force = false
		err = CreateFolder(cxt, path, force)
		var wanted = "Directory already exists on $HOME/test"
		if err.Error() != wanted {
			t.Fatalf(fmt.Sprintf("%s %s", "CreateFolder Function Failed", err.Error()))
		} else {
			t.Logf(fmt.Sprintf("Can not Create Folder OK : %s", path))
		}
	}

}

func TestCreateFileWithForce(t *testing.T) {
	var cxt context.Context
	var path string
	var force bool
	path = "$HOME/test/createfilewithforce.txt"
	force = true
	fileName, err := CreateFile(cxt, path, force)
	if err != nil {
		t.Fatalf(`CreateFolder Function Failed`)
	} else {
		t.Logf(fmt.Sprintf("CreateFile Function OK : %s", fileName))
	}
	newFileName, err := CreateFile(cxt, path, force)
	if err != nil {
		t.Fatalf(`CreateFolder Function Failed`)
	} else {
		t.Logf(fmt.Sprintf("CreateFile Function OK : %s", newFileName))

	}

}

func TestCreateFileWithoutForce(t *testing.T) {
	var cxt context.Context
	var path string
	var force bool
	path = "$HOME/test/createfilewithoutforce.txt"
	force = true
	fileName, err := CreateFile(cxt, path, force)
	if err != nil {
		t.Fatalf(`Create Folder Failed`)
	} else {
		t.Logf(fmt.Sprintf("CreateFile Function OK : %s", fileName))
		force = false
		newFileName, err := CreateFile(cxt, path, force)
		var wanted = fmt.Sprintf("File already exists: %s", path)
		if err.Error() != wanted {
			t.Fatalf(fmt.Sprintf("%s %s", "CreateFile Function Failed", err.Error()))
		} else {
			t.Logf(fmt.Sprintf("Can not Create File OK : %s", newFileName))
		}
	}

}

func TestGenerateRandomString(t *testing.T) {
	length := 20
	var chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var result = GenerateRandomString(length)
	if len(result) != length {
		t.Logf("GenerateRandomString Function Failed; length = %d , len(result) = %d", length, len(result))
	}

	var isContain = strings.ContainsAny(result, chars)
	if isContain {
		t.Logf("GenerateRandomString Function OK")
	} else {
		t.Logf("GenerateRandomString Function Failed")
	}
}

func TestDeleteFile(t *testing.T) {
	var path = "$HOME/test/1.txt"
	var err = DeleteFile(path)
	if err == nil {
		t.Fatalf("DeleteFile Function failed; No such file 1.txt , but function deletes")
	}
	path = "$HOME/test/createfilewithoutforce.txt"
	err = DeleteFile(path)
	if err == nil {
		t.Logf("DeleteFile Function OK")
	} else {
		t.Fatalf("DeleteFile Function Failed")
	}

}

func TestWriteFile(t *testing.T) {
	var path = "$HOME/test/1.txt"
	var writeData = "GoTestData"
	var err = WriteFile(path, writeData)
	if err == nil {
		t.Fatalf("WriteFile Function failed; No such file 1.txt , but function writes")
	}
	path = "$HOME/test/createfilewithforce.txt"
	err = WriteFile(path, writeData)
	if err != nil {
		t.Fatalf("WriteFile Function Failed")
	} else {
		var file, err = os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf(fmt.Sprintf("Write File Function Failed %s ", err.Error()))
		} else {
			data, err := ioutil.ReadAll(file)
			if err != nil {
				t.Fatalf(fmt.Sprintf("Write File Function Failed %s ", err.Error()))
			} else {
				if string(data) == writeData {
					t.Logf("WriteFile Function OK")
				} else {
					t.Fatalf(fmt.Sprintf("Write File Function Failed write = %s result = %s ", writeData, string(data)))
				}
			}
		}

	}
}

func TestCheckFileExist(t *testing.T) {
	testCases := []struct {
		filePath      string
		expectedValue bool
	}{
		{
			filePath:      "$HOME/test/createfilewithforce.txt",
			expectedValue: true,
		},
		{
			filePath:      "$HOME/test/createfilewithforce1.txt",
			expectedValue: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		if CheckFileExist(testCase.filePath) == testCase.expectedValue {
			t.Logf("CheckFileExist Function OK")
		} else {
			t.Fatalf(fmt.Sprintf("CheckFileExist Function Failed: filePath=%s, expectedValue = %t , result = %t", testCase.filePath, testCase.expectedValue, CheckFileExist(testCase.filePath)))
		}
	}

}

func TestContains(t *testing.T) {
	testCases := []struct {
		stringList    []string
		tmpString     string
		expectedValue bool
	}{
		{
			stringList:    []string{"TestString1", "TestString2", "TestString3"},
			tmpString:     "TestString2",
			expectedValue: true,
		},
		{
			stringList:    []string{"TestString1", "TestString2", "TestString3"},
			tmpString:     "TestString",
			expectedValue: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		if Contains(testCase.stringList, testCase.tmpString) == testCase.expectedValue {
			t.Logf("Contains Function OK")
		} else {
			t.Fatalf(fmt.Sprintf("Contains Function Failed: stringList=%s, tmpString=%s, expectedValue = %t , result = %t", testCase.stringList, testCase.tmpString, testCase.expectedValue, Contains(testCase.stringList, testCase.tmpString)))
		}
	}

}

func TestCopyFile(t *testing.T) {
	var ctx context.Context

	// normal case
	var src = "$HOME/test/createfilewithforce.txt"
	var dstFolder = "$HOME"
	var dstFileName = "createfilewithforce.txt"

	var err = CopyFile(ctx, src, dstFolder, dstFileName)
	if err == nil {
		t.Logf("CopyFile Function OK")
	} else {
		t.Logf("CopyFile Function Failed")
	}

	// case that src file doesn't exist
	src = "$HOME/test/createfilewithforce1.txt"
	dstFolder = "$HOME/test/test1"
	dstFileName = "createfilewithforce.txt"

	err = CopyFile(ctx, src, dstFolder, dstFileName)
	if err == nil {
		t.Fatalf(fmt.Sprintf("CopyFile Function Failed; src not availabe but function copies file ; src=%s, dst = %s , dstFile = %s", src, dstFolder, dstFileName))
	} else {
		t.Logf("CopyFile Function OK")

	}

	// case that dstFolder doesn't exist
	src = "$HOME/test/createfilewithforce.txt"
	dstFolder = "$HOME/test/test1"
	dstFileName = "createfilewithforce.txt"

	err = CopyFile(ctx, src, dstFolder, dstFileName)
	if err == nil {
		t.Logf("CopyFile Function OK")
	}

	// case that dstFileName is ""
	src = "$HOME/test/createfilewithforce.txt"
	dstFolder = "$HOME/test/test1"
	dstFileName = ""

	err = CopyFile(ctx, src, dstFolder, dstFileName)
	if err == nil {
		t.Fatalf(fmt.Sprintf("CopyFile Function Failed; dstFileName not availabe but function copies file ; src=%s, dst = %s , dstFile = %s", src, dstFolder, dstFileName))
	} else {
		t.Logf("CopyFile Function OK")

	}

}
