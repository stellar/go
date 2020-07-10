# Recovery Signer: Unique-Signing-Keys

This implementation uses shared signing keys for all accounts registered. The
service can be configured to generate and store unique signing keys for each
user.

The unique signing keys are stored in Postgres and are encrypted using Google
Tink's Hybrid operations. Google Tink's keyset format is used to store the
encryption keys and can be provided to the application as an insecure cleartext
keyset, or as an encrypted keyset.

Encrypted keysets may be encrypted with a remote KMS. Currently only the AWS
KMS is supported.

This implementation makes assumptions and tradeoffs regarding how unique
signing keys are generated and stored. You should review the implementation to
ensure the assumptions and tradeoffs are compatible with your use case,
requirements, and environment.

## Create the AWS KMS Key (Optional)

An AWS KMS key can be used to encrypt the Tink keyset.

1. Create a new key in AWS KMS
2. Note the ARN of the key region and ID of the key created
3. Configure the IAM policies so that only the instance running the service may
   use the key, and other AWS best practices are followed.

## Create the Tink Keyset

Google Tink keysets are used to hold the keys that are used in protecting the
signing keys stored in the database. The Tink keyset given to the application
should be in Tink's JSON format. You can use the official tinkey tool to
generate and manage the keyset, or you can use the built in
`encryption-tink-keyset` commands in recoverysigner to execute some basic
operations.

The keyset should contain at least one key. All keys in the file should be
asymmetric keys.

Create a Tink keyset containing a single key with the create subcommand. The
private key will be printed to stderr.
```
$ recoverysigner encryption-tink-keyset create
```

## Enable Unique Signing Keys

Unique signing keys are enabled by configuring a Tink keyset.

1. Set the `--encryption-tink-keyset` (`ENCRYPTION_TINK_KEYSET`) option to the
   JSON Tink keyset.
2. Set the `--encryption-kms-key-uri` (`ENCRYPTION_KMS_KEY_URI`) option to the
   AWS ARN for the key if the Tink keyset is encrypted with a AWS KMS key.

Note: This feature should not be disabled after it is enabled. Once enabled new
registered accounts will be provided a unique signing key to use along with the
shared signing keys configured on the server. If the feature is disabled by
removing the options after it has been enabled, the accounts will still have
the generated signing keys but the server will return a HTTP 500 Internal
Server Error on requests to sign transactions with a unique signing key.
