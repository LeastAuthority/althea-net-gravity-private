package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/types"
)

func TestQueryValsetConfirm(t *testing.T) {
	var (
		nonce                      = uint64(1)
		myValidatorCosmosAddr, _   = sdk.AccAddressFromBech32("cosmos1ees2tqhhhm9ahlhceh2zdguww9lqn2ckukn86l")
		myValidatorEthereumAddr, _ = types.NewEthAddress("0x3232323232323232323232323232323232323232")
	)
	require.NoError(t, err)
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	input.GravityKeeper.SetValsetConfirm(ctx, types.MsgValsetConfirm{
		Nonce:        nonce,
		Orchestrator: myValidatorCosmosAddr.String(),
		EthAddress:   myValidatorEthereumAddr.GetAddress(),
		Signature:    "alksdjhflkasjdfoiasjdfiasjdfoiasdj",
	})

	specs := map[string]struct {
		src     types.QueryValsetConfirmRequest
		expErr  bool
		expResp types.QueryValsetConfirmResponse
	}{
		/*  Nonce        uint64 `protobuf:"varint,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
		    Orchestrator string `protobuf:"bytes,2,opt,name=orchestrator,proto3" json:"orchestrator,omitempty"`
		    EthAddress   string `protobuf:"bytes,3,opt,name=eth_address,json=ethAddress,proto3" json:"eth_address,omitempty"`
		    Signature    string `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
		}*/

		"all good": {
<<<<<<< HEAD
			src: types.QueryValsetConfirmRequest{Nonce: 1, Address: myValidatorCosmosAddr.String()},

			//expResp:  []byte(`{"type":"gravity/MsgValsetConfirm", "value":{"eth_address":"0x3232323232323232323232323232323232323232", "nonce": "1", "orchestrator": "cosmos1ees2tqhhhm9ahlhceh2zdguww9lqn2ckukn86l",  "signature": "alksdjhflkasjdfoiasjdfiasjdfoiasdj"}}`),
			expResp: types.QueryValsetConfirmResponse{
				types.NewMsgValsetConfirm(1, *myValidatorEthereumAddr, []byte("cosmos1ees2tqhhhm9ahlhceh2zdguww9lqn2ckukn86l"), "alksdjhflkasjdfoiasjdfiasjdfoiasdj")},
=======
			srcNonce: "1",
			srcAddr:  myValidatorCosmosAddr.String(),
			expResp:  []byte(`{"type":"gravity/MsgValsetConfirm", "value":{"eth_address":"0x3232323232323232323232323232323232323232", "nonce": "1", "orchestrator": "gravity1ees2tqhhhm9ahlhceh2zdguww9lqn2ckcxpllh",  "signature": "alksdjhflkasjdfoiasjdfiasjdfoiasdj"}}`),
>>>>>>> 69dc62d (Switch from cosmos to gravity prefixed bech32 addresses)
		},
		"unknown nonce": {
			src:    types.QueryValsetConfirmRequest{Nonce: 999999, Address: myValidatorCosmosAddr.String()},
			expErr: true,
		},
		"invalid address": {
			src: types.QueryValsetConfirmRequest{1, "not a valid addr"},
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.ValsetConfirm(ctx.Context(), &types.QueryValsetConfirmRequest{Nonce: spec.src.Nonce})
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if spec.expResp == (types.QueryValsetConfirmResponse{}) {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, spec.expResp, got)
		})
	}
}

