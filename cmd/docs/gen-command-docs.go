package main

import (
	"log"

	cmd "code.cloudfoundry.org/quarks-secret/cmd/internal"
	"github.com/spf13/cobra/doc"
)

func main() {
	operatorCommand := cmd.NewOperatorCommand()

	err := doc.GenMarkdownTree(operatorCommand, "./docs/commands/")
	if err != nil {
		log.Fatal(err)
	}
}
