package gravity

import (
	"fmt"
	"sort"

	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/keeper"
	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	params := k.GetParams(ctx)
	unhaltBridge(ctx, k, params)
	slashing(ctx, k)
	attestationTally(ctx, k)
	cleanupTimedOutBatches(ctx, k)
	cleanupTimedOutLogicCalls(ctx, k)
	createValsets(ctx, k)
	pruneValsets(ctx, k, params)
	pruneAttestations(ctx, k)
}

// In the event the bridge is halted and governance has decided to reset oracle
// history, we roll back oracle history and reset the parameters
func unhaltBridge(ctx sdk.Context, k keeper.Keeper, params types.Params) {
	if params.ResetBridgeState {
		if params.ResetBridgeNonce == 0 {
			ctx.Logger().Error("Attempted to reset the bridge to an invalid nonce", "nonce", params.ResetBridgeNonce)
		} else {
			ctx.Logger().Info("Gov vote passed: Resetting oracle history", "nonce", params.ResetBridgeNonce)
			pruneAttestationsAfterNonce(ctx, k, uint64(params.ResetBridgeNonce))
		}
		// Reset the parameters now that the bridge is unhalted
		reset := types.DefaultParams()
		k.SetResetBridgeNonce(ctx, reset.ResetBridgeNonce)
		k.SetResetBridgeState(ctx, reset.ResetBridgeState)
	}
}

func createValsets(ctx sdk.Context, k keeper.Keeper) {
	// Auto ValsetRequest Creation.
	// WARNING: do not use k.GetLastObservedValset in this function, it *will* result in losing control of the bridge
	// 1. If there are no valset requests, create a new one.
	// 2. If there is at least one validator who started unbonding in current block. (we persist last unbonded block height in hooks.go)
	//      This will make sure the unbonding validator has to provide an attestation to a new Valset
	//	    that excludes him before he completely Unbonds.  Otherwise he will be slashed
	// 3. If power change between validators of CurrentValset and latest valset request is > 5%

	// get the last valsets to compare against
	latestValset := k.GetLatestValset(ctx)
	lastUnbondingHeight := k.GetLastUnBondingBlockHeight(ctx)

	significantPowerDiff := false
	if latestValset != nil {
		intCurrMembers, err := types.BridgeValidators(k.GetCurrentValset(ctx).Members).ToInternal()
		if err != nil {
			panic(sdkerrors.Wrap(err, "invalid current valset members"))
		}
		intLatestMembers, err := types.BridgeValidators(latestValset.Members).ToInternal()
		if err != nil {
			panic(sdkerrors.Wrap(err, "invalid latest valset members"))
		}

		significantPowerDiff = intCurrMembers.PowerDiff(*intLatestMembers) > 0.05
	}

	if (latestValset == nil) || (lastUnbondingHeight == uint64(ctx.BlockHeight())) || significantPowerDiff {
		// if the conditions are true, put in a new validator set request to be signed and submitted to Ethereum
		k.SetValsetRequest(ctx)
	}
}

func pruneValsets(ctx sdk.Context, k keeper.Keeper, params types.Params) {
	// Validator set pruning
	// prune all validator sets with a nonce less than the
	// last observed nonce, they can't be submitted any longer
	//
	// Only prune valsets after the signed valsets window has passed
	// so that slashing can occur the block before we remove them
	lastObserved := k.GetLastObservedValset(ctx)
	currentBlock := uint64(ctx.BlockHeight())
	tooEarly := currentBlock < params.SignedValsetsWindow
	if lastObserved != nil && !tooEarly {
		earliestToPrune := currentBlock - params.SignedValsetsWindow
		sets := k.GetValsets(ctx)
		for _, set := range sets {
			if set.Nonce < lastObserved.Nonce && set.Height < earliestToPrune {
				k.DeleteValset(ctx, set.Nonce)
			}
		}
	}
}

