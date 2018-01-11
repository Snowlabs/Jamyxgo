package jamyxgo

import (
    "fmt"
    "encoding/json"
    "net"
    "strconv"
    "log"
)

type Command struct {
    Target string `json:"target"`
    Cmd string    `json:"cmd"`
    Opts []string `json:"opts"`
}

// Target holds the connection to the server.
type Target struct {
    ip string
    port int
}

// Constructor for Target
func NewTarget(ip string, port int) (*Target) {
    return &Target { ip, port, }
}

// Return connection to the jamyxer server.
func (target* Target) GetConnection() (net.Conn) {
    conn, err := net.Dial("tcp", target.ip+":"+strconv.Itoa(target.port))
    if err != nil {
        log.Fatal(err)
    }
    return conn
}

// Send a command to the jamyxer server.
func (target* Target) SendCommand(cmd Command) (reply interface {}) {
    conn := target.GetConnection()
    cmd_b, _ := json.Marshal(cmd)

    // log.Println("Sending command:", cmd_b)

    _, err := conn.Write(cmd_b)
    if err != nil {
        log.Fatal(err)
    }

    reply_b := make([]byte, 1024)
    numbytes, err := conn.Read(reply_b)
    if err != nil {
        log.Fatal(err)
    }
    if numbytes == 0 {
        log.Fatal("Numbytes is 0!")
    }

    var r interface {}
    err = json.Unmarshal(reply_b[:numbytes], &r)
    if err != nil {
		fmt.Println("error:", err)
	}
    log.Println(string(reply_b[:numbytes]), r)
    return r
}

func getTargetType(isinput bool) string {
    if isinput { return "i" }
    return "o"
}

func toStringArr(in []interface{}) ([]string) {
    result := make([]string, len(in))
    for i, e := range in{
        if e == nil {
            continue
        }
        result[i] = e.(string)
    }
    return result
}

// ==== Set Volume ====

// Set volume for specified input/output channel.
func (target *Target) VolumeSet(isinput bool, channel string, volume float32) {
    // target.SendCommand("v%ss \"%s\" %f\n", getTargetType(isinput), channel, volume)
    target.SendCommand(Command {
        Target: "myx",
        Cmd: "set",
        Opts: []string{"v", getTargetType(isinput), channel, fmt.Sprintf("%f", volume)}})
}
// Set volume for specified input channel.
func (target *Target) VolumeInputSet(input string, volume float32) { target.VolumeSet(true, input, volume) }
// Set volume for specified output channel.
func (target *Target) VolumeOutputSet(output string, volume float32) { target.VolumeSet(false, output, volume) }

// ==== Set Balance ====

// Set balance for specified input/output channel.
func (target *Target) BalanceSet(isinput bool, channel string, balance float32) {
    // target.SendCommand("v%ss \"%s\" %f\n", getTargetType(isinput), channel, balance)
    target.SendCommand(Command {
        Target: "myx",
        Cmd: "set",
        Opts: []string{"b", getTargetType(isinput), channel, fmt.Sprintf("%f", balance)}})
}
// Set balance for specified input channel.
func (target *Target) BalanceInputSet(input string, balance float32) { target.BalanceSet(true, input, balance) }
// Set balance for specified output channel.
func (target *Target) BalanceOutputSet(output string, balance float32) { target.BalanceSet(false, output, balance) }

// ==== Get Port ====

// Port object that holds all port info
type Port struct {
    Port string
    Ptype string
    IsInput bool
    IsMono bool
    Vol float32
    Bal float32
    Cons []string
    target *Target
}

// Extract Port from interface
func (target *Target) GetPortFromInterface(port map[string]interface{}) Port {
    return Port{
        Port:  port["port"].(string),
        Ptype: port["ptype"].(string),
        IsInput: port["ptype"].(string) == "in",
        IsMono: port["ismono"].(bool),
        Vol:   float32(port["vol"].(float64)),
        Bal:   float32(port["bal"].(float64)),
        Cons:  toStringArr(port["cons"].([]interface{})),
        target: target,
    }
}

// Extract Port object from a command reply
func (target *Target) GetPortFromReply(reply map[string]interface{}) (Port) {
    port := reply["obj"].(map[string]interface{})
    return target.GetPortFromInterface(port)
}

// Get Port object for specified input/output channel.
func (target *Target) GetPort(isinput bool, channel string) (Port) {
    reply := target.SendCommand(Command {
        Target: "myx",
        Cmd: "get",
        Opts: []string{getTargetType(isinput), channel},
    })

    return target.GetPortFromReply(reply.(map[string]interface{}))
}

// Return true is channel name was found in connections
func (port *Port) IsConnectedToChannel(channel string) bool {
    for _, i := range port.Cons {
        if i == channel { return true }
    }
    return false
}

// Return true is port was found in connections
func (port *Port) IsConnectedToPort(other Port) bool {
    return port.IsConnectedToChannel(other.Port)
}

