### Contract test fixture source code

The existing integeration tests refer to .wasm files from the `testdata/` directory location.

#### Any time contract code changes, follow these steps to rebuild the test WASM fixtures:

1. First install latest rust toolchain:
https://www.rust-lang.org/tools/install

2. Update the [`Cargo.toml file`](./Cargo.toml) to have latest git refs to
[`rs-soroban-sdk`](https://github.com/stellar/rs-soroban-sdk) for the `soroban-sdk` and `soroban-auth` dependencies.

3. Recompile the `soroban_token_spec.wasm` by compiling the `rs-soroban-sdk` source code from the same git ref to WASM and copy it to the `contracts/` folder:

```bash
git clone git@github.com:stellar/rs-soroban-sdk.git
cd rs-soroban-sdk/soroban-token-spec
git checkout <insert ref here>
cargo build --target wasm32-unknown-unknown --release
cp ../target/wasm32-unknown-unknown/release/soroban_token_spec.wasm \
   $MONOREPO/services/horizon/internal/integration/contracts
```

(where `$MONOREPO` is where you have `stellar/go` checked out)

4. Compile the contract source code to WASM and copy it to `testdata/`:

```bash
cd ./services/horizon/internal/integration/contracts
cargo build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/*.wasm ../testdata/
```