//nolint: exhaustivestruct
func TestAllValsetConfirmsBynonce(t *testing.T) {
	var (
		nonce                       = uint64(1)
		myValidatorCosmosAddr1, _   = sdk.AccAddressFromBech32("cosmos1u508cfnsk2nhakv80vdtq3nf558ngyvldkfjj9")
		myValidatorCosmosAddr2, _   = sdk.AccAddressFromBech32("cosmos1krtcsrxhadj54px0vy6j33pjuzcd3jj8kmsazv")
		myValidatorCosmosAddr3, _   = sdk.AccAddressFromBech32("cosmos1u94xef3cp9thkcpxecuvhtpwnmg8mhlja8hzkd")
		myValidatorEthereumAddr1, _ = types.NewEthAddress("0x0101010101010101010101010101010101010101")
		myValidatorEthereumAddr2, _ = types.NewEthAddress("0x0202020202020202020202020202020202020202")
		myValidatorEthereumAddr3, _ = types.NewEthAddress("0x0303030303030303030303030303030303030303")
	)

	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	addrs := []string{
		"gravity1u508cfnsk2nhakv80vdtq3nf558ngyvlfxm2hd",
		"gravity1krtcsrxhadj54px0vy6j33pjuzcd3jj8jtz98y",
		"gravity1u94xef3cp9thkcpxecuvhtpwnmg8mhljeh96n9",
	}
	// seed confirmations
	for i := 0; i < 3; i++ {
		addr, _ := sdk.AccAddressFromBech32(addrs[i])
		msg := types.MsgValsetConfirm{}
		msg.EthAddress = gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(i + 1)}, 20)).String()
		msg.Nonce = uint64(1)
		msg.Orchestrator = addr.String()
		msg.Signature = fmt.Sprintf("signature %d", i+1)
		input.GravityKeeper.SetValsetConfirm(ctx, msg)
	}

	specs := map[string]struct {
		src     types.QueryValsetConfirmsByNonceRequest
		expErr  bool
		expResp types.QueryValsetConfirmsByNonceResponse
	}{
		"all good": {
			src:     types.QueryValsetConfirmsByNonceRequest{Nonce: 1},
			expResp: types.QueryValsetConfirmsByNonceResponse{[]*types.MsgValsetConfirm{types.NewMsgValsetConfirm(nonce, *myValidatorEthereumAddr1, myValidatorCosmosAddr1, "1"), types.NewMsgValsetConfirm(nonce, *myValidatorEthereumAddr2, myValidatorCosmosAddr2, "2"), types.NewMsgValsetConfirm(nonce, *myValidatorEthereumAddr3, myValidatorCosmosAddr3, "3")}},
		},
		"unknown nonce": {
			src:     types.QueryValsetConfirmsByNonceRequest{Nonce: 999999},
			expResp: types.QueryValsetConfirmsByNonceResponse{},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.ValsetConfirmsByNonce(ctx.Context(), &types.QueryValsetConfirmsByNonceRequest{Nonce: spec.src.Nonce})
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if spec.expResp.Confirms[0] == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, spec.expResp, got)
		})
	}
}