func slashing(ctx sdk.Context, k keeper.Keeper) {

	params := k.GetParams(ctx)

	// Slash validator for not confirming valset requests, batch requests, logic call requests
	ValsetSlashing(ctx, k, params)
	BatchSlashing(ctx, k, params)
	LogicCallSlashing(ctx, k, params)

}

// Iterate over all attestations currently being voted on in order of nonce and
// "Observe" those who have passed the threshold. Break the loop once we see
// an attestation that has not passed the threshold
func attestationTally(ctx sdk.Context, k keeper.Keeper) {
	params := k.GetParams(ctx)
	// bridge is currently disabled, do not process attestations from Ethereum
	if !params.BridgeActive {
		return
	}

	attmap := k.GetAttestationMapping(ctx)
	// We make a slice with all the event nonces that are in the attestation mapping
	keys := make([]uint64, 0, len(attmap))
	for k := range attmap {
		keys = append(keys, k)
	}
	// Then we sort it
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
	// a slice with one or more attestations at that event nonce. There can be multiple attestations
	// at one event nonce when validators disagree about what event happened at that nonce.
	for _, nonce := range keys {
		// This iterates over all attestations at a particular event nonce.
		// They are ordered by when the first attestation at the event nonce was received.
		// This order is not important.
		for _, att := range attmap[nonce] {
			// We check if the event nonce is exactly 1 higher than the last attestation that was
			// observed. If it is not, we just move on to the next nonce. This will skip over all
			// attestations that have already been observed.
			//
			// Once we hit an event nonce that is one higher than the last observed event, we stop
			// skipping over this conditional and start calling tryAttestation (counting votes)
			// Once an attestation at a given event nonce has enough votes and becomes observed,
			// every other attestation at that nonce will be skipped, since the lastObservedEventNonce
			// will be incremented.
			//
			// Then we go to the next event nonce in the attestation mapping, if there is one. This
			// nonce will once again be one higher than the lastObservedEventNonce.
			// If there is an attestation at this event nonce which has enough votes to be observed,
			// we skip the other attestations and move on to the next nonce again.
			// If no attestation becomes observed, when we get to the next nonce, every attestation in
			// it will be skipped. The same will happen for every nonce after that.
			if nonce == uint64(k.GetLastObservedEventNonce(ctx))+1 {
				k.TryAttestation(ctx, &att)
			}
		}
	}
}

// cleanupTimedOutBatches deletes batches that have passed their expiration on Ethereum
// keep in mind several things when modifying this function
// A) unlike nonces timeouts are not monotonically increasing, meaning batch 5 can have a later timeout than batch 6
//    this means that we MUST only cleanup a single batch at a time
// B) it is possible for ethereumHeight to be zero if no events have ever occurred, make sure your code accounts for this
// C) When we compute the timeout we do our best to estimate the Ethereum block height at that very second. But what we work with
//    here is the Ethereum block height at the time of the last Deposit or Withdraw to be observed. It's very important we do not
//    project, if we do a slowdown on ethereum could cause a double spend. Instead timeouts will *only* occur after the timeout period
//    AND any deposit or withdraw has occurred to update the Ethereum block height.
func cleanupTimedOutBatches(ctx sdk.Context, k keeper.Keeper) {
	ethereumHeight := k.GetLastObservedEthereumBlockHeight(ctx).EthereumBlockHeight
	batches := k.GetOutgoingTxBatches(ctx)
	for _, batch := range batches {
		if batch.BatchTimeout < ethereumHeight {
			k.CancelOutgoingTXBatch(ctx, batch.TokenContract, batch.BatchNonce)
		}
	}
}

