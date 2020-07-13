/*
Copyright 2019 The Kubermatic Authors.

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
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

func generateIntelijJTasks(tasks []Task, root string) error {
	tpl := template.Must(template.New("intelij-task").Funcs(sprig.TxtFuncMap()).Parse(strings.TrimSpace(`
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="{{.Name}}" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="{{ .Package }}" />
    <working_directory value="$PROJECT_DIR$/" />
{{- with .LDFlags }}
    <go_parameters value="-i -ldflags &quot;{{ . }}&quot;" />
{{- end }}
{{- with .Args }}
    <parameters value="{{ . | join " " | html }}" />
{{- end }}
{{- with .Env }}
    <envs>
	{{- range $k, $v := . }}
      <env name="{{ $k }}" value="{{ $v }}" />
	{{- end }}
    </envs>
{{- end }}
    <kind value="DIRECTORY" />
    <filePath value="$PROJECT_DIR/|$PROJECT_DIR$/{{ .Program }}" />
    <package value="{{ .Module }}" />
    <directory value="$PROJECT_DIR$/{{ .Program }}" />
    <method v="2" />
  </configuration>
</component>
`)))

	err := os.MkdirAll(path.Join(root, ".idea", "runConfigurations"), 0755)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		f, err := os.OpenFile(
			path.Join(root, ".idea", "runConfigurations", "kubecarrier_"+strings.ReplaceAll(task.Name, "/", "__")+".xml"),
			// path.Join(root, ".idea", "runConfigurations", "test.xml"),
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			0755,
		)
		if err != nil {
			return err
		}
		if err := tpl.Execute(f, task); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
