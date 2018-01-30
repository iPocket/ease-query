package query

import (
	"net"
	"bytes"
	"encoding/binary"
	"math/rand"
	"strings"
	"strconv"
	"fmt"
	"math"
)

var magic = []byte("\x00\xff\xff\x00\xfe\xfe\xfe\xfe\xfd\xfd\xfd\xfd\x12\x34\x56\x78")
var validDataFormat = "MCPE"

type bedrockResult struct {
	serverId uint64
	msgOfToday string
	onlineCount int32
	maxCount int32
	bedrockNetVer int32
	bedrockGameVer string
}

func (n *bedrockResult) GetServerId() uint64 {
	return n.serverId
}

func (n *bedrockResult) GetResultType() Type {
	return McBedrock
}

func (n *bedrockResult) GetMsgOfToday() string {
	return n.msgOfToday
}

func (n *bedrockResult) GetOnlineCount() int32 {
	return n.onlineCount
}

func (n *bedrockResult) GetMaxCount() int32 {
	return n.maxCount
}

func (n *bedrockResult) GetBedrockNetVer() int32 {
	return n.bedrockNetVer
}

func (n *bedrockResult) GetBedrockGameVer() string {
	return n.bedrockGameVer
}

func (n *bedrockResult) String() string {
	return fmt.Sprintf("BedrockResult{ServerId=%d, " +
		"NetVer=%d, GameVer=%s, MOTD=%s, OnlineCount=%d, MaxCount=%d}",
		n.serverId, n.bedrockNetVer, n.bedrockGameVer, n.msgOfToday, n.onlineCount, n.maxCount)
}

type bedrockConn struct {
	*conn
}

func (c bedrockConn) Pull() (Result, error) {
	return bedrockPullOfflinePingPong(c.addr)
}

func newBedrockConn(addr string) bedrockConn {
	return bedrockConn{
		&conn{
			McBedrock, addr,
		},
	}
}

// Pull RakNet UNCONNECTED_PONG packet
func bedrockPullOfflinePingPong(addr string) (Result, error) {
	var err error
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nilResult, err
	}
	defer func() {conn.Close()} ()

	reqPacket := new(bytes.Buffer)
	reqPacket.WriteByte(0x01)
	pingId :=  rand.Uint64()
	binary.Write(reqPacket, binary.BigEndian, pingId)
	reqPacket.Write(magic)
	binary.Write(reqPacket, binary.BigEndian, reqPacket.Len())
	conn.Write(reqPacket.Bytes())

	resBytes := make([]byte, 4096)
	n, err := conn.Read(resBytes)
	if err != nil {
		return nilResult, err
	}
	defer func() {conn.Close()} ()

	if n < 33 {
		return nil, errBedrockWrongLen
	}
	if resBytes[0] != 0x1c {
		return nil, errBedrockWrongMsgId
	}
	if !bytes.Equal(resBytes[1:9], reqPacket.Bytes()[1:9]) {
		return nil, errBedrockWrongPongId
	}
	sid := binary.BigEndian.Uint64(resBytes[9:17])
	if !bytes.Equal(resBytes[17:33], magic) {
		return nil, errBedrockWrongMagic
	}
	resLen := binary.BigEndian.Uint16(resBytes[33:35])
	if 35 + int(resLen) > n {
		return nil, errBedrockWrongLen
	}
	res := bytes.NewBuffer(resBytes[35:35+resLen]).String() // jump header
	arr := strings.Split(res, ";")
	if len(arr) < 5 {
		return nil, errBedrockWrongFmt
	}
	if arr[0] != validDataFormat {
		return nil, errBedrockWrongFmt
	}
	ret := &bedrockResult{
		serverId: sid,
		msgOfToday: arr[1],
		bedrockNetVer: strToInt32(arr[2], &err),
		bedrockGameVer: arr[3],
		onlineCount: strToInt32(arr[4], &err),
		maxCount: strToInt32(arr[5], &err),
	}
	return ret, err
}

func strToInt32(in string, errOut *error) int32 {
	o, err := strconv.Atoi(in)
	if err != nil {
		*errOut = err
	}
	if o > math.MaxInt32 {
		*errOut = errBedrockWrongFmt
	}
	return int32(o)
}
