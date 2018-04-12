---
title: Transaction Malformed
---

When you submit a malformed transaction to Horizon, Horizon will return a `transaction_malformed` error. There are many ways in which a transaction is malformed, including
* you submitted an empty string
* your base64-encoded string is invalid
* your [XDR](../../learn/xdr.md) structure is invalid
* you have leftover bytes in your [XDR](../../learn/xdr.md) structure

If you are encountering this error, please check the contents of the transaction you are submitting. This error is similar to the [Bad Request](./bad-request.md) error response and, therefore, the [HTTP 400 Error](https://developer.mozilla.org/en-US/docs/Web/HTTP/Response_codes).

## Attributes

As with all errors Horizon returns, `transaction_malformed` follows the [Problem Details for HTTP APIs](https://tools.ietf.org/html/draft-ietf-appsawg-http-problem-00) draft specification guide and thus has the following attributes:

| Attribute | Type   | Description                                                                                                                     |
| --------- | ----   | ------------------------------------------------------------------------------------------------------------------------------- |
| Type      | URL    | The identifier for the error.  This is a URL that can be visited in the browser.                                                |
| Title     | String | A short title describing the error.                                                                                             |
| Status    | Number | An HTTP status code that maps to the error.                                                                                     |
| Detail    | String | A more detailed description of the error.                                                                                       |
| Instance  | String | A token that uniquely identifies this request. Allows server administrators to correlate a client report with server log files. |

In addition, the following additional data is provided in the `extras` field of the error:

| Attribute      | Type   | Description                                        |
|----------------|--------|----------------------------------------------------|
| `envelope_xdr` | String | The submitted data that was malformed in some way. |


## Related

[Bad Request](./bad-request.md)
