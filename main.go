// Package main provides the backup utility.
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5"
	"github.com/walle/targz"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	build   = "HEAD"
)

func h(err error) {
	const errExitCode = 2
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(errExitCode)
	}
}

func main() {
	cfg := newConfig()

	info, err := getDBInfo(cfg)
	h(err)
	h(dump(cfg, info))
	h(addReadme(cfg, info))
	h(compress(cfg))
	h(upload(cfg))
	h(cleanup(cfg))
}

type backupConfig struct {
	name     string
	host     string
	user     string
	pass     string
	bucket   string
	dir      string
	compress bool
	plain    bool
	clean    bool
	time     time.Time
}

type dbInfo struct {
	version string
	dbs     []string
}

func (c *backupConfig) Tarball() string {
	return c.name + ".tar.gz"
}

func (c *backupConfig) UploadKey() string {
	if c.dir == "" {
		return c.Tarball()
	}
	return c.dir + "/" + c.Tarball()
}

func newConfig() *backupConfig {
	cfg := &backupConfig{time: time.Now().UTC()}
	flag.StringVar(&cfg.name, "name", "", "Backup name")
	flag.StringVar(&cfg.host, "host", os.Getenv("PGHOST"), "Database host name")
	flag.StringVar(&cfg.user, "user", os.Getenv("PGUSER"), "Database username")
	flag.StringVar(&cfg.pass, "pass", os.Getenv("PGPASSWORD"), "Database password")
	flag.StringVar(&cfg.bucket, "bucket", "", "S3 bucket name")
	flag.StringVar(&cfg.dir, "dir", "", "S3 bucket directory")
	flag.BoolVar(&cfg.compress, "compress", false, "Compress the backup (ignored with --bucket)")
	flag.BoolVar(&cfg.plain, "text", false, "Plain text format")
	flag.BoolVar(&cfg.clean, "clean", false, "Delete files after upload")

	printVersion := flag.Bool("version", false, "Print version")

	flag.Parse()
	if *printVersion {
		fmt.Printf("%v %v (%v)\n", filepath.Base(os.Args[0]), version, build)
		os.Exit(0)
	}

	if cfg.name == "" || cfg.host == "" || cfg.user == "" || cfg.pass == "" {
		usage()
	}

	return cfg
}

func usage() {
	fmt.Printf(
		"Usage:\n  %v --name <NAME> [--bucket <S3_BUCKET>] [OPTIONS]\n",
		filepath.Base(os.Args[0]),
	)
	const usageExitCode = 1
	os.Exit(usageExitCode)
}

func getDBInfo(cfg *backupConfig) (*dbInfo, error) {
	c, err := pgx.ParseConfig(fmt.Sprintf("user=%v password=%v host=%v", cfg.user, cfg.pass, cfg.host))
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	defer func() { _ = conn.Close(ctx) }()

	info := new(dbInfo)
	if err := conn.QueryRow(ctx, "SHOW server_version").Scan(&info.version); err != nil {
		return nil, err
	}

	rows, err := conn.Query(
		ctx,
		"SELECT datname FROM pg_database WHERE datallowconn",
	)
	if err != nil {
		return nil, err
	}

	info.dbs, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	return info, err
}