// TODO: Check failure modes
//nolint: exhaustivestruct
func TestLastValsetRequests(t *testing.T) {

	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	// seed with maxValsetRequestsReturns + 1 requests
	for i := 0; i < maxValsetRequestsReturned+1; i++ {
		var validators []sdk.ValAddress
		for j := 0; j <= i; j++ {
			// add an validator each block
			valAddr := bytes.Repeat([]byte{byte(j)}, 20)
			ethAddr, err := types.NewEthAddress(gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(j + 1)}, 20)).String())
			require.NoError(t, err)
			input.GravityKeeper.SetEthAddressForValidator(ctx, valAddr, *ethAddr)
			validators = append(validators, valAddr)
		}
		input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(validators...)
		ctx = ctx.WithBlockHeight(int64(100 + i))
		input.GravityKeeper.SetValsetRequest(ctx)
	}

	val1 := types.Valset{
		Nonce:        6,
		Height:       105,
		RewardAmount: sdk.ZeroInt(),
		RewardToken:  "0x0000000000000000000000000000000000000000",
		Members: []*types.BridgeValidator{
			{
				Power:           715827882,
				EthereumAddress: "0x0101010101010101010101010101010101010101",
			},
			{
				Power:           715827882,
				EthereumAddress: "0x0202020202020202020202020202020202020202",
			},
			{
				Power:           715827882,
				EthereumAddress: "0x0303030303030303030303030303030303030303",
			},
			{
				Power:           715827882,
				EthereumAddress: "0x0404040404040404040404040404040404040404",
			},
			{
				Power:           715827882,
				EthereumAddress: "0x0505050505050505050505050505050505050505",
			},
			{
				Power:           715827882,
				EthereumAddress: "0x0606060606060606060606060606060606060606",
			},
		},
	}

	val2 := types.Valset{
		Nonce:        5,
		Height:       104,
		RewardAmount: sdk.ZeroInt(),
		RewardToken:  "0x0000000000000000000000000000000000000000",
		Members: []*types.BridgeValidator{
			{
				Power:           858993459,
				EthereumAddress: "0x0101010101010101010101010101010101010101",
			},
			{
				Power:           858993459,
				EthereumAddress: "0x0202020202020202020202020202020202020202",
			},
			{
				Power:           858993459,
				EthereumAddress: "0x0303030303030303030303030303030303030303",
			},
			{
				Power:           858993459,
				EthereumAddress: "0x0404040404040404040404040404040404040404",
			},
			{
				Power:           858993459,
				EthereumAddress: "0x0505050505050505050505050505050505050505",
			},
		},
	}

	val3 := types.Valset{
		Nonce:        4,
		Height:       103,
		RewardAmount: sdk.ZeroInt(),
		RewardToken:  "0x0000000000000000000000000000000000000000",
		Members: []*types.BridgeValidator{
			{
				Power:           1073741824,
				EthereumAddress: "0x0101010101010101010101010101010101010101",
			},
			{
				Power:           1073741824,
				EthereumAddress: "0x0202020202020202020202020202020202020202",
			},
			{
				Power:           1073741824,
				EthereumAddress: "0x0303030303030303030303030303030303030303",
			},
			{
				Power:           1073741824,
				EthereumAddress: "0x0404040404040404040404040404040404040404",
			},
		},
	}

	val4 := types.Valset{
		Nonce:        3,
		Height:       102,
		RewardAmount: sdk.ZeroInt(),
		RewardToken:  "0x0000000000000000000000000000000000000000",
		Members: []*types.BridgeValidator{
			{
				Power:           1431655765,
				EthereumAddress: "0x0101010101010101010101010101010101010101",
			},
			{
				Power:           1431655765,
				EthereumAddress: "0x0202020202020202020202020202020202020202",
			},
			{
				Power:           1431655765,
				EthereumAddress: "0x0303030303030303030303030303030303030303",
			},
		},
	}

	val5 := types.Valset{
		Nonce:        2,
		Height:       101,
		RewardAmount: sdk.ZeroInt(),
		RewardToken:  "0x0000000000000000000000000000000000000000",
		Members: []*types.BridgeValidator{
			{
				Power:           2147483648,
				EthereumAddress: "0x0101010101010101010101010101010101010101",
			},
			{
				Power:           2147483648,
				EthereumAddress: "0x0202020202020202020202020202020202020202",
			},
		},
	}

	valArray := &types.Valsets{&val1, &val2, &val3, &val4, &val5}

	specs := map[string]struct {
		expResp types.QueryLastValsetRequestsResponse
	}{ // Expect only maxValsetRequestsReturns back
		"limit at 5": {
			expResp: types.QueryLastValsetRequestsResponse{*valArray},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.LastValsetRequests(ctx.Context(), &types.QueryLastValsetRequestsRequest{})
			require.NoError(t, err)
			assert.Equal(t, spec.expResp, got)
		})
	}
}

