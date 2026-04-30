package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ZacharyJo/db-cli/internal/redisdb"
)

var (
	redisHost       string
	redisPort       int
	redisPassword   string
	redisDB         int
	redisMode       string
	redisAddrs      []string
	redisMasterName string
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Connect to Redis and open an interactive session",
	Example: `  # 单机
  db-cli redis -H 127.0.0.1 -P 6379 --password secret

  # 哨兵（主从）
  db-cli redis --mode sentinel --addrs 10.0.0.1:26379,10.0.0.2:26379 --master-name mymaster --password secret

  # 集群
  db-cli redis --mode cluster --addrs 10.0.0.1:7000,10.0.0.2:7001,10.0.0.3:7002 --password secret`,
	RunE: runRedisConnect,
}

var redisExecCmd = &cobra.Command{
	Use:   "exec COMMAND [ARGS...]",
	Short: "Execute a single Redis command and print the result",
	Example: `  db-cli redis exec -H 127.0.0.1 -P 6379 "GET mykey"
  db-cli redis exec --mode cluster --addrs 10.0.0.1:7000,10.0.0.2:7001 "DBSIZE"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRedisExec,
}

func init() {
	rootCmd.AddCommand(redisCmd)
	redisCmd.AddCommand(redisExecCmd)

	for _, cmd := range []*cobra.Command{redisCmd, redisExecCmd} {
		cmd.Flags().StringVarP(&redisHost, "host", "H", "127.0.0.1", "Redis host (single mode)")
		cmd.Flags().IntVarP(&redisPort, "port", "P", 6379, "Redis port (single mode)")
		cmd.Flags().StringVar(&redisPassword, "password", "", "Redis password")
		cmd.Flags().IntVar(&redisDB, "db", 0, "Redis database number (single/sentinel only)")
		cmd.Flags().StringVar(&redisMode, "mode", "single", "Connection mode: single | sentinel | cluster")
		cmd.Flags().StringSliceVar(&redisAddrs, "addrs", nil,
			"Node addresses (comma-separated or repeated). Overrides -H/-P.\n"+
				"  single:   10.0.0.1:6379\n"+
				"  sentinel: 10.0.0.1:26379,10.0.0.2:26379\n"+
				"  cluster:  10.0.0.1:7000,10.0.0.2:7001,10.0.0.3:7002")
		cmd.Flags().StringVar(&redisMasterName, "master-name", "mymaster",
			"Sentinel master name (sentinel mode only)")
	}
}

// buildRedisOptions 将 flag 值转为 redisdb.Options，并校验 --mode 的合法性。
func buildRedisOptions() (redisdb.Options, error) {
	addrs := redisAddrs
	if len(addrs) == 0 {
		addrs = []string{fmt.Sprintf("%s:%d", redisHost, redisPort)}
	}
	// 支持单个 --addrs "a,b,c" 以及多次指定的混合写法，统一展开。
	var flat []string
	for _, a := range addrs {
		for _, part := range strings.Split(a, ",") {
			if s := strings.TrimSpace(part); s != "" {
				flat = append(flat, s)
			}
		}
	}
	mode := redisdb.Mode(redisMode)
	switch mode {
	case redisdb.ModeSingle, redisdb.ModeSentinel, redisdb.ModeCluster:
	default:
		return redisdb.Options{}, fmt.Errorf("unknown --mode %q, must be single|sentinel|cluster", redisMode)
	}
	return redisdb.Options{
		Addrs:      flat,
		Password:   redisPassword,
		DB:         redisDB,
		MasterName: redisMasterName,
		Mode:       mode,
	}, nil
}

func runRedisConnect(_ *cobra.Command, _ []string) error {
	opts, err := buildRedisOptions()
	if err != nil {
		return err
	}
	client, err := redisdb.Connect(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	return redisdb.NewREPL(client).Run()
}

func runRedisExec(_ *cobra.Command, args []string) error {
	opts, err := buildRedisOptions()
	if err != nil {
		return err
	}
	client, err := redisdb.Connect(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	line := strings.Join(args, " ")
	parts := redisdb.SplitArgs(line)
	if len(parts) == 0 {
		return nil
	}
	iargs := make([]interface{}, len(parts))
	for i, p := range parts {
		iargs[i] = p
	}
	result, err := client.Do(context.Background(), iargs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	return nil
}
