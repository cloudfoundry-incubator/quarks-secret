package main

import (
	"os"

	utils "code.cloudfoundry.org/quarks-utils/pkg/cmd"

	cmd "code.cloudfoundry.org/quarks-secret/cmd/internal"
)

const (
	index = `---
title: "Quarks Secret"
linkTitle: "Quarks Secret"
weight: 20
description: >
    Quarks Secret CLI options
---
	`
)

func main() {
	docDir := os.Args[1]
	if err := utils.GenCLIDocsyMarkDown(cmd.NewOperatorCommand(), docDir, index); err != nil {
		panic(err)
	}
}
