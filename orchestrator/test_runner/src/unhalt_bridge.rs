use crate::get_fee;
use crate::happy_path::test_erc20_deposit;
use crate::one_eth;
use crate::utils::*;
use crate::ADDRESS_PREFIX;
use crate::MINER_ADDRESS;
use crate::MINER_PRIVATE_KEY;
use crate::OPERATION_TIMEOUT;
use crate::STAKING_TOKEN;
use crate::STARTING_STAKE_PER_VALIDATOR;
use crate::TOTAL_TIMEOUT;
use bytes::BytesMut;
use clarity::abi::encode_call;
use clarity::abi::Token;
use clarity::{Address as EthAddress, Uint256};
use cosmos_gravity::query::get_attestations;
use cosmos_gravity::query::get_last_event_nonce_for_validator;
use cosmos_gravity::send::MEMO;
use deep_space::address::Address as CosmosAddress;
use deep_space::coin::Coin;
use deep_space::error::CosmosGrpcError;
use deep_space::utils::encode_any;
use deep_space::Contact;
use deep_space::Fee;
use deep_space::Msg;
use ethereum_gravity::send_to_cosmos::send_to_cosmos;
use ethereum_gravity::utils::downcast_uint256;
use futures::future::join;
use gravity_proto::cosmos_sdk_proto::cosmos::gov::v1beta1::VoteOption;
use gravity_proto::cosmos_sdk_proto::cosmos::params::v1beta1::ParamChange;
use gravity_proto::cosmos_sdk_proto::cosmos::params::v1beta1::ParameterChangeProposal;
use gravity_proto::cosmos_sdk_proto::cosmos::staking::v1beta1::QueryValidatorsRequest;
use gravity_proto::cosmos_sdk_proto::cosmos::tx::v1beta1::BroadcastMode;
use gravity_proto::gravity::query_client::QueryClient as GravityQueryClient;
use gravity_proto::gravity::MsgSendToCosmosClaim;
use prost::Message;
use std::str::FromStr;
use std::thread::sleep;
use std::time::Duration;
use std::time::Instant;
use tonic::transport::Channel;
use web30::client::Web3;
use web30::jsonrpc::error::Web3Error;
use web30::types::SendTxOption;
use tokio::time::sleep as delay_for;

