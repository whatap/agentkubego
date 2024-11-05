package internal

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteToJson(w http.ResponseWriter, dataArr any) {
	err := json.NewEncoder(w).Encode(dataArr)
	if err != nil {
		log.Println("fail to getData encode", err)
		return
	}
}
