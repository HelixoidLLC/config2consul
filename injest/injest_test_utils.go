package injest

import (
	"bytes"
	"config2consul/docker_compose"
	"config2consul/log"
	"crypto/tls"
	"errors"
	consulapi "github.com/hashicorp/consul/api"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

func checkIfListenningOnPort(address string) bool {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getHttpResponse(url string) (resp *http.Response, dfrFunc func(), err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err = client.Get(url)
	dfrFunc = func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
	return resp, dfrFunc, err
}

func checkIfHttpResponceNotEqual(url string, unwanted string) bool {
	resp, dfrFunc, err := getHttpResponse(url)
	if dfrFunc != nil {
		defer dfrFunc()
	}
	if err != nil {
		return false
	}

	if resp.StatusCode != 200 {
		return false
	}
	var bodyBuf bytes.Buffer
	if _, err := io.Copy(&bodyBuf, resp.Body); err != nil {
		log.Errorf("ERROR: %v", err)
		return false
	}
	response := bodyBuf.String()
	log.Debugf("HTTP response: '%s'", response)
	if response == unwanted {
		log.Error("HTTP Response is empty")
		return false
	}
	return true
}

type consulTestClient struct {
	address string
	client  *consulapi.Client
}

func createTestProject(projectPath string, CaFile string, CertFile string, KeyFile string) (*consulClient, func(), error) {
	projectName := "testproject"

	project, err := docker_compose.NewDockerComposeProjectFromFile(projectName, projectPath)
	if err != nil {
		return nil, nil, err
	}
	connection, deferFn, err := project.Up()
	if err != nil {
		log.Fatalf("Failed to start docker project: %s", err)
		return nil, deferFn, err
	}
	log.Debugf("Connection: %s", connection)

	// check if Consul container up
	if running, _ := docker_compose.IsRunning(projectName, "consul"); !running {
		log.Fatalf("Consul Container is not running. Aborting ...")
		return nil, deferFn, errors.New("Container is not running. Aborting ...")
	}

	// TODO: define an exit timeout
	// TODO: externalize ports and scheme
	for ok := false; !ok; ok = checkIfHttpResponceNotEqual("https://"+connection+":8501/v1/status/leader", "\"\"") {
		log.Debug("Waiting on HTTP connection https://" + connection + ":8501/v1/status/leader")
		time.Sleep(1000 * time.Millisecond)
	}
	log.Debug("Connected !")

	consul := consulClient{}
	dir := filepath.Dir(projectPath)

	consul.Client = createClient(connection+":8501", "https", "a49e7360-f150-463a-9a29-3eb186ffae1a", filepath.Join(dir, CaFile), filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile))

	return &consul, deferFn, nil
}
