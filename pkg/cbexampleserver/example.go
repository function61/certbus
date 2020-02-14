package cbexampleserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/taskrunner"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
)

func Entrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "example-server",
		Short: "Start demo HTTPS server that demos CertBus integration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()
			if err := exampleServer(ossignal.InterruptOrTerminateBackgroundCtx(rootLogger), rootLogger); err != nil {
				panic(err)
			}
		},
	}
}

func exampleServer(ctx context.Context, logger *log.Logger) error {
	// loadbalancer's CertBus private key for which the certificate private keys are encrypted
	privateKey, err := ioutil.ReadFile("certbus-client.key")
	if err != nil {
		return err
	}

	certBus, err := certbus.New(
		ctx,
		tenantClient(),
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

	tasks.Start("certbus sync", func(ctx context.Context, _ string) error {
		return certBus.Synchronizer(ctx)
	})

	tasks.Start("http server (https://localhost)", func(_ context.Context, _ string) error {
		return removeGracefulServerClosedError(srv.ListenAndServeTLS("", ""))
	})

	// Go's HTTP server doesn't support stopping via context cancel, so we'll need
	// additional goroutine to map cancellation to Shutdown() call
	tasks.Start("http server shutdowner", httpShutdownTask(srv))

	return tasks.Wait()
}

func tenantClient() ehreader.TenantClient {
	client, err := ehreader.TenantConfigFromEnv()
	if err != nil {
		panic(err)
	}

	return client
}

// helper for making HTTP shutdown task. Go's http.Server is weird that we cannot use
// context cancellation to stop it, but instead we have to call srv.Shutdown()
func httpShutdownTask(server *http.Server) func(context.Context, string) error {
	return func(ctx context.Context, _ string) error {
		<-ctx.Done()
		// can't use task ctx b/c it'd cancel the Shutdown() itself
		return server.Shutdown(context.Background())
	}
}

func removeGracefulServerClosedError(httpServerError error) error {
	if httpServerError == http.ErrServerClosed {
		return nil
	} else {
		// some other error
		// (or nil, but http server should always exit with non-nil error)
		return httpServerError
	}
}
