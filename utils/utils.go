package utils

import (
	"context"
	"os"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pkg/errors"
)

// CreateFolder creates the folder in the specified `path`
// Print success info log on successfully ran command, return error if fail
func CreateFolder(ctx context.Context, path string, force bool) error {
	if force {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating directory")
			return errors.Errorf("Failed to create directory: %v \n", err)
		}
		log.WithContext(ctx).Infof("Directory created on %s \n", path)
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Error creating directory")
				return errors.Errorf("Failed to create directory: %v \n", err)
			}
			log.WithContext(ctx).Infof("Directory created on %s \n", path)
		} else {
			log.WithContext(ctx).WithError(err).Error("Directory already exists \n")
			return errors.Errorf("Directory already exists \n")
		}
	}

	return nil
}

// CreateFile creates pastel.conf file
// Print success info log on successfully ran command, return error if fail
func CreateFile(ctx context.Context, fileName string, force bool) (string, error) {

	if force {
		var file, err = os.Create(fileName)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating file")
			return "", errors.Errorf("Failed to create file: %v \n", err)
		}
		defer file.Close()
	} else {
		// check if file exists
		var _, err = os.Stat(fileName)

		// create file if not exists
		if os.IsNotExist(err) {
			var file, err = os.Create(fileName)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Error creating file")
				return "", errors.Errorf("Failed to create file: %v \n", err)
			}
			defer file.Close()
		} else {
			log.WithContext(ctx).WithError(err).Error("File already exists \n")
			return "", errors.Errorf("File already exists \n")
		}
	}

	log.WithContext(ctx).Infof("File created: %s \n", fileName)

	return fileName, nil
}
