package connection

import (
	"fmt"

	"os"

	"github.com/heroku/docker-registry-client/registry"
)

type connectionResult struct {
	srcHub  *registry.Registry
	destHub *registry.Registry
	err     error
}

//InitConnection initializes connections to specified registries
func InitConnection(srcRegistry string, srcUsername string, srcPassword string, srcInsecure bool, destRegistry string, destUsername string, destPassword string, destInsecure bool) (*registry.Registry, *registry.Registry) {
	fmt.Println("Establishing connections...")
	var srcHub *registry.Registry
	var destHub *registry.Registry
	res := make(chan *connectionResult)
	go connect(srcRegistry, srcUsername, srcPassword, srcInsecure, true, res)
	go connect(destRegistry, destUsername, destPassword, destInsecure, false, res)
	for index := 0; index < 2; index++ {
		reg := <-res
		if reg.err != nil {
			os.Exit(1)
		} else {
			if reg.destHub != nil {
				destHub = reg.destHub
			}
			if reg.srcHub != nil {
				srcHub = reg.srcHub
			}

		}
	}

	return srcHub, destHub
}
func connect(url string, username string, password string, insecure bool, src bool, ch chan *connectionResult) {
	var hub *registry.Registry
	var err error
	res := &connectionResult{}
	if insecure {
		hub, err = registry.NewInsecure(url, username, password)

	} else {
		hub, err = registry.New(url, username, password)
	}
	if err != nil {
		res.err = err
		fmt.Println("Cannot connect to registry: " + url)
		fmt.Println("Connection error: " + err.Error())
	}
	if src {
		res.srcHub = hub
	} else {
		res.destHub = hub
	}
	ch <- res
}
