package cmd

import (
	"fmt"
	"os"

	"github.com/combust-labs/firebox/pkg/log"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

var logFormat string
var logLevel string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "firebox",
	Short: "Firecracker toolbox",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.firebox.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Logging config for all settings
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", log.DefaultFormat, "Log format. One of: [json, text]")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level. One of: [trace, debug, info, warn, fatal, panic")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".firebox" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".firebox")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func newLogger() *log.Logger {
	if logger, err := log.NewLogger(log.WithFormat(logFormat), log.WithLevel(logLevel)); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
		return nil
	} else {
		return logger
	}
}
