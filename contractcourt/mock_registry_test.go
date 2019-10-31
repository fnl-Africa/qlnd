package contractcourt

import (
	"github.com/qtumproject/lnd/channeldb"
	"github.com/qtumproject/lnd/invoices"
	"github.com/qtumproject/lnd/lntypes"
	"github.com/qtumproject/lnd/lnwire"
)

type notifyExitHopData struct {
	payHash       lntypes.Hash
	paidAmount    lnwire.MilliSatoshi
	hodlChan      chan<- interface{}
	expiry        uint32
	currentHeight int32
}

type mockRegistry struct {
	notifyChan  chan notifyExitHopData
	notifyErr   error
	notifyEvent *invoices.HodlEvent
}

func (r *mockRegistry) NotifyExitHopHtlc(payHash lntypes.Hash,
	paidAmount lnwire.MilliSatoshi, expiry uint32, currentHeight int32,
	circuitKey channeldb.CircuitKey, hodlChan chan<- interface{},
	eob []byte) (*invoices.HodlEvent, error) {

	r.notifyChan <- notifyExitHopData{
		hodlChan:      hodlChan,
		payHash:       payHash,
		paidAmount:    paidAmount,
		expiry:        expiry,
		currentHeight: currentHeight,
	}

	return r.notifyEvent, r.notifyErr
}

func (r *mockRegistry) HodlUnsubscribeAll(subscriber chan<- interface{}) {}

func (r *mockRegistry) LookupInvoice(lntypes.Hash) (channeldb.Invoice,
	error) {

	return channeldb.Invoice{}, channeldb.ErrInvoiceNotFound
}
