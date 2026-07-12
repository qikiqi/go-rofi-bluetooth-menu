package program

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

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

func parseDevices(input string) []string {
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
	for _, device := range allDevicesSorted {
		log.Info().Msgf("%s: %s", symbol(device.Connected), device.Name)
		tempFile.WriteString(fmt.Sprintf("%s: %s\n", symbol(device.Connected), device.Name))
	}
}

func sortByConnected(allDevices map[string]Device) []Device {
	deviceList := make([]Device, 0, len(allDevices))
	for _, device := range allDevices {
		deviceList = append(deviceList, device)
	}

	sort.Slice(deviceList, func(i, j int) bool {
		return deviceList[i].Connected && !deviceList[j].Connected
	})
	return deviceList
}

func mergeDevices(connected, paired []string) map[string]Device {
	allDevices := make(map[string]Device)

	for _, mac := range connected {
		allDevices[mac] = Device{Name: mac, Connected: true}
	}

	for _, mac := range paired {
		if _, exists := allDevices[mac]; !exists {
			allDevices[mac] = Device{Name: mac, Connected: false}
		}
	}

	return allDevices
}
