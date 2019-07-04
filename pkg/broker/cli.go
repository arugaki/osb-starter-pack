package broker

import (
	"flag"
)

// Options holds the options specified by the broker's code on the command
// line. Users should add their own options here and add flags for them in
// AddFlags.
type Options struct {
	CatalogPath string
	Async       bool

	MysqlAddress  string
	MysqlPort     string
	MysqlUserName string
	MysqlPassword string
	MysqlDB       string
	MysqlActive   int
	MysqlIdle     int

	AuthenticateK8SToken bool
	KubeConfig           string
}

// AddFlags is a hook called to initialize the CLI flags for broker options.
// It is called after the flags are added for the skeleton and before flag
// parse is called.
func AddFlags(o *Options) {
	flag.StringVar(&o.CatalogPath, "catalogPath", "", "The path to the catalog")
	flag.BoolVar(&o.Async, "async", false, "Indicates whether the broker is handling the requests asynchronously.")

	// mysql
	flag.StringVar(&o.MysqlAddress, "mysql-addr", "127.0.0.1", "specify the which mysql host to be used")
	flag.StringVar(&o.MysqlPort, "mysql-port", "3306", "specify the which mysql port to be used")
	flag.StringVar(&o.MysqlUserName, "mysql-username", "root", "specify the which mysql user to be used")
	flag.StringVar(&o.MysqlPassword, "mysql-password", "root", "specify the which mysql password to be used")
	flag.StringVar(&o.MysqlDB, "mysql-database", "root", "specify the which mysql database to be used")
	flag.IntVar(&o.MysqlActive, "mysql-active-connections", 100, "specify the mysql max active connections to be limited")
	flag.IntVar(&o.MysqlIdle, "mysql-idle-connections", 50, "specify the mysql max idle connections to be limited")

	// log level
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
}