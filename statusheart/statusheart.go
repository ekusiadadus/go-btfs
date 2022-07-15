package statusheart

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/bittorrent/go-btfs/spin"
	"github.com/bittorrent/go-btfs/statusheart/abi"
	"github.com/bittorrent/go-btfs/transaction"
	"github.com/ethereum/go-ethereum/common"

	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("status-heart:report")

var (
	statusABI = transaction.ParseABIUnchecked(abi.StatusHeartABI)
	serv      *service
)

const (
	ReportStatusTime   = 10 * time.Second
	statusHeartAddress = "0xE42016a68511BFfdcE74E04DD35DCD7bf75582c8"
)

func Init(transactionService transaction.Service) error {
	var currentAddress common.Address
	if statusHeartAddress == "" {
		return errors.New("no known status heart address for this network")
	} else {
		currentAddress = common.HexToAddress(statusHeartAddress)
	}

	statusHeart := New(currentAddress, transactionService)
	err := statusHeart.CheckReportStatus() // CheckReport when node starts
	if err != nil {
		return err
	}
	return nil
}

type service struct {
	statusAddress      common.Address
	transactionService transaction.Service
}

type Service interface {
	// ReportStatus report status heart info to statusContract
	ReportStatus() (common.Hash, error)

	// CheckReportStatus check report status heart info to statusContract
	CheckReportStatus() error
}

func New(statusAddress common.Address, transactionService transaction.Service) Service {
	serv = &service{
		statusAddress:      statusAddress,
		transactionService: transactionService,
	}

	go func() {
		cycleCheckReport()
	}()
	return serv
}

//report heart status
func (s *service) ReportStatus() (common.Hash, error) {
	if len(spin.GSignedInfo.Peer) <= 0 {
		return common.Hash{}, nil
	}

	peer := spin.GSignedInfo.Peer
	createTime := spin.GSignedInfo.CreatedTime
	version := spin.GSignedInfo.Version
	num := spin.GSignedInfo.Nonce
	bttcAddress := common.HexToAddress(spin.GSignedInfo.BttcAddress)
	signedTime := spin.GSignedInfo.SignedTime
	signature, err := hex.DecodeString(strings.Replace(spin.GSignedInfo.Signature, "0x", "", -1))
	fmt.Println("...... ReportHeartStatus, param = ", peer, createTime, version, num, bttcAddress, signedTime, signature)

	callData, err := statusABI.Pack("reportStatus", peer, createTime, version, num, bttcAddress, signedTime, signature)
	if err != nil {
		return common.Hash{}, err
	}
	fmt.Println("...... ReportHeartStatus, callData, err = ", callData, err)

	request := &transaction.TxRequest{
		To:          &s.statusAddress, //&statusAddress,
		Data:        callData,
		Value:       big.NewInt(0),
		Description: "Report Heart Status",
	}

	txHash, err := s.transactionService.Send(context.Background(), request)
	fmt.Println("...... ReportHeartStatus, txHash, err = ", txHash, err)
	if err != nil {
		return common.Hash{}, err
	}

	// WaitForReceipt takes long time
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("ReportHeartStatus recovered:%+v", err)
			}
		}()
	}()
	return txHash, nil
}

// report heart status
func (s *service) genHashExt(ctx context.Context) (common.Hash, error) {
	peer := "1"
	createTime := uint32(1)
	version := "1"
	num := uint32(3)
	bttcAddress := "0x22df207EC3C8D18fEDeed87752C5a68E5b4f6FbD"

	fmt.Println("...... genHashExt, param = ", peer, createTime, version, num, bttcAddress)

	callData, err := statusABI.Pack("genHashExt", peer, createTime, version, num, common.HexToAddress(bttcAddress))
	if err != nil {
		return common.Hash{}, err
	}

	request := &transaction.TxRequest{
		To:   &s.statusAddress,
		Data: callData,
	}

	result, err := s.transactionService.Call(ctx, request)
	fmt.Println("...... genHashExt - totalStatus, result, err = ", common.Bytes2Hex(result), err)

	if err != nil {
		return common.Hash{}, err
	}

	// WaitForReceipt takes long time
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("genHashExt recovered:%+v", err)
			}
		}()
	}()
	return common.Hash{}, nil
}

func (s *service) CheckReportStatus() error {
	_, err := s.ReportStatus()
	if err != nil {
		log.Errorf("ReportStatus err:%+v", err)
		return err
	}
	return nil
}

func cycleCheckReport() {
	tick := time.NewTicker(ReportStatusTime)
	defer tick.Stop()

	// Force tick on immediate start
	// CheckReport in the for loop
	for ; true; <-tick.C {
		fmt.Println("... CheckReportStatus ......")

		err := serv.CheckReportStatus()
		if err != nil {
			continue
		}
	}
}
