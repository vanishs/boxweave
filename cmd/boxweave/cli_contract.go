package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// The command inventory describes the active parser, not authorization.
// Fixed private workers are dispatched separately.
type cliCommand struct{ group, flags string }

const (
	ledgerFlags          = "db project incus-socket"
	installIdentityFlags = "project service-user incus-socket"
)

var cliCommands = map[string]cliCommand{
	"serve":                                {"service", ledgerFlags + " listen scoped-peer-fresh"},
	"status":                               {"operator", ledgerFlags},
	"doctor":                               {"operator", ledgerFlags + " service-user maintenance summary"},
	"recover":                              {"operator", ledgerFlags + " service-user approve-local action stream recovery-key operation reviewer challenge confirm-uuid reason acknowledge-uncertain-native"},
	"token-create":                         {"operator", ledgerFlags + " ttl"},
	"token-revoke":                         {"operator", ledgerFlags + " id"},
	"oauth-setup":                          {"operator", ledgerFlags + " generate output-file password-file"},
	"oauth-set-origin":                     {"operator", ledgerFlags + " public-origin"},
	"oauth-disable":                        {"operator", ledgerFlags},
	"oauth-status":                         {"operator", ledgerFlags},
	"oauth-list-grants":                    {"operator", ledgerFlags},
	"oauth-revoke-grant":                   {"operator", ledgerFlags + " id"},
	"build-info":                           {"operator", ""},
	"install-stage":                        {"wrapper", installIdentityFlags + " source-binary expect-sha256 scoped-peer-fresh scoped-fresh-review-sha256 approve-stage"},
	"install-activate":                     {"wrapper", installIdentityFlags + " approve-local"},
	"install-status":                       {"wrapper", installIdentityFlags},
	"native-plan":                          {"dependency", "incus-package-version"},
	"native-bootstrap":                     {"dependency", "incus-package-version native-plan-sha256 approve-native-bootstrap"},
	"native-prereq-plan":                   {"dependency", "project"},
	"native-prereq-verify":                 {"dependency", "project native-prereq-sha256 approve-local"},
	"native-prepare-status":                {"dependency", ""},
	"native-prepare-retirement-check":      {"dependency", "project native-prepare-sha256"},
	"native-prepare-rollback":              {"dependency", "native-prepare-sha256 approve-local"},
	"native-prepare-archive":               {"dependency", "native-prepare-sha256 approve-local"},
	"native-migrate-plan":                  {"dependency", "old-incus-package incus-package-version"},
	"native-migrate-apply":                 {"dependency", "old-incus-package incus-package-version native-plan-sha256 confirm-native-host-sha256 approve-native-migrate acknowledge-native-downtime"},
	"native-migrate-status":                {"dependency", ""},
	"native-migrate-reconcile":             {"dependency", "confirm-native-host-sha256 incus-package-version approve-native-migrate"},
	"native-migrate-rollback":              {"dependency", "confirm-native-host-sha256 old-incus-package approve-native-migrate acknowledge-native-downtime"},
	"native-migrate-commit":                {"dependency", "confirm-native-host-sha256 incus-package-version approve-native-migrate"},
	"native-scoped-fresh-plan":             {"wrapper", ""},
	"native-scoped-retirement-plan":        {"wrapper", "project retirement-cold"},
	"native-scoped-retirement-resume-plan": {"wrapper", "project retirement-cold"},
	"native-scoped-retirement-restore":     {"wrapper", "project native-prepare-sha256 approve-local retirement-cold"},
	"native-scoped-retirement-guard":       {"wrapper", "project native-prepare-sha256 approve-local retirement-cold"},
	"native-scoped-guard-boot":             {"service", ""},
	"native-scoped-lifecycle-serve":        {"service", ""},
	"native-scoped-offline-review":         {"wrapper", "project"},
	"native-scoped-offline-progress":       {"wrapper", "project native-prepare-sha256"},
	"native-scoped-offline-final-plan":     {"wrapper", "project native-prepare-sha256"},
	"native-scoped-offline-step":           {"wrapper", "project native-prepare-sha256 approve-local"},
	"native-scoped-offline-restore":        {"wrapper", "project native-prepare-sha256 approve-local"},
	"native-scoped-offline-guard":          {"wrapper", "project native-prepare-sha256 approve-local"},
}

func retiredCLICommand(command string) bool {
	switch command {
	case "upgrade", "rollback", "install-commit", "native-prepare-plan", "native-prepare-apply":
		return true
	default:
		return false
	}
}

// Reuse the original typed values and defaults; register only flags that
// apply to this command. This does not confer any execution authority.
func commandFlagSet(all *flag.FlagSet, spec cliCommand) *flag.FlagSet {
	f := flag.NewFlagSet(all.Name(), flag.ContinueOnError)
	f.SetOutput(all.Output())
	for _, name := range strings.Fields(spec.flags) {
		v := all.Lookup(name)
		if v == nil {
			panic("unregistered CLI flag: " + name)
		}
		f.Var(v.Value, v.Name, v.Usage)
	}
	if all.Name() == "recover" {
		f.Usage = func() {
			fmt.Fprintln(f.Output(), "Reviewed recovery: recover --plan|--apply|--guided --help")
			fmt.Fprintln(f.Output(), "Specialized local Exec log review flags:")
			f.PrintDefaults()
		}
	}
	return f
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: boxweave <command> [flags]")
	for _, group := range []string{"operator", "service", "wrapper", "dependency"} {
		var names []string
		for name, spec := range cliCommands {
			if spec.group == group {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "  %s: %s\n", group, strings.Join(names, " "))
	}
	fmt.Fprintln(os.Stderr, "Use <command> --help for applicable flags. Install and retire via the reviewed wrappers.")
}
