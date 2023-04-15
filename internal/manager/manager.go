package manager

import (
	"fmt"
)

const portRangeStart = 5000
const portRangeEnd = 6000

type ContainerPortManager struct {
	ports []int
}

func NewContainerManager() *ContainerPortManager {
	return &ContainerPortManager{
		ports: []int{},
	}
}

func (cpm *ContainerPortManager) AssignFreePort() (int, error) {
	freePort, err := cpm.getFreePort()
	if err != nil {
		return 0, err
	}

	cpm.ports = append(cpm.ports, freePort)
	return freePort, nil
}

func (cpm *ContainerPortManager) getFreePort() (int, error) {
	for port := portRangeStart; port < portRangeEnd; port++ {
		if cpm.isPortFree(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free port found in range %d-%d", portRangeStart, portRangeEnd)
}

func (cpm *ContainerPortManager) isPortFree(port int) bool {
	for _, p := range cpm.ports {
		if p == port {
			return false
		}
	}

	return true
}
