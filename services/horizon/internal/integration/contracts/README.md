### Contract test fixture source code
#### anytime contract code changes, follow these steps to rebuild the test wasm fixtures:

1. compile from source
First install latest rust toolchain:
https://www.rust-lang.org/tools/install 

and update the ./services/horizon/internal/integration/contracts/Cargo.toml to have latest git refs for
soroban-sdk and soroban-auth packages.

then compile the contract source code to wasm and copy to testdata
```
services/horizon/internal/integration/contracts $ cargo build --target wasm32-unknown-unknown --release
services/horizon/internal/integration/contracts $ cp target/wasm32-unknown-unknown/release/*.wasm ../testdata/
```

recompile the soroban_token_spec.wasm by compiling the rs-soroban-sdk source code from the sae git ref to wasm and copy it to contracts folder
```
rs-soroban-sdk $ cargo build --target wasm32-unknown-unknown --release
rs-soroban-sdk $ cp target/wasm32-unknown-unknown/release/soroban_token_spec.wasm go/services/horizon/internal/integration/contracts
```

2. existing integeration tests refer to .wasm files from that `testdata` directory location.

