package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var configPath = flag.String("config", "config.json", "path to bbrunner config")
var execLabel = flag.String("label", "", "label for this run of exeuction")

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
		log.Printf("Preparing for test %s", testName)
		// add the test name to the available vars
		config.Vars["testName"] = testName

		for _, configName := range testConfig.Configs {
			log.Printf("Preparing configuration '%s'", configName)
			// add the config name to the available vars
			config.Vars["configName"] = configName

			log.Printf("Running the requested %d times...", testConfig.Repeat)
			for i := 0; i < testConfig.Repeat; i++ {
				// add the run number to the available vars
				config.Vars["runNumber"] = fmt.Sprintf("%d", i)

				// create a tmpDir
				tmpDir, err := ioutil.TempDir("", "bbrunner")
				if err != nil {
					log.Fatalf("error creating tmpDir: %v", err)
				}
				// and make that available to the vars as well
				config.Vars["tmpDir"] = tmpDir

				command, err := exec.LookPath(testConfig.Command)
				if err != nil {
					log.Fatalf("failed to locate command: %v", err)
				} else {
					log.Printf("Using command: %s", command)
				}

				// set up args
				args := make([]string, len(testConfig.Args))
				tmplEvalBuffer := &bytes.Buffer{}
				for i, arg := range testConfig.Args {
					tmpl := template.New("")
					_, err := tmpl.Parse(arg)
					if err != nil {
						log.Fatalf("error parsing template '%s' - error %v", arg, err)
					}
					tmpl.Execute(tmplEvalBuffer, config.Vars)
					args[i] = tmplEvalBuffer.String()
					tmplEvalBuffer.Reset()
				}
				log.Printf("With args: %v", args)

				cmd := exec.Command(command, args...)

				// set up env
				env := os.Environ()
				for envKey, envVal := range testConfig.Env {
					tmpl := template.New("")
					_, err := tmpl.Parse(envVal)
					if err != nil {
						log.Fatal(err)
					}
					tmpl.Execute(tmplEvalBuffer, config.Vars)
					envvar := fmt.Sprintf("%s=%s", envKey, tmplEvalBuffer.String())
					log.Printf("Adding Environment Variable: %s", envvar)
					env = append(env, envvar)
					tmplEvalBuffer.Reset()
				}
				cmd.Env = env

				log.Printf("Starting Run %d", i)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("%s\n", output)
					log.Fatalf("error exeucting command: %v", err)
				}
				fmt.Printf("%s\n", output)
				log.Printf("Finished Run %d", i)
				log.Printf("Removing Run tmpDir: %s", tmpDir)
				err = os.RemoveAll(tmpDir)
				if err != nil {
					log.Fatalf("error removing all: %v", err)
				}
			}
		}

		// put comma-separated list of configs into vars
		config.Vars["allConfigs"] = strings.Join(testConfig.Configs, ",")

		// now run aggregates
		for _, aggregate := range testConfig.Aggregates {
			command, err := exec.LookPath(aggregate.Command)
			if err != nil {
				log.Fatalf("failed to locate command: %v", err)
			} else {
				log.Printf("Using command: %s", command)
			}

			// set up args
			args := make([]string, len(aggregate.Args))
			tmplEvalBuffer := &bytes.Buffer{}
			for i, arg := range aggregate.Args {
				tmpl := template.New("")
				_, err := tmpl.Parse(arg)
				if err != nil {
					log.Fatalf("error parsing template '%s' - error %v", arg, err)
				}
				tmpl.Execute(tmplEvalBuffer, config.Vars)
				args[i] = tmplEvalBuffer.String()
				tmplEvalBuffer.Reset()
			}
			log.Printf("With args: %v", args)

			cmd := exec.Command(command, args...)

			log.Printf("Starting Aggregate")
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("%s\n", output)
				log.Fatalf("error exeucting command: %v", err)
			}
			fmt.Printf("%s\n", output)
			log.Printf("Finished Aggregate")
		}
	}
}
