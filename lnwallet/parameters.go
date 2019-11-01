package lnwallet

import (
	"github.com/qtumproject/qtumsuite"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/qtumproject/qlnd/input"
)

// DefaultDustLimit is used to calculate the dust HTLC amount which will be
// send to other node during funding process.
func DefaultDustLimit() qtumsuite.Amount {
	return txrules.GetDustThreshold(input.P2WSHSize, txrules.DefaultRelayFeePerKb)
}
