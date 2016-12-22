/*
 * Copyright 2016 Igor Moochnick
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
