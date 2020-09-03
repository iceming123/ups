package vm

import (
	"fmt"
	"math/big"
	"crypto/rand"
	"strings"
	"io"
	"bytes"
	"sort"
	"errors"
	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/accounts/abi"
	// "github.com/iceming123/ups/core/State"
	// "github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/crypto"
	"github.com/iceming123/ups/crypto/ecies"
	"github.com/iceming123/ups/log"
	// "github.com/iceming123/ups/params"
	"github.com/iceming123/ups/rlp"
)

// 交互式密钥交互过程,卖方设置价格,买方出价，卖方给出加密密钥(以买方公钥加密)，验证该加密密钥。
// 本服务为单向服务合约

var (
	UpsEngineAddress = common.BytesToAddress([]byte{90})
	CleanDealFlag = false
	MaxAutoRedeemHeight = 200
	MaxAutoUnlockedHieght = 500
)
var (
	ErrNotFoundProvider = errors.New("not found the provider from the Key")
	ErrNotFoundDeal = errors.New("not found the deal in the provider")
	ErrNotMatchPrice = errors.New("not match the Price from the provider")
	ErrInvalidParams = errors.New("invalid params")
	ErrInvalidPk = errors.New("uninitialized pubkey in deal")
)
const (
	DealUnPaid byte = iota
	DealPayed
	DealRedeem
	DealFinish
)

var abiEngine abi.ABI

func init() {
	abiEngine, _ = abi.JSON(strings.NewReader(ABIENGINEJSON))
}
/////////////////////////////////////////////////////////////////////////////////////////////
type fileEngine struct{}

func (c *fileEngine) RequiredGas(evm *EVM, input []byte) uint64 {
	var (
		baseGas uint64 = 21000
		method *abi.Method
		err error
	)

	method, err = abiEngine.MethodById(input)
	if err != nil {
		return baseGas
	}
	if gas, ok := FileEngineGas[string(method.Name)]; ok {
		return gas
	} else {
		return baseGas
	}
}

func (c *fileEngine) Run(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	return RunFileEngine(evm, contract, input)
}

// RunStaking execute ups file engine contract
func RunFileEngine(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	var method *abi.Method
	method, err = abiEngine.MethodById(input)
	if err != nil {
		log.Error("No method found")
		return nil, errExecutionReverted
	}

	data := input[4:]

	switch method.Name {
	case "addProvider":
		ret, err = addProvider(evm, contract, data)
	case "postRequestKey":
		ret, err = postRequestKey(evm, contract, data)
	case "getPubKeyFromDeal":
		ret, err = getPubKeyFromDeal(evm, contract, data)
	case "setFileKey":
		ret, err = SetFileKey(evm, contract, data)
	case "getFileKey":
		ret, err = GetFileKey(evm, contract, data)
	default:
		log.Warn("FileEngine call fallback function")
		err = ErrStakingInvalidInput
	}
	if err != nil {
		log.Warn("FileEngine error code", "code", err)
		err = errExecutionReverted
	}
	return ret, err
}

