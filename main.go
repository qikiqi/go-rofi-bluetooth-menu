package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Define the struct
type Device struct {
	Name      string
	Connected bool
}

func getSymbol(status bool) string {
	if status {
		return "󰂱"
	}
	return "󰂲"
}

func runRofi(tempFileName string) (string, error) {
	cmd := exec.Command("rofi", "-dmenu", "-input", tempFileName, "-i", "-p", "Bluetooth", "-keep-right")
	output, err := cmd.Output()
	return string(output), err
}

// containsKey checks if the input string contains any key from the map
func validMAC(input string, devices map[string]Device) bool {
	for key := range devices {
		if strings.Contains(input, key) {
			log.Error().Msg("Found MAC")
			return true
		}
	}
	log.Error().Msg("MAC not found")
	return false
}

func runBluetoothctl(command string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo -e \"%s\" | sudo bluetoothctl", command))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Error running bluetoothctl:", err)
	}
	return out.String()
}

func sanitizeDevice(input string) []string {
	var devices []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Device") {
			line = strings.Replace(line, "Device ", "", 1)
			devices = append(devices, line)
		}
	}
	return devices
}

func getMacFromUserInput(input string) string {
	return strings.Fields(input)[1]
}

func getConnectAction(input string) string {
	if strings.Contains(string(input), "󰂱") {
		return "dis"
	} else {
		return ""
	}
}
func writeRofiTempfile(tempFile *os.File, allDevicesSorted []Device) {
	// Print the map entries
	for _, device := range allDevicesSorted {
		log.Info().Msgf("%s: %s", getSymbol(device.Connected), device.Name)
		tempFile.WriteString(fmt.Sprintf("%s: %s\n", getSymbol(device.Connected), device.Name))
	}
}

func connectDevice(mac string, disconnect string) {
	runBluetoothctl("power on")
	log.Error().Msgf("%sconnect %s", disconnect, mac)
	runBluetoothctl(fmt.Sprintf("%sconnect %s", disconnect, mac))
}

func sortDeviceMapByConnected(allDevices map[string]Device) []Device {
	deviceList := make([]Device, 0, len(allDevices))
	for _, device := range allDevices {
		deviceList = append(deviceList, device)
	}

	// Sort the slice based on the Connected status
	sort.Slice(deviceList, func(i, j int) bool {
		return deviceList[i].Connected && !deviceList[j].Connected
	})
	return deviceList
}

// Function to create the device map
func createDeviceMap(connected, paired []string) map[string]Device {
	allDevices := make(map[string]Device)

	// Add connected devices to the map
	for _, mac := range connected {
		allDevices[mac] = Device{Name: mac, Connected: true}
	}

	// Add paired devices to the map, if not already added
	for _, mac := range paired {
		if _, exists := allDevices[mac]; !exists {
			allDevices[mac] = Device{Name: mac, Connected: false}
		}
	}

	return allDevices
}

func main() {
	// Define a flag for the log level
	logLevel := flag.String("loglevel", "info", "set log level: debug, info, warn, error")
	versionFlag := flag.Bool("version", false, "Print the version information")
	vFlag := flag.Bool("v", false, "Print the version information (shorthand)")
	benchmarkFlag := flag.Bool("benchmark", false, "Launch in benchmark mode")

	flag.Parse()

	// Convert the log level string to a zerolog.Level
	level, err := zerolog.ParseLevel(strings.ToLower(*logLevel))
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}

	// Set the global log level
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	if *versionFlag || *vFlag {
		PrintVersion()
		os.Exit(0)
	}

	tempFile, err := os.CreateTemp("", "bluetooth")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Can't create tempfile")
		return
	}
	defer os.Remove(tempFile.Name())

	connected := sanitizeDevice(runBluetoothctl("devices Connected"))
	paired := sanitizeDevice(runBluetoothctl("devices"))

	allDevices := createDeviceMap(connected, paired)
	allDevicesSorted := sortDeviceMapByConnected(allDevices)

	writeRofiTempfile(tempFile, allDevicesSorted)

	if *benchmarkFlag {
		log.Info().Msg("Benchamrk run finished!")
		os.Exit(0)
	}

	userInput, err := runRofi(tempFile.Name())
	if err != nil {
		log.Error().Msgf("Error running rofi: %v", err)
		log.Error().Msgf("userInput: %s", userInput)
		return
	}

	if !validMAC(userInput, allDevices) {
		log.Error().Msg("Invalid user input (not found in paired devices MAC)")
		return
	}

	connectDevice(getMacFromUserInput(userInput), getConnectAction(userInput))
}