//nolint: exhaustivestruct
// TODO: check that it doesn't accidently return a valset that HAS been signed
// Right now it is basically just testing that any valset comes back
func TestPendingValsetRequests(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	// seed with requests
	for i := 0; i < 6; i++ {
		var validators []sdk.ValAddress
		for j := 0; j <= i; j++ {
			// add an validator each block
			valAddr := bytes.Repeat([]byte{byte(j)}, 20)
			ethAddr, err := types.NewEthAddress(gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(j + 1)}, 20)).String())
			require.NoError(t, err)
			input.GravityKeeper.SetEthAddressForValidator(ctx, valAddr, *ethAddr)
			validators = append(validators, valAddr)
		}
		input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(validators...)
		ctx = ctx.WithBlockHeight(int64(100 + i))
		input.GravityKeeper.SetValsetRequest(ctx)
	}

	specs := map[string]struct {
		expResp types.QueryLastPendingValsetRequestByAddrRequest
	}{
		"find valset": {
			expResp: types.QueryLastPendingValsetRequestByAddrRequest{Address: `[
                                  {
                                    "nonce": "6",
                                    "members": [
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      },
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0202020202020202020202020202020202020202"
                                      },
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0303030303030303030303030303030303030303"
                                      },
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0404040404040404040404040404040404040404"
                                      },
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0505050505050505050505050505050505050505"
                                      },
                                      {
                                        "power": "715827882",
                                        "ethereum_address": "0x0606060606060606060606060606060606060606"
                                      }
                                    ],
                                    "height": "105",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  },
                                  {
                                    "nonce": "5",
                                    "members": [
                                      {
                                        "power": "858993459",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      },
                                      {
                                        "power": "858993459",
                                        "ethereum_address": "0x0202020202020202020202020202020202020202"
                                      },
                                      {
                                        "power": "858993459",
                                        "ethereum_address": "0x0303030303030303030303030303030303030303"
                                      },
                                      {
                                        "power": "858993459",
                                        "ethereum_address": "0x0404040404040404040404040404040404040404"
                                      },
                                      {
                                        "power": "858993459",
                                        "ethereum_address": "0x0505050505050505050505050505050505050505"
                                      }
                                    ],
                                    "height": "104",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  },
                                  {
                                    "nonce": "4",
                                    "members": [
                                      {
                                        "power": "1073741824",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      },
                                      {
                                        "power": "1073741824",
                                        "ethereum_address": "0x0202020202020202020202020202020202020202"
                                      },
                                      {
                                        "power": "1073741824",
                                        "ethereum_address": "0x0303030303030303030303030303030303030303"
                                      },
                                      {
                                        "power": "1073741824",
                                        "ethereum_address": "0x0404040404040404040404040404040404040404"
                                      }
                                    ],
                                    "height": "103",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  },
                                  {
                                    "nonce": "3",
                                    "members": [
                                      {
                                        "power": "1431655765",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      },
                                      {
                                        "power": "1431655765",
                                        "ethereum_address": "0x0202020202020202020202020202020202020202"
                                      },
                                      {
                                        "power": "1431655765",
                                        "ethereum_address": "0x0303030303030303030303030303030303030303"
                                      }
                                    ],
                                    "height": "102",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  },
                                  {
                                    "nonce": "2",
                                    "members": [
                                      {
                                        "power": "2147483648",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      },
                                      {
                                        "power": "2147483648",
                                        "ethereum_address": "0x0202020202020202020202020202020202020202"
                                      }
                                    ],
                                    "height": "101",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  },
                                  {
                                    "nonce": "1",
                                    "members": [
                                      {
                                        "power": "4294967296",
                                        "ethereum_address": "0x0101010101010101010101010101010101010101"
                                      }
                                    ],
                                    "height": "100",
									"reward_amount": "0",
                                    "reward_token": "0x0000000000000000000000000000000000000000"
                                  }
                                ]`},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.LastPendingValsetRequestByAddr(ctx.Context(), &spec.expResp)
			require.NoError(t, err)
			assert.Equal(t, spec.expResp, got, got)
		})
	}
}

//nolint: exhaustivestruct
// TODO: check that it actually returns a batch that has NOT been signed, not just any batch
func TestLastPendingBatchRequest(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	// seed with valset requests and eth addresses to make validators
	// that we will later use to lookup batches to be signed
	for i := 0; i < 6; i++ {
		var validators []sdk.ValAddress
		for j := 0; j <= i; j++ {
			// add an validator each block
			// TODO: replace with real SDK addresses
			valAddr := bytes.Repeat([]byte{byte(j)}, 20)
			ethAddr, err := types.NewEthAddress(gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(j + 1)}, 20)).String())
			require.NoError(t, err)
			input.GravityKeeper.SetEthAddressForValidator(ctx, valAddr, *ethAddr)
			validators = append(validators, valAddr)
		}
		input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(validators...)
		input.GravityKeeper.SetValsetRequest(ctx)
	}

	createTestBatch(t, input)

	specs := map[string]struct {
		expResp types.QueryLastPendingBatchRequestByAddrRequest
	}{
		"find batch": {
			expResp: types.QueryLastPendingBatchRequestByAddrRequest{Address: `{
	"type": "gravity/OutgoingTxBatch",
	"value": {
	"batch_nonce": "1",
	"block": "1234567",
	"transactions": [
		{
		"id": "2",
		"sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
		"dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
		"erc20_token": {
			"amount": "101",
			"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		},
		"erc20_fee": {
			"amount": "3",
			"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		}
		},
		{
		"id": "3",
		"sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
		"dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
		"erc20_token": {
			"amount": "102",
			"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		},
		"erc20_fee": {
			"amount": "2",
			"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		}
		}
	],
	"token_contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
	}
}
			`,
			}},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.LastPendingBatchRequestByAddr(ctx.Context(), &spec.expResp)
			require.NoError(t, err)
			assert.Equal(t, spec.expResp, got, got)
		})
	}
}