// Update the port's properties in place
func (port *Port) Update() {
    *port = port.target.GetPort(port.IsInput, port.Port)
}

// Set the volume of port
func (port *Port) SetVol(vol float32) {
    port.target.VolumeSet(port.IsInput, port.Port, vol)
    port.Update()
}

// Set the balance of port
func (port *Port) SetBal(vol float32) {
    port.target.BalanceSet(port.IsInput, port.Port, vol)
    port.Update()
}

// Wait for volume of port to change and then return vol
func (port* Port) ListenVol() {
    *port = port.target.VolumeListen(port.IsInput, port.Port)
}

// Connect port with channel (other port name)
func (port *Port) ConnectToChannel(channel string) {
    if port.IsInput {
        port.target.ConnectIO(port.Port, channel)
    } else {
        port.target.ConnectIO(channel, port.Port)
    }
    port.Update()
}

// Connect two ports together
func (port *Port) ConnectToPort(other Port) {
    port.ConnectToChannel(other.Port)
    port.Update()
    other.Update() // might not do anything here since other is passed by value
}

// Disconnect port from channel (other port name)
func (port *Port) DisconnectFromChannel(channel string) {
    if port.IsInput {
        port.target.DisconnectIO(port.Port, channel)
    } else {
        port.target.DisconnectIO(channel, port.Port)
    }
}

// Disconnect two ports from eachother
func (port *Port) DisconnectFromPort(other Port) {
    port.DisconnectFromChannel(other.Port)
}

// Toggle Connection with channel (other port name)
func (port *Port) ToggleConnectionWithChannel(channel string) {
    if port.IsInput {
        port.target.ToggleConnectionIO(port.Port, channel)
    } else {
        port.target.ToggleConnectionIO(channel, port.Port)
    }
}

// Toggle Connection between two ports
func (port *Port) ToggleConnectionWithPort(other Port) {
    port.ToggleConnectionWithChannel(other.Port)
}

func (port *Port) SetMonitored() {
    port.target.SetMonitor(port.IsInput, port.Port)
    port.Update()
}

// ==== Get Channels ====

type Ports struct {
    Inputs  []Port
    Outputs []Port
}

// Returns an array of strings representing the names of the input/output channels.
func (target *Target) GetPorts() Ports {
    reply := target.SendCommand(Command { Target: "myx", Cmd: "get", Opts: []string{"channels"} })

    all_ports := make(map[string][]Port)
    all_unconverted_ports := reply.(map[string]interface{})["obj"].(map[string]interface{})
    for k, v := range all_unconverted_ports {
        ports := make([]Port, len(v.([]interface{})))
        for i, e := range v.([]interface{}) {
            ports[i] = target.GetPortFromInterface(e.(map[string]interface{}))
        }
        all_ports[k] = ports
    }

    return Ports { Inputs: all_ports["inputs"], Outputs: all_ports["outputs"] }
}

// ==== Set Connected ====

// Connect input with output
func (target *Target) ConnectIO(input, output string) {
    target.SendCommand(Command { Target: "myx", Cmd: "con", Opts: []string{input, output} })
}
// Toggle connection between input and output
func (target *Target) ToggleConnectionIO(input, output string) {
    target.SendCommand(Command { Target: "myx", Cmd: "tog", Opts: []string{input, output} })
}
// Disconnect input and output
func (target *Target) DisconnectIO(input, output string) {
    target.SendCommand(Command { Target: "myx", Cmd: "dis", Opts: []string{input, output} })
}

// ==== Get Connected ====

// Return true if output & input are connected
func (target *Target) GetConnectedIO(input, output string) bool {
    for _, i := range target.GetPort(false, output).Cons {
        if i == input { return true }
    }
    return false
}

// ==== Set Monitor ====
func (target *Target) SetMonitor(isinput bool, channel string) {
    target.SendCommand(Command { Target: "myx", Cmd: "set", Opts: []string{"monitor", getTargetType(isinput), channel}})
}

// ==== Get Monitor ====

func (target *Target) GetMonitorPort() Port {
    reply := target.SendCommand(Command { Target: "myx", Cmd: "get", Opts: []string{"monitor"} })
    return target.GetPortFromReply(reply.(map[string]interface{}))
}

// ==== Listeners ====
// Listen for volume change for specified channel.
// This is a blocking call waiting for a change in volume and returning it.
func (target *Target) VolumeListen(isinput bool, channel string) Port {
    reply := target.SendCommand(Command {Target: "myx", Cmd: "mon", Opts: []string{"vol", getTargetType(isinput), channel}})
    var port Port = target.GetPortFromReply(reply.(map[string]interface{}))

    return port
}
// Listen for volume for specified input channel.
// This is a blocking call waiting for a change in volume and returning it.
func (target *Target) VolumeInputListen(input string) Port { return target.VolumeListen(true, input) }
// Listen for volume for specified output channel.
// This is a blocking call waiting for a change in volume and returning it.
func (target *Target) VolumeOutputListen(output string) Port { return target.VolumeListen(false, output) }
