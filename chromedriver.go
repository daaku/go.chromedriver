package chromedriver

import (
	"flag"
	"fmt"
	"github.com/nshah/go.freeport"
	"github.com/nshah/go.homedir"
	"github.com/nshah/go.httpzip"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	downloadBase = "http://chromedriver.googlecode.com/files/"
	binaryName   = "chromedriver"
)

var (
	version = flag.String(
		"chromedriver.version",
		"19.0.1068.0",
		"The chromedriver binary version to use.")
	cacheDir = flag.String(
		"chromedriver.cache-dir",
		filepath.Join(homedir.Get(), ".chromedriver"),
		"Location to store or load chromedriver binary from.")
	verbose = flag.Bool(
		"chromedriver.v",
		false,
		"Shows chromdriver server logs and more verbose output.")
	port = flag.Int(
		"chromedriver.port",
		0,
		"Port to bind chromedriver server to. Defaults to random port.")
)

type Server struct {
	Port int
	Cmd  *exec.Cmd
}

func getDownloadUrl() string {
	// TODO consider OS
	return downloadBase + "chromedriver_mac_" + *version + ".zip"
}

func getPort() int {
	if *port != 0 {
		return *port
	}
	port, err := freeport.Get()
	if err != nil {
		log.Fatalf("Failed to find a free port with error %s", err)
	}
	return port
}

func exists(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func Install() (string, error) {
	binaryPath := filepath.Join(*cacheDir, binaryName)
	if exists(binaryPath) {
		return binaryPath, nil
	}

	url := getDownloadUrl()
	zipfile, err := httpzip.ReadURL(url)
	if err != nil {
		return "", fmt.Errorf(
			"Reading zip content from http URL % failed with error %s .", url, err)
	}
	found := false
	for _, file := range zipfile.File {
		if file.Name == binaryName {
			found = true
			fileReader, err := file.Open()
			if err != nil {
				return "", fmt.Errorf(
					"Error reading file stream for file %s in zip zip file "+
						"at URL %s with error %s.",
					binaryName,
					url,
					err)
			}
			defer fileReader.Close()
			err = os.MkdirAll(filepath.Dir(binaryPath), os.FileMode(0777))
			if err != nil {
				return "", fmt.Errorf(
					"Creating directory %s to store binary failed with error %s",
					filepath.Dir(binaryPath), err)
			}
			binaryWriter, err := os.Create(binaryPath)
			if err != nil {
				return "", fmt.Errorf(
					"Error creating output file %s: %s", binaryPath, err)
			}
			defer binaryWriter.Close()
			err = binaryWriter.Chmod(os.FileMode(0777))
			if err != nil {
				return "", fmt.Errorf(
					"Error setting executable bit on file %s with err %s",
					binaryPath, err)
			}
			io.Copy(binaryWriter, fileReader)
			break
		}
	}
	if !found {
		return "", fmt.Errorf(
			"Could not find file %s in the zip file at URL %s.", binaryName, url)
	}
	return binaryPath, nil
}

func Start() (*Server, error) {
	binaryPath, err := Install()
	if err != nil {
		return nil, err
	}
	port := getPort()
	cmd := exec.Command(binaryPath, "-port="+strconv.Itoa(port))
	server := &Server{
		Port: port,
		Cmd:  cmd,
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Failed to start binary %s with error %s.",
			binaryPath, err)
	}
	// TODO be smarter about this
	time.Sleep(500 * time.Millisecond)
	return server, nil
}

func (s *Server) URL() string {
	return "http://0.0.0.0:" + strconv.Itoa(s.Port)
}

func (s *Server) Stop() error {
	return s.Cmd.Process.Kill()
}

func (s *Server) StopOrFatal() {
	err := s.Stop()
	if err != nil {
		log.Fatalf("Failed to kill chromedriver with error %s", err)
	}
}
