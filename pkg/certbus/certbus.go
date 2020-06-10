// Alongside-loadbalancer library
package certbus

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/function61/certbus/pkg/certificatestore"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/logex"
)

type App struct {
	Certs          *certificatestore.DecryptedStore
	certsEncrypted *certificatestore.Store
	reader         *ehreader.Reader
	logl           *logex.Leveled
}

// returns certbus App (meant to be used alongside HTTP server)
func New(
	ctx context.Context,
	tenantCtx ehreader.TenantCtx,
	privateKeyPem string,
	logger *log.Logger,
) (*App, error) {
	certsEncrypted, err := ResolveRealtimeState(ctx, tenantCtx, logger)
	if err != nil {
		return nil, err
	}

	certsDecrypted, err := certificatestore.NewDecryptedStore(certsEncrypted, privateKeyPem)
	if err != nil {
		return nil, err
	}

	// FIXME: a reader is also made inside resolveRealtimeState()
	return &App{
		certsDecrypted,
		certsEncrypted,
		ehreader.New(certsEncrypted, tenantCtx.Client, logger),
		logex.Levels(logger),
	}, nil
}

func (c *App) GetCertificateAdapter() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return certificatestore.DecryptedByHostnameSupportingWildcard(hello.ServerName, c.Certs)
	}
}

func (c *App) Synchronizer(ctx context.Context) error {
	pollInterval := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pollInterval.C:
			// eventually we'll migrate to realtime notifications from eventhorizon,
			// but until then polling will do

			if err := c.reader.LoadUntilRealtime(ctx); err != nil {
				c.logl.Error.Printf("LoadUntilRealtime: %v", err)
			}
		}
	}
}

func ResolveRealtimeState(
	ctx context.Context,
	tenantCtx ehreader.TenantCtx,
	logger *log.Logger,
) (*certificatestore.Store, error) {
	certificates := certificatestore.New(tenantCtx.Tenant, logger)

	if err := ehreader.New(certificates, tenantCtx.Client, logger).LoadUntilRealtime(ctx); err != nil {
		return nil, err
	}

	return certificates, nil
}
