package src

import (
	"server/pkg/util"
)

const (
	codeOK int = -iota
	codeUIDErr
	codeRIDErr
	codeMIDErr
	codeSIDErr
	codeJsepErr
	codeSdpErr
	codeMinfoErr
	codePubErr
	codeSubErr
	codeSfuErr
	codeIslbErr
	codeSfuRpcErr
	codeIslbRpcErr
	codeUnknownErr
)

var codeErr = map[int]string{
	codeOK:         "OK",
	codeUIDErr:     "uid not found",
	codeRIDErr:     "rid not found",
	codeMIDErr:     "mid not found",
	codeSIDErr:     "sid not found",
	codeJsepErr:    "jsep not found",
	codeSdpErr:     "sdp not found",
	codeMinfoErr:   "minfo not found",
	codePubErr:     "pub not found",
	codeSubErr:     "sub not found",
	codeSfuErr:     "sfu not found",
	codeIslbErr:    "islb not found",
	codeSfuRpcErr:  "sfu rpc not found",
	codeIslbRpcErr: "islb rpc not found",
	codeUnknownErr: "unknown error",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}

func invalid(msg map[string]interface{}, key string, reject RejectFunc) bool {
	val := util.Val(msg, key)
	if val == "" {
		switch key {
		case "uid":
			reject(codeUIDErr, codeStr(codeUIDErr))
			return true
		case "rid":
			reject(codeRIDErr, codeStr(codeRIDErr))
			return true
		case "mid":
			reject(codeMIDErr, codeStr(codeMIDErr))
			return true
		case "sid":
			reject(codeSIDErr, codeStr(codeSIDErr))
			return true
		case "jsep":
			reject(codeJsepErr, codeStr(codeJsepErr))
			return true
		case "sdp":
			reject(codeSdpErr, codeStr(codeSdpErr))
			return true
		}
	}
	return false
}
