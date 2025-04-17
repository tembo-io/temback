package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5"
	"github.com/walle/targz"
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
	h(compress(cfg))
	h(upload(cfg))
}

type Config struct {
	Name   string
	DBHost string
	DBUser string
	DBPass string
	Bucket string
	Plain  bool
}

func (c *Config) Tarball() string {
	return c.Name + ".tar.gz"
}

func newConfig() *Config {
	cfg := new(Config)
	flag.StringVar(&cfg.Name, "name", "", "Backup name")
	flag.StringVar(&cfg.DBHost, "host", os.Getenv("PGHOST"), "Database host name")
	flag.StringVar(&cfg.DBUser, "user", os.Getenv("PGUSER"), "Database username")
	flag.StringVar(&cfg.DBPass, "pass", os.Getenv("PGPASSWORD"), "Database password")
	flag.StringVar(&cfg.Bucket, "bucket", "", "S3 bucket name")
	flag.BoolVar(&cfg.Plain, "text", false, "Plain text format")

	flag.Parse()
	if cfg.Name == "" || cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBPass == "" ||
		cfg.Bucket == "" {
		usage()
	}

	return cfg
}

func usage() {
	fmt.Printf(
		"Usage:\n  %v --org-id [ORG_ID] --inst-id [INST_ID] --inst-name [NAME] --conn [URI] --bucket [S3_BUCKET]\n",
		filepath.Base(os.Args[0]),
	)
	os.Exit(1)
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

func dump(cfg *Config, dbs []string) error {
	if err := os.MkdirAll(cfg.Name, 0750); err != nil {
		return err
	}

	type Job struct {
		name string
		cmd  *exec.Cmd
		err  error
	}

	// Assemble the commands.
	jobs := make([]Job, 0, len(dbs)+2)
	jobs = append(jobs,
		Job{
			name: "roles",
			cmd:  exec.Command("pg_dumpall", "-r", "-f", path.Join(cfg.Name, "roles.sql")),
		},
		Job{
			name: "tablespaces",
			cmd:  exec.Command("pg_dumpall", "-t", "-f", path.Join(cfg.Name, "tablespaces.sql")),
		},
	)

	// Setup args according to configuration.
	args := []string{"-C", "-F"}
	ext := ""
	if cfg.Plain {
		args = append(args, "p")
		ext = ".sql"
	} else {
		args = append(args, "d", "-j", "8")
	}
	args = append(args, "-f")

	// Setup the jobs to dump each database.
	for _, db := range dbs {
		jobs = append(jobs,
			Job{
				name: db + " database",
				cmd: exec.Command(
					"pg_dump",
					slices.Concat(args, []string{path.Join(cfg.Name, "db-"+db+ext), db})...,
				),
			},
		)
	}

	// Start the jobs.
	ch := make(chan Job)
	for _, job := range jobs {
		go func(job Job) {
			// defer wg.Done()
			fmt.Printf("Dumping %v\n", job.name)
			job.cmd.Env = append(job.cmd.Env,
				"PGHOST="+cfg.DBHost,
				"PGUSER="+cfg.DBUser,
				"PGPASSWORD="+cfg.DBPass,
				"PATH="+os.Getenv("PATH"),
			)
			job.cmd.Stderr = os.Stderr
			job.cmd.Stdout = os.Stdout
			job.err = job.cmd.Run()
			ch <- job
		}(job)
	}

	// Wait for the commands to finish.
	var ret error
	for range len(jobs) {
		job := <-ch
		if job.err == nil {
			fmt.Printf("Successfully dumped %v\n", job.name)
		} else {
			ret = job.err
			fmt.Printf("Failed to dump dumped %v\n", job.name)
		}
	}

	return ret
}

func compress(cfg *Config) error {
	file := cfg.Tarball()
	fmt.Printf("Archiving %v\n", file)
	return targz.Compress(cfg.Name, file)
}

func upload(cfg *Config) error {
	file := cfg.Tarball()
	fmt.Printf("Uploading %v...", file)

	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(awsCfg)
	_ = client

	fh, err := os.Open(file)
	if err != nil {
		fmt.Println("Failed")
		return fmt.Errorf("cannot open %q: %v", file, err)
	}
	defer fh.Close()

	if _, err := client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:            aws.String(cfg.Bucket),
		Key:               aws.String(cfg.Name),
		Body:              fh,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		ContentType:       aws.String("application/gzip"),
	}); err != nil {
		fmt.Println("Failed")
		return err
	}
	fmt.Println("Done!")
	return nil
}
