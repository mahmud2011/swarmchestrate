package main

import (
	"log"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/cmd"
)

func main() {
	rootCmd := cmd.NewCMDRoot()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
