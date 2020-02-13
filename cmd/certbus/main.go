package main

import (
	"fmt"
	"github.com/function61/certbus/pkg/cbexampleserver"
	"github.com/function61/eventhorizon/pkg/ehcli"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/ossignal"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Cert Bus keeps your TLS certificates fresh",
		Version: dynversion.Version,
	}

	app.AddCommand(listEntry())
	app.AddCommand(newEntry())
	app.AddCommand(inspectEntry())
	app.AddCommand(removeEntry())
	app.AddCommand(renewableEntry())
	app.AddCommand(configUpdateEntry())
	app.AddCommand(configDisplayEntry())

	// Event Horizon administration
	for _, cmd := range ehcli.Entrypoints() {
		app.AddCommand(cmd)
	}

	app.AddCommand(cbexampleserver.Entrypoint())

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func configUpdateEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "conf-update",
		Short: "Update configuration on the event bus",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateConfig(ossignal.InterruptOrTerminateBackgroundCtx(nil), os.Stdin); err != nil {
				panic(err)
			}
		},
	}
}

func configDisplayEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "conf-display",
		Short: "Fetch configuration from the event bus",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := displayConfig(ossignal.InterruptOrTerminateBackgroundCtx(nil), os.Stdout); err != nil {
				panic(err)
			}
		},
	}
}

func listEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "cert-list",
		Short: "List certificates",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := list(ossignal.InterruptOrTerminateBackgroundCtx(nil)); err != nil {
				panic(err)
			}
		},
	}
}

func newEntry() *cobra.Command {
	wildcard := false

	cmd := &cobra.Command{
		Use:   "cert-new [domain]",
		Short: "Issue new certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			createCert := newBasicCertificate
			if wildcard {
				createCert = newWildcardCertificate
			}

			if err := createCert(ossignal.InterruptOrTerminateBackgroundCtx(nil), args[0]); err != nil {
				panic(err)
			}
		},
	}

	cmd.Flags().BoolVarP(&wildcard, "wildcard", "", wildcard, "Create wildcard certificate")

	return cmd
}

func inspectEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "cert-inspect [id]",
		Short: "Inspect a certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := inspect(ossignal.InterruptOrTerminateBackgroundCtx(nil), args[0]); err != nil {
				panic(err)
			}
		},
	}
}

func renewableEntry() *cobra.Command {
	renewFirst := false

	cmd := &cobra.Command{
		Use:   "cert-renewable",
		Short: "List renewable certs",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			after := time.Now()
			if len(args) >= 1 {
				var err error
				after, err = time.Parse("2006-01-02", args[0])
				if err != nil {
					panic(err)
				}
			}

			if err := listRenewable(ossignal.InterruptOrTerminateBackgroundCtx(nil), after, renewFirst); err != nil {
				panic(err)
			}
		},
	}

	cmd.Flags().BoolVarP(&renewFirst, "renew-first", "r", renewFirst, "Renew first renewable cert")

	return cmd
}

func removeEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "cert-remove [id]",
		Short: "Remove a certificate (will also not get automatically renewed)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := remove(ossignal.InterruptOrTerminateBackgroundCtx(nil), args[0]); err != nil {
				panic(err)
			}
		},
	}
}
