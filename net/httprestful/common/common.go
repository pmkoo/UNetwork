package common

import (
	"bytes"
	"fmt"
	"strconv"

	. "UNetwork/common"
	"UNetwork/core/forum"
	"UNetwork/core/ledger"
	tx "UNetwork/core/transaction"
	. "UNetwork/errors"
	. "UNetwork/net/httpjsonrpc"
	Err "UNetwork/net/httprestful/error"
	. "UNetwork/net/protocol"
	"UNetwork/smartcontract/states"
)

var node UNode

const TlsPort int = 443

type ApiServer interface {
	Start() error
	Stop()
}

func SetNode(n UNode) {
	node = n
}

//Node
func GetConnectionCount(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	if node != nil {
		resp["Result"] = node.GetConnectionCnt()
	}

	return resp
}

//Block
func GetBlockHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
	return resp
}
func GetBlockHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	return resp
}
func GetTotalIssued(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	assetid, ok := cmd["Assetid"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var assetHash Uint256

	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	amount, err := ledger.DefaultLedger.Store.GetQuantityIssued(assetHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	resp["Result"] = amount.String()
	return resp
}
func GetBlockInfo(block *ledger.Block) BlockInfo {
	hash := block.Hash()
	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		ConsensusData:    block.Blockdata.ConsensusData,
		NextBookKeeper:   BytesToHexString(block.Blockdata.NextBookKeeper.ToArrayReverse()),
		Program: ProgramInfo{
			Code:      BytesToHexString(block.Blockdata.Program.Code),
			Parameter: BytesToHexString(block.Blockdata.Program.Parameter),
		},
		Hash: BytesToHexString(hash.ToArrayReverse()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return b
}
func GetBlockTransactions(block *ledger.Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		h := block.Transactions[i].Hash()
		trans[i] = BytesToHexString(h.ToArrayReverse())
	}
	hash := block.Hash()
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		Height:       block.Blockdata.Height,
		Transactions: trans,
	}
	return b
}
func getBlock(hash Uint256, getTxBytes bool) (interface{}, int64) {
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return "", Err.UNKNOWN_BLOCK
	}
	if getTxBytes {
		w := bytes.NewBuffer(nil)
		block.Serialize(w)
		return BytesToHexString(w.Bytes()), Err.SUCCESS
	}
	return GetBlockInfo(block), Err.SUCCESS
}
func GetBlockByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	param := cmd["Hash"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	var hash Uint256
	hex, err := HexStringToBytesReverse(param)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}

	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)

	return resp
}
func GetBlockTxsByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	resp["Result"] = GetBlockTransactions(block)
	return resp
}
func GetBlockByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)
	return resp
}

//Asset
func GetAssetByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str := cmd["Hash"].(string)
	hex, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		resp["Error"] = Err.INVALID_ASSET
		return resp
	}
	asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_ASSET
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		asset.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	resp["Result"] = asset
	return resp
}
func GetBalanceByAddr(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)
	var balance Fixed64 = 0
	for _, u := range unspends {
		for _, v := range u {
			balance = balance + v.Value
		}
	}
	resp["Result"] = balance.String()
	return resp
}

