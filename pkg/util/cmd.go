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

package util

import (
	"os"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/ssh/terminal"
	ctrl "sigs.k8s.io/controller-runtime"
	corezap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var ZapLogger *zap.Logger

// CmdLogMixin adds necessary CLI flags for logging and setups the controller runtime log
func CmdLogMixin(cmd *cobra.Command) *cobra.Command {
	dev := cmd.PersistentFlags().Bool("development", terminal.IsTerminal(int(os.Stdout.Fd())), "format output for console")
	v := cmd.PersistentFlags().Int8P("verbose", "v", 0, "verbosity level")

	setupLogger := func() {
		if *dev {
			ZapLogger = corezap.NewRaw(func(options *corezap.Options) {
				level := zap.NewAtomicLevelAt(zapcore.Level(-*v))
				options.Level = &level
				options.Development = *dev
			})
		} else {
			// we need to create ZapLogger manually because we don't want to use Sampler, that does not support arbitrary log levels
			encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
			sink := zapcore.AddSync(os.Stderr)
			ZapLogger = zap.New(zapcore.NewCore(&corezap.KubeAwareEncoder{Encoder: encoder, Verbose: *dev}, sink, zap.NewAtomicLevelAt(zapcore.Level(-*v))), zap.AddCallerSkip(1), zap.ErrorOutput(sink))
		}
		ctrl.SetLogger(zapr.NewLogger(ZapLogger))
	}

	if cmd.PersistentPreRunE != nil {
		parent := cmd.PersistentPreRunE
		cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
			setupLogger()
			return parent(c, args)
		}
		return cmd
	}

	parent := cmd.PersistentPreRun
	cmd.PersistentPreRun = func(c *cobra.Command, args []string) {
		setupLogger()
		if parent != nil {
			parent(c, args)
		}
	}
	return cmd
}