func dump(cfg *backupConfig, info *dbInfo) error {
	fmt.Printf("Backing up to %v\n", cfg.name)
	const dirMode = 0750
	if err := os.MkdirAll(cfg.name, dirMode); err != nil {
		return err
	}

	type Job struct {
		name string
		cmd  *exec.Cmd
		err  error
	}

	// Assemble the commands.
	globals := []string{"roles", "tablespaces"}
	jobs := make([]Job, 0, len(info.dbs)+len(globals))
	for _, g := range globals {
		// #nosec G204
		jobs = append(jobs, Job{name: g, cmd: exec.Command(
			"pg_dumpall", "-r", "-f", path.Join(cfg.name, g+".sql")),
		})
	}

	// Setup args according to configuration.
	args := []string{"-C", "-F"}
	ext := ""
	if cfg.plain {
		args = append(args, "p")
		ext = ".sql"
	} else {
		args = append(args, "d", "-j", "8")
	}
	args = append(args, "-f")

	// Setup the jobs to dump each database.
	for _, db := range info.dbs {
		dbArgs := args
		if db == "postgres" || db == "template1" {
			// Omit -C
			dbArgs = args[1:]
		}
		// #nosec G204
		jobs = append(jobs,
			Job{name: db + " database", cmd: exec.Command(
				"pg_dump",
				slices.Concat(dbArgs, []string{path.Join(cfg.name, "db-"+db+ext), db})...,
			)},
		)
	}

	// Start the jobs.
	ch := make(chan Job)
	for _, job := range jobs {
		go func(job Job) {
			// defer wg.Done()
			fmt.Printf("  Dumping %v\n", job.name)
			job.cmd.Env = append(job.cmd.Env,
				"PGHOST="+cfg.host,
				"PGUSER="+cfg.user,
				"PGPASSWORD="+cfg.pass,
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
	for range jobs {
		job := <-ch
		if job.err == nil {
			fmt.Printf("  Successfully dumped %v\n", job.name)
		} else {
			ret = job.err
			fmt.Printf("  Failed to dump dumped %v\n", job.name)
		}
	}

	return ret
}

//go:embed template.md
var tempSrc string

func addReadme(cfg *backupConfig, info *dbInfo) error {
	fmt.Println("Generating README.md")
	tmpl, err := template.New("test").Parse(tempSrc)
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(cfg.name)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	fh, err := root.Create("README.md")
	if err != nil {
		fmt.Println("Failed")
		return fmt.Errorf("cannot open README.md: %v", err)
	}
	defer func() { _ = fh.Close() }()

	restores := make([]string, len(info.dbs))
	format := "dir"
	if cfg.plain {
		format = "text"
		for i, db := range info.dbs {
			restores[i] = fmt.Sprintf("psql -f %q", "db-"+db+".sql")
		}
	} else {
		for i, db := range info.dbs {
			create := "-C "
			if db == "postgres" || db == "template1" {
				create = ""
			}
			restores[i] = fmt.Sprintf("pg_restore %v-d postgres -j 8 -f %q", create, "db-"+db)
		}
	}

	return tmpl.Execute(fh, map[string]any{
		"Name":      cfg.name,
		"Host":      cfg.host,
		"Date":      cfg.time.Format(time.RFC3339),
		"Version":   info.version,
		"Databases": info.dbs,
		"Restores":  restores,
		"Format":    format,
	})
}

func compress(cfg *backupConfig) error {
	if !cfg.compress && cfg.bucket == "" {
		return nil
	}
	file := cfg.Tarball()
	fmt.Printf("Archiving %v...", file)
	if err := targz.Compress(cfg.name, file); err != nil {
		fmt.Println("Failed")
		return err
	}
	fmt.Println("Success")
	return nil
}

func upload(cfg *backupConfig) error {
	if cfg.bucket == "" {
		return nil
	}
	file := cfg.Tarball()
	fmt.Printf("Uploading %v...", file)

	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(awsCfg)
	_ = client

	root, err := os.OpenRoot(".")
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	fh, err := root.Open(file)
	if err != nil {
		fmt.Println("Failed")
		return fmt.Errorf("cannot open %q: %v", file, err)
	}
	defer func() { _ = fh.Close() }()

	if _, err := client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:            aws.String(cfg.bucket),
		Key:               aws.String(cfg.UploadKey()),
		Body:              fh,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		ContentType:       aws.String("application/gzip"),
	}); err != nil {
		fmt.Println("Failed")
		return err
	}
	fmt.Println("Success")
	return nil
}

func cleanup(cfg *backupConfig) error {
	if !cfg.clean || (cfg.bucket == "" && !cfg.compress) {
		fmt.Println("Done!")
		return nil
	}

	fmt.Println("Cleaning up")
	if cfg.bucket != "" {
		// Backup uploaded, clean up everything.
		for _, path := range []string{cfg.Tarball(), cfg.name} {
			if err := os.RemoveAll(path); err != nil {
				fmt.Println()
				return err
			}
		}
	} else if cfg.compress {
		// Just remove the directory.
		if err := os.RemoveAll(cfg.name); err != nil {
			fmt.Println()
			return err
		}
	}
	fmt.Println("Done!")
	return nil
}
