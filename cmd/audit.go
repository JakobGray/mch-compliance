/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var namespace string
var output string
var resultFile string
var configFile string

// auditCmd represents the audit command
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Checks installation for violations",
	Long: `Scans the resources installed by the multiclusterhub and looks for compliance
violations from a list of pre-defined rules.`,
	Example: `# Define a list of checks to exclude from report
mch-compliance audit -c exemptions.yaml

# Generate report in a parseable json format
mch-compliance audit -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		runAudit()
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().StringVarP(&namespace, "namespace", "n", "open-cluster-management", "Namespace where the hub is installed")
	auditCmd.Flags().StringVarP(&output, "output", "o", "table", "Results format to output")
	auditCmd.Flags().StringVarP(&resultFile, "file", "f", "", "Filepath to save results")
	auditCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file with exemptions")
}

func runAudit() {
	c := &Config{}
	if configFile != "" {
		var err error
		c, err = readConfig(configFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	results, err := c.checkCompliance()
	if err != nil {
		log.Fatal(err)
	}

	var w io.WriteCloser = os.Stdout
	if resultFile != "" {
		fmt.Println("Write to file not implemented")
		w, err = os.Create(resultFile)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Close()
	}

	switch output {
	case "text":
		err = writeText(w, results)
	case "yaml":
		err = writeYAML(w, results)
	case "json":
		err = writeJSON(w, results)
	default:
		err = writeTable(w, results)
	}
	if err != nil {
		log.Fatal(err)
	}
}
