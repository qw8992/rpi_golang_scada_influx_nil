package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	//	"log"
	//	"os"
	"scada/uyeg"
	"time"
)

type RemapFormatV2 struct {
	Version          string     `json:"ver"`
	GatewayID        string     `json:"gateway"`
	MacID            string     `json:"mac"`
	Time             string     `json:"time"`
	Temp             string     `json:"Temp"`
	Humid            string     `json:"Humid"`
	ReactivePower    string     `json:"ReactivePower"`
	ActiveConsum     string     `json:"ActiveConsum"`
	ReactiveConsum   string     `json:"ReactiveConsum"`
	Power            string     `json:"Power"`
	RunningHour      string     `json:"RunningHour"`
	TotalRunningHour string     `json:"TotalRunningHour"`
	MCCounter        string     `json:"MCCounter"`
	PT100            string     `json:"PT100"`
	FaultNumber      string     `json:"FaultNumber"`
	OverCurrR        string     `json:"OverCurrR"`
	OverCurrS        string     `json:"OverCurrS"`
	OverCurrT        string     `json:"OverCurrT"`
	FaultRST         string     `json:"FaultRST"`
	Values           []Depth2V1 `json:"Values"`
}

type Depth2V1 struct {
	Time        string `json:"time"`
	Status      string `json:"status"`
	Curr        string `json:"Curr"`
	CurrR       string `json:"CurrR"`
	CurrS       string `json:"CurrS"`
	CurrT       string `json:"CurrT"`
	Volt        string `json:"Volt"`
	VoltR       string `json:"VoltR"`
	VoltS       string `json:"VoltS"`
	VoltT       string `json:"VoltT"`
	ActivePower string `json:"ActivePower"`
	Ground      string `json:"Ground"`
	V420        string `json:"420"`
}

// UYeGDataCollection 함수는 데이터를 수집하는 함수입니다
func UYeGDataCollection(client *uyeg.ModbusClient, collChan chan<- map[string]interface{}, nullData chan<- map[string]interface{}) {
	var errCount, errCountConn = 0, 0
	ticker := time.NewTicker(10 * time.Millisecond)

	for {
		select {
		case <-client.Done1:
			fmt.Println(fmt.Sprintf("=> %s (%s:%d) 데이터 수집 종료", client.Device.MacId, client.Device.Host, client.Device.Port))
			return
		case <-ticker.C:
			readData := client.GetReadHoldingRegisters()

			if readData == nil {
				Mili := 0
				var vT string
				device := DeviceCount()
				rpFormat := &RemapFormatV2{}
				rpFormat.Values = []Depth2V1{}
				var TimeFormat = "2006-01-02 15:04:05.000"
				var loc, _ = time.LoadLocation("Asia/Seoul")

				for i := device * 30; i > 0; i = i - 30 {
					value := Depth2V1{}

					t := fmt.Sprint(time.Now().In(loc))
					bSecT := t[:len(TimeFormat)-4]
					if Mili == 0 {
						vT = fmt.Sprint(bSecT, ".000")
					} else {
						vT = fmt.Sprint(bSecT, ".", Mili*100)
					}
					value.Time = vT
					value.Curr = "0"
					value.CurrR = "0"
					value.CurrS = "0"
					value.CurrT = "0"
					value.Volt = "0"
					value.VoltR = "0"
					value.VoltS = "0"
					value.VoltT = "0"
					value.ActivePower = "0"
					value.Ground = "0"
					value.V420 = "0"
					value.Status = "false"

					rpFormat.Values = append(rpFormat.Values, value)
				}
				t := fmt.Sprint(time.Now().In(loc))

				bSecT := t[:len(TimeFormat)-4]
				rpFormat.Time = bSecT
				rpFormat.Version = "2"
				rpFormat.GatewayID = client.Device.GatewayId
				rpFormat.MacID = client.Device.MacId
				rpFormat.Temp = "0"
				rpFormat.Humid = "0"
				rpFormat.ActiveConsum = "0"
				rpFormat.RunningHour = "0"
				rpFormat.OverCurrR = "0"
				rpFormat.OverCurrS = "0"
				rpFormat.OverCurrT = "0"
				rpFormat.Power = "0"
				rpFormat.ReactiveConsum = "0"
				rpFormat.ReactivePower = "0"
				rpFormat.TotalRunningHour = "0"
				rpFormat.MCCounter = "0"
				rpFormat.PT100 = "0"
				rpFormat.FaultNumber = "0"
				rpFormat.FaultRST = "0"

				jsonBytes, _ := json.Marshal(rpFormat)
				dataSecond := make(map[string]interface{})
				json.Unmarshal(jsonBytes, &dataSecond)
				fmt.Println(dataSecond)

				nullData <- dataSecond

				//test:= readData;
				ticker.Stop()
				errCount = errCount + 1
				fmt.Println(time.Now().In(Loc).Format(TimeFormat), fmt.Sprintf("Failed to read data Try again (%s:%d)..", client.Device.Host, client.Device.Port))
				log1 := fmt.Sprintf("Failed to read data Try again (%s:%d)..", client.Device.Host, client.Device.Port)
				dbConn.NotResultQueryExec(fmt.Sprintf("INSERT INTO E_LOG(MAC_ID, LOG, CREATE_DATE) VALUES ('%s', '%s', NOW());", client.Device.MacId, log1))
				if errCount > client.Device.RetryCount {
					client.Handler.Close()
					if client.Connect() {
						fmt.Println(time.Now().In(Loc).Format(TimeFormat), fmt.Sprintf("Succeded to reconnect the connection.. (%s:%d)..", client.Device.Host, client.Device.Port))
						errCount = 0
					} else {
						fmt.Println(time.Now().In(Loc).Format(TimeFormat), fmt.Sprintf("Failed to reconnect the connection.. (%s:%d)..", client.Device.Host, client.Device.Port))
						log2 := fmt.Sprintf("Failed to reconnect the connection.. (%s:%d)..", client.Device.Host, client.Device.Port)
						dbConn.NotResultQueryExec(fmt.Sprintf("INSERT INTO E_LOG(MAC_ID, LOG, CREATE_DATE) VALUES ('%s', '%s', NOW());", client.Device.MacId, log2))
						errCountConn = errCountConn + 1

						if errCountConn > client.Device.RetryConnFailedCount {
							derr := make(map[string]interface{})
							derr["Device"] = client.Device
							derr["Error"] = fmt.Sprintf("%s(%s): Connection failed..", client.Device.Name, client.Device.MacId)
							derr["Restart"] = false

							ErrChan <- derr

							dbConn.NotResultQueryExec(fmt.Sprintf("INSERT INTO E_LOG(MAC_ID, LOG, CREATE_DATE) VALUES ('%s', '%s', NOW());", client.Device.MacId, derr["Error"].(string)))
						}
					}
				}
				time.Sleep(time.Duration(client.Device.RetryCycle) * time.Millisecond)
				ticker = time.NewTicker(10 * time.Millisecond)
				continue
			} else {
				errCount = 0
			}

			collChan <- readData
		}
	}
}

func DeviceCount() (device int) {
	db, err := sql.Open("mysql", "root:its@1234@tcp(127.0.0.1:3306)/UYeG")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 하나의 Row를 갖는 SQL 쿼리
	DeviceQuery := fmt.Sprintf("SELECT Count(Enabled) FROM DEVICE WHERE Enabled = '1'")
	err = db.QueryRow(DeviceQuery).Scan(&device)
	if err != nil {
		// log.Fatal(err)
	}
	return device
}
