package ide

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
)

type vsCodeLunchConfig struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Request    string            `json:"request"`
	Mode       string            `json:"mode"`
	Program    string            `json:"program"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	EnvFile    string            `json:"envFile,omitempty"`
	BuildFlags string            `json:"buildFlags,omitempty"`
}

func generateVSCode(tasks []Task, root string) {
	err := os.MkdirAll(path.Join(root, ".vscode"), 0755)
	if err != nil {
		log.Panic(err)
	}
	vscodeLaunchPath := path.Join(root, ".vscode", "launch.json")
	vsCodeConfig := map[string]interface{}{}

	f, err := os.Open(vscodeLaunchPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Panicln("cannot open vscode confgi", err)
	}

	if err == nil {
		if err := json.NewDecoder(f).Decode(&vsCodeConfig); err != nil {
			log.Panicln("cannot decode vsCodeConfig", err)
		}
		if err := f.Close(); err != nil {
			log.Panicln("cannot close f", err)
		}
	}

	_, ok := vsCodeConfig["version"]
	if !ok {
		vsCodeConfig["version"] = "0.2.0"
	}

	vsCodeTasks := make(map[string]vsCodeLunchConfig, len(tasks))
	for _, task := range tasks {
		vsCodeTasks[task.Name] = vsCodeLunchConfig{
			Name:    task.Name,
			Type:    "go",
			Request: "launch",
			Mode:    "auto",
			Program: path.Join("${workspaceFolder}", task.Program),
			Args:    task.Args,
			Env:     task.Env,
			EnvFile: "",
			// for some reason vscode works best with '' but goland with "" surrounding.
			// I'll buy you a beer if you tell me why...
			BuildFlags: "-ldflags '" + task.LDFlags + "'",
		}
	}

	{
		configurations, ok := vsCodeConfig["configurations"]
		if !ok {
			configurations = []interface{}{}
		}
		exitConfigurations := make([]interface{}, 0)
		for _, conf := range configurations.([]interface{}) {
			c := conf.(map[string]interface{})
			task, ok := vsCodeTasks[c["name"].(string)]
			if ok {
				exitConfigurations = append(exitConfigurations, task)
				delete(vsCodeTasks, c["name"].(string))
			} else {
				exitConfigurations = append(exitConfigurations, c)
			}
		}
		for _, task := range vsCodeTasks {
			exitConfigurations = append(exitConfigurations, task)
		}
		vsCodeConfig["configurations"] = exitConfigurations
	}
	b, err := json.MarshalIndent(vsCodeConfig, "", "\t")
	if err != nil {
		log.Panicln("cannot marshal", err)
	}
	if err := ioutil.WriteFile(vscodeLaunchPath, b, 0755); err != nil {
		log.Panicln("cannot write file: ", err)
	}
}
