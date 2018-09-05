package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/cache"
	"github.com/skuid/warden/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:              "Warden",
	Short:            "A microservice for proxying data source access and enforcing DSO regulations",
	PersistentPreRun: initServiceDependencies,
	Version:          version.Name,
}

// Execute is used as entrypoint to the cobra commands
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		zap.L().Error("encountered an error on root command execution", zap.Error(err))
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().Bool("debug", false, "Debug Mode Switch")
	RootCmd.PersistentFlags().Bool("pprof", false, "Profile Mode Switch")
	RootCmd.PersistentFlags().Bool("database_on", false, "Database Switch")
	RootCmd.PersistentFlags().Bool("local_aws_conf_enabled", false, "Used by AWS to switch on loading the local config")

	RootCmd.PersistentFlags().String("dbhost", "", "The persistence database hostname for warden")
	RootCmd.PersistentFlags().Int("dbport", 0, "The persistence database port for warden")
	RootCmd.PersistentFlags().String("dbname", "", "The persistence database name for warden")
	RootCmd.PersistentFlags().String("dbusername", "", "The username for the database connection.")
	RootCmd.PersistentFlags().String("dbpassword", "", "The password for the database connection.")

	RootCmd.PersistentFlags().String("cachehost", "", "The cache hostname for warden to connect to")
	RootCmd.PersistentFlags().Int("cacheport", 0, "The cache port for warden to connect to")
	RootCmd.PersistentFlags().Int("cacheconnectionlimit", 100, "The max number of connections for the cache service to use")

	viper.BindEnv("dbhost", "PGHOST")
	viper.BindEnv("dbport", "PGPORT")
	viper.BindEnv("dbname", "PGDATABASE")
	viper.BindEnv("dbusername", "PGUSER")
	viper.BindEnv("dbpassword", "PGPASSWORD")
	viper.BindEnv("cachehost", "REDIS_HOST")
	viper.BindEnv("cacheport", "REDIS_PORT")
	viper.BindEnv("cacheconnectionlimit", "REDIS_MAX_CONNECTIONS")

	RootCmd.PersistentFlags().String("encryption_key", "", "The encryption key to be used for database encryption")
	RootCmd.PersistentFlags().Bool("use_kms", false, "Switch for using KMS to decrypt DB encryption keys")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if err := viper.BindPFlags(RootCmd.PersistentFlags()); err != nil {
		zap.L().Error("encountered an error on viper flag binding", zap.Error(err))
		os.Exit(1)
	}

	viper.SetEnvPrefix("warden")
	viper.AutomaticEnv()
}

func decryptKMSKey() error {
	if viper.GetBool("local_aws_conf_enabled") {
		err := os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
		if err != nil {
			return err
		}
	}
	key := viper.GetString("encryption_key")
	keyAsBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	sess, err := session.NewSession(
		&aws.Config{CredentialsChainVerboseErrors: aws.Bool(true)},
	)
	if err != nil {
		return err
	}
	svc := kms.New(sess)
	input := &kms.DecryptInput{
		CiphertextBlob: keyAsBytes,
	}
	result, err := svc.Decrypt(input)
	if err != nil {
		return err
	}

	resultAsBytes := make([]byte, len(result.Plaintext))

	keyLength, err := base64.StdEncoding.Decode(resultAsBytes, result.Plaintext)
	if err != nil {
		return err
	}

	resultAsBytes = resultAsBytes[:keyLength]

	viper.Set("encryption_key", resultAsBytes)
	return nil
}

func initServiceDependencies(cmd *cobra.Command, args []string) {
	if viper.GetBool("database_on") {
		if err := picard.CreateConnection(fmt.Sprintf(
			"postgres://%s:%d/%s?sslmode=disable&user=%s&password=%s",
			viper.GetString("dbhost"),
			viper.GetInt("dbport"),
			viper.GetString("dbname"),
			viper.GetString("dbusername"),
			viper.GetString("dbpassword"),
		)); err != nil {
			zap.L().Fatal("STARTUP ERROR: Failed to initialize database connection!", zap.Error(err))
		}

		if viper.GetBool("use_kms") {
			if err := decryptKMSKey(); err != nil {
				zap.L().Fatal("STARTUP ERROR: Failed to decrypt database encryption key using KMS", zap.Error(err))
			}
		}

		if err := picard.SetEncryptionKey([]byte(viper.GetString("encryption_key"))); err != nil {
			zap.L().Fatal("STARTUP ERROR: Failed to set database encryption key!", zap.Error(err))
		}

		if viper.GetString("cachehost") == "" || viper.GetInt("cacheport") == 0 {
			zap.L().Fatal("STARTUP ERROR: REDIS_HOST and REDIS_PORT environment variables were not set!")
		}

		// Setup Redis cache
		cache.Start(viper.GetString("cachehost")+":"+viper.GetString("cacheport"), viper.GetInt("cacheconnectionlimit"))
	}
}