pub async fn unhalt_bridge_test(
    web30: &Web3,
    grpc_client: GravityQueryClient<Channel>,
    contact: &Contact,
    keys: Vec<ValidatorKeys>,
    gravity_address: EthAddress,
    erc20_address: EthAddress,
    validator_out: bool,
) {
    let prefix = contact.get_prefix();
    let mut grpc_client = grpc_client;
    let no_relay_market_config = create_default_test_config();
    let bridge_user = get_user_key();
    info!("Sending bridge user some tokens");
    send_one_eth(bridge_user.eth_address, web30).await;
    send_erc20_bulk(
        one_eth() * 10u64.into(),
        erc20_address,
        &[bridge_user.eth_address],
        web30,
    )
    .await;

    start_orchestrators(
        keys.clone(),
        gravity_address,
        validator_out,
        no_relay_market_config.clone(),
    )
    .await;

    info!("Redistribute stake!");
    redistribute_stake(&keys, contact, &prefix).await;

    info!("Test bridge before false claims!");
    // Test a deposit to increment the event nonce before false claims happen
    let success = test_erc20_deposit(
        web30,
        contact,
        &mut grpc_client,
        bridge_user.cosmos_address,
        gravity_address,
        erc20_address,
        10_000_000_000_000_000u64.into(),
        None,
        None,
    )
    .await;
    if !success {
        panic!("Failed to bridge ERC20!")
    }

    let fee = Fee {
        amount: vec![get_fee()],
        gas_limit: 500_000_000u64,
        granter: None,
        payer: None,
    };
    // These are the nonces each validator is aware of before false claims are submitted
    let (init_val1_nonce, init_val2_nonce, init_val3_nonce) =
        get_nonces(&mut grpc_client, &keys, &prefix).await;
    // At this point we can use any nonce since all the validators have the same state
    let initial_valid_nonce = init_val1_nonce;
    info!(
        "initial_nonce: {} init_val1_nonce: {} init_val2_nonce: {} init_val3_nonce: {}",
        initial_valid_nonce, init_val1_nonce, init_val2_nonce, init_val3_nonce,
    );

    // All nonces should be the same right now
    assert!(
        init_val1_nonce == init_val2_nonce && init_val2_nonce == init_val3_nonce,
        "The initial nonces differed!"
    );

    let initial_height =
        downcast_uint256(web30.eth_get_latest_block().await.unwrap().number).unwrap();

    info!("Two validators submitting false claims!");
    submit_false_claims(
        &keys,
        initial_valid_nonce + 1,
        initial_height + 1,
        &bridge_user,
        &prefix,
        erc20_address,
        contact,
        &fee,
    )
    .await;

    info!("Getting latest nonce after false claims for each validator");
    let (val1_nonce, val2_nonce, val3_nonce) = get_nonces(&mut grpc_client, &keys, &prefix).await;
    info!(
        "initial_nonce: {} val1_nonce: {} val2_nonce: {} val3_nonce: {}",
        initial_valid_nonce, val1_nonce, val2_nonce, val3_nonce,
    );

    // val2_nonce and val3_nonce should be initial + 1 but val1_nonce should not
    assert!(
        val2_nonce == initial_valid_nonce + 1 && val2_nonce == val3_nonce,
        "The false claims validators do not have updated nonces"
    );
    assert_eq!(
        val1_nonce, initial_valid_nonce,
        "The honest validator should not have an updated nonce!"
    );

    info!("Checking that bridge is halted!");

    let halted_bridge_amt = Uint256::from_str("100_000_000_000_000_000").unwrap();
    // Attempt transaction on halted bridge
    let success = test_erc20_deposit(
        web30,
        contact,
        &mut grpc_client,
        bridge_user.cosmos_address,
        gravity_address,
        erc20_address,
        halted_bridge_amt.clone(),
        Some(Duration::from_secs(30)),
        None,
    )
    .await;
    if success {
        panic!("bridge not halted!")
    }

    sleep(Duration::from_secs(30));

    info!("Getting latest nonce after bridge halt check");
    let (val1_nonce, val2_nonce, val3_nonce) = get_nonces(&mut grpc_client, &keys, &prefix).await;
    info!(
        "initial_nonce: {} val1_nonce: {} val2_nonce: {} val3_nonce: {}",
        initial_valid_nonce, val1_nonce, val2_nonce, val3_nonce,
    );

    info!(
        "Bridge successfully locked, starting governance vote to reset nonce to {}.",
        initial_valid_nonce
    );

    info!("Preparing governance proposal!!");
    // Unhalt the bridge
    let deposit = Coin {
        denom: STAKING_TOKEN.to_string(),
        amount: 1_000_000_000u64.into(),
    };
    let _ = submit_and_pass_gov_proposal(initial_valid_nonce, &deposit, contact, &keys)
        .await
        .expect("Governance proposal failed");
    let start = Instant::now();
    loop {
        let (new_val1_nonce, new_val2_nonce, new_val3_nonce) =
            get_nonces(&mut grpc_client, &keys, &prefix).await;
        if new_val1_nonce == val1_nonce
            && new_val2_nonce == val2_nonce
            && new_val3_nonce == val3_nonce
        {
            info!(
                "Nonces have not changed: {}=>{}, {}=>{}, {}=>{}, sleeping before retry",
                new_val1_nonce, val1_nonce, new_val2_nonce, val2_nonce, new_val3_nonce, val3_nonce
            );
            if Instant::now()
                .checked_duration_since(start)
                .unwrap()
                .gt(&Duration::from_secs(10 * 60))
            {
                panic!("10 minutes have elapsed trying to get the validator last nonces to change for val1 and val2!");
            }
            sleep(Duration::from_secs(10));
            continue;
        } else {
            info!(
                "New nonces: {}=>{}, {}=>{}, {}=>{}, sleeping before retry",
                val1_nonce, new_val1_nonce, val2_nonce, new_val2_nonce, val3_nonce, new_val3_nonce
            );
            break;
        }
    }
    let (val1_nonce, val2_nonce, val3_nonce) = get_nonces(&mut grpc_client, &keys, &prefix).await;
    assert!(
        val1_nonce == val2_nonce && val2_nonce == val3_nonce && val1_nonce == initial_valid_nonce,
        "The post-reset nonces are not equal to the initial nonce",
    );

    // After the governance proposal the resync will happen on the next loop.
    // Wait for a bit to replay stuck transactions
    info!("Sleeping so that resync can complete!");
    sleep(Duration::from_secs(30));

    info!("Observing attestations before bridging asset to cosmos!");
    observe_sends_to_cosmos(&grpc_client, true).await;

    let fixed_bridge_amt = Uint256::from_str("50_000_000_000_000_000").unwrap();
    info!("Attempting to resend now that the bridge should be fixed");
    let res = test_erc20_deposit(
        web30,
        contact,
        &mut grpc_client,
        bridge_user.cosmos_address,
        gravity_address,
        erc20_address,
        fixed_bridge_amt.clone(),
        None,
        Some(halted_bridge_amt.clone() + fixed_bridge_amt.clone()),
    )
    .await;
    match res {
        true => info!("Successfully bridged asset!"),
        false => panic!("Failed to bridge ERC20!"),
    }

    info!("res is {:?}", res);
}

