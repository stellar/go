# sep10-homedomaincheck

This tool checks a space separated list of home domains for if they support SEP-10 v3's specification of what a home domain is. It verifies that a challenge transaction issued by the home domains web auth endpoint contains the home domain.

Outputs a CSV to stdout containing for each home domain if it passes or fails, and if it fails what the error is, and regardless, what the challenge transaction was the tool received.

## SEP-10 Version Updates and Compatibility

SEP-10 **v1.X** featured challenge transactions containing a single Manage Data operation containing a `<anchor name> auth` key. SEP-10 **v2.0** updated the Manage Data operation's key to contain a `<domain name> auth` key, where `<domain name>` is the hostname of the server hosting the organization's SEP-1 stellar.toml file. 

For example, if your stellar.toml URL is `https://<domain>/.well-known/stellar.toml`, your challenge transactions should contain a `<domain> auth` key. 

SEP-10 **v2.0** also allowed clients to pass a `home_domain` request parameter in addition to the signed challenge `transaction`. SEP-10 servers that issue JWT tokens for multiple organzations (and by extension, multiple `home_domain`s) should verify that this `home_domain` is one of the known domains your server issues JWT tokens for and include it in the first Manage Data operation's key.

SEP-10 **v2.0** required that clients verify the home domain value, causing incompatibility issues between v2.0 clients and 1.X servers. SEP-10 **v2.1** removed this verification requirement to avoid those issues, and allowed organizations to issue challenge transactions containing additional Manage Data operations.

SEP-10 **v3.0** reintroduced the home domain verification after clarifying the home domain definition as described in the example.

## Install

```
go get github.com/stellar/go/tools/sep10-homedomaincheck
```

## Usage

```
sep10-homedomaincheck [homedomain] [homedomain] ...
```
