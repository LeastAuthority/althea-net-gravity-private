package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/types"
)

const (
	QueryCurrentValset = "currentValset"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err error) {
		switch path[0] {

		// Valsets
		case QueryCurrentValset:
			return queryCurrentValset(ctx, keeper)
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown %s query endpoint", types.ModuleName)
		}
	}
}

func queryCurrentValset(ctx sdk.Context, keeper Keeper) ([]byte, error) {
	valset := keeper.GetCurrentValset(ctx)
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, valset)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

const maxValsetRequestsReturned = 5

type MultiSigUpdateResponse struct {
	Valset     types.Valset `json:"valset"`
	Signatures [][]byte     `json:"signatures,omitempty"`
}

<<<<<<< HEAD
const MaxResults = 100 // todo: impl pagination
=======
// lastPendingBatchRequest gets the latest batch that has NOT been signed by operatorAddr
func lastPendingBatchRequest(ctx sdk.Context, operatorAddr string, keeper Keeper) ([]byte, error) {
	addr, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	var pendingBatchReq types.OutgoingTxBatch
	keeper.IterateOutgoingTXBatches(ctx, func(_ []byte, batch types.InternalOutgoingTxBatch) bool {
		foundConfirm := keeper.GetBatchConfirm(ctx, batch.BatchNonce, batch.TokenContract, addr) != nil
		if !foundConfirm {
			pendingBatchReq = batch.ToExternal()
			return true
		}
		return false
	})
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, pendingBatchReq)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return res, nil
}

const MaxResults = 100 // todo: impl pagination

// Gets MaxResults batches from store. Does not select by token type or anything
func lastBatchesRequest(ctx sdk.Context, keeper Keeper) ([]byte, error) {
	var batches []types.OutgoingTxBatch
	keeper.IterateOutgoingTXBatches(ctx, func(_ []byte, batch types.InternalOutgoingTxBatch) bool {
		batches = append(batches, batch.ToExternal())
		return len(batches) == MaxResults
	})
	if len(batches) == 0 {
		return nil, nil
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, batches)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return res, nil
}

func queryBatchFees(ctx sdk.Context, keeper Keeper) ([]byte, error) {
	val := types.QueryBatchFeeResponse{BatchFees: keeper.GetAllBatchFees(ctx, OutgoingTxBatchSize)}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, val)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return res, nil
}

// Gets MaxResults logic calls from store.
func lastLogicCallRequests(ctx sdk.Context, keeper Keeper) ([]byte, error) {
	var calls []types.OutgoingLogicCall
	keeper.IterateOutgoingLogicCalls(ctx, func(_ []byte, call types.OutgoingLogicCall) bool {
		calls = append(calls, call)
		return len(calls) == MaxResults
	})
	if len(calls) == 0 {
		return nil, nil
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, calls)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return res, nil
}

// queryBatch gets a batch by tokenContract and nonce
func queryBatch(ctx sdk.Context, nonce string, tokenContract string, keeper Keeper) ([]byte, error) {
	parsedNonce, err := types.UInt64FromString(nonce)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}
	contract, err := types.NewEthAddress(tokenContract)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}
	foundBatch := keeper.GetOutgoingTXBatch(ctx, *contract, parsedNonce)
	if foundBatch == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "Can not find tx batch")
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, foundBatch.ToExternal())
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}
	return res, nil
}

// lastPendingLogicCallRequest gets the latest call that has NOT been signed by operatorAddr
func lastPendingLogicCallRequest(ctx sdk.Context, operatorAddr string, keeper Keeper) ([]byte, error) {
	addr, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	var pendingLogicCalls types.OutgoingLogicCall
	found := false
	keeper.IterateOutgoingLogicCalls(ctx, func(_ []byte, call types.OutgoingLogicCall) bool {
		foundConfirm := keeper.GetLogicCallConfirm(ctx, call.InvalidationId, call.InvalidationNonce, addr) != nil
		if !foundConfirm {
			pendingLogicCalls = call
			found = true
			return true
		}
		return false
	})
	if !found {
		return nil, nil
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, pendingLogicCalls)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return res, nil
}

