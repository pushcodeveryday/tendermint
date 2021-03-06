package http

import (
	"fmt"

	"github.com/tendermint/tendermint/lite2/provider"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

// SignStatusClient combines a SignClient and StatusClient.
type SignStatusClient interface {
	rpcclient.SignClient
	rpcclient.StatusClient
}

// http provider uses an RPC client (or SignStatusClient more generally) to
// obtain the necessary information.
type http struct {
	chainID string
	client  SignStatusClient
}

// New creates a HTTP provider, which is using the rpcclient.HTTP
// client under the hood.
func New(chainID, remote string) provider.Provider {
	return NewWithClient(chainID, rpcclient.NewHTTP(remote, "/websocket"))
}

// NewWithClient allows you to provide custom SignStatusClient.
func NewWithClient(chainID string, client SignStatusClient) provider.Provider {
	return &http{
		chainID: chainID,
		client:  client,
	}
}

func (p *http) ChainID() string {
	return p.chainID
}

func (p *http) SignedHeader(height int64) (*types.SignedHeader, error) {
	h, err := validateHeight(height)
	if err != nil {
		return nil, err
	}

	commit, err := p.client.Commit(h)
	if err != nil {
		return nil, err
	}

	// Verify we're still on the same chain.
	if p.chainID != commit.Header.ChainID {
		return nil, fmt.Errorf("expected chainID %s, got %s", p.chainID, commit.Header.ChainID)
	}

	return &commit.SignedHeader, nil
}

func (p *http) ValidatorSet(height int64) (*types.ValidatorSet, error) {
	h, err := validateHeight(height)
	if err != nil {
		return nil, err
	}

	const maxPerPage = 100
	res, err := p.client.Validators(h, 0, maxPerPage)
	if err != nil {
		return nil, err
	}

	var (
		vals = res.Validators
		page = 1
	)

	// Check if there are more validators.
	for len(res.Validators) == maxPerPage {
		res, err = p.client.Validators(h, page, maxPerPage)
		if err != nil {
			return nil, err
		}
		if len(res.Validators) > 0 {
			vals = append(vals, res.Validators...)
		}
		page++
	}

	return types.NewValidatorSet(vals), nil
}

func validateHeight(height int64) (*int64, error) {
	if height < 0 {
		return nil, fmt.Errorf("expected height >= 0, got height %d", height)
	}

	h := &height
	if height == 0 {
		h = nil
	}
	return h, nil
}
