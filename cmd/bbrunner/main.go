//  Copyright (c) 2019 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var configPath = flag.String("config", "config.json", "path to bbrunner config")
var execLabel = flag.String("label", "", "label for this run of exeuction")
var only = flag.String("only", "", "only run this test")

func main() {
	log.Printf("bbrunner started...")
	defer log.Printf("bbrunner complete")

	flag.Parse()

	if *execLabel == "" {
		*execLabel = time.Now().Format("2006-01-02")
	}
	log.Printf("using label: %s", *execLabel)

	configBytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	var config Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Vars: %v", config.Vars)

	// add the execution label to the available vars
	config.Vars["execLabel"] = *execLabel

	for testName, testConfig := range config.Tests {
		if *only != "" && *only != testName {
			log.Printf("Skipping test: %s", testName)
			continue
		}
		log.Printf("Preparing for test %s", testName)
		// add the test name to the available vars
		config.Vars["testName"] = testName

		for _, configName := range testConfig.Configs {
			log.Printf("Preparing configuration '%s'", configName)
			// add the config name to the available vars
			config.Vars["configName"] = configName

			// create a tmpDir
			tmpDir, err := ioutil.TempDir("", "bbrunner")
			if err != nil {
				log.Fatalf("error creating tmpDir: %v", err)
			}
			// and make that available to the vars as well
			config.Vars["tmpDir"] = tmpDir

			// now run setup
			log.Printf("Running Setup")
			for _, setup := range testConfig.Setup {
				err = setup.Run(config.Vars)
				if err != nil {
					log.Fatal(err)
				}
			}

			log.Printf("Running the requested %d times...", testConfig.Repeat)
			for i := 0; i < testConfig.Repeat; i++ {
				// add the run number to the available vars
				config.Vars["runNumber"] = fmt.Sprintf("%d", i)

				// now run tests
				log.Printf("Running Tests")
				for _, test := range testConfig.Tests {
					err = test.Run(config.Vars)
					if err != nil {
						log.Fatal(err)
					}
				}
				log.Printf("Finished Run %d", i)
			}

			log.Printf("Removing Run tmpDir: %s", tmpDir)
			err = os.RemoveAll(tmpDir)
			if err != nil {
				log.Fatalf("error removing all: %v", err)
			}
		}

		// put comma-separated list of configs into vars
		config.Vars["allConfigs"] = strings.Join(testConfig.Configs, ",")

		// now run aggregates
		log.Printf("Running Aggregates")
		for _, aggregate := range testConfig.Aggregates {
			err = aggregate.Run(config.Vars)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
