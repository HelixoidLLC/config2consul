package main

import (
	"config2consul/config"
	"config2consul/injest"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os"
	"runtime"
)

const version = "0.0.8"

var versionFlag bool

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.BoolVar(&versionFlag, "version", false, "prints current version")
	flag.Parse()
}

func main() {
	if versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	if err := config.ReadConfig(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(-1)
	}

	glog.Info("Starting config2consul v" + version)
	glog.Info("Connecting to Consul at: " + config.Conf.Address)

	if len(flag.Args()) == 0 {
		glog.Fatal("Missing path to the ACLs file")
	}
	glog.Info("Applying ACLs from " + flag.Args()[0])

	injest.ImportConfig(injest.ImportPath(flag.Args()[0]))
}
