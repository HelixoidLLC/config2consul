package docker_compose

import (
	"fmt"
	lclient "github.com/docker/libcompose/docker/client"
	"github.com/docker/libcompose/docker/container"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/options"
	"log"
	"github.com/docker/libcompose/docker"
	"golang.org/x/net/context"
	"os"
	"net/url"
	"testing"
	"strings"
)

type ContainerInfo struct {
	ID string
	State string
	Status string
}

type DockerComposeProject struct {
	project project.APIProject
	context docker.Context
	Name string
}

type DockerContainer struct {
	project *DockerComposeProject
	container interface{}
}

func newDockerComposeProject(project project.APIProject, context docker.Context) DockerComposeProject {
	return DockerComposeProject{
		project: project,
		context: context,
	}
}

func (pr *DockerComposeProject) ProjectName() string {
	return pr.context.ProjectName
}

func GetDockerHostIP() string {
	//DOCKER_CERT_PATH=/Users/{username}/.docker/machine/machines/dev
	//DOCKER_HOST=tcp://192.168.99.100:2376
	//DOCKER_MACHINE_NAME=dev
	//DOCKER_TLS_VERIFY=1
	env_docker_host := os.Getenv("DOCKER_HOST")
	if env_docker_host == "" {
		return ""
	}
	docker_host, err := url.Parse(env_docker_host)
	if err != nil {
		return ""
	}
	parts := strings.Split(docker_host.Host, ":")
	return parts[0]
}

func NewDockerComposeProjectFromString(composeProject string, t *testing.T) (*DockerComposeProject, error) {
	context := docker.Context{
		Context: project.Context{
			ComposeBytes: [][]byte{[]byte(composeProject)},
			ProjectName:  "test-project",
		},
	}
	pr, err := docker.NewProject(&context, nil)

	if err != nil {
		log.Fatal(err)
	}

	dpr := newDockerComposeProject(pr, context)
	return &dpr, err
}

func NewDockerComposeProjectFromFile(projectName string, composeFilePath string) (*DockerComposeProject, error) {
	context := docker.Context{
		Context: project.Context{
			ComposeFiles: []string{composeFilePath},
			ProjectName: projectName,
		},
	}
	pr, err := docker.NewProject(&context, nil)

	if err != nil {
		log.Fatal(err)
	}

	dpr := newDockerComposeProject(pr, context)
	return &dpr, err
}

func (pr *DockerComposeProject) Up() (string, func(), error) {
	err := pr.project.Up(context.Background(), options.Up{})

	if err != nil {
		log.Fatal(err)
	}

	dfrFunc := func() {
		pr.project.Down(context.Background(), options.Down{
			//RemoveVolume:  false,
			//RemoveOrphans: false,
			//RemoveImages:  "none",
			RemoveVolume:  true,
			RemoveOrphans: true,
			RemoveImages:  "local",
		})
	}

	return GetDockerHostIP(), dfrFunc, err
}

func IsRunning(projectName string, containerName string) (bool, error) {
	name := fmt.Sprintf("%s_%s_1", projectName, containerName)

	client, _ := lclient.Create(lclient.Options{})
	container, err := container.Get(context.Background(), client, name)
	if err != nil {
		return false, err
	}

	return container.State.Running, nil
}