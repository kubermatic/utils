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
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/ssh/terminal"
	ctrl "sigs.k8s.io/controller-runtime"
	corezap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var ZapLogger *zap.Logger

// BuildLogger build logr logger using log level, dev flag and writer
// WARN: we are setting global variable `ZapLogger` here
func BuildLogger(level int8, dev bool, w io.Writer) logr.Logger {
	if dev {
		ZapLogger = corezap.NewRaw(func(options *corezap.Options) {
			level := zap.NewAtomicLevelAt(zapcore.Level(-level))
			options.Level = &level
			options.Development = dev
			options.DestWritter = w
		})
	} else {
		// we need to create ZapLogger manually because we don't want to use Sampler, that does not support arbitrary log levels
		encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		sink := zapcore.AddSync(w)
		ZapLogger = zap.New(
			zapcore.NewCore(
				&corezap.KubeAwareEncoder{Encoder: encoder, Verbose: dev},
				sink,
				zap.NewAtomicLevelAt(zapcore.Level(-level))),
			zap.AddCallerSkip(1),
			zap.ErrorOutput(sink),
			zap.AddStacktrace(zap.NewAtomicLevelAt(zap.ErrorLevel)),
		)

	}
	return zapr.NewLogger(ZapLogger)
}

// CmdLogMixin adds necessary CLI flags for logging and setups the controller runtime log
func CmdLogMixin(cmd *cobra.Command) *cobra.Command {
	dev := cmd.PersistentFlags().Bool("development", terminal.IsTerminal(int(os.Stdout.Fd())), "format output for console")
	v := cmd.PersistentFlags().Int8P("verbose", "v", 0, "verbosity level")
	_ = viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))

	setupLogger := func() { ctrl.SetLogger(BuildLogger(*v, *dev, cmd.ErrOrStderr())) }

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