//nolint: exhaustivestruct
func createTestBatch(t *testing.T, input TestInput) {
	var (
		mySender            = bytes.Repeat([]byte{1}, 20)
		myReceiver          = "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934"
		myTokenContractAddr = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		now                 = time.Now().UTC()
	)
	receiver, err := types.NewEthAddress(myReceiver)
	require.NoError(t, err)
	tokenContract, err := types.NewEthAddress(myTokenContractAddr)
	require.NoError(t, err)
	// mint some voucher first
	token, err := types.NewInternalERC20Token(sdk.NewInt(99999), myTokenContractAddr)
	require.NoError(t, err)
	allVouchers := sdk.Coins{token.GravityCoin()}
	err = input.BankKeeper.MintCoins(input.Context, types.ModuleName, allVouchers)
	require.NoError(t, err)

	// set senders balance
	input.AccountKeeper.NewAccountWithAddress(input.Context, mySender)
	err = input.BankKeeper.SendCoinsFromModuleToAccount(input.Context, types.ModuleName, mySender, allVouchers)
	require.NoError(t, err)

	// add some TX to the pool
	for i, v := range []uint64{2, 3, 2, 1} {
		amountToken, err := types.NewInternalERC20Token(sdk.NewInt(int64(i+100)), myTokenContractAddr)
		require.NoError(t, err)
		amount := amountToken.GravityCoin()
		feeToken, err := types.NewInternalERC20Token(sdk.NewIntFromUint64(v), myTokenContractAddr)
		require.NoError(t, err)
		fee := feeToken.GravityCoin()
		_, err = input.GravityKeeper.AddToOutgoingPool(input.Context, mySender, *receiver, amount, fee)
		require.NoError(t, err)
		// Should create:
		// 1: amount 100, fee 2
		// 2: amount 101, fee 3
		// 3: amount 102, fee 2
		// 4: amount 103, fee 1
	}
	// when
	input.Context = input.Context.WithBlockTime(now)

	// tx batch size is 2, so that some of them stay behind
	_, err = input.GravityKeeper.BuildOutgoingTXBatch(input.Context, *tokenContract, 2)
	require.NoError(t, err)
	// Should have 2 and 3 from above
	// 1 and 4 should be unbatched
}

//nolint: exhaustivestruct
func TestQueryAllBatchConfirms(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	var (
		tokenContract      = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		validatorAddr, err = sdk.AccAddressFromBech32("gravity1mgamdcs9dah0vn0gqupl05up7pedg2mvc3tzjl")
	)
	require.NoError(t, err)

	input.GravityKeeper.SetBatchConfirm(ctx, &types.MsgConfirmBatch{
		Nonce:         1,
		TokenContract: tokenContract,
		EthSigner:     "0xf35e2cc8e6523d683ed44870f5b7cc785051a77d",
		Orchestrator:  validatorAddr.String(),
		Signature:     "signature",
	})

	batchConfirms, err := k.BatchRequestByNonce(ctx.Context(), &types.QueryBatchRequestByNonceRequest{Nonce: 1, ContractAddress: tokenContract})
	require.NoError(t, err)

	expectedJSON := []byte(`[{"eth_signer":"0xf35e2cc8e6523d683ed44870f5b7cc785051a77d", "nonce":"1", "signature":"signature", "token_contract":"0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", "orchestrator":"gravity1mgamdcs9dah0vn0gqupl05up7pedg2mvc3tzjl"}]`)

	assert.Equal(t, expectedJSON, batchConfirms, "json is equal")
}

