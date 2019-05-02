## Keystore Spec

### Problem

We need a keystore service that supports non-custodial applications.
It will make the process of stellarizing any applications easier as
they don't have to implement the logic to create a stellar account
and handle the encrypted private key themselves.

It is also intended to be the service that wallet SDK talks to.

### Authentication

For simplicity we will have each application spin up their own keystore
server, so there won’t be any routing logic in the keystore server that
directs requests to the correct client server to authenticate. Since we
don’t anticipate a lot of requests to the keystore from each user, we
should be able to tolerate having another round trip for relaying the
auth token to the client server.

Clients will have to configure their server's address used for
authentication when booting up the keystore.

An auth token needs to be passed for all requests, and that's why there
is no need for a userid argument to the endpoints below.

<img src=attachments/2019-04-24-keystore-auth-flows.png>

### Encrypted Key

EncryptedKey Object:

```typescript
interface EncryptedKey {
	keyType: string;
	publicKey: string;
	path?: string;
	extra?: any;
	encrypterName: string;
	encryptedPrivateKey: string;
	salt: string;
}
```

### Encrypted Key Data

EncryptedKeyData Object:

```typescript
interface EncryptedKeyData {
	keyType: string;
	publicKey: string;
	path?: string;
	extra?: any;
	encrypterName: string;
	encryptedPrivateKey: string;
	salt: string;
	creationTime: number;
	modifiedTime: number;	
}
```

### /store-keys

Store Keys Request:

```typescript
interface StoreKeysRequest {
	encryptedKeys: EncryptedKey[];
}
```

Store Keys Response:

```typescript
interface StoreKeysResponse {
	encryptedKeys: EncryptedKeyData[];
}
```

<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### /load-all-keys

Load All Keys Request:

This endpoint currently doesn't take any parameters.
We can potentially add some filters in the request.

Load All Keys Response:

```typescript
interface LoadAllKeysResponse {
	encryptedKeys: EncryptedKeyData[];
}
```
<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### /load-key

Load Key Request:

```typescript
interface LoadKeyRequest {
	publicKey: string;
}
```

Load Key Response:

```typescript
type LoadKeyResponse = EncryptedKeyData;
```

<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### /update-keys

Update Keys Request:

```typescript
interface UpdateKeysRequest {
	encryptedKeys: EncryptedKey[];
}
```

Update Keys Response:

```typescript
interface UpdateKeysResponse {
	encryptedKeys: EncryptedKeyData[];
}
```
<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### /remove-key

Remove Key Request:

```typescript
interface RemoveKeyRequest {
	publicKey: string;
}
```

Remove Key Response:

```typescript
type RemoveKeyResponse = EncryptedKeyData;
```

<details><summary>Errors</summary>

TBD
```json
{
	"code": "some error code",
	"message": "some error message",
	"retriable": false,
}
```
</details>

### Required Changes in Client Server

Applications using the keytore will have to implement an endpoint
that takes an auth token and returns a userid.
