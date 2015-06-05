package rancher

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/runconfig"
	"github.com/gorilla/websocket"
	rancherClient "github.com/rancherio/go-rancher/client"
	"github.com/rancherio/go-rancher/hostaccess"
	"github.com/rancherio/rancher-compose/librcompose/docker"
	"github.com/rancherio/rancher-compose/librcompose/project"
)

const (
	LB_IMAGE = "rancher/load-balancer"
)

var (
	colorPrefix chan string = make(chan string)
)

type IsDone func(*rancherClient.Resource) (bool, error)

type ContainerInspect struct {
	Name       string
	Config     *runconfig.Config
	HostConfig *runconfig.HostConfig
}

type RancherService struct {
	tty           bool
	name          string
	serviceConfig *project.ServiceConfig
	context       *Context
}

func (r *RancherService) Name() string {
	return r.name
}

func (r *RancherService) Config() *project.ServiceConfig {
	return r.serviceConfig
}

func NewService(name string, config *project.ServiceConfig, context *Context) *RancherService {
	return &RancherService{
		tty:           terminal.IsTerminal(int(os.Stdout.Fd())),
		name:          name,
		serviceConfig: config,
		context:       context,
	}
}

func (r *RancherService) Create() error {
	service, err := r.findExisting(r.name)

	if err == nil && service == nil {
		service, err = r.createService()
	}

	return err
}

func (r *RancherService) Up() error {
	var err error

	defer func() {
		if err == nil && r.context.Log {
			go r.Log()
		}
	}()

	service, err := r.findExisting(r.name)

	if err == nil && service == nil {
		service, err = r.createService()
	}

	if err != nil {
		return err
	}

	if service.State == "active" {
		return nil
	}

	if service.Actions["activate"] != "" {
		service, err = r.context.Client.Service.ActionActivate(service)
		err = r.wait(service)
	}

	return err
}

func (r *RancherService) Down() error {
	service, err := r.findExisting(r.name)

	if err == nil && service == nil {
		return nil
	}

	if err != nil {
		return err
	}

	if service.State == "inactive" {
		return nil
	}

	service, err = r.context.Client.Service.ActionDeactivate(service)
	return r.wait(service)
}

func (r *RancherService) Delete() error {
	service, err := r.findExisting(r.name)

	if err == nil && service == nil {
		return nil
	}

	if err != nil {
		return err
	}

	if service.State == "inactive" {
		return nil
	}

	err = r.context.Client.Service.Delete(service)
	if err != nil {
		return err
	}

	return r.wait(service)
}