func GetLockedAsset(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, a := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !a || !k {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	tmpID, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	asset, err := Uint256ParseFromBytes(tmpID)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	type locked struct {
		Lock   uint32
		Unlock uint32
		Amount string
	}
	ret := []*locked{}
	lockedAsset, _ := ledger.DefaultLedger.Store.GetLockedFromProgramHash(programHash, asset)
	for _, v := range lockedAsset {
		a := &locked{
			Lock:   v.Lock,
			Unlock: v.Unlock,
			Amount: v.Amount.String(),
		}
		ret = append(ret, a)
	}
	resp["Result"] = ret

	return resp
}

func GetUserInfo(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	name, ok := cmd["Username"].(string)
	if !ok || len(name) > MaxUserNameLen || len(name) < MinUserNameLen {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	userInfo, err := ledger.DefaultLedger.Store.GetUserInfo(name)
	if err != nil {
		resp["Error"] = Err.INVALID_USER
		return resp
	}
	totalInfo, err := ledger.DefaultLedger.Store.GetTokenInfo(name, forum.TotalToken)
	if err != nil {
		resp["Error"] = Err.INVALID_ASSET
		return resp
	}
	withdrawInfo, err := ledger.DefaultLedger.Store.GetTokenInfo(name, forum.WithdrawnToken)
	if err != nil {
		resp["Error"] = Err.INVALID_ASSET
		return resp
	}
	type info struct {
		ProgramHash    string
		Reputation     string
		TotalToken     string
		WithdrawnToken string
	}
	ret := &info{
		ProgramHash:    BytesToHexString(userInfo.UserProgramHash.ToArrayReverse()),
		Reputation:     userInfo.Reputation.String(),
		TotalToken:     totalInfo.Number.String(),
		WithdrawnToken: withdrawInfo.Number.String(),
	}
	resp["Result"] = ret

	return resp
}

func GetUserArticleInfo(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	name, ok := cmd["Username"].(string)
	if !ok || len(name) > MaxUserNameLen || len(name) < MinUserNameLen {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	articleInfo, err := ledger.DefaultLedger.Store.GetUserArticleInfo(name)
	if err != nil {
		resp["Error"] = Err.INVALID_USER
		return resp
	}
	type info struct {
		ParentTxnHash string `json:",omitempty"`
		ContentHash   string
		ContentType   string
	}
	var ret []*info
	for _, v := range articleInfo {
		t, p := "", ""
		switch v.ContentType {
		case forum.Post:
			t = "post"
		case forum.Reply:
			t = "reply"
		}
		var zeroHash Uint256
		if v.ParentTxnHash != zeroHash {
			p = BytesToHexString(v.ParentTxnHash.ToArrayReverse())
		}
		tmp := &info{
			ParentTxnHash: p,
			ContentHash:   BytesToHexString(v.ContentHash.ToArray()),
			ContentType:   t,
		}
		ret = append(ret, tmp)
	}
	resp["Result"] = ret

	return resp
}

func GetLikeInfo(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	txnHash, ok := cmd["Posthash"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	tmp, err := HexStringToBytesReverse(txnHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	hash, err := Uint256ParseFromBytes(tmp)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}

	likeInfo, err := ledger.DefaultLedger.Store.GetLikeInfo(hash)
	if err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	resp["Result"] = likeInfo

	return resp
}

func GetBalanceByAsset(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)
	var balance Fixed64 = 0
	for k, u := range unspends {
		assid := BytesToHexString(k.ToArrayReverse())
		for _, v := range u {
			if assetid == assid {
				balance = balance + v.Value
			}
		}
	}
	resp["Result"] = balance.String()
	return resp
}
func GetUnspends(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160

	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	type Result struct {
		AssetId   string
		AssetName string
		Utxo      []UTXOUnspentInfo
	}
	var results []Result
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)

	for k, u := range unspends {
		assetid := BytesToHexString(k.ToArrayReverse())
		asset, err := ledger.DefaultLedger.Store.GetAsset(k)
		if err != nil {
			resp["Error"] = Err.INTERNAL_ERROR
			return resp
		}
		var unspendsInfo []UTXOUnspentInfo
		for _, v := range u {
			unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{BytesToHexString(v.Txid.ToArrayReverse()), v.Index, v.Value.String()})
		}
		results = append(results, Result{assetid, asset.Name, unspendsInfo})
	}
	resp["Result"] = results
	return resp
}
func GetUnspendOutput(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}

	var programHash Uint160
	var assetHash Uint256
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(programHash, assetHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		resp["Result"] = err
		return resp
	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: BytesToHexString(v.Txid.ToArrayReverse()), Index: v.Index, Value: v.Value.String()})
	}
	resp["Result"] = UTXOoutputs
	return resp
}

//Transaction
func GetTransactionByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_TRANSACTION
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		tx.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	tran := TransArryByteToHexString(tx)
	resp["Result"] = tran
	return resp
}
func SendRawTransaction(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str, ok := cmd["Data"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	bys, err := HexStringToBytes(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var txn tx.Transaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	if txn.TxType != tx.TransferAsset && txn.TxType != tx.RegisterUser &&
		txn.TxType != tx.PostArticle && txn.TxType != tx.ReplyArticle &&
		txn.TxType != tx.LikeArticle && txn.TxType != tx.Withdrawal {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	var hash Uint256
	hash = txn.Hash()
	if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
		resp["Error"] = int64(errCode)
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	//TODO 0xd1 -> tx.InvokeCode
	if txn.TxType == 0xd1 {
		if userid, ok := cmd["Userid"].(string); ok && len(userid) > 0 {
			resp["Userid"] = userid
		}
	}
	return resp
}

//stateupdate
func GetStateUpdate(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	namespace, ok := cmd["Namespace"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	key, ok := cmd["Key"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	fmt.Println(cmd, namespace, key)
	//TODO get state from store
	return resp
}

func ResponsePack(errCode int64) map[string]interface{} {
	resp := map[string]interface{}{
		"Action":  "",
		"Result":  "",
		"Error":   errCode,
		"Desc":    "",
		"Version": "1.0.0",
	}
	return resp
}
func GetContract(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint160
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	//TODO GetContract from store
	contract, err := ledger.DefaultLedger.Store.GetContract(hash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	c := new(states.ContractState)
	b := bytes.NewBuffer(contract)
	c.Deserialize(b)
	var params []int
	for _, v := range c.Code.ParameterTypes {
		params = append(params, int(v))
	}
	codehash := c.Code.CodeHash()
	funcCode := &FunctionCodeInfo{
		Code:           BytesToHexString(c.Code.Code),
		ParameterTypes: params,
		ReturnType:     int(c.Code.ReturnType),
		CodeHash:       BytesToHexString(codehash.ToArrayReverse()),
	}
	programHash := c.ProgramHash
	result := DeployCodeInfo{
		Name:        c.Name,
		Author:      c.Author,
		Email:       c.Email,
		Version:     c.Version,
		Description: c.Description,
		Language:    int(c.Language),
		Code:        new(FunctionCodeInfo),
		ProgramHash: BytesToHexString(programHash.ToArrayReverse()),
	}

	result.Code = funcCode
	resp["Result"] = result
	return resp
}
