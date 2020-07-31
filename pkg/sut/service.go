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

package sut

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8c.io/utils/pkg/ide"
)

type SUTFlags struct {
	LabelSelector string
	ExtraArgs     []string
	LDFlags       string
	DeploymentNN  string
	Workdir       string
	TaskName      string
	ProjectRoot   string
	Program       string
	ConfigFlags   *genericclioptions.ConfigFlags
}

func NewSUTFlags() *SUTFlags {
	return &SUTFlags{
		ConfigFlags: genericclioptions.NewConfigFlags(false),
		TaskName:    "SUT",
		ProjectRoot: ".",
		Program:     "cmd/",
	}
}

func (f *SUTFlags) NewCommand(log logr.Logger, use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Long:  strings.TrimSpace(``),
		Short: "system under test",
		RunE: func(cmd *cobra.Command, args []string) error {
			// k8s client setup
			clientConfig := f.ConfigFlags.ToRawKubeConfigLoader()
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			k8sClient, err := client.New(cfg, client.Options{})
			if err != nil {
				return fmt.Errorf("client: %w", err)
			}
			ctx, closeCtx := context.WithCancel(context.Background())
			defer closeCtx()

			// get the deployment info from the k8s cluster
			deployment, container, err := getDeploymentAndContainer(cmd, ctx, k8sClient, f.DeploymentNN, f.LabelSelector)
			if err != nil {
				return fmt.Errorf("getting getDeploymentAndContainer: %w", err)
			}
			log.Info("selected deployment: " + deployment.Namespace + "/" + deployment.Name)

			// SUT workdir setup
			if f.Workdir == "" {
				tmpDir, err := ioutil.TempDir("", "sut-")
				if err != nil {
					log.Info("created temp directory", "dir", tmpDir)
					return fmt.Errorf("tmpdir: %w", err)
				}
				f.Workdir = tmpDir
			}
			log.Info("using workdir", "dir", f.Workdir)
			rootMount := path.Join(f.Workdir, "rootfs")
			envJson := path.Join(f.Workdir, "env.json")
			logFile := path.Join(f.Workdir, "telepresence.log")
			taskKubeconfig := path.Join(f.Workdir, "kubeconfig")

			// configure telepresence args
			telepresenceArgs := []string{
				"--swap-deployment", deployment.Name,
				"--namespace", deployment.Namespace,
				"--mount", rootMount,
				"--env-json", envJson,
				"--logfile", logFile,
			}
			for _, portSpec := range container.Ports {
				telepresenceArgs = append(telepresenceArgs, "--expose", fmt.Sprint(portSpec.ContainerPort))
			}
			telepresenceArgs = append(telepresenceArgs,
				"--run",
				"bash",
				"-c",
				"\"while true; do sleep 3600; done\"",
			)

			log.Info("=== manually run telepresence! ===")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "telepresence", strings.Join(telepresenceArgs, " "),
				"\nWait for message \"T: Setup complete. Lunching your command.\" and press any key to continue...",
			)
			pause(cmd.InOrStdin())

			if err := writeServiceKubeconfig(clientConfig, rootMount, taskKubeconfig); err != nil {
				return fmt.Errorf("write kubeconfig: %w", err)
			}

			taskArgs, taskEnv, err := generateTaskArgsAndEnv(container, rootMount, f.ExtraArgs, envJson)
			if err != nil {
				return err
			}
			taskEnv["KUBECONFIG"] = taskKubeconfig

			root, err := filepath.Abs(f.ProjectRoot)
			if err != nil {
				return err
			}
			goModPath := path.Join(root, "go.mod")
			data, err := ioutil.ReadFile(goModPath)
			if err != nil {
				return fmt.Errorf("cannot read go.mod file, is project root a go module?: %w", err)
			}
			gomod, err := modfile.Parse(goModPath, data, nil)
			if err != nil {
				return err
			}
			// generate tasks
			task := ide.Task{
				Name:    f.TaskName,
				Program: f.Program,
				Args:    taskArgs,
				Env:     taskEnv,
				LDFlags: f.LDFlags,
				Module:  gomod.Module.Mod.Path,
			}
			_, task.Package = path.Split(task.Module)
			{
				b, err := json.MarshalIndent(task, "", "\t")
				if err != nil {
					return fmt.Errorf("marshal task: %w", err)
				}
				log.Info("generating IDE tasks\n" + string(b))
			}

			return ide.GenerateTasks([]ide.Task{task}, f.ProjectRoot)
		},
	}
	cmd.Flags().StringVar(&f.TaskName, "task-name", f.TaskName, "IDE task name this tool should generate")
	cmd.Flags().StringVar(&f.ProjectRoot, "project-root", f.ProjectRoot, "project root where IDE tasks should be generated")
	cmd.Flags().StringVar(&f.LDFlags, "ld-flags", f.LDFlags, "ld-flags to pass to go compiler upon running this")
	cmd.Flags().StringVar(&f.DeploymentNN, "deployment-nn", f.DeploymentNN, "deployment-nn signal the deployement namespace name which should be selected. If none (default) a fzf based picker shall be shown")
	cmd.Flags().StringVar(&f.Workdir, "workdir", f.Workdir, "sut working for logs, rootfs mountpoints, etc. default to new temp dir")
	cmd.Flags().StringArrayVar(&f.ExtraArgs, "extra-flags", f.ExtraArgs, "extra flags to pass to the running task")
	cmd.Flags().StringVar(&f.Program, "program", f.Program, "program is the path within project root which is the main package")
	cmd.Flags().StringVarP(&f.LabelSelector, "selector", "l", f.LabelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	return cmd

}

func pause(in io.Reader) {
	b := make([]byte, 1)
	_, _ = in.Read(b)
}