// cleanupTimedOutBatches deletes logic calls that have passed their expiration on Ethereum
// keep in mind several things when modifying this function
// A) unlike nonces timeouts are not monotonically increasing, meaning call 5 can have a later timeout than batch 6
//    this means that we MUST only cleanup a single call at a time
// B) it is possible for ethereumHeight to be zero if no events have ever occurred, make sure your code accounts for this
// C) When we compute the timeout we do our best to estimate the Ethereum block height at that very second. But what we work with
//    here is the Ethereum block height at the time of the last Deposit or Withdraw to be observed. It's very important we do not
//    project, if we do a slowdown on ethereum could cause a double spend. Instead timeouts will *only* occur after the timeout period
//    AND any deposit or withdraw has occurred to update the Ethereum block height.
func cleanupTimedOutLogicCalls(ctx sdk.Context, k keeper.Keeper) {
	ethereumHeight := k.GetLastObservedEthereumBlockHeight(ctx).EthereumBlockHeight
	calls := k.GetOutgoingLogicCalls(ctx)
	for _, call := range calls {
		if call.Timeout < ethereumHeight {
			k.CancelOutgoingLogicCall(ctx, call.InvalidationId, call.InvalidationNonce)
		}
	}
}

// prepValsetConfirms loads all confirmations into a hashmap indexed by validatorAddr
// reducing the lookup time dramatically and separating out the task of looking up
// the orchestrator for each validator
func prepValsetConfirms(ctx sdk.Context, k keeper.Keeper, nonce uint64) map[string]types.MsgValsetConfirm {
	confirms := k.GetValsetConfirms(ctx, nonce)
	// bytes are incomparable in go, so we convert the sdk.ValAddr bytes to a string
	ret := make(map[string]types.MsgValsetConfirm)
	for _, confirm := range confirms {
		// TODO this presents problems for delegate key rotation see issue #344
		confVal, _ := sdk.AccAddressFromBech32(confirm.Orchestrator)
		val, foundValidator := k.GetOrchestratorValidator(ctx, confVal)
		if !foundValidator {
			panic("Confirm from validator we can't identify?")
		}
		ret[val.GetOperator().String()] = confirm
	}
	return ret
}

func ValsetSlashing(ctx sdk.Context, k keeper.Keeper, params types.Params) {
	// don't slash in the beginning before there aren't even SignedValsetsWindow blocks yet
	if uint64(ctx.BlockHeight()) <= params.SignedValsetsWindow {
		return
	}

	unslashedValsets := k.GetUnSlashedValsets(ctx, params.SignedValsetsWindow)

	// unslashedValsets are sorted by nonce in ASC order
	// Question: do we need to sort each time? See if this can be epoched
	for _, vs := range unslashedValsets {
		confirms := prepValsetConfirms(ctx, k, vs.Nonce)

		// SLASH BONDED VALIDTORS who didn't attest valset request
		currentBondedSet := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
		for _, val := range currentBondedSet {
			consAddr, _ := val.GetConsAddr()
			valSigningInfo, exist := k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)

			//  Slash validator ONLY if he joined before valset is created
			if exist && uint64(valSigningInfo.StartHeight) < vs.Height {
				// Check if validator has confirmed valset or not
				_, found := confirms[val.GetOperator().String()]
				// slash validators for not confirming valsets
				if !found {
					cons, _ := val.GetConsAddr()
					k.StakingKeeper.Slash(ctx, cons, ctx.BlockHeight(), val.ConsensusPower(sdk.DefaultPowerReduction), params.SlashFractionValset)
					if !val.IsJailed() {
						k.StakingKeeper.Jail(ctx, cons)
						// Our unbonding hook SHOULD be triggered after the above jail
						// but is not when triggered by the endblocker TODO investigate why
						k.SetLastUnBondingBlockHeight(ctx, uint64(ctx.BlockHeight()))
					}

				}
			}
		}

		// SLASH UNBONDING VALIDATORS who didn't attest valset request
		blockTime := ctx.BlockTime().Add(k.StakingKeeper.GetParams(ctx).UnbondingTime)
		blockHeight := ctx.BlockHeight()
		unbondingValIterator := k.StakingKeeper.ValidatorQueueIterator(ctx, blockTime, blockHeight)
		defer unbondingValIterator.Close()

		// All unbonding validators
		for ; unbondingValIterator.Valid(); unbondingValIterator.Next() {
			unbondingValidators := k.DeserializeValidatorIterator(unbondingValIterator.Value())

			for _, valAddr := range unbondingValidators.Addresses {
				addr, err := sdk.ValAddressFromBech32(valAddr)
				if err != nil {
					panic(err)
				}
				validator, _ := k.StakingKeeper.GetValidator(ctx, sdk.ValAddress(addr))
				valConsAddr, _ := validator.GetConsAddr()
				valSigningInfo, exist := k.SlashingKeeper.GetValidatorSigningInfo(ctx, valConsAddr)

				// Only slash validators who joined after valset is created and they are unbonding and UNBOND_SLASHING_WINDOW hasn't passed
				if exist && valSigningInfo.StartHeight < int64(vs.Height) && validator.IsUnbonding() && vs.Height < uint64(validator.UnbondingHeight)+params.UnbondSlashingValsetsWindow {
					// Check if validator has confirmed valset or not
					_, found := confirms[validator.GetOperator().String()]

					// slash validators for not confirming valsets
					if !found {
						k.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), validator.ConsensusPower(sdk.DefaultPowerReduction), params.SlashFractionValset)
						if !validator.IsJailed() {
							k.StakingKeeper.Jail(ctx, valConsAddr)
							// Our unbonding hook SHOULD be triggered after the above jail
							// but is not when triggered by the endblocker TODO investigate why
							k.SetLastUnBondingBlockHeight(ctx, uint64(ctx.BlockHeight()))
						}
					}
				}
			}
		}
		// then we set the latest slashed valset  nonce
		k.SetLastSlashedValsetNonce(ctx, vs.Nonce)
	}
}

