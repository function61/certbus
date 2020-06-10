package cbexampleserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/osutil"
	"github.com/function61/gokit/taskrunner"
	"github.com/spf13/cobra"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "example-server",
		Short: "Start demo HTTPS server that demos CertBus integration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			osutil.ExitIfError(exampleServer(
				osutil.CancelOnInterruptOrTerminate(rootLogger),
				rootLogger))
		},
	}
}

func exampleServer(ctx context.Context, logger *log.Logger) error {
	// loadbalancer's CertBus private key for which the certificate private keys are encrypted
	privateKey, err := ioutil.ReadFile("certbus-client.key")
	if err != nil {
		return err
	}

	tenantCtx, err := ehreader.TenantCtxFrom(ehreader.ConfigFromEnv)
	if err != nil {
		return err
	}

	certBus, err := certbus.New(
		ctx,
		*tenantCtx,
		string(privateKey),
		logex.Prefix("certbus", logger))
	if err != nil {
		return err
	}

	routes := http.NewServeMux()
	routes.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "greetings from %s\n", req.URL.Path)
	})

	srv := &http.Server{
		Addr:    ":443",
		Handler: routes,
		TLSConfig: &tls.Config{
			// this integrates CertBus into your server - certificates are fetched
			// dynamically from CertBus's dynamically managed state
			GetCertificate: certBus.GetCertificateAdapter(),
		},
	}

	// you don't have to use taskrunner, but it makes graceful stopping simpler
	tasks := taskrunner.New(ctx, logger)

	tasks.Start("certbus sync", func(ctx context.Context) error {
		return certBus.Synchronizer(ctx)
	})

	tasks.Start("http server (https://localhost)", func(_ context.Context) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServeTLS("", ""))
	})

	// Go's HTTP server doesn't support stopping via context cancel, so we'll need
	// additional goroutine to map cancellation to Shutdown() call
	tasks.Start("http server shutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}
