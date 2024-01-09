package main

// ZhfdResult 是智慧房东接口请求结果
type ZhfdResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Records []*RentInfo `json:"records"`
	} `json:"data"`
}

// RentInfo 存放用户的水电信息
type RentInfo struct {
	MeterAddForms []*struct {
		ResidualElectricity string `json:"residualElectricity"` // 剩余电量
	} `json:"meterAddForms"`

	WaterAddForms []*struct {
		Remarks         string `json:"remarks"`         // 备注
		RechargeTonnage string `json:"rechargeTonnage"` // 剩余水量
	} `json:"waterAddForms"`
}
