package c2

type ConnectionFile struct {
	Defaults  *Defaults  `hcl:"defaults,block"`
	Instances []Instance `hcl:"server,block"`
	AGNames   []AGName   `hcl:"ag_name,block"`
}

type Defaults struct {
	Tags              *[]string `hcl:"tags"`
	Credential        *string   `hcl:"credential"`
	IgnoreBackups     *bool     `hcl:"ignore_backups"`
	IgnoreBackupsList *[]string `hcl:"ignore_backups_list"`
}

type Instance struct {
	ID                string    `hcl:"id,label"`
	Server            *string   `hcl:"server"`
	DisplayName       *string   `hcl:"display_name"`
	Tags              *[]string `hcl:"tags"`
	Key               *string   `hcl:"key"`
	Credential        *string   `hcl:"credential"`
	IgnoreBackups     *bool     `hcl:"ignore_backups"`
	IgnoreBackupsList *[]string `hcl:"ignore_backups_list"`

	// This an alias for multiple machines
	// such as a Listener or static DNS
	Alias *bool `hcl:"alias"`
}

type AGName struct {
	Domain      string `hcl:"domain"`
	Name        string `hcl:"name"`
	DisplayName string `hcl:"display_name"`
}
