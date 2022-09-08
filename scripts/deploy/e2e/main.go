/*
This is a tool for triggering TF Cloud runs/deploys from master commits.
Usage:
	go run ./scripts/deploy/e2e \
		--tf-token XXX \
		--workspace dev \
		--xmtp-e2e-image xmtp/xmtpd-e2e@sha256:XXX \
		--git-commit=$(git rev-parse HEAD)
*/
package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/jessevdk/go-flags"
	"github.com/xmtp/xmtp-node-go/internal/terraform"
	"go.uber.org/zap"
)

const (
	e2eRunnerImagePrefix = "xmtp/xmtpd-e2e@sha256:"
)

var options struct {
	TFToken        string `long:"tf-token" description:"Terraform token"`
	Workspace      string `long:"workspace" description:"TF cloud workspace" choice:"dev" choice:"production"`
	Organization   string `long:"organization" default:"xmtp" choice:"xmtp"`
	ContainerImage string `long:"container-image"`
	GitCommit      string `long:"git-commit"`
}

func main() {
	ctx := context.Background()

	log, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	_, err = flags.NewParser(&options, flags.Default).Parse()
	if err != nil {
		log.Fatal("parsing options", zap.Error(err))
	}

	tfc, err := tfe.NewClient(&tfe.Config{
		Token: options.TFToken,
	})
	if err != nil {
		log.Fatal("creating terraform client", zap.Error(err))
	}

	deployer, err := terraform.NewDeployer(ctx, log, tfc, &terraform.Config{
		Organization: options.Organization,
		Workspace:    options.Workspace,
	})
	if err != nil {
		log.Fatal("creating deployer", zap.Error(err))
	}

	if options.ContainerImage == "" {
		log.Fatal("Must specify container-image")
	}

	if !strings.HasPrefix(options.ContainerImage, e2eRunnerImagePrefix) {
		log.Fatal("Invalid e2e image %s", zap.String("image", options.ContainerImage))
	}

	msg := fmt.Sprintf("triggered from commit %s", options.GitCommit)
	out, err := exec.Command("git", "log", "--oneline", "-n 1").Output()
	if err != nil {
		log.Error("getting git commit message", zap.Error(err))
	} else {
		msg = string(out)
	}

	err = deployer.Deploy("xmtpd_e2e_image", options.ContainerImage, msg)
	if err != nil {
		log.Fatal("deploying", zap.Error(err))
	}
}