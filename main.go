package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

type CLIOptions struct {
	ConfigFiles struct {
		List []string `positional-arg-name:"config-files"`
	} `positional-args:"yes" required:"yes"`
}

func main() {

	options := &CLIOptions{}
	_, err := flags.Parse(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		os.Exit(2)
	}

	err = realMain(options)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		os.Exit(3)
	}

}

func realMain(opts *CLIOptions) error {
	configFiles := opts.ConfigFiles.List

	config, err := LoadConfigFiles(configFiles)
	if err != nil {
		return err
	}

	targetGroups := map[string]*TargetGroup{}

	for _, tgConfig := range config.TargetGroups {
		if tgConfig.TargetGroupARN == "" {
			return fmt.Errorf("cannot have target group with empty ARN")
		}
		if tgConfig.ServiceName == "" {
			return fmt.Errorf("no 'service' specified for target group %q", tgConfig.TargetGroupARN)
		}

		if targetGroups[tgConfig.TargetGroupARN] != nil {
			return fmt.Errorf("duplicate declaration of target group %q", tgConfig.TargetGroupARN)
		}

		tg, err := NewTargetGroup(
			tgConfig,
			config.Consul,
			config.AWS,
		)

		if err != nil {
			log.Printf("skipping target group %q: %s\n", tgConfig.TargetGroupARN, err)
			continue
		}

		targetGroups[tgConfig.TargetGroupARN] = tg
	}

	if len(targetGroups) == 0 {
		log.Println(
			"No valid target_group blocks found in config!",
		)
	}

	// If we get down here then our config seems valid enough and so
	// we'll start up a goroutine for each target group and then hang
	// out waiting for a signal.
	for _, tg := range targetGroups {
		go tg.KeepSyncing()
	}

	c := make(chan struct{})
	c <- struct{}{}
	return nil
}
