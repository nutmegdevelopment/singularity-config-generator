package main // import "github.com/nutmegdevelopment/singularity-config-generator"

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultConfigFile                            = "singularity.yml"
	scheduledExpectedRuntimeMillisDefault        = 360000
	killOldNonLongRunningTasksAfterMillisDefault = 10000

	// Default output file names.
	deployFilename  = "singularity-deploy.json"
	requestFilename = "singularity-request.json"
)

var (
	debug           = false
	configFile      string
	deployTemplate  *template.Template
	requestTemplate *template.Template
	commandLineVars = make(stringmap)
)

// SingularityConfigData is used to store the config yaml template data
type SingularityConfigData map[string]interface{}

// SingularityConfig ...
type SingularityConfig struct {
	Command                               string
	DeployID                              string            `yaml:"deploy-id"`
	Env                                   map[string]string `yaml:"env,omitempty"`
	KillOldNonLongRunningTasksAfterMillis int               `yaml:"kill-old-non-long-running-tasks-after-millis"`
	NumRetriesOnFailure                   int               `yaml:"num-retries-on-failure"`
	Owners                                []string
	RequestType                           string            `yaml:"request-type"`
	RequiredSlaveAttributes               map[string]string `yaml:"required-slave-attributes"`
	Schedule                              string
	ScheduledExpectedRuntimeMillis        int    `yaml:"scheduled-expected-runtime-millis"`
	RequestID                             string `yaml:"request-id"`
	Arguments                             []string
	ContainerInfo                         SingularityContainerInfo `yaml:"container-info,omitempty"`
	Resources                             struct {
		NumPorts int     `yaml:"num-ports" json:"numPorts,omitempty"`
		MemoryMb float64 `yaml:"memory-mb" json:"memoryMb,omitempty"`
		CPUs     float64 `yaml:"cpus" json:"cpus,omitempty"`
		DiskMb   float64 `yaml:"disk-mb" json:"diskMb,omitempty"`
	} `json:"resources"`
	URIs []string `yaml:"uris"`
}

