package dex

import (
	"context"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Orderfactory simplifies creating orders, trades and cancelOrders objects
// Pair is the token pair for which the order is created
// Exchange is the Ethereum address of the exchange smart contract
// CurrentOrderID increments for each new order
type OrderFactory struct {
	Client         *ethclient.Client
	Pair           *TokenPair
	Wallet         *Wallet
	Exchange       common.Address
	Params         *OrderParams
	TradeNonce     uint64
	OrderNonce     uint64
	CurrentOrderID uint64
	NonceGenerator *rand.Rand
}

type OrderParams struct {
	FeeMake *big.Int
	FeeTake *big.Int
	Nonce   *big.Int
	Expires *big.Int
}

// NewOrderFactory returns an order factory from a given token pair and a given wallet
func NewOrderFactory(p *TokenPair, w *Wallet) *OrderFactory {

	rpcClient, err := rpc.DialWebsocket(context.Background(), "ws://127.0.0.1:8546", "")
	if err != nil {
		log.Printf("Could not create order factory")
		return nil
	}

	client := ethclient.NewClient(rpcClient)

	params := &OrderParams{
		FeeMake: big.NewInt(0),
		FeeTake: big.NewInt(0),
		Nonce:   big.NewInt(0),
		Expires: big.NewInt(1e18),
	}

	source := rand.NewSource(time.Now().UnixNano())
	ng := rand.New(source)

	return &OrderFactory{
		Pair:           p,
		Wallet:         w,
		Exchange:       config.Exchange,
		Params:         params,
		CurrentOrderID: 0,
		Client:         client,
		NonceGenerator: ng,
	}
}

func (f *OrderFactory) SetExchangeAddress(exchange common.Address) error {
	f.Exchange = exchange
	return nil
}

func (f *OrderFactory) NewOrderMessage(tokenBuy Token, amountBuy int64, tokenSell Token, amountSell int64) (*Message, *Order, error) {
	o, err := f.NewOrder(tokenBuy, amountBuy, tokenSell, amountBuy)
	if err != nil {
		return nil, nil, err
	}

	p := &OrderPayload{Order: o}
	return &Message{MessageType: PLACE_ORDER, Payload: p}, o, nil
}

// NewOrder creates a new Order object
func (f *OrderFactory) NewOrder(tokenBuy Token, amountBuy int64, tokenSell Token, amountSell int64) (*Order, error) {
	o := &Order{}

	o.Id = f.CurrentOrderID
	o.ExchangeAddress = f.Exchange
	o.TokenBuy = tokenBuy.Address
	o.SymbolBuy = tokenBuy.Symbol
	o.TokenSell = tokenSell.Address
	o.SymbolSell = tokenSell.Symbol
	o.AmountBuy = big.NewInt(amountBuy)
	o.AmountSell = big.NewInt(amountSell)
	o.Expires = f.Params.Expires
	o.FeeMake = f.Params.FeeMake
	o.FeeTake = f.Params.FeeTake
	o.Nonce = big.NewInt(int64(f.NonceGenerator.Intn(1000)))
	o.Maker = f.Wallet.Address
	o.Price = 0
	o.Amount = 0
	o.PairID = f.Pair.ID
	o.Sign(f.Wallet)

	log.Printf("Order is equal to %v", o)

	f.OrderNonce++
	f.CurrentOrderID++
	return o, nil
}

func (f *OrderFactory) NewTrade(o *Order, amount int64) (*Trade, error) {
	t := &Trade{}

	t.OrderHash = o.Hash
	t.PairID = f.Pair.ID
	t.Taker = f.Wallet.Address
	t.TradeNonce = big.NewInt(int64(f.NonceGenerator.Intn(1000)))
	t.Amount = big.NewInt(amount)
	t.Sign(f.Wallet)

	log.Printf("Trade is equal to %v", t)

	f.TradeNonce++
	return t, nil
}

// NewOrderCancel creates a new OrderCancel object from an Order
func (f *OrderFactory) NewOrderCancel(o *Order) (*OrderCancel, error) {
	oc := &OrderCancel{}

	oc.OrderId = o.Id
	oc.PairID = f.Pair.ID
	oc.OrderHash = o.Hash
	oc.Sign(f.Wallet)
	return oc, nil
}

func (f *OrderFactory) NewCancelOrderMessage(o *Order) (*Message, error) {
	oc, err := f.NewOrderCancel(o)
	if err != nil {
		return nil, err
	}

	p := &OrderCancelPayload{OrderCancel: oc}
	return &Message{MessageType: CANCEL_ORDER, Payload: p}, nil
}
