package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"resin-supervisor/gosuper/systemd"
)

// Compile the expression once, usually at init time.
// Use raw strings to avoid having to quote the backslashes.
var dockerMatch = regexp.MustCompile(`(docker[0-9]+)|(rce[0-9]+)`)

// API response sent from gosupervisor
type APIResponse struct {
	Data  interface{}
	Error string
}

type PurgeBody struct {
	ApplicationId interface{}
}

func jsonResponse(writer http.ResponseWriter, response interface{}, status int) {
	jsonBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Could not marshal JSON for %+v\n", response)
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	writer.Write(jsonBody)
}

func parseJsonBody(destination interface{}, request *http.Request) error {
	decoder := json.NewDecoder(request.Body)
	return decoder.Decode(&destination)
}

func parsePurgeBody(request *http.Request) (appId string, err error) {
	var body PurgeBody
	if err = parseJsonBody(&body, request); err != nil {
		return
	}
	switch v := body.ApplicationId.(type) {
	case string:
		appId = v
	case float64:
		if v != 0 {
			appId = strconv.Itoa(int(v))
		}
	default:
		log.Printf("Invalid appId type %T\n", v)
	}
	return
}

func responseSender(writer http.ResponseWriter) func(interface{}, string, int) {
	return func(data interface{}, errorMsg string, statusCode int) {
		jsonResponse(writer, APIResponse{data, errorMsg}, statusCode)
	}
}

func PurgeHandler(writer http.ResponseWriter, request *http.Request) {
	log.Println("Purging /data")

	sendResponse := responseSender(writer)
	sendError := func(err error) {
		sendResponse("Error", err.Error(), http.StatusInternalServerError)
	}
	sendBadRequest := func(errorMsg string) {
		sendResponse("Error", errorMsg, http.StatusBadRequest)
	}

	if appId, err := parsePurgeBody(request); err != nil {
		sendBadRequest("Invalid request")
	} else if appId == "" {
		sendBadRequest("applicationId is required")
	} else if !IsValidAppId(appId) {
		sendBadRequest(fmt.Sprintf("Invalid applicationId '%s'", appId))
	} else if _, err = os.Stat(ResinDataPath + appId); err != nil {
		if os.IsNotExist(err) {
			sendResponse("Error", fmt.Sprintf("Invalid applicationId '%s': Directory does not exist", appId), http.StatusNotFound)
		} else {
			sendError(err)
		}
	} else if err = os.RemoveAll(ResinDataPath + appId); err != nil {
		sendError(err)
	} else if err = os.Mkdir(ResinDataPath+appId, 0755); err != nil {
		sendError(err)
	} else {
		sendResponse("OK", "", http.StatusOK)
	}
}

func inASecond(theFunc func()) {
	time.Sleep(time.Duration(time.Second))
	theFunc()
}

func RebootHandler(writer http.ResponseWriter, request *http.Request) {
	log.Println("Rebooting")

	sendResponse := responseSender(writer)
	sendResponse("OK", "", http.StatusAccepted)
	go inASecond(func() { systemd.Logind.Reboot(false) })
}

func ShutdownHandler(writer http.ResponseWriter, request *http.Request) {
	log.Println("Shutting down")

	sendResponse := responseSender(writer)
	sendResponse("OK", "", http.StatusAccepted)
	go inASecond(func() { systemd.Logind.PowerOff(false) })
}

// This function returns all active IPs of the interfaces that arent docker/rce and loopback
func ipAddress() (ipAddresses []string, err error) {

	ifaces, err := net.Interfaces()
	if err != nil {
		return ipAddresses, err
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp == 0) || (iface.Flags&net.FlagLoopback != 0) || dockerMatch.MatchString(iface.Name) {
			continue // Interface down or Interface is loopback or Interface is a docker IP
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ipAddresses, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				log.Printf("Warning: Unrecognised type %T\n", v)
				continue
			}
			if ip == nil {
				continue
			}
			if ip = ip.To4(); ip == nil {
				continue // This isnt an IPv4 Addresss
			}
			ipAddresses = append(ipAddresses, ip.String())
		}
	}
	return
}

//IPAddressHandler is used to reply back with an array of the IPaddress used by the system.
func IPAddressHandler(writer http.ResponseWriter, request *http.Request) {
	sendResponse := responseSender(writer)
	sendError := func(err string) {
		sendResponse("Error", err, http.StatusInternalServerError)
	}

	if ipAddr, err := ipAddress(); err != nil {
		sendError("Invalid request")
	} else {
		payload := make(map[string][]string)
		payload["IPAddresses"] = ipAddr
		sendResponse(payload, "", http.StatusOK)
	}
}
