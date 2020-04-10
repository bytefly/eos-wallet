package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
)

var m sync.Mutex

func Respond(w http.ResponseWriter, code int, payload interface{}) {
	ret := make(map[string]interface{})
	ret["Code"] = code
	if code >= 0 && code < 300 {
		ret["Msg"] = "Success"
	} else {
		ret["Msg"] = "Failure"
	}

	if payload != nil {
		ret["Data"] = payload
	}

	response, _ := json.Marshal(ret)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	Respond(w, code, map[string]string{"error": msg})
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("404: %s %s\n", r.Method, r.URL)
	RespondWithError(w, 404, "Not found")
}

func GetBalanceHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var balance *big.Int
		var err error

		address := r.URL.Query().Get("address")
		if address == "" {
			RespondWithError(w, 400, "Missing 'address' field")
			return
		}
		if !VerifyAddress(address) {
			log.Println("Invalid address:", address)
			RespondWithError(w, 400, "Invalid address")
			return
		}

		// Retrieve EOS balance
		balance, err = GetAddressBalance(config, address)
		if err != nil {
			log.Println("get eos balance of", address, "err:", err)
			RespondWithError(w, 500, fmt.Sprintf("Could not retrieve EOS balance: %v", err))
			return
		}

		log.Println("get eos balance of", address, ":", balance.String())
		Respond(w, 0, map[string]string{"balance": LeftShift(balance.String(), 4)})
		return
	}
}

/*
func SendEthHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	isWithdraw := false
	branch := 0
	var index int
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEthHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		from := ""
		to := r.Form.Get("to")
		amount := r.Form.Get("amount")

		if from == "" {
			from, err = CreateNewAddress(config.Xpub, 1, 0)
			if err != nil {
				log.Println("create from address error: ", err)
				RespondWithError(w, 500, "Couldn't get from address")
				return
			}
			isWithdraw = true
			branch = 1
			index = 0
		}

		if to == "" {
			log.Println("Got Send Ethereum order but to field is missing")
			RespondWithError(w, 400, "Missing to field")
			return
		} else {
			_, ok := addrs.Load(to[2:])
			if ok {
				log.Println("to address is in our wallet")
				RespondWithError(w, 500, "Couldn't launch transfering within the same wallet")
				return
			}
		}

		if !VerifyAddress(config, from) || !VerifyAddress(config, to) {
			log.Println("Invalid from/to address:", from, to)
			RespondWithError(w, 400, "Invalid from/to address")
			return
		}

		if !isWithdraw {
			v, ok := addrs.Load(from[2:])
			if !ok {
				log.Println("from address is not in our wallet")
				RespondWithError(w, 400, "Invalid from address")
				return
			}
			index = int(v.(uint32))
		}
		log.Println("send eth from", from, "to", to, "amount:", amount)

		private, err := GetPrivateKey(config.Xpriv, branch, index)
		if err != nil {
			log.Println("get private key fail")
			RespondWithError(w, 500, "get pirvate key fail")
			return
		}

		if amount == "" {
			log.Println("Got Send Ethereum order but 'amount' field is missing")
			RespondWithError(w, 400, "Missing 'amount' field")
			return
		}

		_, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			RespondWithError(w, 400, "Could not convert amount")
			return
		}

		bgAmountInt := new(big.Int)
		bgAmountInt.SetString(RightShift(amount, 18), 10)
		tx, err := SendEthCoin(config, bgAmountInt, private, to, nil, nil)
		if err != nil {
			log.Println("send eth err:", err)
			RespondWithError(w, 500, fmt.Sprintf("Could not send Ethereum coin: %v", err))
			return
		}

		Respond(w, 0, map[string]string{"txhash": tx})
	}
}

func PrepareTrezorEthSignHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEthHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		from := r.Form.Get("from")
		amountStr := r.Form.Get("amount")
		to := r.Form.Get("to")

		log.Println("PrepareTrezorEthSign:", from, to, amountStr)
		if from == "" || amountStr == "" {
			log.Println("parameters not enought")
			RespondWithError(w, 400, "Missing some fields")
			return
		}

		amount, ok := new(big.Int).SetString(RightShift(amountStr, 18), 10)
		if !ok {
			log.Println("invalid amount")
			RespondWithError(w, 400, "Invalid amount")
			return
		}

		if to != "" && !VerifyAddress(config, to) {
			log.Println("invalid to address:", to)
			RespondWithError(w, 400, "invalid to address")
			return
		} else if to == "" {
			to, _ = CreateNewAddress(config.Xpub, 1, 0)
		}

		unsignedTx, err := PrepareTrezorEthSign(config, from, amount, to)
		if err != nil {
			RespondWithError(w, 500, fmt.Sprintf("prepare trezor Eth Sign err: %v", err))
		} else {
			Respond(w, 0, map[string]string{"unsignedTx": unsignedTx})
		}
	}
}

func SendSignedEthTxHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEthHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		from := r.Form.Get("from")
		amountStr := r.Form.Get("amount")
		to := r.Form.Get("to")
		R := r.Form.Get("r")
		S := r.Form.Get("s")
		V := r.Form.Get("v")

		log.Println("sendSignedEth:", from, to, amountStr)
		if from == "" || amountStr == "" ||
			R == "" || S == "" || V == "" {
			log.Println("parameters not enought")
			RespondWithError(w, 400, "Missing some fields")
			return
		}

		amount, ok := new(big.Int).SetString(RightShift(amountStr, 18), 10)
		if !ok {
			log.Println("invalid amount")
			RespondWithError(w, 400, "Invalid amount")
			return
		}

		rbs, err := hex.DecodeString(R[2:])
		if err != nil {
			log.Println("invalid r sig")
			RespondWithError(w, 400, "Invalid sig")
			return
		}
		sbs, err := hex.DecodeString(S[2:])
		if err != nil {
			log.Println("invalid s sig")
			RespondWithError(w, 400, "Invalid sig")
			return
		}
		vbs, err := hex.DecodeString(V[2:])
		if err != nil {
			log.Println("invalid v sig")
			RespondWithError(w, 400, "Invalid sig")
			return
		}

		if !VerifyAddress(config, from) {
			log.Println("invalid from address:", to)
			RespondWithError(w, 400, "invalid from address")
			return
		}

		if to != "" && !VerifyAddress(config, to) {
			log.Println("invalid to address:", to)
			RespondWithError(w, 400, "invalid to address")
			return
		} else if to == "" {
			to, _ = CreateNewAddress(config.Xpub, 1, 0)
		}

		vbs[0] -= 35
		vbs[0] -= byte(config.ChainId.Uint64() << 1)
		sig := make([]byte, 0)
		sig = append(sig, rbs...)
		sig = append(sig, sbs...)
		sig = append(sig, vbs...)
		hash, err := SendSignedEthTx(config, from, amount, to, sig)
		if err != nil {
			log.Println("send tx err:", err)
			RespondWithError(w, 500, fmt.Sprintf("send tx err: %v", err))
			return
		}
		Respond(w, 0, map[string]string{"hash": hash})
	}
}
*/
