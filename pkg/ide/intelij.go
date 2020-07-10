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