// prepBatchConfirms loads all confirmations into a hashmap indexed by validatorAddr
// reducing the lookup time dramatically and separating out the task of looking up
// the orchestrator for each validator
func prepBatchConfirms(ctx sdk.Context, k keeper.Keeper, batch *types.InternalOutgoingTxBatch) map[string]types.MsgConfirmBatch {
	confirms := k.GetBatchConfirmByNonceAndTokenContract(ctx, batch.BatchNonce, batch.TokenContract)
	// bytes are incomparable in go, so we convert the sdk.ValAddr bytes to a string (note this is NOT bech32)
	ret := make(map[string]types.MsgConfirmBatch)
	for _, confirm := range confirms {
		// TODO this presents problems for delegate key rotation see issue #344
		confVal, _ := sdk.AccAddressFromBech32(confirm.Orchestrator)
		val, foundValidator := k.GetOrchestratorValidator(ctx, confVal)
		if !foundValidator {
			panic("Confirm from validator we can't identify?")
		}
		ret[val.GetOperator().String()] = confirm
	}
	return ret
}

func BatchSlashing(ctx sdk.Context, k keeper.Keeper, params types.Params) {

	// We look through the full bonded set (the active set)
	// and we slash users who haven't signed a batch confirmation that is >15hrs in blocks old
	maxHeight := uint64(0)

	// don't slash in the beginning before there aren't even SignedBatchesWindow blocks yet
	if uint64(ctx.BlockHeight()) > params.SignedBatchesWindow {
		maxHeight = uint64(ctx.BlockHeight()) - params.SignedBatchesWindow
	} else {
		// we can't slash anyone if this window has not yet passed
		return
	}

	unslashedBatches := k.GetUnSlashedBatches(ctx, maxHeight)
	for _, batch := range unslashedBatches {

		// SLASH BONDED VALIDTORS who didn't attest batch requests
		currentBondedSet := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
		confirms := prepBatchConfirms(ctx, k, batch)
		for _, val := range currentBondedSet {
			// Don't slash validators who joined after batch is created
			consAddr, _ := val.GetConsAddr()
			valSigningInfo, exist := k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
			if exist && valSigningInfo.StartHeight > int64(batch.Block) {
				continue
			}

			_, found := confirms[val.GetOperator().String()]
			if !found {
				cons, _ := val.GetConsAddr()
				k.StakingKeeper.Slash(ctx, cons, ctx.BlockHeight(), val.ConsensusPower(sdk.DefaultPowerReduction), params.SlashFractionBatch)
				if !val.IsJailed() {
					k.StakingKeeper.Jail(ctx, cons)
					// Our unbonding hook SHOULD be triggered after the above jail
					// but is not when triggered by the endblocker TODO investigate why
					k.SetLastUnBondingBlockHeight(ctx, uint64(ctx.BlockHeight()))
				}
			}
		}
		// then we set the latest slashed batch block
		k.SetLastSlashedBatchBlock(ctx, batch.Block)
	}
}