//nolint: exhaustivestruct
func TestQueryLogicCalls(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	var (
		logicContract            = "0x510ab76899430424d209a6c9a5b9951fb8a6f47d"
		payload                  = []byte("fake bytes")
		tokenContract            = "0x7580bfe88dd3d07947908fae12d95872a260f2d8"
		invalidationId           = []byte("GravityTesting")
		invalidationNonce uint64 = 1
	)

	// seed with valset requests and eth addresses to make validators
	// that we will later use to lookup calls to be signed
	for i := 0; i < 6; i++ {
		var validators []sdk.ValAddress
		for j := 0; j <= i; j++ {
			// add an validator each block
			// TODO: replace with real SDK addresses
			valAddr := bytes.Repeat([]byte{byte(j)}, 20)
			ethAddr, err := types.NewEthAddress(gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(j + 1)}, 20)).String())
			require.NoError(t, err)
			input.GravityKeeper.SetEthAddressForValidator(ctx, valAddr, *ethAddr)
			validators = append(validators, valAddr)
		}
		input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(validators...)
	}

	token := []types.ERC20Token{{
		Contract: tokenContract,
		Amount:   sdk.NewIntFromUint64(5000),
	}}

	call := types.OutgoingLogicCall{
		Transfers:            token,
		Fees:                 token,
		LogicContractAddress: logicContract,
		Payload:              payload,
		Timeout:              10000,
		InvalidationId:       invalidationId,
		InvalidationNonce:    uint64(invalidationNonce),
	}
	k.SetOutgoingLogicCall(ctx, call)

	res := k.GetOutgoingLogicCall(ctx, invalidationId, invalidationNonce)

	require.Equal(t, call, *res)

	_, err := k.OutgoingLogicCalls(ctx.Context(), &types.QueryOutgoingLogicCallsRequest{})
	require.NoError(t, err)

	var valAddr sdk.AccAddress = bytes.Repeat([]byte{byte(1)}, 20)
	_, err = k.LastPendingLogicCallByAddr(ctx.Context(), &types.QueryLastPendingLogicCallByAddrRequest{Address: valAddr.String()})
	require.NoError(t, err)

	require.NoError(t, err)
}

//nolint: exhaustivestruct
func TestQueryLogicCallsConfirms(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	var (
		logicContract            = "0x510ab76899430424d209a6c9a5b9951fb8a6f47d"
		payload                  = []byte("fake bytes")
		tokenContract            = "0x7580bfe88dd3d07947908fae12d95872a260f2d8"
		invalidationId           = []byte("GravityTesting")
		invalidationNonce uint64 = 1
	)

	// seed with valset requests and eth addresses to make validators
	// that we will later use to lookup calls to be signed
	for i := 0; i < 6; i++ {
		var validators []sdk.ValAddress
		for j := 0; j <= i; j++ {
			// add an validator each block
			// TODO: replace with real SDK addresses
			valAddr := bytes.Repeat([]byte{byte(j)}, 20)
			ethAddr, err := types.NewEthAddress(gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(j + 1)}, 20)).String())
			require.NoError(t, err)
			input.GravityKeeper.SetEthAddressForValidator(ctx, valAddr, *ethAddr)
			validators = append(validators, valAddr)
		}
		input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(validators...)
	}

	token := []types.ERC20Token{{
		Contract: tokenContract,
		Amount:   sdk.NewIntFromUint64(5000),
	}}

	call := types.OutgoingLogicCall{
		Transfers:            token,
		Fees:                 token,
		LogicContractAddress: logicContract,
		Payload:              payload,
		Timeout:              10000,
		InvalidationId:       invalidationId,
		InvalidationNonce:    uint64(invalidationNonce),
	}
	k.SetOutgoingLogicCall(ctx, call)

	var valAddr sdk.AccAddress = bytes.Repeat([]byte{byte(1)}, 20)

	confirm := types.MsgConfirmLogicCall{
		InvalidationId:    hex.EncodeToString(invalidationId),
		InvalidationNonce: 1,
		EthSigner:         "test",
		Orchestrator:      valAddr.String(),
		Signature:         "test",
	}

	k.SetLogicCallConfirm(ctx, &confirm)

	res := k.GetLogicConfirmByInvalidationIDAndNonce(ctx, invalidationId, 1)
	assert.Equal(t, len(res), 1)
}

//nolint: exhaustivestruct
// TODO: test that it gets the correct batch, not just any batch.
// Check with multiple nonces and tokenContracts
func TestQueryBatch(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	var (
		tokenContract = "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
	)

	createTestBatch(t, input)

	batch, err := k.BatchConfirms(ctx.Context(), &types.QueryBatchConfirmsRequest{Nonce: 1, ContractAddress: tokenContract})
	require.NoError(t, err)

	expectedJSON := []byte(`{
		"type": "gravity/OutgoingTxBatch",
		"value": {
		  "transactions": [
			{
			  "erc20_fee": {
				"amount": "3",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "101",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "2"
			},
			{
			  "erc20_fee": {
				"amount": "2",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "102",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "3"
			}
		  ],
		  "batch_nonce": "1",
		  "block": "1234567",
		  "token_contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		}
	  }
	  `)

	// TODO: this test is failing on the empty representation of valset members
	assert.Equal(t, string(expectedJSON), batch, batch)
}

