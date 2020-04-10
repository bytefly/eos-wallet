package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/token"
)

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
