package main

import (
	"os"

	utils "code.cloudfoundry.org/quarks-utils/pkg/cmd"

	cmd "code.cloudfoundry.org/quarks-secret/cmd/internal"
)

func main() {
	docDir := os.Args[1]
	if err := utils.GenCLIDocsyMarkDown(cmd.NewOperatorCommand(), docDir); err != nil {
		panic(err)
	}
}
