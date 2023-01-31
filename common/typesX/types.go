package typesX

type ChannelData struct {
	ID                      int64         `json:"id, optional"`
	Code                    string        `json:"code, optional"`
	Name                    string        `json:"name, optional"`
	IsProxy                 string        `json:"isProxy, optional"`
	IsNzPre                 string        `json:"isNzPre, optional"`
	ApiUrl                  string        `json:"apiUrl, optional"`
	CurrencyCode            string        `json:"currencyCode, optional"`
	ChannelWithdrawCharge   float64       `json:"channelWithdrawCharge, optional"`
	Balance                 float64       `json:"balance, optional"`
	Status                  string        `json:"status, optional"`
	Device                  string        `json:"device,optional"`
	MerId                   string        `json:"merId, optional"`
	MerKey                  string        `json:"merKey, optional"`
	PayUrl                  string        `json:"payUrl, optional"`
	PayQueryUrl             string        `json:"payQueryUrl, optional"`
	PayQueryBalanceUrl      string        `json:"payQueryBalanceUrl, optional"`
	ProxyPayUrl             string        `json:"proxyPayUrl, optional"`
	ProxyPayQueryUrl        string        `json:"proxyPayQueryUrl, optional"`
	ProxyPayQueryBalanceUrl string        `json:"proxyPayQueryBalanceUrl, optional"`
	WhiteList               string        `json:"whiteList, optional"`
	PayTypeMapList          []PayTypeMap  `json:"payTypeMapList, optional" gorm:"-"`
	PayTypeMap              string        `json:"payTypeMap, optional"`
	ChannelPort             string        `json:"channelPort, optional"`
	WithdrawBalance         float64       `json:"withdrawBalance, optional"`
	ProxypayBalance         float64       `json:"proxypayBalance, optional"`
	BankCodeMapList         []BankCodeMap `json:"bankCodeMapList, optional" gorm:"-"`
	Banks                   []Bank        `json:"banks, optional" gorm:"many2many:ch_channel_banks;foreignKey:Code;joinForeignKey:channel_code;references:bank_no;joinReferences:bank_no"`
}

type PayTypeMap struct {
	PayType string `json:"payType"`
	TypeNo  string `json:"typeNo"`
	MapCode string `json:"mapCode"`
}

type BankCodeMap struct {
	ID          int64  `json:"id, optional"`
	ChannelCode string `json:"channelCode, optional"`
	BankNo      string `json:"bankNo"`
	MapCode     string `json:"mapCode"`
}

type BankCodeMapX struct {
	BankCodeMap
	BankName string `json:"bankName"`
}

type Bank struct {
	ID           int64         `json:"id"`
	BankNo       string        `json:"bankNo"`
	BankName     string        `json:"bankName"`
	BankNameEn   string        `json:"bankNameEn"`
	Abbr         string        `json:"abbr"`
	BranchNo     string        `json:"branchNo"`
	BranchName   string        `json:"branchName"`
	City         string        `json:"city"`
	Province     string        `json:"province"`
	CurrencyCode string        `json:"currencyCode"`
	Status       string        `json:"status"`
	ChannelDatas []ChannelData `json:"channelDatas, optional" gorm:"many2many:ch_channel_banks:foreignKey:bankNo;joinForeignKey:bank_no;references:code;joinReferences:channel_code"`
}

type ChannelPayType struct {
	ID                int64   `json:"id, optional"`
	Code              string  `json:"code, optional"`
	ChannelCode       string  `json:"channelCode, optional"`
	PayTypeCode       string  `json:"payTypeCode, optional"`
	Fee               float64 `json:"fee, optional"`
	HandlingFee       float64 `json:"handlingFee, optional"`
	MaxInternalCharge float64 `json:"maxInternalCharge, optional"`
	DailyTxLimit      float64 `json:"dailyTxLimit, optional"`
	SingleMinCharge   float64 `json:"singleMinCharge, optional"`
	SingleMaxCharge   float64 `json:"singleMaxCharge, optional"`
	FixedAmount       string  `json:"fixedAmount, optional"`
	BillDate          int64   `json:"billDate, optional"`
	Status            string  `json:"status, optional"`
	IsProxy           string  `json:"isProxy, optional"`
	Device            string  `json:"device, optional"`
	MapCode           string  `json:"mapCode, optional"`
}

