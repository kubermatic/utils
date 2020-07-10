/*
Copyright 2019 The KubeCarrier Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ide

import (
	_ "golang.org/x/mod/modfile"
)

// Task defines single task
type Task struct {
	Name    string
	Program string
	Args    []string
	Env     map[string]string
	LDFlags string
	// go package, e.g. utils
	Package string
	// go module, e.g. github.com/kubermatic/utils
	Module string
}

func GenerateTasks(tasks []Task, root string) error {
	if err := generateIntelijJTasks(tasks, root); err != nil {
		return err
	}
	generateVSCode(tasks, root)
	return nil
}
