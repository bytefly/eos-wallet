package main

import (
	"github.com/eoscanada/eos-go"
	"log"
	"math/big"
)

const (
	TYPE_BLOCK_HASH = iota
	TYPE_TXN_HASH
)

type ObjMessage struct {
	Type   int
	Hash   string
	Number *big.Int
}

var blkPool = make(map[uint64][]NotifyMessage)
var minAmount = new(big.Int).SetUint64(1000) //0.1000 EOS

func GetNewerBlock(config *Config, ch chan<- ObjMessage) error {
	api := eos.New(config.RPCURL)

	info, err := api.GetInfo()
	if err != nil {
		log.Println("get info err:", err)
		return err
	}

	bgInt := new(big.Int)
	bgInt.SetInt64(int64(info.HeadBlockNum))
	ch <- ObjMessage{TYPE_BLOCK_HASH, info.HeadBlockID.String(), bgInt}
	return nil
}

func Listener(config *Config, ch <-chan ObjMessage, notifyChannel chan<- NotifyMessage, last_id uint64) {
	for message := range ch {
		switch message.Type {
		case TYPE_BLOCK_HASH:
			// Recovery: We get a recent block, but the last we parsed is more older that current - 1
			last := new(big.Int)
			last.SetUint64(last_id)

			stop := new(big.Int)
			stop.Sub(message.Number, big.NewInt(1))
			//log.Printf("Recovery: Doing block %s - %s", last.Text(10), message.Number.String())
			for last.Cmp(message.Number) <= 0 {
				//time.Sleep(10000 * time.Millisecond)
				txns, err := ReadBlock(config, last)
				if err != nil {
					log.Println("Listener:", err)
					break
				}

				if last.Cmp(stop) < 0 {
					for _, txn := range txns {
						notifyChannel <- txn
					}

					notifyChannel <- NotifyMessage{
						MessageType: NOTIFY_TYPE_ADMIN,
						Amount:      last,
					}
				} else {
					//put it in map
					if len(txns) > 0 {
						blkPool[last.Uint64()] = txns
						log.Println("add txs to", last.Uint64(), "txs size:", len(txns))
					}
					//scan and broadcast 1 confirms
					txns, ok := blkPool[last.Uint64()-1]
					if ok {
						for _, txn := range txns {
							notifyChannel <- txn
						}

						notifyChannel <- NotifyMessage{
							MessageType: NOTIFY_TYPE_ADMIN,
							Amount:      new(big.Int).Sub(last, big.NewInt(1)),
						}

						// delete unconfirmed height txs map
						delete(blkPool, last.Uint64()-1)
						log.Println("delete txs in block", last.Uint64()-1)
					}
				}

				last.SetUint64(last.Uint64() + 1)
				config.LastBlock = last.Uint64()
			}

			//log.Printf("Recovery is over: Done up to block %s", last.Text(10))
			last_id = last.Uint64()
		}
	}
}

func Notifier(config *Config, ch <-chan NotifyMessage) {
	var (
		from string
		to   string
	)

	for message := range ch {
		if message.MessageType == NOTIFY_TYPE_NONE {
			continue
		}

		if message.MessageType == NOTIFY_TYPE_ADMIN {
			continue
		}

		from = message.AddressFrom
		to = message.AddressTo
		amount := LeftShift(message.Amount.String(), 4)
		symbol := "EOS"
		findFrom := from == config.Account
		findTo := to == config.Account
		fee := "0"

		if !findTo && !findFrom {
			continue
		}

		if findFrom {
			if findTo {
				log.Printf("token transfer within the same wallet (%s: %s -> %s %s)\n", symbol, from, to, amount)
			}
		}
		// handle confirmed wallet transaction
		if findTo && !findFrom && message.Memo != "" {
			log.Printf("%s %s tokens deposit to the wallet, %s -> %s, memo: %s tx: %s\n", symbol, amount, from, to, message.Memo, message.TxHash)
			if message.Amount.Cmp(minAmount) < 0 { //ignore tiny deposit
				log.Println("amount is too small, ignored")
			} else {
				uid, err := ParseMemoToUID(message.Memo)
				if err != nil {
					log.Println("user not found, ignored")
					continue
				}
				log.Println("user ID:", uid)
				// call the deposit interface
				storeTokenDepositTx(config, symbol, message.TxHash, message.Memo, amount)
			}
		} else if !findTo && findFrom {
			log.Printf("%s %s tokens withdraw from the wallet, %s -> %s, tx: %s fee: %s\n", symbol, amount, from, to, message.TxHash, fee)
			// call the withdraw interface
			storeTokenWithdrawTx(config, symbol, message.TxHash, to, amount, fee)
		}
	}
}