//nolint: exhaustivestruct
func TestLastBatchesRequest(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper

	createTestBatch(t, input)
	createTestBatch(t, input)

	lastBatches, err := k.OutgoingTxBatches(ctx.Context(), &types.QueryOutgoingTxBatchesRequest{})
	require.NoError(t, err)

	expectedJSON := []byte(`[
		{
		  "transactions": [
			{
			  "erc20_fee": {
				"amount": "3",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "101",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "6"
			},
			{
			  "erc20_fee": {
				"amount": "2",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "102",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "7"
			}
		  ],
		  "batch_nonce": "2",
		  "block": "1234567",
		  "token_contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		},
		{
		  "transactions": [
			{
			  "erc20_fee": {
				"amount": "3",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "101",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "2"
			},
			{
			  "erc20_fee": {
				"amount": "2",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "dest_address": "0x320915BD0F1bad11cBf06e85D5199DBcAC4E9934",
			  "erc20_token": {
				"amount": "102",
				"contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
			  },
			  "sender": "gravity1qyqszqgpqyqszqgpqyqszqgpqyqszqgpkrnxg5",
			  "id": "3"
			}
		  ],
		  "batch_nonce": "1",
		  "block": "1234567",
		  "token_contract": "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"
		}
	  ]
	  `)

	assert.Equal(t, expectedJSON, lastBatches, "json is equal")
}

//nolint: exhaustivestruct
// tests setting and querying eth address and orchestrator addresses
func TestQueryCurrentValset(t *testing.T) {
	var (
		ethAddress                = "0xb462864E395d88d6bc7C5dd5F3F5eb4cc2599255"
		valAddress sdk.ValAddress = bytes.Repeat([]byte{0x2}, 20)
	)
	addr, err := types.NewEthAddress(ethAddress)
	require.NoError(t, err)
	input := CreateTestEnv(t)
	input.GravityKeeper.StakingKeeper = NewStakingKeeperMock(valAddress)
	ctx := input.Context
	input.GravityKeeper.SetEthAddressForValidator(ctx, valAddress, *addr)

	currentValset := input.GravityKeeper.GetCurrentValset(ctx)

	bridgeVal := types.BridgeValidator{EthereumAddress: ethAddress, Power: 4294967296}
	internalBridgeVal, err := bridgeVal.ToInternal()
	require.NoError(t, err)
	internalBridgeVals := types.InternalBridgeValidators([]*types.InternalBridgeValidator{internalBridgeVal})
	expectedValset, err := types.NewValset(1, 1234567, internalBridgeVals, sdk.NewIntFromUint64(0), *types.ZeroAddress())
	require.NoError(t, err)
	assert.Equal(t, *expectedValset, currentValset)
}

//nolint: exhaustivestruct
func TestQueryERC20ToDenom(t *testing.T) {
	var (
		erc20, err = types.NewEthAddress("0xb462864E395d88d6bc7C5dd5F3F5eb4cc2599255")
		denom      = "uatom"
	)
	require.NoError(t, err)
	response := types.QueryERC20ToDenomResponse{
		Denom:            denom,
		CosmosOriginated: true,
	}
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	input.GravityKeeper.setCosmosOriginatedDenomToERC20(ctx, denom, *erc20)

	queriedDenom, err := k.ERC20ToDenom(ctx.Context(), &types.QueryERC20ToDenomRequest{erc20.GetAddress()})
	require.NoError(t, err)
	correctBytes, err := codec.MarshalJSONIndent(types.ModuleCdc, response)
	require.NoError(t, err)

	assert.Equal(t, correctBytes, queriedDenom)
}