// prepLogicCallConfirms loads all confirmations into a hashmap indexed by validatorAddr
// reducing the lookup time dramatically and separating out the task of looking up
// the orchestrator for each validator
func prepLogicCallConfirms(ctx sdk.Context, k keeper.Keeper, call types.OutgoingLogicCall) map[string]*types.MsgConfirmLogicCall {
	confirms := k.GetLogicConfirmByInvalidationIDAndNonce(ctx, call.InvalidationId, call.InvalidationNonce)
	// bytes are incomparable in go, so we convert the sdk.ValAddr bytes to a string (note this is NOT bech32)
	ret := make(map[string]*types.MsgConfirmLogicCall)
	for _, confirm := range confirms {
		// TODO this presents problems for delegate key rotation see issue #344
		confVal, _ := sdk.AccAddressFromBech32(confirm.Orchestrator)
		val, foundValidator := k.GetOrchestratorValidator(ctx, confVal)
		if !foundValidator {
			panic("Confirm from validator we can't identify?")
		}
		ret[val.GetOperator().String()] = &confirm
	}
	return ret
}

func LogicCallSlashing(ctx sdk.Context, k keeper.Keeper, params types.Params) {

	// We look through the full bonded set (the active set)
	// and we slash users who haven't signed a batch confirmation that is >15hrs in blocks old
	maxHeight := uint64(0)

	// don't slash in the beginning before there aren't even SignedBatchesWindow blocks yet
	if uint64(ctx.BlockHeight()) > params.SignedLogicCallsWindow {
		maxHeight = uint64(ctx.BlockHeight()) - params.SignedLogicCallsWindow
	} else {
		// we can't slash anyone if this window has not yet passed
		return
	}

	unslashedLogicCalls := k.GetUnSlashedLogicCalls(ctx, maxHeight)
	for _, call := range unslashedLogicCalls {

		// SLASH BONDED VALIDTORS who didn't attest batch requests
		currentBondedSet := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
		confirms := prepLogicCallConfirms(ctx, k, call)
		for _, val := range currentBondedSet {
			// Don't slash validators who joined after batch is created
			consAddr, _ := val.GetConsAddr()
			valSigningInfo, exist := k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
			if exist && valSigningInfo.StartHeight > int64(call.Block) {
				continue
			}

			_, found := confirms[val.GetOperator().String()]
			if !found {
				cons, _ := val.GetConsAddr()
				k.StakingKeeper.Slash(ctx, cons, ctx.BlockHeight(), val.ConsensusPower(sdk.DefaultPowerReduction), params.SlashFractionLogicCall)
				if !val.IsJailed() {
					k.StakingKeeper.Jail(ctx, cons)
					// Our unbonding hook SHOULD be triggered after the above jail
					// but is not when triggered by the endblocker TODO investigate why
					k.SetLastUnBondingBlockHeight(ctx, uint64(ctx.BlockHeight()))
				}
			}
		}
		// then we set the latest slashed logic call block
		k.SetLastSlashedLogicCallBlock(ctx, call.Block)
	}
}

