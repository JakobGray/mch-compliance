package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"gopkg.in/yaml.v2"
)

func writeJSON(w io.Writer, res Checklist) error {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", string(b))
	return nil
}

func writeYAML(w io.Writer, res Checklist) error {
	b, err := yaml.Marshal(res)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", string(b))
	return nil
}

func writeText(w io.Writer, res Checklist) error {
	for _, d := range res.Deployments {
		for _, c := range d.Checks {
			fmt.Fprintf(w, "[%s][%s][%s] %s\n", d.Name, c.Category, c.Rule, c.Message)
		}
	}
	return nil
}

func writeTable(w io.Writer, res Checklist) error {
	tbw := new(tabwriter.Writer)
	// Format in tab-separated columns with a tab stop of 8.
	tbw.Init(w, 0, 8, 0, '\t', 0)

	fmt.Fprintf(tbw, "%s\t%s\t%s\t%s\n", "DEPLOYMENT", "CATEGORY", "RULE", "MESSAGE")
	for _, d := range res.Deployments {
		for _, c := range d.Checks {
			fmt.Fprintf(tbw, "%s\t%s\t%s\t%s\n", d.Name, c.Category, c.Rule, c.Message)
		}
	}
	tbw.Flush()
	return nil
}
