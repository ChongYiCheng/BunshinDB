package main

import (
    // "encoding/json"
    "log"
    "fmt"
    // badger "github.com/dgraph-io/badger"
    // "net/http"
    "os"
    // "os/exec"
    // "strings"
    "bufio"
    // "bytes"
    "net"
    "errors"
    // glog "github.com/golang/glog"
)



type ShoppingCart struct {
    Items []string
    Vector_clock [][]string
}

type Message struct{
    SenderIP string
    SenderPort string
    MessageType string
    Data map[string][]byte //Key-Value pair. Value is byte-array representation of ShoppingCart
    VectorClock [][]string
    Query string //Just a key string for receiver to query
    ResponseCode string //200,404 etc.
}



func handle(err interface{}){
	if err != nil{
		log.Fatal(err)
	}
}


func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func parseCommandLine(command string) ([]string, error) {
	//Finite state machine to handle arguments with white spaces enclosed within quotes
	//Handles escaped stuff too
    var args []string
    state := "start"
    current := ""
    quote := "\""
    escapeNext := true
    for i := 0; i < len(command); i++ {
        c := command[i]

        if state == "quotes" {
            if string(c) != quote {
                current += string(c)
            } else {
                args = append(args, current)
                current = ""
                state = "start"
            }
            continue
        }

        if (escapeNext) {
            current += string(c)
            escapeNext = false
            continue
        }

        if (c == '\\') {
            escapeNext = true
            continue
        }

        if c == '"' || c == '\'' {
            state = "quotes"
            quote = string(c)
            continue
        }

        if state == "arg" {
            if c == ' ' || c == '\t' {
                args = append(args, current)
                current = ""
                state = "start"
            } else {
                current += string(c)
            }
            continue
        }

        if c != ' ' && c != '\t' {
            state = "arg"
            current += string(c)
        }
    }

    if state == "quotes" {
        return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
    }

    if current != "" {
        args = append(args, current)
    }

    return args, nil
}



func main(){
    if len(os.Args) != 4{
        fmt.Printf("Usage of program is: %s , <client/server> <PORT> <DBPath>\r\n", os.Args[0])
        os.Exit(0)
    }

    serverList := []string{"8005", "8006", "8007"}

    if os.Args[1] == "client" {
        client := &Client{}
        currentIP, err := externalIP()
        client.ip = currentIP
        fmt.Printf("Setting Client's IP to be %s\r\n", client.ip)
        handle(err)
        client.port = os.Args[2]
        client.serverList = serverList

        // go client.Start()
        // //Start of CLI interactivity
        reader := bufio.NewReader(os.Stdin)
        fmt.Printf("Node@%s:%s$ ",client.ip,client.port)
        for {
            fmt.Printf("Node@%s:%s$ ",client.ip,client.port)
            cmdString, err := reader.ReadString('\n')
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
            }
            err = client.runCommand(cmdString)
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
            }
        }
    } else if os.Args[1] == "server" {
        server := &Server{}
        server.DBPath = os.Args[3]
        currentIP, err := externalIP()
        server.ip = currentIP
        fmt.Printf("Setting Server's IP to be %s\r\n",server.ip)
        handle(err)
        server.port = os.Args[2]
        server.serverList = serverList

        go server.Start()
        //Start of CLI interactivity
        reader := bufio.NewReader(os.Stdin)
        fmt.Printf("Node@%s:%s$ ",server.ip,server.port)
        for {
            fmt.Printf("Node@%s:%s$ ",server.ip,server.port)
            cmdString, err := reader.ReadString('\n')
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
            }
            err = server.runCommand(cmdString)
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
            }
        }
    }
    	
}

