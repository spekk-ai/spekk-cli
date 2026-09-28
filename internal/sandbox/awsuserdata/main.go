// Command awsuserdata writes the UserData block of the AWS CloudFormation
// template from cloud-init.yaml. `go generate ./internal/sandbox` runs it
// from that directory, so the template path is relative to it.
package main

import (
	"fmt"
	"os"

	"github.com/spekk-ai/spekk-cli/internal/sandbox"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "awsuserdata:", err)
		os.Exit(1)
	}
}

func run() error {
	template, err := os.ReadFile(sandbox.AWSTemplateFile)
	if err != nil {
		return err
	}
	updated, err := sandbox.ReplaceBlockScalarAfter(string(template), sandbox.AWSUserDataHeader, sandbox.AWSUserData())
	if err != nil {
		return err
	}
	if updated == string(template) {
		return nil
	}
	return os.WriteFile(sandbox.AWSTemplateFile, []byte(updated), 0o644)
}