// SingularityPortMapping - see:
// https://github.com/HubSpot/Singularity/blob/master/Docs/reference/api.md#model-SingularityPortMapping
type SingularityPortMapping struct {
	HostPort          int                        `yaml:"hostPort" json:"hostPort"`
	ContainerPort     int                        `yaml:"containerPort" json:"containerPort" json:"containerPort"`
	ContainerPortType SingularityPortMappingType `yaml:"containerPortType,omitempty" json:"containerPortType,omitempty"`
	Protocol          string                     `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	HostPortType      SingularityPortMappingType `yaml:"hostPortType,omitempty" json:"hostPortType,omitempty"`
}

// SingularityPortMappingType - see:
// https://github.com/HubSpot/Singularity/blob/master/Docs/reference/api.md#model-SingularityPortMappingType
type SingularityPortMappingType struct {
}

// SingularityVolume - see:
// https://github.com/HubSpot/Singularity/blob/master/Docs/reference/api.md#model-SingularityVolume
type SingularityVolume struct {
	HostPath      string `json:"hostPath,omitempty"`
	ContainerPath string `json:"containerPath,omitempty"`
	Mode          string `json:"mode,omitempty"`
}

// Init ...
func (s *SingularityConfig) Init() {
	// Initialize fields that you do NOT want to have 'null' values,
	// like slices where you want to see '[]' in the JSON when it is
	// empty.
	s.ContainerInfo.Volumes = make([]SingularityVolume, 0)

	// Set any default values.
	if s.ScheduledExpectedRuntimeMillis == 0 {
		s.ScheduledExpectedRuntimeMillis = scheduledExpectedRuntimeMillisDefault
	}
	if s.KillOldNonLongRunningTasksAfterMillis == 0 {
		s.KillOldNonLongRunningTasksAfterMillis = killOldNonLongRunningTasksAfterMillisDefault
	}
}

// SingularityRequestTemplate ...
// Make sure that guaranteed items are at the end - like id - so that preceeding
// elements can add a trailing comma "," if they exist.
const SingularityRequestTemplate = `
{
    {{.WriteOwners -}}
	{{.WriteRequiredSlaveAttributes -}}
	{{.WriteSchedule -}}
	"killOldNonLongRunningTasksAfterMillis": {{.KillOldNonLongRunningTasksAfterMillis}},
	"numRetriesOnFailure": {{.NumRetriesOnFailure}},
	"requestType": "{{.RequestType -}}",
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
		{{.WriteContainerInfo}}
		{{.WriteEnv}}
        {{.WriteResources}}
        "requestId": "{{.RequestID}}",
        "id": "{{.DeployID}}"
    }
}
`

// SingularityContainerInfo - see:
// https://github.com/HubSpot/Singularity/blob/master/Docs/reference/api.md#model-SingularityContainerInfo
type SingularityContainerInfo struct {
	Docker  SingularityDockerInfo `json:"docker"`
	Type    string                `json:"type"`
	Volumes []SingularityVolume   `json:"volumes,omitempty"`
}

// SingularityDockerInfo - see:
// https://github.com/HubSpot/Singularity/blob/master/Docs/reference/api.md#model-SingularityDockerInfo
type SingularityDockerInfo struct {
	ForcePullImage   bool                     `json:"forcePullImage,omitempty"`
	Privileged       bool                     `json:"privileged"`
	Network          string                   `json:"network"`
	Image            string                   `json:"image"`
	Parameters       map[string]string        `json:"parameters,omitempty"`
	DockerParameters []map[string]string      `yaml:"dockerParameters,omitempty" json:"dockerParameters,omitempty"`
	PortMappings     []SingularityPortMapping `yaml:"portMappings,omitempty" json:"portMappings,omitempty"`
}

// WriteContainerInfo handles adding a containerInfo section if one is
// required.  Internal sections are only added if they are needed to avoid
// having null values.
func (s SingularityConfig) WriteContainerInfo() string {
	if s.ContainerInfo.Type == "" {
		return ""
	}

	return marshalJSON("containerInfo", s.ContainerInfo)
}

// marshalJSON takes an element name and an interface.  The interface is
// marshalled into a JSON string and appended to the element name to
// create a "key": "value", pair.
// A trailing comma is always added as elements written using this method
// are not expected to be the last elements in the JSON object (the 'id'
// element is always last to allow trailing commas to now break the JSON).
func marshalJSON(elementName string, i interface{}) string {
	j, err := json.Marshal(i)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"data":  fmt.Sprintf("%+v", i),
		}).Error("Error marshalling to json string")
	}

	return fmt.Sprintf(`"%s": %s,`, elementName, string(j))
}

// WriteOwners ...
func (s SingularityConfig) WriteOwners() string {
	if len(s.Owners) == 0 {
		return ""
	}

	return marshalJSON("owners", s.Owners)
}

// WriteResources is a map with
func (s SingularityConfig) WriteResources() string {
	return marshalJSON("resources", s.Resources)
}

// WriteSchedule ...
func (s SingularityConfig) WriteSchedule() string {
	if s.Schedule == "" {
		return ""
	}
	return marshalJSON("schedule", s.Schedule)
}

// WriteEnv ...
func (s SingularityConfig) WriteEnv() string {
	if len(s.Env) == 0 {
		return ""
	}
	return marshalJSON("env", s.Env)
}

// WriteRequiredSlaveAttributes ...
func (s SingularityConfig) WriteRequiredSlaveAttributes() string {
	if s.RequiredSlaveAttributes == nil {
		return ""
	}
	return marshalJSON("requiredSlaveAttributes", s.RequiredSlaveAttributes)

}

// WriteArguments ...
func (s SingularityConfig) WriteArguments() string {
	if len(s.Arguments) == 0 {
		return ""
	}
	return marshalJSON("arguments", s.Arguments)
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

// Write a file to the local filesystem.  Return an error if unsuccessful.
func writeFile(filename string, b []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	out, err := f.Write(b)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"bytes":    out,
		"filename": filename,
	}).Info("File created")

	return nil
}

// checkJSON tries to unmarshal the provided JSON into an interface{} - if
// not successful then the generated error is returned.
func checkJSON(b []byte) error {
	var iface interface{}
	err := json.Unmarshal(b, &iface)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"json": string(b),
	}).Debug("JSON is valid")

	return nil
}

func init() {
	var err error
	requestTemplate, err = template.New("Request template").Parse(SingularityRequestTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to parse the request template")
	}
	deployTemplate, err = template.New("Deploy template").Parse(SingularityDeployTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to parse the deploy template")
	}
}

// process performs three functions:
// 1, generates JSON from the provided template and SingularityConfig instance.
// 2, checks that the generated JSON is valid JSON.
// 3, writes the JSON to a local file.
func process(tmpl *template.Template, singularityConfig SingularityConfig, filename string) error {
	var jsonOutput = new(bytes.Buffer)

	// Create the JSON
	err := tmpl.Execute(jsonOutput, singularityConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Unable to execute the template")
		return err
	}
	log.WithFields(log.Fields{
		"json": jsonOutput.String(),
	}).Debug("Generated JSON")

	// Check that the JSON is valid.
	err = checkJSON(jsonOutput.Bytes())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"json":  jsonOutput.String(),
		}).Error("Invalid request JSON")
		return err
	}

	// Write the JSON to a file.
	err = writeFile(filename, jsonOutput.Bytes())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Unable to write JSON file")
		return err
	}

	return nil
}

func loadConfig() SingularityConfig {
	var singularityConfig SingularityConfig
	singularityConfig.Init()

	// Load the config through the go templating engine
	configTemplate, err := template.ParseFiles(configFile)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to parse the config file (go template)")
	}

	// Load vars from the command line
	var singularityConfigData SingularityConfigData
	for k, v := range commandLineVars {
		singularityConfigData[k] = v
	}

	// Exexute the template with the provided vars
	var rawConfig = new(bytes.Buffer)
	err = configTemplate.Execute(rawConfig, singularityConfigData)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to execute config template")
	}
	log.WithFields(log.Fields{
		"templateResult": rawConfig,
	}).Debug("Templated the config file")

	// Unmarshal the templated YAML config.
	err = yaml.Unmarshal(rawConfig.Bytes(), &singularityConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"filename":              configFile,
			"error":                 err,
			"message":               "Check that all expected replacements have been correctly applied",
			"yaml-after-templating": rawConfig.String(),
		}).Fatal("Unable to unmarshal config file")
	}
	log.WithFields(log.Fields{
		"config": singularityConfig,
	}).Debug("Unmarshalled config")

	return singularityConfig
}

func main() {
	flag.BoolVar(&debug, "debug", false, "debug output.")
	flag.StringVar(&configFile, "config-file", defaultConfigFile, "The name of the config file")
	flag.Var(&commandLineVars, "var", "[] of variables in the form of: key=value - multiple -var flags can be used, one per key/value pair.")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	singularityConfig := loadConfig()

	err := process(requestTemplate, singularityConfig, requestFilename)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": requestFilename,
		}).Fatal("Unrecoverable error occurred processing request")
	}

	err = process(deployTemplate, singularityConfig, deployFilename)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": deployFilename,
		}).Fatal("Unrecoverable error occurred processing deploy")
	}
}
