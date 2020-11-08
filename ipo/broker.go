package main

import (
	"fmt"
	"log"
	"strings"
	"golang.org/x/net/html/atom"
)

type Stock struct {
	name string
	ipoPrice int //1手价格
	freezeDays int //冻结天数
	growthRate float64 //预估增值
	lotWinningRate float64 //1手中签率
	winningRateGrowth float64 //单手中签率增幅，线性模型
}

func NewStock(name string, price, fd int, gr, lwr, wrg float64) *Stock {
	v := &Stock{
		name: name,
		ipoPrice: price,
		freezeDays: fd,
		growthRate: gr,
		lotWinningRate: lwr,
		winningRateGrowth: wrg,
	}
	return v
}

// 预估整体中签率
func (v *Stock) winningRate(lotCnt int) float64 {
	if lotCnt == 1 {
		return v.lotWinningRate
	}
	return v.lotWinningRate + float64(lotCnt - 1) * v.winningRateGrowth
}

func (v *Stock) ValidLotCnt(lotCnt int) int {
	seeds := []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 14, 16, 18, 20, 30, 40, 50, 60, 70, 80, 90, 100, 120, 140, 160, 180, 200, 400, 600, 800,
	}
	for _, s := range seeds {
		if s == lotCnt {
			return s
		}
	}
	return 0
}

func (v *Stock) String() string {
	return fmt.Sprintf("%v, 价格:%v, 冻资天数:%v, 增长率:%v, 一手中签率:%v, 中签率增长:%v", v.name, v.ipoPrice, v.freezeDays, v.growthRate, v.lotWinningRate, v.winningRateGrowth)
}

type Broker struct {
	name                   string
	cashSubscribeFee       int     //现金申购费
	financingSubscribeFee  int     //融资申购费
	financingRate          float64 //融资利率
	financingMultipleTimes int     //杠杆
}

func NewBroker(name string, cf, ff int, fr float64, times int) *Broker {
	v := &Broker{
		name:                   name,
		cashSubscribeFee:       cf,
		financingSubscribeFee:  ff,
		financingRate:          fr,
		financingMultipleTimes: times,
	}
	return v
}

func (v *Broker) String() string {
	return fmt.Sprintf("%v, 现金申购费:%v, 融资申购费:%v, 融资利率:%v, 融资倍数:%v", v.name, v.cashSubscribeFee, v.financingSubscribeFee, v.financingRate, v.financingMultipleTimes)
}

// 申购成本
type IpoCost struct {
	details []string
	stock *Stock
	broker *Broker
	cash int
	cashLot int //现金申购几手
	financeLot int //融资申购几手
	useFinancing bool //是否融资申购
}

func NewIpoCost(stock *Stock, broker *Broker, cash int, useFinancing bool, cashLot, financeLot int) *IpoCost {
	v := &IpoCost{
		stock: stock,
		broker: broker,
		cash: cash,
		useFinancing: useFinancing,
		cashLot: cashLot,
		financeLot: financeLot,
	}
	return v
}

// 计算融资成本
func (v *IpoCost) calc() (cost int) {
	v.details = []string{}

	if !v.useFinancing { //现金申购
		cost = v.broker.cashSubscribeFee
		v.details = append(v.details, fmt.Sprintf("现金申购费:%v", cost))
		return
	}

	// 融资申购
	pureFinancingLot := v.financeLot * (v.broker.financingMultipleTimes - 1) / v.broker.financingMultipleTimes
	v.details = append(v.details, fmt.Sprintf("纯融资手数：%v", pureFinancingLot))
	pureFinancingMoney := pureFinancingLot * v.stock. ipoPrice
	v.details = append(v.details, fmt.Sprintf("融资额:%v", pureFinancingMoney))
	financingCost := int(float64(pureFinancingMoney) * v.broker.financingRate * float64(v.stock.freezeDays) / 365.0)
	v.details = append(v.details, fmt.Sprintf("融资费用:%v", financingCost))
	v.details = append(v.details, fmt.Sprintf("融资申购费:%v", v.broker.financingSubscribeFee))
	cost = financingCost + v.broker.financingSubscribeFee
	v.details = append(v.details, fmt.Sprintf("融资总费用:%v", cost))
	return
}

func (v *IpoCost) String() string {
	return strings.Join(v.details, "\r\n")
}

// 申购收入
type IpoEarning struct {
	stock *Stock
	broker *Broker
	cash int
	useFinancing bool
	lot int
	details []string
}

func NewIpoEarning(stock *Stock, broker *Broker, cash int, useFinancing bool, lot int) *IpoEarning {
	v := &IpoEarning{
		stock: stock,
		broker: broker,
		cash: cash,
		useFinancing: useFinancing,
		lot: lot,
	}
	return v
}

func (v *IpoEarning) calc() (earning int) {
	v.details = []string{}

	v.details = append(v.details, fmt.Sprintf("总手数:%v", v.lot))
	winningRate := v.stock.winningRate(v.lot)
	v.details = append(v.details, fmt.Sprintf("中签率:%v", winningRate))
	earning = int(float64(v.stock.ipoPrice) * winningRate * v.stock.growthRate)
	v.details = append(v.details, fmt.Sprintf("单手价格：%v, 预估上市后增长:%v", v.stock.ipoPrice, v.stock.growthRate))
	v.details = append(v.details, fmt.Sprintf("预估收入:%v", earning))
	return
}

