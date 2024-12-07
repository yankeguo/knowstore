package main

import (
	"flag"
	"log"
	"time"

	"github.com/yankeguo/rg"
)

var (
	optKubeconfig         string
	optContainerdStateDir string
	optInterval           time.Duration
)

func main() {

	flag.StringVar(&optKubeconfig, "kubeconfig", "", "the kubeconfig file, if empty, use in-cluster config")
	flag.StringVar(&optContainerdStateDir, "containerd.state.dir", "/run/containerd", "the containerd state dir")
	flag.DurationVar(&optInterval, "interval", 10*time.Minute, "the interval to refresh, set to 0 to run once")
	flag.Parse()

loop:
	if err := once(); err != nil {
		log.Println("execution failed:", err.Error())
	}

	if optInterval > 0 {
		time.Sleep(optInterval)
		goto loop
	}
}

func once() (err error) {
	defer rg.Guard(&err)

	return
}