async fn redistribute_stake(keys: &[ValidatorKeys], contact: &Contact, _prefix: &str) {
    let validators = contact
        .get_validators_list(QueryValidatorsRequest::default())
        .await
        .unwrap();
    for validator in validators.validators {
        info!(
            "Validator {} has {} tokens",
            validator.operator_address, validator.tokens
        );
    }
    // let num_validators = keys.len();
    // let controlling_stake = (STARTING_STAKE_PER_VALIDATOR * (num_validators as u128)) as f64 * 0.45;
    // let stake_needed: f64 = controlling_stake - STARTING_STAKE_PER_VALIDATOR as f64;
    // info!(
    //     "Controlling stake is {} stake needed is {}",
    //     controlling_stake, stake_needed
    // );
    let delegate_address = keys[0]
        .validator_key
        .to_address(&format!("{}valoper", *ADDRESS_PREFIX))
        .unwrap();
    // Want validator 0 to have 50% voting power, and they start with 33.3...%
    // 1/2 = 1/3 + 2X where 2X is the delegate amount from the other two validators
    // X = (1/2 - 1/3) / 2 = ((3 * sspv / 2) - sspv) / 2
    // (3s/2 - 2s/2) / 2 = (s/2)/2 = s/4
    let delegate_amount = (STARTING_STAKE_PER_VALIDATOR as f64) / 4.0;
    for (i, k) in keys.iter().enumerate() {
        if i == 0 {
            // Don't delegate from validator 1 to itself
            continue;
        }
        let amount = Coin {
            denom: (*STAKING_TOKEN).to_string(),
            amount: Uint256::from_str(delegate_amount.to_string().as_str()).unwrap(),
        };

        let res = contact
            .delegate_to_validator(
                delegate_address,
                amount,
                get_fee(),
                k.validator_key,
                Some(TOTAL_TIMEOUT),
            )
            .await;
        info!(
            "Delegated {} from validator {} to validator 1, response {:?}",
            delegate_amount,
            &(i + 1),
            res,
        )
    }
    let validators = contact
        .get_validators_list(QueryValidatorsRequest::default())
        .await
        .unwrap();
    for validator in validators.validators {
        info!(
            "Validator {} has {} tokens",
            validator.operator_address, validator.tokens
        );
    }
}

