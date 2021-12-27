package serra

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Setfile struct {
	Description string   `description`
	Cards       []string `cards`
	Value       []Value  `value`
}

type Value struct {
	Date  string
	Value float32
}

// Read formatted yaml file
func (s *Setfile) ReadFile(path string) *Setfile {

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		LogMessage("Could not open file", "red")
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, s)
	if err != nil {
		LogMessage(fmt.Sprintf("Unmarshal %v", err), "red")
		os.Exit(1)
	}

	return s
}

func (s *Setfile) Write(path string) *Setfile {
	data, err := yaml.Marshal(*s)
	if err != nil {
		LogMessage(fmt.Sprintf("Marshal %v", err), "red")
		os.Exit(1)
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		LogMessage("Could not write file", "red")
		os.Exit(1)
	}

	return s
}
