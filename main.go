package main

import (
	"code.cloudfoundry.org/cfdev/env"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"code.cloudfoundry.org/cfdev/cfanalytics"
	"code.cloudfoundry.org/cfdev/cfanalytics/toggle"
	"code.cloudfoundry.org/cfdev/cmd"
	"code.cloudfoundry.org/cfdev/config"
	"code.cloudfoundry.org/cfdev/errors"
	cfdevos "code.cloudfoundry.org/cfdev/os"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"code.cloudfoundry.org/cli/plugin"
	"github.com/spf13/cobra"
	"gopkg.in/segmentio/analytics-go.v3"
)

type Command interface {
	Run(args []string) error
}

type Plugin struct {
	UI        terminal.UI
	Analytics *cfanalytics.Analytics
	Root      *cobra.Command
	Version   plugin.VersionType
}

const (
	boshIP   = "10.144.0.2"
	routerIP = "10.144.0.34"
	domain   = "dev.cfdev.sh"
)

func main() {
	exitChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		close(exitChan)
	}()

	configureEnvironmentVariables()

	ui := terminal.NewUI(
		os.Stdin,
		os.Stdout,
		terminal.NewTeePrinter(os.Stdout),
		trace.NewLogger(os.Stdout, false, "", ""),
	)

	conf, err := config.NewConfig()
	if err != nil {
		ui.Failed(err.Error())
		os.Exit(1)
	}

	analyticsToggle := toggle.New(filepath.Join(conf.CFDevHome, "analytics", "analytics.txt"))
	baseAnalyticsClient, _ := analytics.NewWithConfig(conf.AnalyticsKey, analytics.Config{
		Logger: analytics.StdLogger(log.New(ioutil.Discard, "", 0)),
	})

	o := cfdevos.OS{}
	osVersion, err := o.Version()
	if err != nil {
		osVersion = "unknown-os-version"
	}

	analyticsClient := cfanalytics.New(analyticsToggle, baseAnalyticsClient, conf.CliVersion.Original, osVersion, env.IsBehindProxy(), exitChan, ui)
	defer analyticsClient.Close()

	v := conf.CliVersion

	cfdev := &Plugin{
		UI:        ui,
		Analytics: analyticsClient,
		Root:      cmd.NewRoot(exitChan, ui, conf, analyticsClient, analyticsToggle),
		Version:   plugin.VersionType{Major: v.Major, Minor: v.Minor, Build: v.Build},
	}

	plugin.Start(cfdev)
}

func configureEnvironmentVariables() {
	fetchNoProxyVariables := func() []string {
		noProxyVars := os.Getenv("NO_PROXY")
		if noProxyVars != "" {
			noProxyVars = os.Getenv("no_proxy")
		}

		collection := strings.Split(noProxyVars, ",")
		return append(collection, boshIP, routerIP, "."+domain)
	}

	os.Unsetenv("BOSH_ALL_PROXY")
	os.Setenv("NO_PROXY", strings.Join(fetchNoProxyVariables(), ","))
}

func (p *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "cfdev",
		Version: p.Version,
		Commands: []plugin.Command{
			{
				Name:     "dev",
				HelpText: "Start and stop a single vm CF deployment running on your workstation",
				UsageDetails: plugin.Usage{
					Usage: p.Root.UsageString(),
				},
			},
		},
	}
}

func (p *Plugin) Run(connection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		p.Analytics.Event(cfanalytics.UNINSTALL, nil)
		return
	}

	if len(args) >= 2 && (strings.ToLower(args[1]) != "telemetry" && strings.ToLower(args[1]) != "start") {
		if err := p.Analytics.PromptOptInIfNeeded(""); err != nil {
			p.UI.Failed(err.Error())
			p.Analytics.Close()
			os.Exit(1)
		}
	}

	p.Root.SetArgs(args)
	if err := p.Root.Execute(); err != nil {
		p.UI.Failed(err.Error())
		extraData := map[string]interface{}{"errors": errors.SafeError(err)}
		p.Analytics.Event(cfanalytics.ERROR, extraData)
		p.Analytics.Close()
		os.Exit(1)
	}
}
