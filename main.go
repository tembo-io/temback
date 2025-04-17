package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"

	"github.com/jackc/pgx/v5"
)

func h(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func main() {
	cfg := newConfig()

	dbs, err := list_databases(cfg)
	h(err)
	h(dump(cfg, dbs))
}

type Config struct {
	OrgID    string
	InstID   string
	InstName string
	DBHost   string
	DBUser   string
	DBPass   string
	Plain    bool
}

func (c *Config) DumpDir() string {
	return fmt.Sprintf("%v-%v-%v", c.OrgID, c.InstID, c.InstName)
}

func newConfig() *Config {
	cfg := new(Config)
	flag.StringVar(&cfg.OrgID, "org-id", "", "Organization ID")
	flag.StringVar(&cfg.InstID, "inst-id", "", "Instance ID")
	flag.StringVar(&cfg.InstName, "inst-name", "", "Instance Name")
	flag.StringVar(&cfg.DBHost, "host", os.Getenv("PGHOST"), "Database host name")
	flag.StringVar(&cfg.DBUser, "user", os.Getenv("PGUSER"), "Database username")
	flag.StringVar(&cfg.DBPass, "pass", os.Getenv("PGPASSWORD"), "Database password")
	flag.BoolVar(&cfg.Plain, "text", false, "Plain text format")

	flag.Parse()
	if cfg.OrgID == "" || cfg.InstID == "" || cfg.InstName == "" || cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBPass == "" {
		usage()
	}

	return cfg
}

func usage() {
	fmt.Printf("%v --org-id [ORG_ID] --inst-id [INST_ID] --inst-name [NAME] --conn [URI]\n", os.Args[0])
	os.Exit(1)
}

func dump(cfg *Config, dbs []string) error {
	dir := cfg.DumpDir()
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	type Job struct {
		name string
		cmd  *exec.Cmd
	}

	// Assemble the commands.
	jobs := make([]Job, 0, len(dbs)+2)
	jobs = append(jobs,
		Job{
			name: "roles",
			cmd:  exec.Command("pg_dumpall", "-r", "-f", path.Join(dir, "roles.sql")),
		},
		Job{
			name: "tablespaces",
			cmd:  exec.Command("pg_dumpall", "-t", "-f", path.Join(dir, "tablespaces.sql")),
		},
	)

	opts := []string{"-C", "-F"}
	ext := ".sql"
	if cfg.Plain {
		opts = append(opts, "p")
	} else {
		opts = append(opts, "d", "-j", "8")
		ext = ""
	}
	opts = append(opts, "-f")

	for _, db := range dbs {
		jobs = append(jobs,
			Job{
				name: db + " database",
				cmd: exec.Command(
					"pg_dump",
					slices.Concat(opts, []string{path.Join(dir, "db-"+db+ext), db})...,
				),
			},
		)
	}

	// Start the jobs.
	for _, job := range jobs {
		fmt.Printf("Dumping %v\n", job.name)
		job.cmd.Env = append(job.cmd.Env,
			"PGHOST="+cfg.DBHost,
			"PGUSER="+cfg.DBUser,
			"PGPASSWORD="+cfg.DBPass,
			"PATH="+os.Getenv("PATH"),
		)
		job.cmd.Stderr = os.Stderr
		job.cmd.Stdout = os.Stdout
		if err := job.cmd.Start(); err != nil {
			return err
		}
	}

	// Wait for the commands to finish.
	var ret error
	for _, job := range jobs {
		if err := job.cmd.Wait(); err != nil {
			ret = err
		}
		fmt.Printf("Finished %v\n", job.name)
	}

	return ret
}

func list_databases(cfg *Config) ([]string, error) {

	c, err := pgx.ParseConfig(fmt.Sprintf("user=%v password=%v host=%v", cfg.DBUser, cfg.DBPass, cfg.DBHost))
	if err != nil {
		return nil, err
	}
	conn, err := pgx.ConnectConfig(context.Background(), c)
	if err != nil {
		return nil, err
	}

	defer conn.Close(context.Background())

	rows, err := conn.Query(
		context.Background(),
		"SELECT datname FROM pg_database WHERE datallowconn",
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowTo[string])
}
