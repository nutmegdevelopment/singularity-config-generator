package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"strings"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// SingularityConfig ...
type SingularityConfig struct {
	DeployID                              string `yaml:"deploy-id"`
	KillOldNonLongRunningTasksAfterMillis int    `yaml:"kill-old-non-long-running-tasks-after-millis"`
	NumRetriesOnFailure                   int    `yaml:"num-retries-on-failure"`
	Owners                                []string
	RequestType                           string            `yaml:"request-type"`
	RequiredSlaveAttributes               map[string]string `yaml:"required-slave-attributes"`
	Schedule                              string
	ScheduledExpectedRuntimeMillis        int    `yaml:"scheduled-expected-runtime-millis"`
	RequestID                             string `yaml:"request-id"`
	Arguments                             []string
	ContainerInfo                         struct {
		Type   string
		Docker struct {
			Network string
			Image   string
		}
	} `yaml:"container-info"`
	Resources map[string]string
}

// SingularityRequestTemplate ...
// Make sure that guaranteed items are at the end - like id - so that preceeding
// elements can add a trailing comma "," if they exist.
const SingularityRequestTemplate = `
{
    "requestType": "{{.RequestType -}}",
    {{.WriteOwners}}
    "numRetriesOnFailure": {{.NumRetriesOnFailure}},
    "killOldNonLongRunningTasksAfterMillis": {{.KillOldNonLongRunningTasksAfterMillis}},
    {{.WriteRequiredSlaveAttributes}}
    "scheduledExpectedRuntimeMillis": {{.ScheduledExpectedRuntimeMillis}},
    "id": "{{.RequestID -}}"
}
`

// SingularityDeployTemplate ...
// Make sure that guaranteed items are at the end - like id - so that preceeding
// elements can add a trailing comma "," if they exist.
const SingularityDeployTemplate = `
{
    "deploy": {
        {{.WriteArguments}}
        "containerInfo": {
            "type": "DOCKER",
            "docker": {
                "network": "BRIDGE",
                "image": "registry.nutmeg.co.uk:8443/marathon-lb-zdd:latest"
            }
        },
        {{.WriteResources}}
        "requestId": "{{.RequestID}}",
        "id": "{{.DeployID}}"
    }
}
`

// WriteOwners ...
func (s SingularityConfig) WriteOwners() string {
	return WriteSlice("owners", s.Owners)

}

// WriteResources ...
func (s SingularityConfig) WriteResources() string {
	return WriteMap("resources", s.Resources)

}

// WriteRequiredSlaveAttributes ...
func (s SingularityConfig) WriteRequiredSlaveAttributes() string {
	return WriteMap("requiredSlaveAttributes", s.RequiredSlaveAttributes)

}

// WriteMap loops over the entries in a map and creates a JSON formatted
// string - with trailing comma ",".  If the map is empty then an empty
// string is returned.
func WriteMap(key string, m map[string]string) string {
	if len(m) == 0 {
		return ""
	}

	out := new(bytes.Buffer)
	out.WriteString(fmt.Sprintf(`"%s": {`, key))
	mapIndex := 0
	for key, value := range m {
		if mapIndex > 0 {
			out.WriteString(",")
		}
		out.WriteString(fmt.Sprintf(`"%s":"%s"`, key, makeStringJSONSafe(value)))
		mapIndex++
	}
	out.WriteString(`},`)

	return out.String()
}

// WriteArguments ...
func (s SingularityConfig) WriteArguments() string {
	return WriteSlice("arguments", s.Arguments)

}

// WriteSlice loops over the entries in a alice and creates a JSON formatted
// string - with trailing comma ",".  If the slice is empty then an empty
// string is returned.
func WriteSlice(key string, s []string) string {
	if len(s) == 0 {
		return ""
	}

	out := new(bytes.Buffer)
	out.WriteString(fmt.Sprintf(`"%s": [`, key))
	for index, value := range s {
		if index > 0 {
			out.WriteString(",")
		}
		out.WriteString(fmt.Sprintf(`"%s"`, makeStringJSONSafe(value)))
	}
	out.WriteString(`],`)

	return out.String()
}

func makeStringJSONSafe(s string) string {
	s = strings.Replace(s, `"`, `\"`, -1)
	return s
}

// Read in a file.
func readFile(filename string) ([]byte, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Read in a file and fatal error if there is a problem.
func readFileOrDie(filename string) []byte {
	b, err := readFile(filename)
	if err != nil {
		log.Fatalf("Unable to read file: %s. %s", filename, err)
	}
	return b
}

func main() {
	var singularityConfig SingularityConfig
	var err error

	var requestJSON = new(bytes.Buffer)
	var requestTemplate = template.New("Request template")
	requestTemplate, err = requestTemplate.Parse(SingularityRequestTemplate)

	var deployJSON = new(bytes.Buffer)
	var deployTemplate = template.New("Deploy template")
	deployTemplate, err = deployTemplate.Parse(SingularityDeployTemplate)

	yamlFile := readFileOrDie("singularity.yml")
	log.WithFields(log.Fields{
		"yaml": string(yamlFile),
	}).Debug("Read YAML config file")

	err = yaml.Unmarshal(yamlFile, &singularityConfig)
	if err != nil {
		log.Fatal("Unable to unmarshal yaml file")
	}
	log.WithFields(log.Fields{
		"config": singularityConfig,
	}).Debug("Unmarshalled config")

	requestTemplate.Execute(requestJSON, singularityConfig)
	log.WithFields(log.Fields{
		"json": requestJSON.String(),
	}).Info("Request JSON")
	deployTemplate.Execute(deployJSON, singularityConfig)
	log.WithFields(log.Fields{
		"json": deployJSON.String(),
	}).Info("Deploy JSON")
}
