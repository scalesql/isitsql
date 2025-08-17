package app

type IsItSQLTOML struct {
	Repository struct {
		Host       string `toml:"host"`
		Database   string `toml:"database"`
		Credential string `toml:"credential"`
	} `toml:"repository"`
}
