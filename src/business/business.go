package business

import (
	"calldb"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//	"errmsg"
//	"strconv"
type msg_info struct {
	Header json.RawMessage `json:"header"`
	Body   json.RawMessage `json:"body"`
	Mac    string          `json:"mac"`
}

type msg_head struct {
	TransCode string `json:"transCode"`
	PartnerID string `json:"partnerID"`
	MessageID string `json:"messageID"`
	Timestamp string `json:"timestamp"`
}

func pack_msg(header, body, passwd string) []byte {
	head_body := fmt.Sprintf("\"header\":%s,\"body\":%s%s", header, body, passwd)
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(head_body))
	cipherStr := md5Ctx.Sum(nil)
	fmt.Println(hex.EncodeToString(cipherStr))
	content := fmt.Sprintf("{\"header\":%s,\"body\":%s,\"mac\":%s}", header, body, hex.EncodeToString(cipherStr))
	protocol := fmt.Sprintf("HTTP/1.1 200 OK \r\nContent-Length: %d\r\n\r\n%s", len(content), content)
	return []byte(protocol)

}

func pack_param_content(proc_anser, err_msg string, ret_code int) string {
	var re_content string
	parse_content := strings.Split(proc_anser, "$")
	parse_len := len(parse_content)
	gameinfo := strings.Split(parse_content[parse_len-1], "#")

	if ret_code != 0 {
		re_content = fmt.Sprintf("{\"ErrCode\":%d,\"retMsg\":\"%s\"}", ret_code, err_msg)
	} else {
		re_content = fmt.Sprintf("{\"ErrCode\":%d,\"WLSSTinfo\":\"%s\",\"FTPIPAddress\":\"%s\",\"FTPPort\":%s,\"FTPUser\":\"%s\",\"FTPPassword\":\"%s\",\"FTPPath\":\"%s\",\"UPG_FTPIPAddress\":\"%s\",\"UPG_FTPPort\":%s,\"UPG_FTPUser\":\"%s\",\"UPG_FTPPassword\":\"%s\",\"UPG_FTPPath\":\"%s\"",
			ret_code, parse_content[0], parse_content[1], parse_content[2], parse_content[3], parse_content[4], parse_content[5], parse_content[6], parse_content[7], parse_content[8], parse_content[9], parse_content[10])
	}

	game_count := len(gameinfo)
	var str []string = make([]string, game_count)
	for i := 0; i < game_count; i++ {
		game := strings.Split(gameinfo[i], "&")

		str[i] = fmt.Sprintf("{\"GameCode\":\"%s\",\"MaxBetQuantity\":%s,\"MaxMul\":%s,\"ShortTermGame\":%s,\"IssueStarttime\":\"%s\",\"IssueInterval\":%s,\"IssueMax\":%s,\"IssuePrizing\":%s}", game[0], game[1], game[2], game[3], game[4], game[5], game[6], game[7])
		if i == 0 {
			re_content += ",\"GameInfo\":[" + str[i] + ","
		} else if i == game_count-1 {
			re_content += str[i] + "]}"
		} else {
			re_content += str[i] + ","
		}

	}
	return re_content
}

