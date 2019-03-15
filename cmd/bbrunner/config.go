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
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
)

type Config struct {
	Vars  map[string]string     `json:"vars"`
	Tests map[string]TestConfig `json:"tests"`
}

type TestConfig struct {
	Setup      []Command `json:"setup"`
	Tests      []Command `json:"tests"`
	Repeat     int       `json:"repeat"`
	Configs    []string  `json:"configs"`
	Aggregates []Command `json:"aggregates"`
}

type Command struct {
	Name string            `json:"command"`
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

func (c *Command) Run(vars map[string]string) error {
	command, err := exec.LookPath(c.Name)
	if err != nil {
		return fmt.Errorf("failed to locate command: %v", err)
	} else {
		log.Printf("Using command: %s", command)
	}

	// set up args
	args := make([]string, len(c.Args))
	tmplEvalBuffer := &bytes.Buffer{}
	for i, arg := range c.Args {
		tmpl := template.New("")
		_, err := tmpl.Parse(arg)
		if err != nil {
			return fmt.Errorf("error parsing template '%s' - error %v", arg, err)
		}
		tmpl.Execute(tmplEvalBuffer, vars)
		args[i] = tmplEvalBuffer.String()
		tmplEvalBuffer.Reset()
	}
	log.Printf("With args: %v", args)

	cmd := exec.Command(command, args...)

	// set up env
	env := os.Environ()
	for envKey, envVal := range c.Env {
		tmpl := template.New("")
		_, err := tmpl.Parse(envVal)
		if err != nil {
			log.Fatal(err)
		}
		tmpl.Execute(tmplEvalBuffer, vars)
		envvar := fmt.Sprintf("%s=%s", envKey, tmplEvalBuffer.String())
		log.Printf("Adding Environment Variable: %s", envvar)
		env = append(env, envvar)
		tmplEvalBuffer.Reset()
	}
	cmd.Env = env

	log.Printf("Starting Command")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", output)
		return fmt.Errorf("error exeucting command: %v", err)
	}
	fmt.Printf("%s\n", output)
	log.Printf("Finished Command")
	return nil
}
