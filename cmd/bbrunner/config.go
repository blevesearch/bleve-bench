package main

type Config struct {
	Vars  map[string]string     `json:"vars"`
	Tests map[string]TestConfig `json:"tests"`
}

type TestConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	Repeat     int               `json:"repeat"`
	Configs    []string          `json:"configs"`
	Aggregates []Aggregate       `json:"aggregates"`
}

type Aggregate struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
