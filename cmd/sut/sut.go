package main

import (
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubermatic/utils/pkg/sut"
	"github.com/kubermatic/utils/pkg/util"
)

func main() {
	cmd := sut.NewSUTFlags().NewCommand(ctrl.Log, "sut")
	cmd = util.CmdLogMixin(cmd)
	if err := cmd.Execute(); err != nil {
		ctrl.Log.Error(err, "error during execution")
		os.Exit(2)
	}
}
