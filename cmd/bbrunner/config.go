package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os/exec"
)

type Config struct {
	Vars  map[string]string     `json:"vars"`
	Tests map[string]TestConfig `json:"tests"`
}

type TestConfig struct {
	Setup      []Command         `json:"setup"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	Repeat     int               `json:"repeat"`
	Configs    []string          `json:"configs"`
	Aggregates []Command         `json:"aggregates"`
}

type Command struct {
	Name string   `json:"command"`
	Args []string `json:"args"`
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
