### Contract test fixture source code
#### anytime contracct code changes, follow these steps to rebuild the test wasm fixtures:

1. compile from source
First install latest rust toolchain:
https://www.rust-lang.org/tools/install 

and update the ./services/horizon/internal/integration/contracts/Cargo.toml to have latest git refs for
soroban-sdk and soroban-auth packages.

then compile the contract source code to wasm
```
services/horizon/internal/integration/contracts $ cargo build --target wasm32-unknown-unknown --release
```

2. copy the resulting .wasm files in  to ./services/horizon/internal/integration/testdata/
3. existing integeration tests refer to .wasm files from that `testdata` directory location.

