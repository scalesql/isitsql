File-Based Configuration
========================
More and more applications I write use configuration files.  This lets me version them in GIT.  IsItSQL uses JSON files that are painful to edit.  Starting with 1.8, IsItSQL supports file-based configuration for the servers using `.HCL` files.

File-based configuration looks for servers to monitor in `*.hcl` files in the `servers` folder.  It supports multiple HCL files in subdirectories.

Getting Started
---------------
In the `optional` folder, there are two executables:

* `cfg2file.exe` - Reads the existing `connections.json` and writes `servers/servers.isitsql.hcl` and `ag_names.hcl`.
* `linter.exe` - Monitors the HCL files in the `servers` folder for syntax errors and duplicate keys.

The steps to migrate to HCL file-based configuration are:

1. Copy these two exeutables up to the IsItSQL folder
2. Run `cfg2file.exe`.  It will make the `servers` folder and create `servers.isitsql.hcl` and `ag_names.hcl` in that folder.
3. Restart IsItSQL.  Look for a log entry like `config mode: FILE`.  The old mode will log `config mode: GUI`.
4. Edit your HCL files.  IsItSQL polls them for changes every minute and completely rereads them every hour.

File Structure
==============
The `servers` folder can have any number of HCL files.  A given HCL file can have:

* One `defaults` block
* One or more `server` blocks
* One or more `ag_name` blocks

Server Block
------------
The simplest form of the HCL entry is:

```hcl
server "D40\SQL2014" {}
```

The full form is: 

```hcl
server "db-txn" {
    server = "D40\SQL2019"
    display_name "txn-server"
    key = "txn1"
    tags = ["prod", "dc1"]
    credential = "sqlmonitor"
    ignore_backups = false
    ignore_backups_list = ["a", "b"]
}
```

The fields are:

* Identifier or ID.  The string in quotes after `server` is the Identifier or ID (`db-txn` in this example).  It must be unique across all files.  I typically use the server name for this.
* `server` is what IsItSQL will try to connect to.  It is like `Server` in a connection string.  If this isn't provided, it will use the Identifier.
* `display_name` is what you see on the screen.  If this isn't provided, it will use the Identifier.
* `key` is the primary key for the internal map that stores the data.  It must be unique across all files.  This is also the URL for the server: `http://localhost:8143/server/txn1`.  If this isn't provided, it will use the Identifier.  This will be converted to lower-case for the internal map and the URLs.
* `tags` are the tags for the server
* `credential` is the name of a shared credential.  If this isn't provided, it defaults to a trusted connection.
* `ignore_backups` tells IsItSQL to ignore missing backups for this server.
* `ignore_backups_list` tells IsItSQL to ignore backups for the listed databases.

## Defaults Block
Each HCL file can have a defaults section:

```hcl
defaults {
    tags = ["prod", "dc1"]
    credential = "sqlmonitor"
    ignore_backups = false
    ignore_backups_list = ["a", "b"]
}
```

These defaults are assigned to all servers in the file.

* For `credential` and `ignore_backups`, the `server` block will override the default.
* For `tags` and `ignore_backups_list`, the defaults and the `server` block will be merged, sorted, and converted to lower-case.

## Availability Group Names Block
We can assign display names to Availability Groups.  This is used in the Availability Group page to display a different name on the screen.  I try to use a static DNS entry on top of the Listener.  This lets me display that static DNS instead of the Listener.

```hcl
ag_name {
  domain       = "PROD"
  name         = "Listen01P"
  display_name = "db-txn"
}
```

Editing with VSCode
====================
* Run the Linter in the terminal.  It will monitor the files and report any issues as they are saved.  It polls every second.  `linter -?` will display the options.
* Use the HCL plugin for syntax coloring.  I use the Hashicorp HCL plugin.
* VSCode autosaves by default.  This can lead to file saves during partial edits which the Linter will complain about.

Notes
=====
* Beware backslashes.  I think I have handled all the cases, but you may need to escape them using double-backslashes.
* I use multiple files with defaults.  For example, I have the following files right now.  I will likely create folders for each domain shortly.
    * `prod.core.hcl` -- all production servers with no defaults in the CORE domain
    * `prod.office.hcl` -- all production servers in the OFFICE domain.  Has a default shared credential.
    * `dev.office.hcl` -- all DEV servers in the OFFICE domain.  Has a default shared credential.
    * `monaco.office.hcl` -- all the serves in the Monaco business unit.

