/*
Copyright 2016 The Kubernetes Authors.

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

package templates

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"k8s.io/kubectl/pkg/util/term"
)

type FlagExposer interface {
	ExposeFlags(cmd *cobra.Command, flags ...string) FlagExposer
}

func ActsAsRootCommand(cmd *cobra.Command, filters []string, groups ...CommandGroup) FlagExposer {
	if cmd == nil {
		panic("nil root command")
	}
	templater := &templater{
		RootCmd:       cmd,
		CommandGroups: groups,
		Filtered:      filters,
	}
	cmd.SetFlagErrorFunc(templater.FlagErrorFunc())
	cmd.SilenceUsage = true
	cmd.SetUsageFunc(templater.UsageFunc())
	cmd.SetHelpFunc(templater.HelpFunc())
	return templater
}

// UseOptionsTemplates makes a command print only the flags every command accepts.
func UseOptionsTemplates(cmd *cobra.Command) {
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		if !c.HasInheritedFlags() {
			return nil
		}
		out := term.NewResponsiveWriter(c.OutOrStderr())
		_, err := io.WriteString(out, "The following options can be passed to any command:\n\n"+flagsUsages(c.InheritedFlags()))
		return err
	})
	cmd.SetHelpFunc(func(*cobra.Command, []string) {})
}

type templater struct {
	RootCmd *cobra.Command
	CommandGroups
	Filtered []string
}

func (templater *templater) FlagErrorFunc(exposedFlags ...string) func(*cobra.Command, error) error {
	return func(c *cobra.Command, err error) error {
		c.SilenceUsage = true
		switch c.CalledAs() {
		case "options":
			return fmt.Errorf("%s\nRun '%s' without flags.", err, c.CommandPath())
		default:
			return fmt.Errorf("%s\nSee '%s --help' for usage.", err, c.CommandPath())
		}
	}
}

func (templater *templater) ExposeFlags(cmd *cobra.Command, flags ...string) FlagExposer {
	cmd.SetUsageFunc(templater.UsageFunc(flags...))
	return templater
}

func (templater *templater) HelpFunc() func(*cobra.Command, []string) {
	return func(c *cobra.Command, s []string) {
		out := term.NewResponsiveWriter(c.OutOrStdout())
		if _, err := io.WriteString(out, templater.help(c)); err != nil {
			c.Println(err)
		}
	}
}

func (templater *templater) UsageFunc(exposedFlags ...string) func(*cobra.Command) error {
	return func(c *cobra.Command) error {
		out := term.NewResponsiveWriter(c.OutOrStderr())
		_, err := io.WriteString(out, templater.usage(c, exposedFlags))
		return err
	}
}

// help is the long description, or the short one, followed by the usage.
func (templater *templater) help(c *cobra.Command) string {
	var b strings.Builder
	if text := c.Long; text != "" || c.Short != "" {
		if text == "" {
			text = c.Short
		}
		b.WriteString(strings.TrimSpace(text))
	}
	if c.Runnable() || c.HasSubCommands() {
		b.WriteString(c.UsageString())
	}
	return b.String()
}

// usage renders aliases, examples, subcommands, options, the usage line and the help hints.
func (templater *templater) usage(c *cobra.Command, exposedFlags []string) string {
	visible := visibleFlags(flagsNotIntersected(c.LocalFlags(), c.PersistentFlags()))
	exposed := flag.NewFlagSet("exposed", flag.ContinueOnError)
	for _, name := range exposedFlags {
		if f := c.Flags().Lookup(name); f != nil {
			exposed.AddFlag(f)
		}
	}

	var b strings.Builder
	b.WriteString("\n\n")
	if len(c.Aliases) > 0 {
		b.WriteString("Aliases:\n" + c.NameAndAliases() + "\n\n")
	}
	if c.HasExample() {
		b.WriteString("Examples:\n" + trimRight(c.Example) + "\n\n")
	}
	if c.HasAvailableSubCommands() {
		b.WriteString(templater.cmdGroupsString(c) + "\n\n")
	}
	if visible.HasFlags() || exposed.HasFlags() {
		b.WriteString("Options:\n")
		if visible.HasFlags() {
			b.WriteString(trimRight(flagsUsages(visible)))
		}
		if exposed.HasFlags() {
			if visible.HasFlags() {
				b.WriteString("\n")
			}
			b.WriteString(trimRight(flagsUsages(exposed)))
		}
		b.WriteString("\n\n")
	}
	if useLine := c.UseLine(); c.Runnable() && useLine != "" && useLine != templater.rootCmdName(c) {
		b.WriteString("Usage:\n  " + templater.usageLine(c) + "\n\n")
	}
	if c.HasSubCommands() {
		b.WriteString("Use \"")
		for _, name := range templater.reverseParentsNames(c) {
			b.WriteString(name + " ")
		}
		b.WriteString("<command> --help\" for more information about a given command.\n")
	}
	if optionsCmd := templater.optionsCmdFor(c); optionsCmd != "" {
		b.WriteString("Use \"" + optionsCmd + "\" for a list of global command-line options (applies to all commands).\n")
	}
	return b.String()
}

func trimRight(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }

func (templater *templater) cmdGroups(c *cobra.Command, all []*cobra.Command) []CommandGroup {
	if len(templater.CommandGroups) > 0 && c == templater.RootCmd {
		all = filter(all, templater.Filtered...)
		return AddAdditionalCommands(templater.CommandGroups, "Other Commands:", all)
	}
	all = filter(all, "options")
	return []CommandGroup{
		{
			Message:  "Available Commands:",
			Commands: all,
		},
	}
}

func (t *templater) cmdGroupsString(c *cobra.Command) string {
	groups := []string{}
	for _, cmdGroup := range t.cmdGroups(c, c.Commands()) {
		cmds := []string{cmdGroup.Message}
		for _, cmd := range cmdGroup.Commands {
			if cmd.IsAvailableCommand() {
				cmds = append(cmds, fmt.Sprintf("  %-*s   %s", cmd.NamePadding(), cmd.Name(), cmd.Short))
			}
		}
		groups = append(groups, strings.Join(cmds, "\n"))
	}
	return strings.Join(groups, "\n\n")
}

func (t *templater) rootCmdName(c *cobra.Command) string {
	return t.rootCmd(c).CommandPath()
}

func (t *templater) reverseParentsNames(c *cobra.Command) []string {
	reverseParentsNames := []string{}
	parents := t.parents(c)
	for i := len(parents) - 1; i >= 0; i-- {
		reverseParentsNames = append(reverseParentsNames, parents[i].Name())
	}
	return reverseParentsNames
}

func (t *templater) isRootCmd(c *cobra.Command) bool {
	return t.rootCmd(c) == c
}

func (t *templater) parents(c *cobra.Command) []*cobra.Command {
	parents := []*cobra.Command{c}
	for current := c; !t.isRootCmd(current) && current.HasParent(); {
		current = current.Parent()
		parents = append(parents, current)
	}
	return parents
}

func (t *templater) rootCmd(c *cobra.Command) *cobra.Command {
	if c != nil && !c.HasParent() {
		return c
	}
	if t.RootCmd == nil {
		panic("nil root cmd")
	}
	return t.RootCmd
}

func (t *templater) optionsCmdFor(c *cobra.Command) string {
	if !c.Runnable() {
		return ""
	}
	rootCmdStructure := t.parents(c)
	for i := len(rootCmdStructure) - 1; i >= 0; i-- {
		cmd := rootCmdStructure[i]
		if _, _, err := cmd.Find([]string{"options"}); err == nil {
			return cmd.CommandPath() + " options"
		}
	}
	return ""
}

func (t *templater) usageLine(c *cobra.Command) string {
	usage := c.UseLine()
	suffix := "[options]"
	if c.HasFlags() && !strings.Contains(usage, suffix) {
		usage += " " + suffix
	}
	return usage
}

// flagsUsages will print out the kubectl help flags
func flagsUsages(f *flag.FlagSet) string {
	flagBuf := new(bytes.Buffer)
	wrapLimit, err := term.GetWordWrapperLimit()
	if err != nil {
		wrapLimit = 0
	}
	printer := NewHelpFlagPrinter(flagBuf, wrapLimit)

	f.VisitAll(func(flag *flag.Flag) {
		if flag.Hidden {
			return
		}
		printer.PrintHelpFlag(flag)
	})

	return flagBuf.String()
}

// getFlagFormat will output the flag format
func getFlagFormat(f *flag.Flag) string {
	var format string
	format = "--%s=%s:\n%s%s"
	if f.Value.Type() == "string" {
		format = "--%s='%s':\n%s%s"
	}

	if len(f.Shorthand) > 0 {
		format = "    -%s, " + format
	} else {
		format = "    %s" + format
	}

	return format
}

func flagsNotIntersected(l *flag.FlagSet, r *flag.FlagSet) *flag.FlagSet {
	f := flag.NewFlagSet("notIntersected", flag.ContinueOnError)
	l.VisitAll(func(flag *flag.Flag) {
		if r.Lookup(flag.Name) == nil {
			f.AddFlag(flag)
		}
	})
	return f
}

func visibleFlags(l *flag.FlagSet) *flag.FlagSet {
	hidden := "help"
	f := flag.NewFlagSet("visible", flag.ContinueOnError)
	l.VisitAll(func(flag *flag.Flag) {
		if flag.Name != hidden {
			f.AddFlag(flag)
		}
	})
	return f
}

func filter(cmds []*cobra.Command, names ...string) []*cobra.Command {
	out := []*cobra.Command{}
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		skip := false
		for _, name := range names {
			if name == c.Name() {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, c)
	}
	return out
}
