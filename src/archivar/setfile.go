package archivar

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Setfile struct {
	Description string   `description`
	Cards       []string `cards`
}

// Read formatted yaml file
func (c *Setfile) ReadFile(path string) *Setfile {

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		LogMessage("Could not open file", "red")
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		LogMessage(fmt.Sprintf("Unmarshal %v", err), "red")
		os.Exit(1)
	}

	return c
}
