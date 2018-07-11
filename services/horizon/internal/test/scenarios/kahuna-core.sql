running recipe

Error! (ParamContractError): Contract violation for argument 1 of 1:
        Expected: Stellar::KeyPair,
        Actual: :bumper
        Value guarded in: StellarCoreCommander::Transactor::next_sequence
        With Contract: Stellar::KeyPair => Num
        At: /Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bundler/gems/stellar_core_commander-b54494191507/lib/stellar_core_commander/transactor.rb:343 

/Users/nullstyle/go/src/github.com/stellar/go/services/horizon/internal/test/scenarios/kahuna.rb:375:in `run_recipe'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bundler/gems/stellar_core_commander-b54494191507/lib/stellar_core_commander/transactor.rb:53:in `instance_eval'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bundler/gems/stellar_core_commander-b54494191507/lib/stellar_core_commander/transactor.rb:53:in `run_recipe'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bundler/gems/stellar_core_commander-b54494191507/bin/scc:89:in `run'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bundler/gems/stellar_core_commander-b54494191507/bin/scc:152:in `<top (required)>'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bin/scc:23:in `load'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/bin/scc:23:in `<top (required)>'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli/exec.rb:74:in `load'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli/exec.rb:74:in `kernel_load'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli/exec.rb:28:in `run'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli.rb:424:in `exec'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/vendor/thor/lib/thor/command.rb:27:in `run'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/vendor/thor/lib/thor/invocation.rb:126:in `invoke_command'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/vendor/thor/lib/thor.rb:387:in `dispatch'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli.rb:27:in `dispatch'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/vendor/thor/lib/thor/base.rb:466:in `start'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/cli.rb:18:in `start'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/exe/bundle:30:in `block in <top (required)>'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/lib/bundler/friendly_errors.rb:124:in `with_friendly_errors'
/Users/nullstyle/.rbenv/versions/2.5.1/lib/ruby/gems/2.5.0/gems/bundler-1.16.2/exe/bundle:22:in `<top (required)>'
/Users/nullstyle/.rbenv/versions/2.5.1/bin/bundle:23:in `load'
/Users/nullstyle/.rbenv/versions/2.5.1/bin/bundle:23:in `<main>'

