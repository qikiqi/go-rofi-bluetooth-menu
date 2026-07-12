package program

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/qikiqi/go-rofi-bluetooth-menu/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Run executes the rofi Bluetooth menu.
func Run() {
	logLevel := flag.String("loglevel", "info", "set log level: debug, info, warn, error")
	versionFlag := flag.Bool("version", false, "Print the version information")
	vFlag := flag.Bool("v", false, "Print the version information (shorthand)")
	benchmarkFlag := flag.Bool("benchmark", false, "Launch in benchmark mode")

	flag.Parse()

	level, err := zerolog.ParseLevel(strings.ToLower(*logLevel))
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	if *versionFlag || *vFlag {
		version.PrintVersion()
		os.Exit(0)
	}

	ctx := context.Background()
	bt := bluetoothctlRunner{}
	menu := rofiMenu{}

	tempFile, err := os.CreateTemp("", "bluetooth")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Can't create tempfile")
		return
	}
	defer os.Remove(tempFile.Name())

	connected := parseDevices(bt.Run(ctx, "devices Connected"))
	paired := parseDevices(bt.Run(ctx, "devices"))

	allDevices := mergeDevices(connected, paired)
	allDevicesSorted := sortByConnected(allDevices)

	writeRofiTempfile(tempFile, allDevicesSorted)

	if *benchmarkFlag {
		log.Info().Msg("Benchamrk run finished!")
		os.Exit(0)
	}

	userInput, err := menu.Prompt(ctx, tempFile.Name())
	if err != nil {
		log.Error().Msgf("Error running rofi: %v", err)
		log.Error().Msgf("userInput: %s", userInput)
		return
	}

	if !validMAC(userInput, allDevices) {
		log.Error().Msg("Invalid user input (not found in paired devices MAC)")
		return
	}

	connectDevice(ctx, bt, getMacFromUserInput(userInput), getConnectAction(userInput))
}