// queryLogicCall gets a logic call by nonce and invalidation id
func queryLogicCall(ctx sdk.Context, invalidationId string, invalidationNonce string, keeper Keeper) ([]byte, error) {
	nonce, err := types.UInt64FromString(invalidationNonce)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	foundCall := keeper.GetOutgoingLogicCall(ctx, []byte(invalidationId), nonce)
	if foundCall == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "Can not find logic call")
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, foundCall)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}
	return res, nil
}

// allLogicCallConfirms returns all the confirm messages for a given nonce
// When nothing found an empty json array is returned. No pagination.
func queryAllLogicCallConfirms(ctx sdk.Context, invalidationId string, invalidationNonce string, keeper Keeper) ([]byte, error) {
	nonce, err := types.UInt64FromString(invalidationNonce)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	invalidationIdBytes, err := hex.DecodeString(invalidationId)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	var confirms []*types.MsgConfirmLogicCall
	keeper.IterateLogicConfirmByInvalidationIDAndNonce(ctx, invalidationIdBytes, nonce, func(_ []byte, c *types.MsgConfirmLogicCall) bool {
		confirms = append(confirms, c)
		return false
	})
	if len(confirms) == 0 {
		return nil, nil
	}
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, confirms)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryGravityID(ctx sdk.Context, keeper Keeper) ([]byte, error) {
	gravityID := keeper.GetGravityID(ctx)
	res, err := codec.MarshalJSONIndent(types.ModuleCdc, gravityID)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	} else {
		return res, nil
	}
}

func queryDenomToERC20(ctx sdk.Context, denom string, keeper Keeper) ([]byte, error) {
	cosmos_originated, erc20, err := keeper.DenomToERC20Lookup(ctx, denom)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	var response types.QueryDenomToERC20Response
	response.CosmosOriginated = cosmos_originated
	response.Erc20 = erc20.GetAddress()
	bytes, err := codec.MarshalJSONIndent(types.ModuleCdc, response)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	} else {
		return bytes, nil
	}
}

func queryERC20ToDenom(ctx sdk.Context, ERC20 string, keeper Keeper) ([]byte, error) {
	ethAddr, err := types.NewEthAddress(ERC20)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "invalid ERC20 in query: %s", ERC20)
	}
	cosmos_originated, denom := keeper.ERC20ToDenomLookup(ctx, *ethAddr)
	var response types.QueryERC20ToDenomResponse
	response.CosmosOriginated = cosmos_originated
	response.Denom = denom
	bytes, err := codec.MarshalJSONIndent(types.ModuleCdc, response)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	} else {
		return bytes, nil
	}
}

func queryPendingSendToEth(ctx sdk.Context, senderAddr string, k Keeper) ([]byte, error) {
	batches := k.GetOutgoingTxBatches(ctx)
	unbatched_tx := k.GetUnbatchedTransactions(ctx)
	res := types.QueryPendingSendToEthResponse{
		TransfersInBatches: []types.OutgoingTransferTx{},
		UnbatchedTransfers: []types.OutgoingTransferTx{},
	}
	for _, batch := range batches {
		for _, tx := range batch.Transactions {
			if tx.Sender.String() == senderAddr {
				res.TransfersInBatches = append(res.TransfersInBatches, tx.ToExternal())
			}
		}
	}
	for _, tx := range unbatched_tx {
		if tx.Sender.String() == senderAddr {
			res.UnbatchedTransfers = append(res.UnbatchedTransfers, tx.ToExternal())
		}
	}
	bytes, err := codec.MarshalJSONIndent(types.ModuleCdc, res)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	} else {
		return bytes, nil
	}
}
>>>>>>> 765cdc8 (Remove pointers in IterateOutgoingTXBatches)