// Iterate over all attestations currently being voted on in order of nonce
// and prune those that are older than the current nonce and no longer have any
// use. This could be combined with create attestation and save some computation
// but (A) pruning keeps the iteration small in the first place and (B) there is
// already enough nuance in the other handler that it's best not to complicate it further
func pruneAttestations(ctx sdk.Context, k keeper.Keeper) {
	attmap := k.GetAttestationMapping(ctx)
	// We make a slice with all the event nonces that are in the attestation mapping
	keys := make([]uint64, 0, len(attmap))
	for k := range attmap {
		keys = append(keys, k)
	}
	// Then we sort it
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// we delete all attestations earlier than the current event nonce
	// minus some buffer value. This buffer value is purely to allow
	// frontends and other UI components to view recent oracle history
	const eventsToKeep = 1000
	lastNonce := uint64(k.GetLastObservedEventNonce(ctx))
	var cutoff uint64
	if lastNonce <= eventsToKeep {
		return
	} else {
		cutoff = lastNonce - eventsToKeep
	}

	// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
	// a slice with one or more attestations at that event nonce. There can be multiple attestations
	// at one event nonce when validators disagree about what event happened at that nonce.
	for _, nonce := range keys {
		// This iterates over all attestations at a particular event nonce.
		// They are ordered by when the first attestation at the event nonce was received.
		// This order is not important.
		for _, att := range attmap[nonce] {
			// delete all before the cutoff
			if nonce < cutoff {
				k.DeleteAttestation(ctx, att)
			}
		}
	}
}

// Iterate over all attestations currently being voted on in order of nonce
// and prune those that are older than nonceCutoff
func pruneAttestationsAfterNonce(ctx sdk.Context, k keeper.Keeper, nonceCutoff uint64) {
	// Decide on the most recent nonce we can actually roll back to
	lastObserved := k.GetLastObservedEventNonce(ctx)
	if nonceCutoff < lastObserved {
		ctx.Logger().Error("Attempted to reset to a nonce before the last \"observed\" event, which is not allowed", "lastObserved", lastObserved, "nonce", nonceCutoff)
		return
	}

	// Get relevant event nonces
	attmap := k.GetAttestationMapping(ctx)
	keys := make([]uint64, 0, len(attmap))
	for k := range attmap {
		keys = append(keys, k)
	}
	// Sort the nonces for iteration
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// Discover all affected validators whose LastEventNonce must be reset to nonceCutoff

	numValidators := len(k.StakingKeeper.GetBondedValidatorsByPower(ctx))
	// void and setMember are necessary for sets to work
	type void struct{}
	var setMember void
	// Initialize a Set of validators
	affectedValidatorsSet := make(map[string]void, numValidators)

	// Delete all reverted attestations, keeping track of the validators who attested to any of them
	for _, nonce := range keys {
		for _, att := range attmap[nonce] {
			// we delete all attestations earlier than the cutoff event nonce
			if nonce > nonceCutoff {
				ctx.Logger().Info(fmt.Sprintf("Deleting attestation at height %v", att.Height))
				for _, vote := range att.Votes {
					if _, ok := affectedValidatorsSet[vote]; !ok { // if set does not contain vote
						affectedValidatorsSet[vote] = setMember // add key to set
					}
				}

				k.DeleteAttestation(ctx, att)
			}
		}
	}

	// Reset the last event nonce for all validators affected by history deletion
	for vote := range affectedValidatorsSet {
		val, err := sdk.ValAddressFromBech32(vote)
		if err != nil {
			panic(sdkerrors.Wrap(err, "invalid validator address affected by bridge reset"))
		}
		valLastNonce := k.GetLastEventNonceByValidator(ctx, val)
		if valLastNonce > nonceCutoff {
			ctx.Logger().Info("Resetting validator's last event nonce due to bridge unhalt", "validator", vote, "lastEventNonce", valLastNonce, "resetNonce", nonceCutoff)
			k.SetLastEventNonceByValidator(ctx, val, nonceCutoff)
		}
	}
}
