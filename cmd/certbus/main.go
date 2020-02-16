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

	app.AddCommand(certSubcommandsEntry())

	app.AddCommand(configSubcommandsEntry())

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

func configSubcommandsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conf",
		Short: "Configuration subcommands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Update configuration on the event bus",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateConfig(ossignal.InterruptOrTerminateBackgroundCtx(nil), os.Stdin); err != nil {
				panic(err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "display",
		Short: "Fetch configuration from the event bus (warning: shows secrets)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := displayConfig(ossignal.InterruptOrTerminateBackgroundCtx(nil), os.Stdout); err != nil {
				panic(err)
			}
		},
	})

	return cmd
}

func certSubcommandsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Certificate management subcommands",
	}

	cmd.AddCommand(listEntry())
	cmd.AddCommand(mkEntry())
	cmd.AddCommand(inspectEntry())
	cmd.AddCommand(renewableEntry())
	cmd.AddCommand(removeEntry())

	return cmd
}

func listEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List certificates",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := list(ossignal.InterruptOrTerminateBackgroundCtx(nil)); err != nil {
				panic(err)
			}
		},
	}
}

func mkEntry() *cobra.Command {
	wildcard := false
	subdomain := false

	cmd := &cobra.Command{
		Use:   "mk [domain]",
		Short: "Issue new certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if wildcard && subdomain {
				panic("cannot apply both wildcard and subdomain certificate at the same time")
			}

			createCert := newBasicCertificate
			if subdomain {
				createCert = newSubdomainCertificate
			}
			if wildcard {
				createCert = newWildcardCertificate
			}

			if err := createCert(ossignal.InterruptOrTerminateBackgroundCtx(nil), args[0]); err != nil {
				panic(err)
			}
		},
	}

	cmd.Flags().BoolVarP(&wildcard, "wildcard", "", wildcard, "Create wildcard certificate")
	cmd.Flags().BoolVarP(&subdomain, "subdomain", "", subdomain, "Create subdomain certificate (no 'www.' prefix)")

	return cmd
}

func inspectEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "cat [id]",
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
		Use:   "renewable",
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
		Use:   "rm [id]",
		Short: "Remove a certificate (will also not get automatically renewed)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := remove(ossignal.InterruptOrTerminateBackgroundCtx(nil), args[0]); err != nil {
				panic(err)
			}
		},
	}
}