func (r *RancherService) findExisting(name string) (*rancherClient.Service, error) {
	logrus.Debugf("Finding service %s", name)
	services, err := r.context.Client.Service.List(&rancherClient.ListOpts{
		Filters: map[string]interface{}{
			"environmentId": r.context.Environment.Id,
			"name":          name,
			"removed_null":  nil,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(services.Data) == 0 {
		return nil, nil
	}

	logrus.Debugf("Found service %s", name)
	return &services.Data[0], nil
}

func (r *RancherService) createLbService() (*rancherClient.Service, error) {
	var lbConfig *rancherClient.LoadBalancerConfig

	fmt.Printf("!!!! %#v\n", r.context.RancherConfig)
	if config, ok := r.context.RancherConfig[r.name]; ok {
		lbConfig = config.LoadBalancerConfig
	}

	_, err := r.context.Client.LoadBalancerService.Create(&rancherClient.LoadBalancerService{
		Name:               r.name,
		LoadBalancerConfig: lbConfig,
		LaunchConfig: rancherClient.Container{
			Ports: r.serviceConfig.Ports,
		},
		Scale:         int64(r.getConfiguredScale()),
		EnvironmentId: r.context.Environment.Id,
	})

	if err != nil {
		return nil, err
	}

	return r.findExisting(r.name)
}

func (r *RancherService) createNormalService() (*rancherClient.Service, error) {
	fmt.Println("!!!image " + r.serviceConfig.Image)

	launchConfig, err := r.createLaunchConfig()
	if err != nil {
		return nil, err
	}

	volumesFrom, err := r.getVolumesFrom()
	if err != nil {
		return nil, err
	}

	if r.serviceConfig.Build != "" {
		needBuild := true
		for _, remote := range project.ValidRemotes {
			if strings.HasPrefix(r.serviceConfig.Build, remote) {
				needBuild = false
				break
			}
		}

		if needBuild {
			image, url, err := Upload(r.context.Project, r.name)
			if err != nil {
				return nil, err
			}
			logrus.Infof("Build for %s available at %s", r.name, url)
			r.serviceConfig.Build = url

			if r.serviceConfig.Image == "" {
				r.serviceConfig.Image = image
			}

			launchConfig.Build = &rancherClient.DockerBuild{
				Context:    url,
				Dockerfile: r.serviceConfig.Dockerfile,
			}
			launchConfig.ImageUuid = "docker:" + image
		}

	}

	fmt.Printf("!!!! %#v\n", launchConfig.Build)
	return r.context.Client.Service.Create(&rancherClient.Service{
		Name:                   r.name,
		LaunchConfig:           launchConfig,
		Scale:                  int64(r.getConfiguredScale()),
		EnvironmentId:          r.context.Environment.Id,
		DataVolumesFromService: volumesFrom,
	})
}

func (r *RancherService) getConfiguredScale() int {
	scale := 1
	if config, ok := r.context.RancherConfig[r.Name()]; ok {
		if config.Scale > 0 {
			scale = config.Scale
		}
	}

	return scale
}

func (r *RancherService) createService() (*rancherClient.Service, error) {
	logrus.Infof("Creating service %s", r.name)

	var service *rancherClient.Service
	var err error

	if r.serviceConfig.Image == LB_IMAGE {
		service, err = r.createLbService()
	} else {
		service, err = r.createNormalService()
	}

	if err != nil {
		return nil, err
	}

	links, err := r.getLinks()
	if err == nil && len(links) > 0 {
		_, err = r.context.Client.Service.ActionSetservicelinks(service, &rancherClient.SetServiceLinksInput{
			ServiceLinks: links,
		})
	}

	err = r.wait(service)
	return service, err
}

func (r *RancherService) getLinks() (map[string]interface{}, error) {
	result := map[string]interface{}{}

	for _, link := range r.serviceConfig.Links {
		parts := strings.SplitN(link, ":", 2)
		name := parts[0]
		alias := name
		if len(parts) == 2 {
			alias = parts[1]
		}
		linkedService, err := r.findExisting(name)
		if err != nil {
			return nil, err
		}

		if linkedService == nil {
			logrus.Warnf("Failed to find service %s to link to", name)
		} else {
			result[alias] = linkedService.Id
		}
	}

	return result, nil
}

func (r *RancherService) getVolumesFrom() ([]string, error) {
	result := []string{}

	for _, name := range r.serviceConfig.VolumesFrom {
		volumesFromService, err := r.findExisting(name)
		if err != nil {
			return nil, err
		}

		if volumesFromService == nil {
			return nil, fmt.Errorf("Failed to find service %s to get volume from", name)
		} else {
			result = append(result, volumesFromService.Id)
		}
	}

	return result, nil
}

func (r *RancherService) createLaunchConfig() (rancherClient.Container, error) {
	var result rancherClient.Container

	schemasUrl := strings.SplitN(r.context.Client.Schemas.Links["self"], "/schemas", 2)[0]
	scriptsUrl := schemasUrl + "/scripts/transform"

	config, hostConfig, err := docker.Convert(r.serviceConfig)
	if err != nil {
		return result, err
	}

	dockerContainer := &ContainerInspect{
		Config:     config,
		HostConfig: hostConfig,
	}

	if r.serviceConfig.Name != "" {
		dockerContainer.Name = "/" + r.serviceConfig.Name
	} else {
		dockerContainer.Name = "/" + r.name
	}

	err = r.context.Client.Post(scriptsUrl, dockerContainer, &result)
	if r.serviceConfig.Build != "" {
		result.Build = &rancherClient.DockerBuild{
			Remote:     r.serviceConfig.Build,
			Dockerfile: r.serviceConfig.Dockerfile,
		}
	}

	return result, err
}

func (r *RancherService) wait(service *rancherClient.Service) error {
	for {
		if service.Transitioning != "yes" {
			return nil
		}

		time.Sleep(150 * time.Millisecond)

		err := r.context.Client.Reload(&service.Resource, service)
		if err != nil {
			return err
		}
	}
}

func (r *RancherService) waitInstance(service *rancherClient.Instance) error {
	for {
		if service.Transitioning != "yes" {
			return nil
		}

		time.Sleep(150 * time.Millisecond)

		err := r.context.Client.Reload(&service.Resource, service)
		if err != nil {
			return err
		}
	}
}

func (r *RancherService) Scale(count int) error {
	service, err := r.findExisting(r.name)
	if err != nil {
		return err
	}

	service.Scale = int64(count)

	service, err = r.context.Client.Service.Update(service, service)
	if err != nil {
		return err
	}

	return r.wait(service)
}

func (r *RancherService) containers() ([]rancherClient.Container, error) {
	service, err := r.findExisting(r.name)
	if err != nil {
		return nil, err
	}

	var instances rancherClient.ContainerCollection

	err = r.context.Client.GetLink(service.Resource, "instances", &instances)
	if err != nil {
		return nil, err
	}

	return instances.Data, nil
}

func (r *RancherService) Restart() error {
	containers, err := r.containers()
	if err != nil {
		return err
	}

	for _, container := range containers {
		logrus.Infof("Restarting container: %s", container.Name)
		instance, err := r.context.Client.Container.ActionRestart(&container)
		if err != nil {
			return err
		}

		r.waitInstance(instance)
		if instance.State != "running" {
			return fmt.Errorf("Failed to restart %s, in state: %s", instance.Name, instance.State)
		}
	}

	return nil
}

func (r *RancherService) Log() error {
	if r.serviceConfig.Image == LB_IMAGE {
		return nil
	}

	containers, err := r.containers()
	if err != nil {
		logrus.Errorf("Failed to list containers to log: %v", err)
		return err
	}

	for _, container := range containers {
		conn, err := (*hostaccess.RancherWebsocketClient)(r.context.Client).GetHostAccess(container.Resource, "logs", nil)
		if err != nil {
			logrus.Errorf("Failed to get logs for %s: %v", container.Name, err)
			continue
		}

		go r.pipeLogs(&container, conn)
	}

	return nil
}

func (r *RancherService) getLogFmt(container *rancherClient.Container) (string, string) {
	pad := 0
	for name, _ := range r.context.Project.Configs {
		if len(name) > pad {
			pad = len(name)
		}
	}
	pad += 3

	logFmt := "%s | %s"
	if r.tty {
		logFmt = <-colorPrefix + " %s"
	}

	name := fmt.Sprintf("%-"+strconv.Itoa(pad)+"s", strings.TrimPrefix(container.Name, r.context.ProjectName+"_"))

	return logFmt, name
}

func (r *RancherService) pipeLogs(container *rancherClient.Container, conn *websocket.Conn) {
	defer conn.Close()

	logFmt, name := r.getLogFmt(container)

	for {
		messageType, bytes, err := conn.ReadMessage()
		if messageType != websocket.TextMessage {
			continue
		}

		if err == io.EOF {
			return
		} else if err != nil {
			logrus.Errorf("Failed to read log: %v", err)
			return
		}

		if len(bytes) <= 3 {
			continue
		}

		message := string(bytes[3:])

		i := strings.Index(message, " ")
		if i > 0 {
			message = message[i+1:]
		}

		message = fmt.Sprintf(logFmt, name, string(message))

		if "01" == string(bytes[:2]) {
			fmt.Printf(message)
		} else {
			fmt.Fprint(os.Stderr, message)
		}
	}
}

func generateColors() {
	i := 0
	color_order := []string{
		"36",   // cyan
		"33",   // yellow
		"32",   // green
		"35",   // magenta
		"31",   // red
		"34",   // blue
		"36;1", // intense cyan
		"33;1", // intense yellow
		"32;1", // intense green
		"35;1", // intense magenta
		"31;1", // intense red
		"34;1", // intense blue
	}

	for {
		colorPrefix <- fmt.Sprintf("\033[%sm%%s |\033[0m", color_order[i])
		i = (i + 1) % len(color_order)
	}
}

func init() {
	go generateColors()
}