func (v *IpoEarning) String() string {
	return strings.Join(v.details, "\r\n")
}

type IpoScheme struct {
	stock         *Stock
	broker        *Broker
	cash          int
	userFinancing bool //使用融资
	cashLot       int //现金申购数
	financingLot  int //融资申购数
	logs          []string
	cost          *IpoCost
	earning       *IpoEarning
}

func NewIpoScheme(stock *Stock, broker *Broker, cash int, useFinancing bool) *IpoScheme {
	v := &IpoScheme{
		stock:  stock,
		broker: broker,
		cash: cash,
		userFinancing: useFinancing,
	}
	// 现金可以申购几手
	v.cashLot = v.stock.ValidLotCnt(v.cash / v.stock.ipoPrice)
	// 融资可以申购几手
	v.financingLot = v.stock.ValidLotCnt(v.cash * v.broker.financingMultipleTimes / v.stock.ipoPrice)
	v.cost = NewIpoCost(v.stock, v.broker, v.cash, v.userFinancing, v.cashLot, v.financingLot)
	lot := v.cashLot
	if useFinancing {
		lot = v.financingLot
	}
	v.earning = NewIpoEarning(v.stock, v.broker, v.cash, v.userFinancing, lot)
	return v
}

func (v *IpoScheme) profit() int {
	v.logs = []string{}

	if !v.userFinancing {
		v.logs = append(v.logs, "现金申购")
	} else {
		v.logs = append(v.logs, "融资申购")
	}

	cost := v.cost.calc()
	v.logs = append(v.logs, fmt.Sprintf("成本:%v", v.cost))

	earning := v.earning.calc()
	v.logs = append(v.logs, fmt.Sprintf("收入:%v", v.earning))

	p := earning - cost
	v.logs = append(v.logs, fmt.Sprintf("盈利:%v", p))
	return p
}

func (v *IpoScheme) String() string {
	return strings.Join(v.logs, "\r\n")
}

// ipo 打新资金分配
type IpoArrange struct {
	stock *Stock
	brokers []*Broker
	cash int
	all_schemes []*IpoScheme
}

func NewIpoArrange(stock *Stock, cash int, brokers []*Broker) *IpoArrange {
	v := &IpoArrange{
		stock: stock,
		brokers: brokers,
		cash: cash,
		all_schemes: []*IpoScheme{},
	}
	return v
}

func (v *IpoArrange) arrange() {
	v.arrangeImpl([]*IpoScheme{}, 0)

	
}

// first currentSchemeList is empty, layer = 0
func (v *IpoArrange) arrangeImpl(currentSchemeList []*IpoScheme, layer int) {
	sum := 0
	for _, scheme := range currentSchemeList {
		sum += scheme.cash
	}
	if sum > v.cash {
		return
	}
	if sum == v.cash {
		v.all_schemes = append(v.all_schemes, currentSchemeList...)
		return
	}
	if layer >= len(v.brokers) {
		v.all_schemes = append(v.all_schemes, currentSchemeList...)
		return
	}

	// k指的是当前投入的资金
	for k := 0; k + sum > v.cash; {
		if k == 0 { //不投资该券商
			v.arrangeImpl(currentSchemeList, layer + 1)
		} else {
			// 纯现金投资该券商
			scheme := NewIpoScheme(v.stock, v.brokers[layer], k, false)
			currentSchemeList = append(currentSchemeList, scheme)
			v.arrangeImpl(currentSchemeList, layer + 1)
			currentSchemeList = append(currentSchemeList[:0], currentSchemeList[1:]...)

			// 融资投资该券商
			scheme = NewIpoScheme(v.stock, v.brokers[layer], k, true)
			currentSchemeList = append(currentSchemeList, scheme)
			v.arrangeImpl(currentSchemeList, layer + 1)
			currentSchemeList = append(currentSchemeList[:0], currentSchemeList[1:]...)

		}
		k += v.stock.ipoPrice
	}

}

func main()  {
	s := NewStock("test", 10700, 5, 0.06, 0.05, 0.007)
	log.Println(fmt.Sprintf("stock:%v", s))

	brokers := []*Broker{}
	brokers = append(brokers, &Broker{name: "富途", cashSubscribeFee:50, financingSubscribeFee: 100, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "长桥", cashSubscribeFee:49, financingSubscribeFee: 100, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "华盛", cashSubscribeFee:49, financingSubscribeFee: 100, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "东方财富国际版", cashSubscribeFee:49, financingSubscribeFee: 100, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "老虎", cashSubscribeFee:100, financingSubscribeFee: 100, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "辉立", cashSubscribeFee:0, financingSubscribeFee: 0, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "华泰", cashSubscribeFee:0, financingSubscribeFee: 0, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "艾德", cashSubscribeFee:0, financingSubscribeFee: 0, financingRate: 0.03, financingMultipleTimes: 10})
	brokers = append(brokers, &Broker{name: "耀才", cashSubscribeFee:0, financingSubscribeFee: 0, financingRate: 0.03, financingMultipleTimes: 10})

	for _, b := range brokers {
		log.Println(b)
	}
}