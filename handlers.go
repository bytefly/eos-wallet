package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
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

func GetMemoHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		arg := r.URL.Query().Get("uid")
		if arg == "" {
			RespondWithError(w, 400, "missing uid")
			return
		}

		uid, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			RespondWithError(w, 500, fmt.Sprintf("invalid uid"))
			return
		}

		memo := CreateMemoByUID(uid)
		log.Println("create memo of", uid, ":", memo)
		Respond(w, 0, map[string]string{"memo": memo})
		return
	}
}

func GetBalanceHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var balance *big.Int
		var err error

		address := r.URL.Query().Get("address")
		if address == "" {
			address = config.Account
		}
		if !VerifyAddress(config, address) {
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

func SendEosHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEosHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		to := r.Form.Get("to")
		amount := r.Form.Get("amount")
		memo := r.Form.Get("memo")

		log.Println("send EOS to", to, "amount:", amount)
		if to == "" {
			log.Println("Got Send EOS order but to field is missing")
			RespondWithError(w, 400, "Missing to field")
			return
		}
		if !VerifyAddress(config, to) {
			log.Println("Invalid to address:", to)
			RespondWithError(w, 400, "Invalid to address")
			return
		}

		if amount == "" {
			log.Println("Got Send EOS order but 'amount' field is missing")
			RespondWithError(w, 400, "Missing 'amount' field")
			return
		}

		_, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			RespondWithError(w, 400, "Could not convert amount")
			return
		}

		bgAmountInt := new(big.Int)
		bgAmountInt.SetString(RightShift(amount, 4), 10)
		tx, err := SendEosCoin(config, to, bgAmountInt.Int64(), memo)
		if err != nil {
			log.Println("send EOS err:", err)
			RespondWithError(w, 500, fmt.Sprintf("Could not send EOS: %v", err))
			return
		}

		Respond(w, 0, map[string]string{"txhash": tx})
	}
}

func PrepareTrezorEosSignHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEosHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		to := r.Form.Get("to")
		amountStr := r.Form.Get("amount")
		memo := r.Form.Get("memo")

		log.Println("PrepareTrezorEosSign:", to, amountStr)
		if to == "" || amountStr == "" {
			log.Println("parameters not enought")
			RespondWithError(w, 400, "Missing some fields")
			return
		}

		amount, ok := new(big.Int).SetString(RightShift(amountStr, 4), 10)
		if !ok {
			log.Println("invalid amount")
			RespondWithError(w, 400, "Invalid amount")
			return
		}

		if !VerifyAddress(config, to) {
			log.Println("invalid to address:", to)
			RespondWithError(w, 400, "invalid to address")
			return
		}

		unsignedTx, err := PrepareTrezorEosSign(config, to, amount.Int64(), memo)
		if err != nil {
			RespondWithError(w, 500, fmt.Sprintf("prepare trezor Eos Sign err: %v", err))
		} else {
			Respond(w, 0, map[string]string{"unsignedTx": unsignedTx})
		}
	}
}

func SendSignedEosTxHandler(config *Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("SendEosHandler: Could not parse body parameters")
			RespondWithError(w, 400, "Could not parse parameters")
			return
		}

		amountStr := r.Form.Get("amount")
		to := r.Form.Get("to")
		memo := r.Form.Get("memo")
		sig := r.Form.Get("sig")

		log.Println("sendSignedEos:", to, amountStr)
		if to == "" || amountStr == "" || sig == "" {
			log.Println("parameters not enought")
			RespondWithError(w, 400, "Missing some fields")
			return
		}

		amount, ok := new(big.Int).SetString(RightShift(amountStr, 4), 10)
		if !ok {
			log.Println("invalid amount")
			RespondWithError(w, 400, "Invalid amount")
			return
		}

		if !VerifyAddress(config, to) {
			log.Println("invalid to address:", to)
			RespondWithError(w, 400, "invalid to address")
			return
		}

		hash, err := SendSignedEosTx(config, to, amount.Int64(), memo, sig)
		if err != nil {
			log.Println("send tx err:", err)
			RespondWithError(w, 500, fmt.Sprintf("send tx err: %v", err))
			return
		}
		Respond(w, 0, map[string]string{"hash": hash})
	}
}
