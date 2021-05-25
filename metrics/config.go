// Copyright 2021 The go-popcateum Authors
// This file is part of go-popcateum.
//
// The go-popcateum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-popcateum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-popcateum library. If not, see <http://www.gnu.org/licenses/>.

package metrics

// Config contains the configuration for the metric collection.
type Config struct {
	Enabled          bool   `toml:",omitempty"`
	EnabledExpensive bool   `toml:",omitempty"`
	HTTP             string `toml:",omitempty"`
	Port             int    `toml:",omitempty"`
	EnableInfluxDB   bool   `toml:",omitempty"`
	InfluxDBEndpoint string `toml:",omitempty"`
	InfluxDBDatabase string `toml:",omitempty"`
	InfluxDBUsername string `toml:",omitempty"`
	InfluxDBPassword string `toml:",omitempty"`
	InfluxDBTags     string `toml:",omitempty"`
}

// DefaultConfig is the default config for metrics used in go-popcateum.
var DefaultConfig = Config{
	Enabled:          false,
	EnabledExpensive: false,
	HTTP:             "127.0.0.1",
	Port:             6060,
	EnableInfluxDB:   false,
	InfluxDBEndpoint: "http://localhost:8086",
	InfluxDBDatabase: "gpop",
	InfluxDBUsername: "test",
	InfluxDBPassword: "test",
	InfluxDBTags:     "host=localhost",
}
