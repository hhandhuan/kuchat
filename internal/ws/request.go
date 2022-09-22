package ws

import "encoding/json"

type MsgID struct {
	ID uint32 `json:"id"`
}

type Request struct {
	MsgID uint32
	Data  []byte
	Conn  *Connection
}

func (r *Request) GetConnection() *Connection {
	return r.Conn
}

func (r *Request) GetData() []byte {
	return r.Data
}

func (r *Request) GetMsgID() uint32 {
	return r.MsgID
}

func (r *Request) Parse(obj any) error {
	return json.Unmarshal(r.Data, &obj)
}
