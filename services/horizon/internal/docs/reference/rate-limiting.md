---
title: Rate Limiting
---

In order to provide service stability, Horizon limits the number of requests a
client can perform within a one hour window.  By default this is set to 3600
requests per hour—an average of one request per second.

## Response headers for rate limiting

Every response from Horizon sets advisory headers to inform clients of their
standing with rate limiting system:

|          Header         |                               Description                                |
| ----------------------- | ------------------------------------------------------------------------ |
| `X-RateLimit-Limit`     | The maximum number of requests that the current client can make in one hour. |
| `X-RateLimit-Remaining` | The number of remaining requests for the current window.                 |
| `X-RateLimit-Reset`     | Seconds until a new window starts.                                        |

In addition, a `Retry-After` header will be set when the current client is being
throttled.