///////////////API///////////////////////////////////////////////////////////////////////////
// addProvider
func addProvider(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		Key	 []byte
		Price  *big.Int
	}{}
	method, _ := abiStaking.Methods["addProvider"]

	err = method.Inputs.Unpack(&args, input)
	if err != nil {
		log.Error("Unpack addProvider error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	engine := NewEngine()
	err = engine.Load(evm.StateDB)
	if err != nil {
		log.Error("engine load error", "error", err)
		return nil, err
	}
	err = engine.tryAddProvider(string(args.Key), from, args.Price)
	if err != nil {
		log.Error("tryAddProvider", "address", contract.caller.Address(),"Key",args.Key, "Price", args.Price, "error", err)
		return nil, err
	}

	err = engine.Save(evm.StateDB)
	if err != nil {
		log.Error("engine save State error", "error", err)
		return nil, err
	}

	event := abiStaking.Events["AddProvider"]
	logData, err := event.Inputs.PackNonIndexed(args.Key, args.Price)
	if err != nil {
		log.Error("Pack AddProvider log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID(),
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	return nil, nil
}
// postRequestKey pay and lock the coin to the conctrat for get the password
func postRequestKey(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		Key	 []byte
		Pk   []byte
	}{}
	method, _ := abiStaking.Methods["postRequestKey"]

	err = method.Inputs.Unpack(&args, input)
	if err != nil {
		log.Error("Unpack postRequestKey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	engine := NewEngine()
	err = engine.Load(evm.StateDB)
	if err != nil {
		log.Error("engine load error", "error", err)
		return nil, err
	}
	err = engine.tryAddConsumer(newConsumer(string(args.Key),contract.Value(),args.Pk), evm.Context.BlockNumber.Uint64())
	if err != nil {
		log.Error("postRequestKey", "address", from,"Key",string(args.Key), "pk", args.Pk, "error", err)
		return nil, err
	}

	err = engine.Save(evm.StateDB)
	if err != nil {
		log.Error("engine save State error", "error", err)
		return nil, err
	}

	event := abiStaking.Events["PostRequestKey"]
	logData, err := event.Inputs.PackNonIndexed(args.Key, args.Pk)
	if err != nil {
		log.Error("Pack PostRequestKey log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID(),
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	return nil, nil
}
// getPubKeyFromDeal
func getPubKeyFromDeal(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		Key	 	 []byte
		Addr 	 common.Address
	}{}
	method, _ := abiStaking.Methods["getPubKeyFromDeal"]

	err = method.Inputs.Unpack(&args, input)
	if err != nil {
		log.Error("Unpack getPubKeyFromDeal error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	engine := NewEngine()
	err = engine.Load(evm.StateDB)
	if err != nil {
		log.Error("engine load error", "error", err)
		return nil, err
	}
	if pk,err := engine.GetPubKeyByProvider(string(args.Key),args.Addr,from); err != nil {
		log.Error("getPubKeyFromDeal", "address", from,"Key",string(args.Key), "consumerAddr", args.Addr, "error", err)
		return nil, err
	} else {
		return method.Outputs.Pack(pk)
	}
}
// SetFileKey
func SetFileKey(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {
	args := struct {
		Key	 	 []byte
		EcPass   []byte
		Addr 	 common.Address
	}{}
	method, _ := abiStaking.Methods["setFileKey"]

	err = method.Inputs.Unpack(&args, input)
	if err != nil {
		log.Error("Unpack setFileKey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	engine := NewEngine()
	err = engine.Load(evm.StateDB)
	if err != nil {
		log.Error("engine load error", "error", err)
		return nil, err
	}
	err = engine.SetPasswordByProvider(string(args.Key),args.EcPass,args.Addr,from)
	if err != nil {
		log.Error("setFileKey", "address", from,"Key",string(args.Key), "EcPass", args.EcPass, "error", err)
		return nil, err
	}

	err = engine.Save(evm.StateDB)
	if err != nil {
		log.Error("engine save State error", "error", err)
		return nil, err
	}

	event := abiStaking.Events["SetFileKey"]
	logData, err := event.Inputs.PackNonIndexed(args.Key, args.EcPass,args.Addr)
	if err != nil {
		log.Error("Pack SetFileKey log error", "error", err)
		return nil, err
	}
	topics := []common.Hash{
		event.ID(),
		common.BytesToHash(from[:]),
	}
	logN(evm, contract, topics, logData)
	return nil, nil
}
// GetFileKey
func GetFileKey(evm *EVM, contract *Contract, input []byte) (ret []byte, err error) {	
	args := struct {
		Key	 	 []byte
		Addr 	 common.Address
	}{}
	method, _ := abiStaking.Methods["getFileKey"]

	err = method.Inputs.Unpack(&args, input)
	if err != nil {
		log.Error("Unpack setFileKey error", "err", err)
		return nil, ErrStakingInvalidInput
	}

	from := contract.caller.Address()
	engine := NewEngine()
	err = engine.Load(evm.StateDB)
	if err != nil {
		log.Error("engine load error", "error", err)
		return nil, err
	}
	if ec,err := engine.GetPasswordByConsumer(string(args.Key),args.Addr,from); err != nil {
		log.Error("setFileKey", "address", from,"Key",string(args.Key), "providerAddress", args.Addr, "error", err)
		return nil, err
	} else {
		return method.Outputs.Pack(ec)
	}
}
///////////////API///////////////////////////////////////////////////////////////////////////

func matchPrice(p *provider,c *consumer) error {
	if p == nil || c == nil {
		return ErrInvalidParams
	}
	e := p.getService(c.getKey())
	if e == nil {
		return ErrNotFoundProvider
	}
	if e.getPrice().Cmp(c.getPrice()) == 0 {
		return nil
 	} else {
		 return ErrNotMatchPrice
	}
}
// will be store to State
func storeBalance(val *big.Int) error {
	// add balance to State
	return nil
}
func transToProvider(val *big.Int,addr common.Address) error {
	// trans to the provider from the upsEngineAddress 
	return nil
} 
func redeemToConsumer(val *big.Int,addr common.Address) error {
	return nil
}
func allBalance() *big.Int {
	// all balance of the upsEngineAddress
	return nil
}
func Encrypt(pk,msg []byte) ([]byte,error) {
	if p,err := crypto.UnmarshalPubkey(pk); err != nil {
		return nil,err
	} else {
		if cr, err := ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(p), msg, nil, nil); err != nil {
			return nil,err
		} else {
			return cr,nil
		}
	}
}
func Decrypt(priv,msg []byte) ([]byte,error) {
	if p,err := crypto.ToECDSA(priv);err != nil {
		return nil,err
	} else {
		priKey := ecies.ImportECDSA(p)
		if dcr, err := priKey.Decrypt(msg, nil, nil); err != nil {
			return nil,err
		} else {
			return dcr,nil
		}
	}
}

type deal struct {
	Key string
	Buyer common.Address
	BuyerPk []byte
	Password []byte
	Height 	uint64
	Price 	*big.Int
	State byte
}
func (d *deal) setPassword(ec []byte) {
	d.Password = make([]byte,len(ec))
	copy(d.Password,ec)
}
func (d *deal) getPassword() []byte {
	ps := make([]byte,len(d.Password))
	copy(ps,d.Password)
	return ps
}
func (d *deal) getPk() []byte{
	if len(d.BuyerPk) > 0 {
		pk := make([]byte,len(d.BuyerPk))
		copy(pk,d.BuyerPk)
		return pk
	}
	return nil
}
func (d *deal) getHeight() uint64 {
	return d.Height
}
func (d *deal) getPrice() *big.Int {
	return new(big.Int).Set(d.Price)
}
func (d *deal) isPayed() bool {
	return d.State == DealPayed
}
func (d *deal) canRedeem(ch uint64) bool {
	if int64(ch - d.Height) > int64(MaxAutoRedeemHeight) && d.State == DealUnPaid {
		return true
	}
	return false
}
func (d *deal) canClear() bool {
	if d.State == DealRedeem || d.State == DealFinish {
		return true
	}
	return false
}
func (d *deal) redeemed() {
	d.State = DealRedeem
}
func (d *deal) payed() {
	d.State = DealPayed
}
func (d *deal) finish() {
	d.State = DealFinish
}
func (d *deal) doRedeem() error {
	return redeemToConsumer(d.Price,d.Buyer)
}

type Entry struct {
	Key string
	Price *big.Int
	Buyer bool
}
func newEntry(Key string, Price *big.Int) *Entry {
	return &Entry{
		Key:	Key,
		Price: new(big.Int).Set(Price),
	}
}
func (e *Entry) getPrice() *big.Int {
	return e.Price
}

type StoreService struct {
	Key  	string
	En      *Entry
}
type SortStoreService []*StoreService
func (vs SortStoreService) Len() int {
	return len(vs)
}
func (vs SortStoreService) Less(i, j int) bool {
	return bytes.Compare([]byte(vs[i].Key), []byte(vs[j].Key)) == -1
}
func (vs SortStoreService) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

type provider struct {
	Addr 		common.Address
	Service 	map[string]*Entry
	DealList 	[]*deal
}
func (p *provider) getService(Key string) *Entry {
	v,ok := p.Service[Key]
	if !ok {
		return nil
	}
	return v
}
func (p *provider) getPrice(Key string) *big.Int {
	e := p.getService(Key)
	if e == nil {
		return nil
	}
	return e.getPrice()
}
func (p *provider) getDealResult(Key string,addr common.Address) *deal {
	for _,v := range p.DealList {
		if Key == v.Key && bytes.Equal(addr[:],v.Buyer[:]) {
			return v
		}
	}
	return nil
}
func (p *provider) addDealResult(height uint64,Key string,addr common.Address,pk []byte,Price *big.Int) {
	d := p.getDealResult(Key,addr)
	if d != nil {
		return 
	}
	p.DealList = append(p.DealList,&deal{
		Key:	Key,
		Buyer:	addr,
		BuyerPk:	pk,
		Height:	height,
		Price:	new(big.Int).Set(Price),
		State:	DealUnPaid,
	})
	return 
}
func (p *provider) setPassword(Key string, addr common.Address,ec []byte) error {
	d := p.getDealResult(Key,addr)
	if d == nil {
		return ErrNotFoundDeal
	}
	d.setPassword(ec)
	d.payed()
	return nil
}
func (p *provider) addEntry(Key string,Price *big.Int) error {
	// disable to change the Price 
	_,ok := p.Service[Key]
	if !ok {
		p.Service[Key] = newEntry(Key,Price)
	}
	return nil
}
func (p *provider) getAddress() common.Address {
	return p.Addr
}
func (p *provider) isSuppy(Key string) bool {
	for k := range p.Service {
		if k == Key {
			return true
		}
	}
	return false
}
func (p *provider) clearDealList() {
	tmp := make([]*deal,0,0)
	for _,d := range p.DealList {
		if !d.canClear() {
			tmp = append(tmp,d)
		}
	}
	p.DealList = tmp
}
func (p *provider) serviceToSlice() SortStoreService {
	v1 := make([]*StoreService, 0, 0)
	for k, v := range p.Service {
		v1 = append(v1, &StoreService{
			Key: 		k,
			En:      	v,
		})
	}
	sort.Sort(SortStoreService(v1))
	return SortStoreService(v1)
}
func (p *provider) serviceFromSlice(v1 SortStoreService) {
	enInfos := make(map[string]*Entry)
	for _, v := range v1 {
		enInfos[v.Key] = v.En
	}
	p.Service = enInfos
}
func (p *provider) DecodeRLP(s *rlp.Stream) error {
	eb := struct {
		Addr 		common.Address
		Value1 		SortStoreService
		DealList 	[]*deal
	}{}
	if err := s.Decode(&eb); err != nil {
		return err
	}
	p.serviceFromSlice(eb.Value1)
	p.Addr,p.DealList = eb.Addr,eb.DealList
	return nil
}

func (p *provider) EncodeRLP(w io.Writer) error {
	tmp := struct {
		Addr 		common.Address
		Value1 		SortStoreService
		DealList 	[]*deal
	}{ 
		Addr:		p.Addr,
		Value1:		p.serviceToSlice(),
		DealList:	p.DealList,
	 }

	return rlp.Encode(w, &tmp)
}

type consumer struct {
	Key 	string
	Price   *big.Int
	Pk 		[]byte	
}
func newConsumer(Key string,Price *big.Int,pk []byte) *consumer {
	return &consumer{
		Key:	Key,
		Price:	new(big.Int).Set(Price),
		Pk:		pk,
	}
}
func (c *consumer) getKey() string {
	return c.Key
}
func (c *consumer) getPrice() *big.Int {
	return new(big.Int).Set(c.Price)
}
func (c *consumer) getAddr() common.Address {
	return common.BytesToAddress(crypto.Keccak256(c.Pk[1:])[12:])
}
////////////////////////////////////////////////////////////////////////////////////////////////
type StoreEngine struct {
	Address common.Address
	Pov      *provider
}
type SortStoreEngine []*StoreEngine
func (vs SortStoreEngine) Len() int {
	return len(vs)
}
func (vs SortStoreEngine) Less(i, j int) bool {
	return bytes.Compare(vs[i].Address[:], vs[j].Address[:]) == -1
}
func (vs SortStoreEngine) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

type Engine struct {
	maker  map[common.Address]*provider
}
func NewEngine() *Engine {
	return &Engine{
		maker: 	make(map[common.Address]*provider),
	}
}

func (en *Engine) Load(State StateDB) error {
	Key := common.BytesToHash(UpsEngineAddress[:])
	data := State.GetPOSState(UpsEngineAddress, Key)
	lenght := len(data)
	if lenght == 0 {
		return errors.New("Load data = 0")
	}
	var temp Engine
	if err := rlp.DecodeBytes(data, &temp); err != nil {
		log.Error("Invalid Engine entry RLP", "err", err)
		return errors.New(fmt.Sprintf("Invalid Engine entry RLP %s", err.Error()))
	}
	en.maker = temp.maker
	return nil
}
func (en *Engine) Save(State StateDB) error {
	Key := common.BytesToHash(UpsEngineAddress[:])
	data, err := rlp.EncodeToBytes(en)
	if err != nil {
		log.Crit("Failed to RLP encode Engine", "err", err)
	}
	State.SetPOSState(UpsEngineAddress, Key, data)
	return err
}
/////////////////////////////////////////////////////////////////
func (en *Engine) toSlice() SortStoreEngine {
	v1 := make([]*StoreEngine, 0, 0)
	for k, v := range en.maker {
		v1 = append(v1, &StoreEngine{
			Address: 	k,
			Pov:      	v,
		})
	}
	sort.Sort(SortStoreEngine(v1))
	return SortStoreEngine(v1)
}
func (en *Engine) fromSlice(v1 SortStoreEngine) {
	enInfos := make(map[common.Address]*provider)
	for _, v := range v1 {
		enInfos[v.Address] = v.Pov
	}
	en.maker = enInfos
}
func (en *Engine) DecodeRLP(s *rlp.Stream) error {
	eb := struct {
		Value1 SortStoreEngine
	}{}
	if err := s.Decode(&eb); err != nil {
		return err
	}
	en.fromSlice(eb.Value1)
	return nil
}
func (en *Engine) EncodeRLP(w io.Writer) error {
	tmp := struct {
		Value1 SortStoreEngine
	}{ en.toSlice() }

	return rlp.Encode(w, &tmp)
}

/////////////////////////////////////////////////////////////////

func (en *Engine) getProviderByKey(Key string) *provider {
	for _,v := range en.maker {
		if v.isSuppy(Key) {
			return v
		}
	}
	return nil
}
func (en *Engine) getProviderByAddr(own common.Address) *provider {
	v,ok := en.maker[own]
	if ok {
		return v
	}
	return nil
}
func (en *Engine) addProvider(p *provider) {
	en.maker[p.getAddress()] = p
}

func (en *Engine) matchByConsumer(c *consumer) (*provider,error) {
	p := en.getProviderByKey(c.getKey())
	if p == nil {
		return nil,ErrNotFoundProvider
	}
	err := matchPrice(p,c)
	
	return p,err
}

// try to add balance to contract address and locked it
func (en *Engine) tryAddConsumer(c *consumer,height uint64) error {
	// 1. add consumer and match the provider
	// 2. the consumer store the money to the contract and locked it
	// 3. the provider set the Password encrypted by consumer's pk
	p,err := en.matchByConsumer(c)
	if err != nil {
		return err
	}
	p.addDealResult(height,c.Key,c.getAddr(),c.Pk,c.getPrice())
	return nil
}
// make sure the caller match the provider
func (en *Engine) tryAddProvider(Key string,own common.Address,Price *big.Int) error {
	p := en.getProviderByAddr(own)
	if p == nil {
		service := make(map[string]*Entry)
		service[Key] = newEntry(Key,Price)
		en.addProvider(&provider{
			Addr: own,
			Service: service,
			DealList:	make([]*deal,0,0),
		})
		return nil
	} 
	return p.addEntry(Key,Price)
}
// check provider's address by caller in contract,make sure the caller match the provider
func (en *Engine) batchGetPkFromDeals(keys []string,addrs []common.Address,own common.Address) ([][]byte,error) {
	l := len(keys)
	if len(keys) != len(addrs) {
		return nil, ErrInvalidParams
	}
	p := en.getProviderByAddr(own)
	if p == nil {
		return nil,ErrNotFoundProvider
	}
	res := make([][]byte,0,0)
	for i:=0;i<l;i++ {
		k,a := keys[i],addrs[i]
		d := p.getDealResult(k,a)
		if d != nil {
			pk := d.getPk()
			if pk != nil {
				res = append(res,pk)
				continue
			}
		} 
		res = append(res,[]byte{0})		// invalid pk
	}
	return res,nil
}
func (en *Engine) batchSetPassword(keys []string,addrs []common.Address,ecPass [][]byte,own common.Address) (int,error) {
	l := len(keys)
	if len(keys) != len(addrs) || l != len(ecPass) {
		return 0, ErrInvalidParams
	}
	p := en.getProviderByAddr(own)
	if p == nil {
		return 0,ErrNotFoundProvider
	}
	for i:=0;i<l;i++ {
		k,a,ec := keys[i],addrs[i],ecPass[i]
		if err := p.setPassword(k,a,ec); err != nil {
			return i,err
		}
	}
	return 0,nil
}

func (en *Engine) GetPubKeyByProvider(Key string, addr,own common.Address) ([]byte,error) {
	p := en.getProviderByAddr(own)
	if p == nil {
		return nil,ErrNotFoundProvider
	}
	d := p.getDealResult(Key,addr)
	if d == nil {
		return nil,ErrNotFoundDeal
	}
	pk := d.getPk()
	if pk == nil {
		return nil,ErrInvalidPk
	}
	return pk,nil
}
func (en *Engine) SetPasswordByProvider(Key string, ecPass []byte,addr,own common.Address) error {
	p := en.getProviderByAddr(own) 
	if p == nil {
		return ErrNotFoundProvider
	}
	
	return p.setPassword(Key,addr,ecPass)
}
func (en *Engine) GetPasswordByConsumer(Key string, addr, own common.Address) ([]byte,error) {
	p := en.getProviderByAddr(addr)
	if p == nil {
		return nil,ErrNotFoundProvider
	}
	d := p.getDealResult(Key,own)
	if d == nil {
		return nil,ErrNotFoundDeal
	}
	return d.getPassword(),nil
}
// the balance will auto translate to the consumer for no deal when more than MaxAutoRedeemHeight height 
func (en *Engine) actionRedeemForConsumer(ch uint64) error {
	// check the not finished deal
	for _,p := range en.maker {
		for _,d := range p.DealList {
			if d.canRedeem(ch) {
				d.doRedeem()
				d.redeemed()
			}
		}
	}
	return nil
}
func (en *Engine) actionUnlockedForProvider(ch uint64) error {
	for _,p := range en.maker {
		for _,d := range p.DealList {
			if d.isPayed() && int64(ch - d.getHeight()) > int64(MaxAutoUnlockedHieght) {
				transToProvider(d.getPrice(),p.getAddress())
				d.finish()
			}
		}
	}
	return nil
}
func (en *Engine) actionClearDeal(ch uint64) error {
	if ch % 200 == 0 {
		for _,p := range en.maker {
			p.clearDealList()
		}
	}
	return nil 
}
func (en *Engine) DoAction(ch uint64) error {
	err := en.actionRedeemForConsumer(ch)
	if err != nil {
		return err
	}
	err = en.actionUnlockedForProvider(ch)
	if err != nil {
		return err
	}
	return en.actionClearDeal(ch)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////

const ABIENGINEJSON = `
[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "price",
				"type": "uint256"
			}
		],
		"name": "AddProvider",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "pk",
				"type": "bytes"
			}
		],
		"name": "PostRequestKey",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"indexed": true,
				"internalType": "bytes",
				"name": "ecPass",
				"type": "bytes"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "addr",
				"type": "address"
			}
		],
		"name": "SetFileKey",
		"type": "event"
	},
	{
		"constant": false,
		"inputs": [
			{
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"internalType": "uint256",
				"name": "price",
				"type": "uint256"
			}
		],
		"name": "addProvider",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"internalType": "address",
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getFileKey",
		"outputs": [
			{
				"internalType": "bytes",
				"name": "ec",
				"type": "bytes"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"internalType": "address",
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getPubKeyFromDeal",
		"outputs": [
			{
				"internalType": "bytes",
				"name": "pk",
				"type": "bytes"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"internalType": "bytes",
				"name": "data",
				"type": "bytes"
			}
		],
		"name": "postRequestKey",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"internalType": "string",
				"name": "key",
				"type": "string"
			},
			{
				"internalType": "bytes",
				"name": "ecPass",
				"type": "bytes"
			},
			{
				"internalType": "address",
				"name": "addr",
				"type": "address"
			}
		],
		"name": "setFileKey",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`

// FileEngineGas defines all method gas
var FileEngineGas = map[string]uint64{
	"addProvider":       	3000000,
	"postRequestKey":      	60000,
	"getPubKeyFromDeal":    30000,
	"setFileKey":          	40000,
	"getFileKey":           30000,
}