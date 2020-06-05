package dice

type ConnectUri struct {
	Host     string `toml:"host" json:"host"`
	Port     int    `toml:"port" json:"port"`
	Database string `toml:"db" json:"db"`
	Username string `toml:"username" json:"username"`
	Password string `toml:"password" json:"password"`
	SSL      bool   `toml:"ssl" json:"ssl"`
}

type Structure struct {
	Type       string      `toml:"type"`
	Primary    bool        `toml:"primary"`
	Attributes []string    `toml:"attr"`
	Default    interface{} `toml:"default"`
	Constraint string      `toml:"constraint"`
	Mixins     []string    `toml:"mixins"`
}

type Schema struct {
	Table     string `toml:"table"`
	ModelName string `toml:"model"`
	Columns   map[string]Structure
}
