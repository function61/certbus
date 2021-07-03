package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/certbus/pkg/cbexampleserver"
	"github.com/function61/eventhorizon/pkg/ehcli"
	"github.com/function61/gokit/aws/lambdautils"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/osutil"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/spf13/cobra"
)

func main() {
	if lambdautils.InLambda() {
		// assume scheduled events => renew first renewable
		lambda.StartHandler(lambdautils.NoPayloadAdapter(func(ctx context.Context) error {
			return listRenewable(
				ctx,
				time.Now(),
				true)
		}))
		return
	}

	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Cert Bus keeps your TLS certificates fresh",
		Version: dynversion.Version,
	}

	app.AddCommand(certSubcommandsEntry())

	app.AddCommand(configSubcommandsEntry())

	// Event Horizon administration
	app.AddCommand(ehcli.Entrypoint())

	app.AddCommand(cbexampleserver.Entrypoint())

	osutil.ExitIfError(app.Execute())
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
			osutil.ExitIfError(updateConfig(
				osutil.CancelOnInterruptOrTerminate(nil),
				os.Stdin))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "display",
		Short: "Fetch configuration from the event bus (warning: shows secrets)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(displayConfig(
				osutil.CancelOnInterruptOrTerminate(nil),
				os.Stdout))
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
	cmd.AddCommand(renewEntry())
	cmd.AddCommand(removeEntry())

	return cmd
}

func listEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List certificates",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(list(osutil.CancelOnInterruptOrTerminate(nil)))
		},
	}
}

func mkEntry() *cobra.Command {
	wildcard := false
	subdomain := false
	dns := true

	cmd := &cobra.Command{
		Use:   "mk [domain]",
		Short: "Issue new certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if wildcard && subdomain {
				panic("cannot apply both wildcard and subdomain certificate at the same time")
			}

			challengeType := func() challenge.Type {
				if dns {
					return challenge.DNS01
				} else {
					return challenge.HTTP01
				}
			}()

			createCert := newBasicCertificate
			if subdomain {
				createCert = newSubdomainCertificate
			}
			if wildcard {
				createCert = newWildcardCertificate
			}

			osutil.ExitIfError(createCert(
				osutil.CancelOnInterruptOrTerminate(nil),
				args[0],
				challengeType))
		},
	}

	cmd.Flags().BoolVarP(&wildcard, "wildcard", "", wildcard, "Create wildcard certificate, please take care you don't have wildcard CNAME (mutually exclusive with --subdomain)")
	cmd.Flags().BoolVarP(&subdomain, "subdomain", "", subdomain, "Create subdomain certificate (no 'www.' prefix)")
	cmd.Flags().BoolVarP(&dns, "dns", "", dns, "Use DNS-01 challenge")

	return cmd
}

func inspectEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "cat [id]",
		Short: "Inspect a certificate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(inspect(osutil.CancelOnInterruptOrTerminate(nil), args[0]))
		},
	}
}

func renewableEntry() *cobra.Command {
	renewFirst := false

	cmd := &cobra.Command{
		Use:   "renewable [at]",
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

			osutil.ExitIfError(listRenewable(
				osutil.CancelOnInterruptOrTerminate(nil),
				after,
				renewFirst))
		},
	}

	cmd.Flags().BoolVarP(&renewFirst, "renew-first", "r", renewFirst, "Renew first renewable cert")

	return cmd
}

func renewEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "renew [id]",
		Short: "Renew a cert",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(renew(
				osutil.CancelOnInterruptOrTerminate(nil),
				args[0]))
		},
	}
}

func removeEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "rm [id]",
		Short: "Remove a certificate (will also not get automatically renewed)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(remove(
				osutil.CancelOnInterruptOrTerminate(nil),
				args[0]))
		},
	}
}
