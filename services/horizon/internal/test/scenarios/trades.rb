run_recipe File.dirname(__FILE__) + "/_common_accounts.rb"

use_manual_close

create_account :scott,  :master, 100
create_account :bartek, :master, 100
create_account :usd_gateway, :master, 100
create_account :eur_gateway, :master, 100

close_ledger

trust :scott,  :usd_gateway, "USD"
trust :bartek, :usd_gateway, "USD"
trust :scott,  :eur_gateway, "EUR"
trust :bartek, :eur_gateway, "EUR"

close_ledger

payment :usd_gateway, :scott,  ["USD", :usd_gateway, 500]

close_ledger

payment :eur_gateway, :bartek, ["EUR", :eur_gateway, 500]

close_ledger

# Offers below should be applied in separate ledgers. Transactions withing a single
# ledger are applied in a random order, this makes synthetic offer IDs in `/trades`
# endpoint to be different when tests are regenerated.
offer :bartek, {buy:["USD", :usd_gateway], with:["EUR", :eur_gateway]}, 100, 1.0
close_ledger
offer :bartek, {buy:["USD", :usd_gateway], with:["EUR", :eur_gateway]}, 100, 0.9
close_ledger
offer :bartek, {buy:["USD", :usd_gateway], with:["EUR", :eur_gateway]}, 100, 0.8
close_ledger
offer :scott, {sell:["USD", :usd_gateway], for:["EUR", :eur_gateway]}, 50, 1.0
close_ledger
offer :scott, {sell:["USD", :usd_gateway], for: :native}, 50, 1.0
close_ledger
offer :scott, {sell:["USD", :usd_gateway], for:["EUR", :eur_gateway]}, 20, 1.0
