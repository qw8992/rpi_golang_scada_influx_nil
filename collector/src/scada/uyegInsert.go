package main

import (
	"log"
	"reflect"
	"sort"
	"strings"
	"time"
	"fmt"
	client "github.com/Heo-youngseo/influxdb1-client/v2"
)

//influxDB 사용하기 위한 모듈을 import, DB 연결 정보를 전역변수로 선언
const (
	database = "UYeG_influx"
	username = "its"
	password = "its@1234"
)

//influxDB 연결 & 수집된 데이터를 InfluxDB에 저장하는 함수 실행
func influxDataInsert(chInserData chan map[string]interface{}, chnull chan map[string]interface{}) {
	for {
		select {
		case <-chInserData:
			c := influxDBClient()
			createMetrics(c, chInserData)

		case <-chnull:
			c := influxDBClient()
			createMetrics(c, chnull)
		}
	}
}

//influxDB에 연결시키는 함수
func influxDBClient() client.Client {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://192.168.100.92:8086",
		Username: username,
		Password: password,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		fmt.Println("influxDBClient")
		log.Fatalln("Error: ", err)
	}
	return c
}

//수집한 데이터를 influxdb에 저장하는 함수, 데이터를 받으면 새로운 배치포인트를 만듦
func createMetrics(c client.Client, chInserData chan map[string]interface{}) {

	for {
		select {
		case data := <-chInserData:
			bp, err := client.NewBatchPoints(client.BatchPointsConfig{
				Database:  database,
				Precision: "ms",
			})

			if err != nil {
				fmt.Println("create1")
				log.Fatalln("Error: ", err)
			}

			//1초 데이터의 키를 오름차순으로 정렬하여 string으로 조인하고 변경해야할 이름을 수정
			values := data["Values"]

			//1초 데이터 key 오름차순으로 정렬
			keySec := orderKey(data)
			tempStrSec := strings.Join(keySec[:], ",")
			tempStrSec = strings.Replace(tempStrSec, ",time", "", 1)
			tempStrSec = strings.Replace(tempStrSec, ",ver", "", 1)
			tempStrSec = strings.Replace(tempStrSec, ",gateway", "", 1)
			tempStrSec = strings.Replace(tempStrSec, ",mac", "", 1)
			tempStrSec = strings.Replace(tempStrSec, ",Values", "", 1)
			arrKeySec := strings.Split(tempStrSec, ",")

			switch reflect.TypeOf(values).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(values)

				//0.1초 데이터 key 오름차순으로 정렬
				keyMilli := orderKey(s.Index(0).Interface().(map[string]interface{}))
				tempStrMilli := strings.Join(keyMilli[:], ",")
				tempStrMilli = strings.Replace(tempStrMilli, "time", "DataSavedTime", 1)
				tempStrMilli = strings.Replace(tempStrMilli, "420", "`420`", 1)

				//insert문 생성
				//데이터를 차례대로 불러와서 fields에 추가, 추가 할때 tag로는 mac주소와 gatewayID로 한다(기본키 개념)
				for i := 0; i < s.Len(); i++ {

					dataMilli := s.Index(i).Interface().(map[string]interface{})

					tags := map[string]string{
						"mac":     data["mac"].(string),
						"gateway": data["gateway"].(string),
					}

					fields := make(map[string]interface{})
					for j := 0; j < len(arrKeySec); j++ {
						fields[arrKeySec[j]] = data[arrKeySec[j]]
					}

					for j := 0; j < len(keyMilli); j++ {
						fields[keyMilli[j]] = dataMilli[keyMilli[j]]
					}

					//마지막으로 string으로 된 시간을 time형으로 변경하고 point를 추가한 후 batchpoint에 저장
					date := dataMilli["time"].(string)
					t, err := time.Parse("2006-01-02 15:04:05.000", date)
					point, err := client.NewPoint(
						"SmartEOCR",
						tags,
						fields,
						t,
					)

					if err != nil {
						fmt.Println("Create")
						log.Fatalln("Error: ", err)
					}

					bp.AddPoint(point)
				}
			}

			//influxDB에 Insert
			go func() {
				err = c.Write(bp)
				if err != nil {
					log.Fatal(err)
				}
			}()
		}
	}
}

// map의 key를 오름차순으로 재정렬하여 오름차순한 key 배열을 반환하는 함수
func orderKey(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys) //sort by key
	return keys
}
