package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/btcsuite/btcd/btcec"
	"github.com/eoscanada/eos-go/btcsuite/btcutil"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/eoscanada/eos-go/token"
)

type TransferData struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Quantity string `json:"quantity"`
	Memo     string `json:"memo"`
}

type Auth struct {
	Actor      string `json:"actor"`
	Permission string `json:"permission"`
}

type TransferAction struct {
	Account       string `json:"account"`
	Authorization []Auth `json:"authorization"`
	Name          string `json:"name"`
	TransferData  `json:"data"`
}

type Transaction struct {
	ChainId string           `json:"chainId"`
	Actions []TransferAction `json:"actions"`
}

type EosTrezorTx struct {
	Path        string       `json:"path"`
	Transaction *Transaction `json:"transaction"`
}

func GetAddressBalance(config *Config, address string) (*big.Int, error) {
	api := eos.New(config.RPCURL)
	bals, err := api.GetCurrencyBalance(eos.AccountName(address), "EOS", "eosio.token")
	if err != nil {
		return nil, err
	}

	bgInt := new(big.Int)
	if len(bals) > 0 {
		bgInt.SetInt64(int64(bals[0].Amount))
	}
	return bgInt, nil
}

func ParseTransaction(tx *eos.SignedTransaction, id string, ts int64) []NotifyMessage {
	var ans []NotifyMessage

	for _, action := range tx.Transaction.Actions {
		account := action.Account
		name := action.Name
		if name == "transfer" && account == "eosio.token" {
			transfer, _ := action.ActionData.Data.(*token.Transfer)
			from := string(transfer.From)
			dest := string(transfer.To)
			quantity := transfer.Quantity
			memo := transfer.Memo
			//filter empty memo and long memo(longer than 12)
			if quantity.Symbol.Symbol == "EOS" && memo != "" && len(memo) <= 12 {
				if fDebug {
					log.Printf("EOS: %s => %s / Value: %d Memo: %s\n", from, dest, int64(quantity.Amount), memo)
				}

				ans = append(ans, NotifyMessage{
					MessageType: NOTIFY_TYPE_TX,
					AddressFrom: from,
					AddressTo:   dest,
					Amount:      big.NewInt(int64(quantity.Amount)),
					Memo:        memo,
					TxHash:      id,
					BlockTime:   ts,
				})
			}
		}
	}

	return ans
}

func ReadBlock(config *Config, number *big.Int) ([]NotifyMessage, error) {
	var err error
	var messages []NotifyMessage

	api := eos.New(config.RPCURL)
	block, err := api.GetBlockByNum(uint32(number.Uint64()))
	if err != nil {
		return messages, fmt.Errorf("ReadBlock failed: %v", err)
	}

	for _, tx := range block.SignedBlock.Transactions {
		status := tx.TransactionReceiptHeader.Status.String()
		if status != "executed" {
			log.Println("tx is bad:", status)
		} else {
			id := tx.Transaction.ID.String()
			if tx.Transaction.Packed == nil {
				fmt.Println("packed is nil")
				continue
			}

			signedTx, err := tx.Transaction.Packed.Unpack()
			if err == nil && packHash == "" || packHash == id {
				msgs := ParseTransaction(signedTx, id, block.SignedBlock.SignedBlockHeader.Timestamp.Time.Unix())
				if len(msgs) > 0 {
					messages = append(messages, msgs...)
				}
			}
		}
	}

	return messages, nil
}

func VerifyAddress(addr string) bool {
	if len(addr) > 12 {
		return false
	}
	for _, b := range addr {
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '5') || b == '.' {
			continue
		}
		return false
	}

	_, err := eos.StringToName(addr)
	if err != nil {
		return false
	}

	return true
}

func SendEosCoin(config *Config, to string, amount int64, memo string) (string, error) {
	wif, _ := ExtractPrivPubKey(config.Xpriv, 0)
	keyBag := eos.NewKeyBag()
	keyBag.Add(wif)

	api := eos.New(config.RPCURL)
	api.SetSigner(keyBag)

	actions := []*eos.Action{token.NewTransfer(eos.AccountName(config.Account), eos.AccountName(to), eos.NewEOSAsset(amount), memo)}
	tx := eos.NewTransaction(actions, nil)
	chainId, _ := hex.DecodeString(config.EosId)
	rsp, err := api.SignPushTransaction(tx, eos.Checksum256(chainId), eos.CompressionZlib)

	if rsp == nil {
		return "", err
	}
	return rsp.TransactionID, err
}

func PrepareTrezorEosSign(config *Config, to string, amount int64, memo string) (string, error) {
	trezorTx := &EosTrezorTx{
		//hardcode, change it if needed
		Path: "m/44'/194'/0'/1/0",
		Transaction: &Transaction{
			ChainId: config.EosId,
			Actions: []TransferAction{
				TransferAction{
					Account: "eosio.token",
					Name:    "transfer",
					Authorization: []Auth{
						Auth{
							Actor:      config.Account,
							Permission: "active",
						},
					},
					TransferData: TransferData{
						From:     config.Account,
						To:       to,
						Quantity: eos.NewEOSAsset(amount).String(),
						Memo:     memo,
					},
				},
			},
		},
	}

	bs, _ := json.Marshal(&trezorTx)
	return string(bs), nil
}

func SendSignedEosTx(config *Config, to string, amount int64, memo string, sig string) (string, error) {
	actions := []*eos.Action{token.NewTransfer(eos.AccountName(config.Account), eos.AccountName(to), eos.NewEOSAsset(amount), memo)}
	stx := eos.NewSignedTransaction(eos.NewTransaction(actions, nil))
	signature, _ := ecc.NewSignature(sig)
	stx.Signatures = append(stx.Signatures, signature)
	packedTx, _ := stx.Pack(eos.CompressionZlib)

	api := eos.New(config.RPCURL)
	rsp, err := api.PushTransaction(packedTx)

	if rsp == nil {
		return "", err
	}
	return rsp.TransactionID, err
}

func ExtractPrivPubKey(xpriv string, index int) (wif, pkStr string) {
	masterKey, err := hdkeychain.NewKeyFromString(xpriv)
	if err != nil {
		return
	}

	acctExt, err := masterKey.Child(uint32(index))
	if err != nil {
		return
	}
	privKey, _ := acctExt.ECPrivKey()

	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), privKey.Serialize())
	wifObj, _ := btcutil.NewWIF(priv, 0x80, false)
	eccPub, _ := ecc.NewPublicKeyFromData(append([]byte{0x00}, pub.SerializeCompressed()...))

	wif, pkStr = wifObj.String(), eccPub.String()
	return
}