//nolint: exhaustivestruct
func TestQueryDenomToERC20(t *testing.T) {
	var (
		erc20, err = types.NewEthAddress("0xb462864E395d88d6bc7C5dd5F3F5eb4cc2599255")
		denom      = "uatom"
	)
	require.NoError(t, err)
	response := types.QueryDenomToERC20Response{
		Erc20:            erc20.GetAddress(),
		CosmosOriginated: true,
	}
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	input.GravityKeeper.setCosmosOriginatedDenomToERC20(ctx, denom, *erc20)

	queriedERC20, err := k.DenomToERC20(ctx.Context(), &types.QueryDenomToERC20Request{denom})
	require.NoError(t, err)

	correctBytes, err := codec.MarshalJSONIndent(types.ModuleCdc, response)
	require.NoError(t, err)

	assert.Equal(t, correctBytes, queriedERC20)
}

//nolint: exhaustivestruct
func TestQueryPendingSendToEth(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	k := input.GravityKeeper
	var (
		now                 = time.Now().UTC()
		mySender, err1      = sdk.AccAddressFromBech32("gravity1ahx7f8wyertuus9r20284ej0asrs085ceqtfnm")
		myReceiver          = "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7"
		myTokenContractAddr = "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5" // Pickle
		token, err2         = types.NewInternalERC20Token(sdk.NewInt(99999), myTokenContractAddr)
		allVouchers         = sdk.NewCoins(token.GravityCoin())
	)
	require.NoError(t, err1)
	require.NoError(t, err2)
	receiver, err := types.NewEthAddress(myReceiver)
	require.NoError(t, err)
	tokenContract, err := types.NewEthAddress(myTokenContractAddr)
	require.NoError(t, err)

	// mint some voucher first
	require.NoError(t, input.BankKeeper.MintCoins(ctx, types.ModuleName, allVouchers))
	// set senders balance
	input.AccountKeeper.NewAccountWithAddress(ctx, mySender)
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, mySender, allVouchers))

	// CREATE FIRST BATCH
	// ==================

	// add some TX to the pool
	for i, v := range []uint64{2, 3, 2, 1} {
		amountToken, err := types.NewInternalERC20Token(sdk.NewInt(int64(i+100)), myTokenContractAddr)
		require.NoError(t, err)
		amount := amountToken.GravityCoin()
		feeToken, err := types.NewInternalERC20Token(sdk.NewIntFromUint64(v), myTokenContractAddr)
		require.NoError(t, err)
		fee := feeToken.GravityCoin()
		_, err = input.GravityKeeper.AddToOutgoingPool(ctx, mySender, *receiver, amount, fee)
		require.NoError(t, err)
		// Should create:
		// 1: amount 100, fee 2
		// 2: amount 101, fee 3
		// 3: amount 102, fee 2
		// 4: amount 104, fee 1
	}

	// when
	ctx = ctx.WithBlockTime(now)

	// tx batch size is 2, so that some of them stay behind
	// Should contain 2 and 3 from above
	_, err = input.GravityKeeper.BuildOutgoingTXBatch(ctx, *tokenContract, 2)
	require.NoError(t, err)

	// Should receive 1 and 4 unbatched, 2 and 3 batched in response
	response, err := k.GetPendingSendToEth(ctx.Context(), &types.QueryPendingSendToEth{mySender.String()})
	require.NoError(t, err)
	expectedJSON := []byte(`{
  "transfers_in_batches": [
    {
      "id": "2",
      "sender": "gravity1ahx7f8wyertuus9r20284ej0asrs085ceqtfnm",
      "dest_address": "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7",
      "erc20_token": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "101"
      },
      "erc20_fee": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "3"
      }
    },
    {
      "id": "3",
      "sender": "gravity1ahx7f8wyertuus9r20284ej0asrs085ceqtfnm",
      "dest_address": "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7",
      "erc20_token": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "102"
      },
      "erc20_fee": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "2"
      }
    }
  ],
  "unbatched_transfers": [
    {
      "id": "1",
      "sender": "gravity1ahx7f8wyertuus9r20284ej0asrs085ceqtfnm",
      "dest_address": "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7",
      "erc20_token": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "100"
      },
      "erc20_fee": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "2"
      }
    },
    {
      "id": "4",
      "sender": "gravity1ahx7f8wyertuus9r20284ej0asrs085ceqtfnm",
      "dest_address": "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7",
      "erc20_token": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "103"
      },
      "erc20_fee": {
        "contract": "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5",
        "amount": "1"
      }
    }
  ]}
	  `)

	assert.Equal(t, expectedJSON, response, "json is equal")
}
