package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
)

var HYPEREXECUTE_CLI_DOWNLOAD_LINK = "https://downloads.lambdatest.com/hyperexecute/darwin/hyperexecute"

func download(url, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		log.Println("failed to create file " + filename + ": " + err.Error())
		return err
	}
	defer out.Close()

	res, err := http.Get(url)
	if err != nil {
		log.Println("failed to download " + url + ": " + err.Error())
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		log.Println("failed to write binary: " + err.Error())
		return err
	}

	return nil
}

func buildPayload(payloadFile *os.File, files map[string]string) error {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			log.Println("failed to create archive file " + name + ": " + err.Error())
			return err
		}

		if _, err = f.Write([]byte(content)); err != nil {
			log.Println("failed to write archive content for " + name + ": " + err.Error())
			return err
		}
	}

	if err := w.Close(); err != nil {
		log.Println("failed to close archive: " + err.Error())
		return err
	}

	_, err := buf.WriteTo(payloadFile)
	if err != nil {
		log.Println("failed to write to zip file: " + err.Error())
	}
	return err
}

func runOnHyperExecute(pwd, executableName string, files map[string]string) error {
	payloadFile, err := os.CreateTemp(pwd, "runonhyex")
	if err != nil {
		log.Println("failed to create temporary file: " + err.Error())
		return err
	}
	defer os.Remove(payloadFile.Name())

	if err = buildPayload(payloadFile, files); err != nil {
		return err
	}

	args := []string{"--no-track", "--use-zip", payloadFile.Name()}
	log.Printf("running hyperexecute with arguments %v\n", args)
	cmd := exec.Command(filepath.Join(pwd, executableName), args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("\n%s\n", output)
	return err
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Println(err.Error())
		return
	}

	if os.Getenv("LT_USERNAME") == "" {
		log.Println("Please set LT_USERNAME environment variable.")
		return
	}

	if os.Getenv("LT_ACCESS_KEY") == "" {
		log.Println("Please set LT_ACCESS_KEY environment variable.")
		return
	}

	if runtime.GOOS != "darwin" {
		log.Panic("platform (" + runtime.GOOS + ") not supported")
	}
	url := HYPEREXECUTE_CLI_DOWNLOAD_LINK

	executableName := path.Base(url)

	if _, err := os.Stat(executableName); errors.Is(err, os.ErrNotExist) {
		log.Println("binary missing, downloading to " + executableName)
		if err = download(url, executableName); err != nil {
			return
		}

		if err = os.Chmod(executableName, 0744); err != nil {
			log.Println("failed to mark binary as executable: " + err.Error())
			return
		}
	}

	files := map[string]string{}
	files[".hyperexecute.yaml"] = "version: 0.1\nrunson: win\nautosplit: true\nconcurrency: 1\ntestDiscovery:\n  type: raw\n  mode: dynamic\n  command: echo test\n\ntestRunnerCommand: node --version"

	runOnHyperExecute(pwd, executableName, files)
}
