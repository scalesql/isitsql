C2
==
Sample configuration file

```
ag_name {
    domain = "prod"
    name = "ag_name"
    display_name = "txn-host"
}

defaults {
    tags = ["a", "b"]
    credential = "credential_name"
    ignore_backups = true
    ignore_backups_list = ["db1", "db2"]
    dns_suffix = "static.us.loc"
}

server "ID" {
    server = "abc.domain.com"
    host = "abc"
    dns_suffix = "domain.com"
    display_name = "abc"
    tags = ["a", "b"]
    key = "my-key"
    credential = "credential_name"
    ignore_backups = true
    ignore_backups_list = ["db1", "db2"]
    alias = true 
}
```

What to connect to 
------------------
This describes how we set Connection.Server from an Instance.  Options for the name include `(host(.domain)|ip4|ip6)[((,|:))|\instance]`


I need to end up with: server,port, and instance.

### Scenarios #1
* server "10.10.32.10" {}
* server "sd-db-txn" {server="10.10.1.2"}
* server "sd-db-txn" {host="10.10.1.2,1437"}
* server "db-txn" {server = "db-txn.static.us.loc"}
* server "db-txn" {host = "db-txn", dns_suffix="static.us.loc"}
* server "db-ibobr" {host="db-ibobr" port=1437 dns_suffix="domain.com"}
* server "db-ibobr" {server="db-ibobr.domain.com:1437"}

### Scenarios #2
* If server is an IP4 or IP6 address, use that.  Port?  Instance?
* 
* If host is set, apply an optional dns_suffix and use that
* Use the ID




## Availabile Fields
* ID
* Server
* Host (new)
* DNSSuffix (new)

I want:
* ID and DNS suffix from above, if host name




Today we set with COALESCE(server, ID).  We will switch to:

1. Set defaults for the fields above
2. If Server is set, use that and return
3. If host is set, use that else use ID
4. If (host|ID) is not an IP address, add the dns_suffix