func generateTaskArgsAndEnv(container *corev1.Container, rootMount string, extraArgs []string, envJson string) ([]string, map[string]string, error) {
	// translate path args from the container to IDE task
	volumeReplacementMap := make(map[string]string)
	for _, mount := range container.VolumeMounts {
		volumeReplacementMap[mount.MountPath] = path.Join(rootMount, mount.MountPath)
	}
	envBytes, err := ioutil.ReadFile(envJson)
	if err != nil {
		return nil, nil, fmt.Errorf("reading envJson: %w", err)
	}
	env := make(map[string]string)
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return nil, nil, fmt.Errorf("cannot unmarshall the environment")
	}
	hostContainerArgs := make([]string, 0, len(container.Args))
	for _, arg := range container.Args {
		arg, err := k8sExpandEnvArg(arg, env)
		if err != nil {
			return nil, nil, fmt.Errorf("expanding arg: %w", err)
		}
		for containerPath, hostPath := range volumeReplacementMap {
			arg = strings.ReplaceAll(arg, containerPath, hostPath)
		}
		hostContainerArgs = append(hostContainerArgs, arg)
	}
	hostContainerArgs = append(hostContainerArgs, extraArgs...)
	for k, v := range env {
		if !strings.HasPrefix(k, "TELEPRESENCE") {
			for containerPath, hostPath := range volumeReplacementMap {
				if strings.HasPrefix(v, containerPath) {
					v = strings.ReplaceAll(v, containerPath, hostPath)
				}
			}
			env[k] = v
		}
	}
	return hostContainerArgs, env, nil
}

func getDeploymentAndContainer(cmd *cobra.Command, ctx context.Context, cl client.Client, deploymentNN string, labelSelector string) (*appsv1.Deployment, *corev1.Container, error) {
	deployment := &appsv1.Deployment{}
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, nil, err
	}
	if deploymentNN == "" {
		depl := &appsv1.DeploymentList{}
		if err := cl.List(ctx, depl, &client.ListOptions{
			LabelSelector: client.MatchingLabelsSelector{
				Selector: selector,
			},
		}); err != nil {
			return nil, nil, fmt.Errorf("listing deployments: %w", err)
		}
		deployment, err = pickDeployment(ctx, depl, cmd.ErrOrStderr())
		if err != nil {
			return nil, nil, fmt.Errorf("picking deployment: %w", err)
		}
	} else {
		split := strings.Split(deploymentNN, "/")
		if len(split) != 2 {
			return nil, nil, fmt.Errorf("deployment namespace name should be in namespace/name format. Found more than 1 '/' char")
		}
		if err := cl.Get(ctx, types.NamespacedName{
			Namespace: split[0],
			Name:      split[1],
		}, deployment); err != nil {
			return nil, nil, fmt.Errorf("getting deployment: %w", err)
		}
	}
	if len(deployment.Spec.Template.Spec.Containers) > 1 {
		return nil, nil, fmt.Errorf("expected only 1 container, found: %d", len(deployment.Spec.Template.Spec.Containers))
	}
	return deployment, &deployment.Spec.Template.Spec.Containers[0], err
}

func writeServiceKubeconfig(clientConfig clientcmd.ClientConfig, rootMount string, kubeconfigPath string) error {
	rawCfg, err := clientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("rawCfg: %w", err)
	}
	kubeconfigContext := rawCfg.Contexts[rawCfg.CurrentContext]
	rawCfg.AuthInfos[kubeconfigContext.AuthInfo] = &api.AuthInfo{
		TokenFile: path.Join(rootMount, "var", "run", "secrets", "kubernetes.io", "serviceaccount", "token"),
	}
	if err := clientcmd.WriteToFile(rawCfg, kubeconfigPath); err != nil {
		return fmt.Errorf("marshall raw cfg: %w", err)
	}
	return nil
}

func pickDeployment(ctx context.Context, depl *appsv1.DeploymentList, stdErr io.Writer) (*appsv1.Deployment, error) {
	choices := bytes.NewBufferString("")
	choiceToDep := make(map[string]*appsv1.Deployment)
	for _, dep := range depl.Items {
		choice := fmt.Sprintf("%-40s %s", dep.Name, dep.Namespace)
		choiceToDep[choice] = dep.DeepCopy()
		_, _ = fmt.Fprintln(choices, choice)
	}

	fzfCmd := exec.CommandContext(ctx, "fzf", "--header="+fmt.Sprintf("%-40s %s", "NAME", "NAMESPACE"))
	fzfCmd.Stdin = choices
	fzfCmd.Stderr = stdErr
	picker, err := fzfCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot exec fzf: %w", err)
	}
	deployment, ok := choiceToDep[strings.TrimSpace(string(picker))]
	if !ok {
		return nil, fmt.Errorf("invalid choice")
	}
	return deployment, nil
}

func k8sExpandEnvArg(arg string, env map[string]string) (string, error) {
	for {
		idx := strings.Index(arg, "$(")
		if idx == -1 {
			return arg, nil
		}

		// handle escaping
		for idx > 0 && arg[idx-1:idx] == "$" {
			newIdx := strings.Index(arg[idx+2:], "$(")
			if newIdx == -1 {
				return arg, nil
			}
			idx += newIdx + 2
		}

		// find closing paren
		delta := strings.Index(arg[idx:], ")")
		if delta == -1 {
			return "", fmt.Errorf("unmatched (")
		}
		envElem := arg[idx+2 : idx+delta]
		envVal, ok := env[envElem]
		if !ok {
			return "", fmt.Errorf("cannot find %s in supplied envVars", envVal)
		}
		arg = arg[:idx] + envVal + arg[idx+delta+1:]
	}
}