async fn submit_and_pass_gov_proposal(
    nonce: u64,
    deposit: &Coin,
    contact: &Contact,
    keys: &[ValidatorKeys],
) -> Result<bool, CosmosGrpcError> {
    let mut params_to_change: Vec<ParamChange> = Vec::new();
    // this does not
    let reset_state = ParamChange {
        subspace: "gravity".to_string(),
        key: "ResetBridgeState".to_string(),
        value: serde_json::to_string(&true).unwrap(),
    };
    info!("Submit and pass gov proposal: nonce is {}", nonce);
    params_to_change.push(reset_state);
    let reset_nonce = ParamChange {
        subspace: "gravity".to_string(),
        key: "ResetBridgeNonce".to_string(),
        value: format!("\"{}\"", nonce),
    };
    params_to_change.push(reset_nonce);
    let proposal = ParameterChangeProposal {
        title: "Reset Bridge State".to_string(),
        description: "Test resetting bridge state to before things were messed up".to_string(),
        changes: params_to_change,
    };
    let any = encode_any(
        proposal,
        "/cosmos.params.v1beta1.ParameterChangeProposal".to_string(),
    );

    let res = contact
        .create_gov_proposal(
            any.clone(),
            deposit.clone(),
            get_fee(),
            keys[0].validator_key,
            Some(TOTAL_TIMEOUT),
        )
        .await;
    info!("Proposal response is {:?}", res);
    if res.is_err() {
        return Err(res.unwrap_err());
    }

    // Vote yes on all proposals with all validators
    let proposals = contact
        .get_governance_proposals_in_voting_period()
        .await
        .unwrap();
    info!("Found proposals: {:?}", proposals.proposals);
    for proposal in proposals.proposals {
        for key in keys.iter() {
            info!("Voting yes on governance proposal");
            let res = contact
                .vote_on_gov_proposal(
                    proposal.proposal_id,
                    VoteOption::Yes,
                    get_fee(),
                    key.validator_key,
                    Some(TOTAL_TIMEOUT),
                )
                .await
                .unwrap();
            contact.wait_for_tx(res, TOTAL_TIMEOUT).await.unwrap();
        }
    }
    Ok(true)
}

async fn get_nonces(
    grpc_client: &mut GravityQueryClient<Channel>,
    keys: &[ValidatorKeys],
    prefix: &str,
) -> (u64, u64, u64) {
    let nonce1 = get_last_event_nonce_for_validator(
        grpc_client,
        keys[0].orch_key.to_address(prefix).unwrap(),
        prefix.to_string(),
    )
    .await
    .unwrap();
    let nonce2 = get_last_event_nonce_for_validator(
        grpc_client,
        keys[1].orch_key.to_address(prefix).unwrap(),
        prefix.to_string(),
    )
    .await
    .unwrap();
    let nonce3 = get_last_event_nonce_for_validator(
        grpc_client,
        keys[2].orch_key.to_address(prefix).unwrap(),
        prefix.to_string(),
    )
    .await
    .unwrap();
    (nonce1, nonce2, nonce3)
}

fn create_claim(
    nonce: u64,
    height: u64,
    token_contract: &EthAddress,
    initiator_eth_addr: &EthAddress,
    receiver_cosmos_addr: &CosmosAddress,
    orchestrator_addr: &CosmosAddress,
) -> MsgSendToCosmosClaim {
    MsgSendToCosmosClaim {
        event_nonce: nonce,
        block_height: height,
        token_contract: token_contract.to_string(),
        amount: one_eth().to_string(),
        cosmos_receiver: receiver_cosmos_addr.to_string(),
        ethereum_sender: initiator_eth_addr.to_string(),
        orchestrator: orchestrator_addr.to_string(),
    }
}
async fn create_message(
    claim: &MsgSendToCosmosClaim,
    contact: &Contact,
    key: &ValidatorKeys,
    fee: &Fee,
    prefix: &str,
) -> Vec<u8> {
    let msg_url = "/gravity.v1.MsgSendToCosmosClaim";

    let msg = Msg::new(msg_url, claim.clone());
    let args = contact
        .get_message_args(key.orch_key.to_address(prefix).unwrap(), fee.clone())
        .await
        .unwrap();
    let msgs = vec![msg];
    key.orch_key.sign_std_msg(&msgs, args, MEMO).unwrap()
}

#[allow(clippy::too_many_arguments)]
async fn submit_false_claims(
    keys: &[ValidatorKeys],
    nonce: u64,
    height: u64,
    bridge_user: &BridgeUserKey,
    prefix: &str,
    erc20_address: EthAddress,
    contact: &Contact,
    fee: &Fee,
) {
    let mut fut_1 = None;
    let mut fut_2 = None;
    for (i, k) in keys.iter().enumerate() {
        if i == 0 {
            info!("Skipping validator 0 for false claims");
            continue;
        }
        let claim = create_claim(
            nonce,
            height,
            &erc20_address,
            &bridge_user.eth_address,
            &bridge_user.cosmos_address,
            &k.orch_key.to_address(prefix).unwrap(),
        );
        info!("Oracle number {} submitting false deposit {:?}", i, &claim);
        let msg_bytes = create_message(&claim, contact, k, fee, prefix).await;

        let response = contact
            .send_transaction(msg_bytes, BroadcastMode::Sync)
            .await
            .unwrap();
        let fut = contact.wait_for_tx(response, OPERATION_TIMEOUT);

        if i == 1 {
            fut_1 = Some(fut);
        } else {
            fut_2 = Some(fut);
        }
    }
    let join_res = join(fut_1.unwrap(), fut_2.unwrap()).await;
    match join_res.0 {
        Ok(success) => {
            info!("Received success from claim_1's wait_for_tx {:?}", success);
        }
        Err(err) => {
            info!("Received an error from claim_1 {}", err);
        }
    }
    match join_res.1 {
        Ok(success) => {
            info!("Received success from claim_2's wait_for_tx {:?}", success);
        }
        Err(err) => {
            info!("Received an error from claim_2 {}", err);
        }
    }
}

