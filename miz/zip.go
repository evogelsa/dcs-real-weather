package miz

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/evogelsa/DCS-real-weather/v2/config"
	"github.com/evogelsa/DCS-real-weather/v2/logger"
)

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to dest, taken from https://golangcode.com/unzip-files-in-go/
func Unzip() ([]string, error) {
	logger.Infoln("unpacking mission file...")

	src := config.Get().RealWeather.Mission.Input
	logger.Infoln("source file:", src)
	dest := "mission_unpacked"

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
		if !strings.HasPrefix(
			fpath,
			filepath.Clean(dest)+string(os.PathSeparator),
		) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return filenames, err
			}
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(
			fpath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			f.Mode(),
		)
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}

	logger.Infoln("unpacked mission file")
	for _, fn := range filenames {
		logger.Debugf("unzipped: %s", fn)
	}

	return filenames, nil
}

// Zip takes the unpacked mission and recreates the mission file
// taken from https://golangcode.com/create-zip-files-in-go/
func Zip() error {
	logger.Infoln("repacking mission file...")

	baseFolder := "mission_unpacked/"

	dest := config.Get().RealWeather.Mission.Output
	outFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	addFiles(w, baseFolder, "")

	err = w.Close()
	if err != nil {
		return fmt.Errorf("error closing output file: %v", err)
	}

	logger.Infoln("repacked mission file")

	return nil
}

// Clean will remove the unpacked mission from directory
func Clean() {
	directory := "mission_unpacked/"
	os.RemoveAll(directory)
	logger.Infoln("removed unpacked mission")
}

// addFiles handles adding each file in directory to zip archive
// taken from https://golangcode.com/create-zip-files-in-go/
func addFiles(w *zip.Writer, basePath, baseInZip string) error {
	files, err := os.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("error reading directory %v: %v", basePath, err)
	}

	for _, file := range files {
		logger.Debugf("zipped: %s", basePath+file.Name())
		if !file.IsDir() {
			dat, err := os.ReadFile(basePath + file.Name())
			if err != nil {
				return fmt.Errorf(
					"error reading file %v: %v",
					basePath+file.Name(),
					err,
				)
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				return fmt.Errorf(
					"error creating file %v: %v",
					baseInZip+file.Name(),
					err,
				)
			}

			_, err = f.Write(dat)
			if err != nil {
				return fmt.Errorf("error writing data: %v", err)
			}

		} else if file.IsDir() {
			newBase := basePath + file.Name() + "/"
			err := addFiles(w, newBase, baseInZip+file.Name()+"/")
			if err != nil {
				return fmt.Errorf("error adding files from %v: %v", baseInZip+file.Name()+"/", err)
			}
		}
	}

	return nil
}
