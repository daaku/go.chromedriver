// Package chromedriver provides APIs to install and launch
// chromedriver servers. This allows for an embeddable webdriver
// environment. It provides some command line flags to control the
// global configuration.
package chromedriver

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/daaku/go.homedir"
	"github.com/daaku/go.httpzip"
	"github.com/facebookgo/freeport"
)

const (
	downloadBase = "https://chromedriver.storage.googleapis.com/"
	binaryBase   = "chromedriver"
)

var (
	version = flag.String(
		"chromedriver.version",
		"2.27",
		"chromedriver binary version to use")
	cacheDir = flag.String(
		"chromedriver.cache-dir",
		filepath.Join(homedir.Get(), ".chromedriver"),
		"location to store or load chromedriver binary from")
	verbose = flag.Bool(
		"chromedriver.v",
		false,
		"shows chromdriver server logs and more verbose output")
	port = flag.Int(
		"chromedriver.port",
		0,
		"port to bind chromedriver server to")

	once         = &sync.Once{}
	binaryPath   string
	installError error
)

// Server represents a running chromedriver instance.
type Server struct {
	Port int
	Cmd  *exec.Cmd
}

func init() {
	binaryPath = filepath.Join(*cacheDir, binaryBase+"-"+*version)
}

func getDownloadURL() string {
	os := map[string]string{
		"darwin":  "mac64",
		"linux":   "linux64",
		"windows": "win32",
	}[runtime.GOOS]
	return downloadBase + *version + "/chromedriver_" + os + ".zip"
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

// Fetch and install the chromedriver server binary if necessary.
func install() error {
	once.Do(func() {
		installError = realInstall()
	})
	return installError
}

func realInstall() error {
	if exists(binaryPath) {
		return nil
	}

	url := getDownloadURL()
	zipfile, err := httpzip.ReadURL(url)
	if err != nil {
		return fmt.Errorf(
			"reading zip content from http URL %s failed with error %s .", url, err)
	}
	found := false
	for _, file := range zipfile.File {
		if file.Name == binaryBase {
			found = true
			fileReader, err := file.Open()
			if err != nil {
				return fmt.Errorf(
					"error reading file stream for file %s in zip zip file "+
						"at URL %s with error %s.",
					binaryBase,
					url,
					err)
			}
			defer fileReader.Close()
			err = os.MkdirAll(filepath.Dir(binaryPath), os.FileMode(0777))
			if err != nil {
				return fmt.Errorf(
					"creating directory %s to store binary failed with error %s",
					filepath.Dir(binaryPath), err)
			}
			binaryWriter, err := os.Create(binaryPath)
			if err != nil {
				return fmt.Errorf(
					"error creating output file %s: %s", binaryPath, err)
			}
			defer binaryWriter.Close()
			err = binaryWriter.Chmod(os.FileMode(0777))
			if err != nil {
				return fmt.Errorf(
					"Error setting executable bit on file %s with err %s",
					binaryPath, err)
			}
			io.Copy(binaryWriter, fileReader)
			break
		}
	}
	if !found {
		return fmt.Errorf(
			"Could not find file %s in the zip file at URL %s.", binaryBase, url)
	}
	return nil
}

// Start a new chromedriver server. It is bound to a random port. This
// will install the server if necessary.
func Start() (*Server, error) {
	err := install()
	if err != nil {
		return nil, err
	}
	port := getPort()
	cmd := exec.Command(binaryPath, "-port="+strconv.Itoa(port))
	cmd.Dir = *cacheDir
	server := &Server{
		Port: port,
		Cmd:  cmd,
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout pipe for chromedriver server: %s.", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr pipe for chromedriver server: %s.", err)
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Failed to start binary %s with error %s.",
			binaryPath, err)
	}
	if *verbose {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	// TODO be smarter about this
	time.Sleep(500 * time.Millisecond)
	return server, nil
}

// URL returns the webdriver server URL.
func (s *Server) URL() string {
	return "http://0.0.0.0:" + strconv.Itoa(s.Port)
}

// Stop this server.
func (s *Server) Stop() error {
	return s.Cmd.Process.Kill()
}

// StopOrFatal stops this server, and fatals if it can't be stopped.
func (s *Server) StopOrFatal() {
	err := s.Stop()
	if err != nil {
		log.Fatalf("Failed to kill chromedriver with error %s", err)
	}
}
