package lnd

import (
	"context"

	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btclog"
	"github.com/lightninglabs/neutrino"
	sphinx "github.com/lightningnetwork/lightning-onion"
	"github.com/qtumproject/qlnd/autopilot"
	"github.com/qtumproject/qlnd/build"
	"github.com/qtumproject/qlnd/chainntnfs"
	"github.com/qtumproject/qlnd/chanbackup"
	"github.com/qtumproject/qlnd/chanfitness"
	"github.com/qtumproject/qlnd/channeldb"
	"github.com/qtumproject/qlnd/channelnotifier"
	"github.com/qtumproject/qlnd/contractcourt"
	"github.com/qtumproject/qlnd/discovery"
	"github.com/qtumproject/qlnd/htlcswitch"
	"github.com/qtumproject/qlnd/invoices"
	"github.com/qtumproject/qlnd/lnrpc/autopilotrpc"
	"github.com/qtumproject/qlnd/lnrpc/chainrpc"
	"github.com/qtumproject/qlnd/lnrpc/invoicesrpc"
	"github.com/qtumproject/qlnd/lnrpc/routerrpc"
	"github.com/qtumproject/qlnd/lnrpc/signrpc"
	"github.com/qtumproject/qlnd/lnrpc/walletrpc"
	"github.com/qtumproject/qlnd/lnrpc/wtclientrpc"
	"github.com/qtumproject/qlnd/lnwallet"
	"github.com/qtumproject/qlnd/monitoring"
	"github.com/qtumproject/qlnd/netann"
	"github.com/qtumproject/qlnd/peernotifier"
	"github.com/qtumproject/qlnd/routing"
	"github.com/qtumproject/qlnd/signal"
	"github.com/qtumproject/qlnd/sweep"
	"github.com/qtumproject/qlnd/watchtower"
	"github.com/qtumproject/qlnd/watchtower/wtclient"
	"google.golang.org/grpc"
)

// Loggers per subsystem.  A single backend logger is created and all subsystem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by
// calling logWriter.InitLogRotator.
var (
	logWriter = build.NewRotatingLogWriter()

	// Loggers that need to be accessible from the lnd package can be placed
	// here. Loggers that are only used in sub modules can be added directly
	// by using the addSubLogger method.
	ltndLog = build.NewSubLogger("LTND", logWriter.GenSubLogger)
	peerLog = build.NewSubLogger("PEER", logWriter.GenSubLogger)
	rpcsLog = build.NewSubLogger("RPCS", logWriter.GenSubLogger)
	srvrLog = build.NewSubLogger("SRVR", logWriter.GenSubLogger)
	fndgLog = build.NewSubLogger("FNDG", logWriter.GenSubLogger)
	utxnLog = build.NewSubLogger("UTXN", logWriter.GenSubLogger)
	brarLog = build.NewSubLogger("BRAR", logWriter.GenSubLogger)
	atplLog = build.NewSubLogger("ATPL", logWriter.GenSubLogger)
)

// Initialize package-global logger variables.
func init() {
	setSubLogger("LTND", ltndLog, signal.UseLogger)
	setSubLogger("ATPL", atplLog, autopilot.UseLogger)
	setSubLogger("PEER", peerLog, nil)
	setSubLogger("RPCS", rpcsLog, nil)
	setSubLogger("SRVR", srvrLog, nil)
	setSubLogger("FNDG", fndgLog, nil)
	setSubLogger("UTXN", utxnLog, nil)
	setSubLogger("BRAR", brarLog, nil)

	addSubLogger("LNWL", lnwallet.UseLogger)
	addSubLogger("DISC", discovery.UseLogger)
	addSubLogger("NTFN", chainntnfs.UseLogger)
	addSubLogger("CHDB", channeldb.UseLogger)
	addSubLogger("HSWC", htlcswitch.UseLogger)
	addSubLogger("CMGR", connmgr.UseLogger)
	addSubLogger("CRTR", routing.UseLogger)
	addSubLogger("BTCN", neutrino.UseLogger)
	addSubLogger("CNCT", contractcourt.UseLogger)
	addSubLogger("SPHX", sphinx.UseLogger)
	addSubLogger("SWPR", sweep.UseLogger)
	addSubLogger("SGNR", signrpc.UseLogger)
	addSubLogger("WLKT", walletrpc.UseLogger)
	addSubLogger("ARPC", autopilotrpc.UseLogger)
	addSubLogger("INVC", invoices.UseLogger)
	addSubLogger("NANN", netann.UseLogger)
	addSubLogger("WTWR", watchtower.UseLogger)
	addSubLogger("NTFR", chainrpc.UseLogger)
	addSubLogger("IRPC", invoicesrpc.UseLogger)
	addSubLogger("CHNF", channelnotifier.UseLogger)
	addSubLogger("CHBU", chanbackup.UseLogger)
	addSubLogger("PROM", monitoring.UseLogger)
	addSubLogger("WTCL", wtclient.UseLogger)
	addSubLogger("PRNF", peernotifier.UseLogger)

	addSubLogger(routerrpc.Subsystem, routerrpc.UseLogger)
	addSubLogger(wtclientrpc.Subsystem, wtclientrpc.UseLogger)
	addSubLogger(chanfitness.Subsystem, chanfitness.UseLogger)
}

// addSubLogger is a helper method to conveniently create and register the
// logger of a sub system.
func addSubLogger(subsystem string, useLogger func(btclog.Logger)) {
	logger := build.NewSubLogger(subsystem, logWriter.GenSubLogger)
	setSubLogger(subsystem, logger, useLogger)
}

// setSubLogger is a helper method to conveniently register the logger of a sub
// system.
func setSubLogger(subsystem string, logger btclog.Logger,
	useLogger func(btclog.Logger)) {

	logWriter.RegisterSubLogger(subsystem, logger)
	if useLogger != nil {
		useLogger(logger)
	}
}

// logClosure is used to provide a closure over expensive logging operations so
// don't have to be performed when the logging level doesn't warrant it.
type logClosure func() string

// String invokes the underlying function and returns the result.
func (c logClosure) String() string {
	return c()
}

// newLogClosure returns a new closure over a function that returns a string
// which itself provides a Stringer interface so that it can be used with the
// logging system.
func newLogClosure(c func() string) logClosure {
	return logClosure(c)
}

// errorLogUnaryServerInterceptor is a simple UnaryServerInterceptor that will
// automatically log any errors that occur when serving a client's unary
// request.
func errorLogUnaryServerInterceptor(logger btclog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		resp, err := handler(ctx, req)
		if err != nil {
			// TODO(roasbeef): also log request details?
			logger.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return resp, err
	}
}

// errorLogStreamServerInterceptor is a simple StreamServerInterceptor that
// will log any errors that occur while processing a client or server streaming
// RPC.
func errorLogStreamServerInterceptor(logger btclog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		err := handler(srv, ss)
		if err != nil {
			logger.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return err
	}
}
