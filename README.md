# Harmony Tx Sender
Harmony tx sender is a tool to bulk send transactions on Harmony's blockchain.

## Installation

Until I write a wrapper script that downloads the binary, do this for now:

```
git clone https://github.com/SebastianJ/harmony-tx-sender.git && cd harmony-tx-sender
rm -rf harmony-tx-sender && git pull && go build
```

## Usage

```
NAME:
   Harmony Tx Sender CLI App - This is the entry point for starting a new Harmony tx sender
USAGE:
   main [global options]
   
AUTHOR:
   Sebastian Johnsson
   
GLOBAL OPTIONS:
   --node value          Which node endpoint to use for API commands (default: "https://api.s0.pga.hmny.io")
   --from value          Which address to send tokens from (must exist in the keystore)
   --from-shard value    What shard to send tokens from (default: 0)
   --passphrase value    Passphrase to use for unlocking the keystore
   --to-shard value      What shard to send tokens to (default: 0)
   --amount value        How many tokens to send per tx (default: 1)
   --tx-count value      How many transactions to send in total (default: 1000)
   --tx-pool-size value  How many transactions to send simultaneously (default: 100)
   --receivers value     Which file to use for receiver addresses (default: "./data/receivers.txt")
   --tx-data value       Which file to use for tx data (default: "./data/tx_data.txt")
   --help, -h            show help
   --version, -v         print the version
   
VERSION:
   go1.12/darwin-amd64
```

Create the file data/receivers.txt and add the receiver addresses you want to send tokens to.
If you want to use tx data for your transactions, create the file data/tx_data.txt and add your tx data. Do note that tx data is a bit hit and miss.

```
./harmony-tx-sender --from ADDRESS --passphrase WALLET_PASSPHRASE --node https://api.s0.pga.hmny.io --from-shard 0 --to-shard 0 --tx-count 1000 --tx-pool-size 100 --amount 1
```