func proc_get_param(msginfo msg_info, h msg_head) ([]byte, error) {
	content := struct {
		//Deviceid string `json:"DeviceID"`
		Content json.RawMessage `json:content`
	}{}
	parm := struct {
		Deviceid string `json:"DeviceID"`
	}{}
	var body_msg string
	err := json.Unmarshal(msginfo.Body, &content)
	if err != nil {
		fmt.Printf("json unmarshal error!,err:%v\n", err)
	}
	fmt.Printf("content:%s\n", string(content.Content))
	err = json.Unmarshal(content.Content, &parm)
	if err != nil {
		fmt.Printf("json unmarshal error!,err11111:%v\n", err)
	}
	fmt.Printf("parm:%s\n", parm.Deviceid)
	out_ret, out_ansStr, out_sqlCode, out_errMsg, err := calldb.Call_procedure("wlas_getparam", parm.Deviceid)
	if err != nil {
		fmt.Printf("calldb.Call_procedure is error!")
	}
	fmt.Println(out_ret, out_ansStr, out_sqlCode, out_errMsg)
	now := time.Now()
	year, mon, day := now.Date()
	hour, min, sec := now.Clock()
	curr_time := fmt.Sprintf("%d%d%d%d%d%d", year%100, mon, day, hour, min, sec)
	//anser := fmt.Sprintf("HTTP/1.1 200 OK \r\nContent-Length: %d\r\n\r\n%s", len(out_ansStr), out_ansStr)
	head_msg := fmt.Sprintf("{\"transCode\":\"%s\",\"partnerID\":\"%s\",\"messageID\":\"%s\",\"timestamp\":\"%s\"}", "5001", "100001", "17120808010100000000001", curr_time)
	ret_content := pack_param_content(out_ansStr, out_errMsg, out_ret)
	fmt.Println(ret_content)
	body_msg = fmt.Sprintf("{\"content\":%s}", ret_content)
	fmt.Println(body_msg)
	/*
		if out_ret == 0 {
			//out_errMsg = errmsg.ERR_SUCCESS
			body_msg = fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"retMsg\":\"%s\",\"other\":{\"%s\"}}", out_ret, out_errMsg, out_ansStr)
		} else {

			body_msg = fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"retMsg\":\"%s\",\"other\":{\"%s\"}}", out_ret, out_errMsg, out_ansStr)
		}
	*/

	//body_msg := fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"retMsg\":\"%s\",\"DateTime\":\"%s\"}}", out_ret, out_ansStr)
	anser := pack_msg(head_msg, body_msg, "1a2b3c")
	fmt.Println(string(anser))
	return anser, err
}

func proc_get_time(msginfo msg_info, h msg_head) ([]byte, error) {
	var body_msg string
	out_ret, out_ansStr, out_sqlCode, out_errMsg, err := calldb.Call_procedure("wlas_gettime", "")
	if err != nil {
		fmt.Printf("calldb.Call_procedure is error!")
	}
	fmt.Println(out_ret, out_ansStr, out_sqlCode, out_errMsg)

	now := time.Now()
	year, mon, day := now.Date()
	hour, min, sec := now.Clock()
	curr_time := fmt.Sprintf("%d%d%d%d%d%d", year%100, mon, day, hour, min, sec)
	//anser := fmt.Sprintf("HTTP/1.1 200 OK \r\nContent-Length: %d\r\n\r\n%s", len(out_ansStr), out_ansStr)
	head_msg := fmt.Sprintf("{\"transCode\":\"%s\",\"partnerID\":\"%s\",\"messageID\":\"%s\",\"timestamp\":\"%s\"}", "5001", "100001", "17120808010100000000001", curr_time)
	if out_ret == 0 {
		/* 成功 不返回错误描述信息retMsg */
		body_msg = fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"DateTime\":\"%s\"}}", out_ret, out_ansStr)
	} else {
		body_msg = fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"retMsg\":\"%s\"}}", out_ret, out_errMsg)
	}
	//body_msg := fmt.Sprintf("{\"content\":{\"ErrCode\":\"%d\",\"retMsg\":\"%s\",\"DateTime\":\"%s\"}}", out_ret, out_ansStr)
	anser := pack_msg(head_msg, body_msg, "1a2b3c")
	return anser, err
}

func Transaction(message []byte) ([]byte, error) {
	var anser []byte
	msg := msg_info{}
	err := json.Unmarshal(message, &msg)
	if err != nil {
		fmt.Printf("json unmarshal error!,err:%v\n", err)
	}
	//fmt.Printf("msg:%v,%v,%s\n", msg.Header, msg.Body, msg.Mac)
	h := msg_head{}
	err = json.Unmarshal(msg.Header, &h)
	if err != nil {
		fmt.Printf("json unmarshal error!,err:%v\n", err)
	}
	//fmt.Printf("msg:%v\n", h)
	/* get time */
	if h.TransCode == "1001" {
		anser, err = proc_get_time(msg, h)
		//return anser, err
	} else if h.TransCode == "1002" {
		anser, err = proc_get_param(msg, h)
	}
	return anser, err
}