type Order struct {
	ID                      int64   `json:"id"`
	Type                    string  `json:"type"` //收支方式  (代付 DF 支付 ZF 下發 XF 內充 NC)
	MerchantCode            string  `json:"merchantCode"`
	TransAt                 string  `json:"transAt"`
	OrderNo                 string  `json:"orderNo"`
	BalanceType             string  `json:"balanceType"`
	BeforeBalance           float64 `json:"beforeBalance"`
	TransferAmount          float64 `json:"transferAmount"`
	Balance                 float64 `json:"balance"`
	FrozenAmount            float64 `json:"frozenAmount"`
	Status                  string  `json:"status"` //訂單狀態(0:待處理 1:處理中 2:交易中 20:成功 30:失敗 31:凍結)
	IsLock                  string  `json:"isLock"`
	RepaymentStatus         string  `json:"repaymentStatus"` //还款状态：(0：不需还款、1:待还款、2：还款成功、3：还款失败)
	Memo                    string  `json:"memo"`
	ErrorType               string  `json:"errorType, optional"`
	ErrorNote               string  `json:"errorNote, optional"`
	ChannelCode             string  `json:"channelCode"`
	ChannelPayTypesCode     string  `json:"channelPayTypesCode"`
	PayTypeCode             string  `json:"payTypeCode"`
	Fee                     float64 `json:"fee"`
	HandlingFee             float64 `json:"handlingFee"`
	InternalChargeOrderPath string  `json:"internalChargeOrderPath"`
	CurrencyCode            string  `json:"currencyCode"`
	MerchantBankAccount     string  `json:"merchantBankAccount"`  //商戶銀行帳號
	MerchantBankNo          string  `json:"merchantBankNo"`       //商戶銀行代碼
	MerchantBankName        string  `json:"merchantBankName"`     //商戶姓名
	MerchantBankBranch      string  `json:"merchantBankBranch"`   //商戶銀行分行名
	MerchantBankProvince    string  `json:"merchantBankProvince"` //商戶開戶縣市名
	MerchantBankCity        string  `json:"merchantBankCity"`     //商戶開戶縣市名
	MerchantAccountName     string  `json:"merchantAccountName"`  //開戶行名(代付、下發)
	ChannelBankAccount      string  `json:"channelBankAccount"`   //渠道銀行帳號
	ChannelBankNo           string  `json:"channelBankNo"`        //渠道銀行代碼
	ChannelBankName         string  `json:"channelBankName"`      //渠道银行名称
	ChannelAccountName      string  `json:"channelAccountName"`   //渠道账户姓名
	OrderAmount             float64 `json:"orderAmount"`          // 订单金额
	ActualAmount            float64 `json:"actualAmount"`         // 实际金额
	TransferHandlingFee     float64 `json:"transferHandlingFee"`
	MerchantOrderNo         string  `json:"merchantOrderNo"` //商戶訂單編號
	ChannelOrderNo          string  `json:"channelOrderNo"`  //渠道订单编号
	Source                  string  `json:"source"`          //1:平台 2:API
	SourceOrderNo           string  `json:"sourceOrderNo"`   //來源訂單編號(From NC)
	CallBackStatus          string  `json:"callBackStatus, optional"`
	NotifyUrl               string  `json:"notifyUrl, optional"`
	PageUrl                 string  `json:"pageUrl, optional"`
	PersonProcessStatus     string  `json:"personProcessStatus, optional"` //人工处理状态：(0:待處理1:處理中2:成功3:失敗 10:不需处理)
	IsMerchantCallback      string  `json:"isMerchantCallback, optional"`  //是否已经回调商户(0：否、1:是、2:不需回调)(透过API需提供的资讯)
	CreatedBy               string  `json:"createdBy, optional"`
	UpdatedBy               string  `json:"updatedBy, optional"`
}

type TransactionLogData struct {
	MerchantNo      string      `json:"merchantNo"`
	MerchantOrderNo string      `json:"merchantOrderNo"`
	OrderNo         string      `json:"orderNo"`
	LogType         string      `json:"logType"`
	LogSource       string      `json:"logSource"`
	Content         interface{} `json:"content"`
	ErrCode         string      `json:"errCode"`
	ErrMsg          string      `json:"errMsg"`
	TraceId         string      `json:"traceId"`
}

type TxLog struct {
	ID              int64  `json:"id"`
	MerchantCode    string `json:"merchantCode, optional"`
	OrderNo         string `json:"orderNo, optional"`
	MerchantOrderNo string `json:"merchantOrderNo, optional"`
	ChannelOrderNo  string `json:"channelOrderNo, optional"`
	LogType         string `json:"logType, optional"`
	LogSource       string `json:"logSource, optional"`
	Content         string `json:"content, optional"`
	Log             string `json:"log, optional"`
	CreatedAt       string `json:"createdAt, optional"`
	ErrorCode       string `json:"errorCode, optional"`
	ErrorMsg        string `json:"errorMsg, optional"`
	TraceId         string `json:"traceId, optional"`
}