#[allow(clippy::too_many_arguments)]
async fn bridge_asset(
    web30: &Web3,
    contact: &Contact,
    grpc_client: &mut GravityQueryClient<Channel>,
    dest: CosmosAddress,
    gravity_address: EthAddress,
    erc20_address: EthAddress,
    amount: Uint256,
    timeout: Option<Duration>,
) -> Result<bool, Web3Error> {
    let start_coin = check_cosmos_balance("gravity", dest, contact).await;
    const SEND_TO_COSMOS_GAS_LIMIT: u128 = 100_000;

    info!(
        "Sending to Cosmos from {} to {} with amount {}",
        *MINER_ADDRESS, dest, amount
    );

    let mut options: Vec<SendTxOption> = vec![];
    let tx_id = send_to_cosmos(
        erc20_address,
        gravity_address,
        amount.clone(),
        dest.clone(),
        *MINER_PRIVATE_KEY,
        timeout.clone(),
        web30,
        vec![],
    ).await.unwrap();

    if let Some(duration) = timeout {
        web30
            .wait_for_transaction(tx_id.clone(), duration, None)
            .await?;
    }

    delay_for(Duration::from_secs(10)).await;
    observe_sends_to_cosmos(&grpc_client, true).await;


    let start = Instant::now();
    let duration = match timeout {
        Some(w) => w,
        None => TOTAL_TIMEOUT,
    };
    while Instant::now() - start < duration {
        match (
            start_coin.clone(),
            check_cosmos_balance("gravity", dest, contact).await,
        ) {
            (Some(start_coin), Some(end_coin)) => {
                if start_coin.amount + amount.clone() == end_coin.amount
                    && start_coin.denom == end_coin.denom
                {
                    info!(
                        "Successfully bridged ERC20 {}{} to Cosmos! Balance is now {}{}",
                        amount, start_coin.denom, end_coin.amount, end_coin.denom
                    );
                    return Ok(true);
                }
            }
            (None, Some(end_coin)) => {
                if amount == end_coin.amount {
                    info!(
                        "Successfully bridged ERC20 {}{} to Cosmos! Balance is now {}{}",
                        amount, end_coin.denom, end_coin.amount, end_coin.denom
                    );
                    return Ok(true);
                } else {
                    return Ok(false);
                }
            }
            _ => {}
        }
        info!("Waiting for ERC20 deposit");
        //observe_sends_to_cosmos(&mut grpc_client.clone(), false).await;
        contact.wait_for_next_block(TOTAL_TIMEOUT).await.unwrap();
    }
    Ok(false)
}

async fn observe_sends_to_cosmos(
    grpc_client: &GravityQueryClient<Channel>,
    print_others: bool,
) {
    let mut grpc_client = &mut grpc_client.clone();
    let attestations = get_attestations(&mut grpc_client, None)
        .await
        .expect("Something happened while getting attestations after delegating to validator");
    for (i, attestation) in attestations.into_iter().enumerate() {
        let claim = attestation.clone().claim.unwrap();
        if  print_others && claim.type_url != "/gravity.v1.MsgSendToCosmosClaim"{
            info!("attestation {}: {:?}", i, &attestation);
            continue;
        }
        let mut buf = BytesMut::with_capacity(claim.value.len());
        buf.extend_from_slice(&claim.value);

        // Here we use the `T` type to decode whatever type of message this attestation holds
        // for use in the `f` function
        let decoded = MsgSendToCosmosClaim::decode(buf);

        info!(
            "attestation {}: votes {:?}\n decoded{:?}",
            i, &attestation.votes, decoded
        );
    }

}