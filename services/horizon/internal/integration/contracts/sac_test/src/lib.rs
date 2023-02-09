#![no_std]
use soroban_sdk::{
    contractimpl, contracttype, Address, BytesN, Env,
};

mod token {
    soroban_sdk::contractimport!(
        file = "../soroban_token_spec.wasm"
    );
}

#[contracttype]
pub enum DataKey {
    Token,
}

fn get_token(e: &Env) -> BytesN<32> {
    e.storage().get_unchecked(&DataKey::Token).unwrap()
}

pub struct SACTest;

#[contractimpl]
impl SACTest {

    pub fn init(e: Env, contract: BytesN<32>) {
        e.storage().set(&DataKey::Token, &contract);
    }

    pub fn get_token(e: Env) -> BytesN<32> {
        get_token(&e)
    }

    pub fn burn_self(env: Env, amount: i128) {
        let client = token::Client::new(&env, &get_token(&env));
        client.burn(&env.current_contract_address(), &amount);
    }

    pub fn xfer(env: Env, to: Address, amount: i128) {
        let client = token::Client::new(&env, &get_token(&env));
        client.xfer(&env.current_contract_address(), &to, &amount);
    }
}

#[test]
fn test() {
    use soroban_sdk::testutils::Address as _;

    let env = Env::default();
    let admin = Address::random(&env);
    let token_contract_id = env.register_stellar_asset_contract(admin.clone());

    let contract_id = env.register_contract(None, SACTest);
    let contract = SACTestClient::new(&env, &contract_id);
    let contract_address = Address::from_contract_id(&env, &contract_id);
    contract.init(&token_contract_id);

    let token = token::Client::new(&env, &contract.get_token());
    assert_eq!(token.decimals(), 7);
    
    token.mint(&admin, &contract_address, &1000);

    contract.burn_self(&400);
    assert_eq!(token.balance(&contract_address), 600);

    let user = Address::random(&env);
    contract.xfer(&user, &100);
    assert_eq!(token.balance(&contract_address), 500);
    assert_eq!(token.balance(&user), 100);
}
