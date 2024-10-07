#![no_std]
use soroban_sdk::{contract, contractimpl, Env, Symbol};

#[contract]
pub struct Contract;

#[contractimpl]
impl Contract {
    pub fn __constructor(env: Env, key: Symbol) {
        env.storage().persistent().set(&key, &1_u32);
        env.storage().instance().set(&key, &2_u32);
        env.storage().temporary().set(&key, &3_u32);
    }

    pub fn get_data(env: Env, key: Symbol) -> u32 {
        env.storage().persistent().get::<_, u32>(&key).unwrap()
            + env.storage().instance().get::<_, u32>(&key).unwrap()
            + env.storage().temporary().get::<_, u32>(&key).unwrap()
    }
}
