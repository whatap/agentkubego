package config

import (
	"fmt"
)

var (
	imdsV2Supported   bool = false
	token             string
	ImdsTokenDuration int = 60 // 토큰 유효 기간

)

func getImdsv2Token() (err error, token string) {
	// 토큰 발급 URL
	tokenUrl := "http://169.254.169.254/latest/api/token"
	// 토큰 요청을 위한 헤더 설정
	tokenHeader := map[string]string{
		"X-aws-ec2-metadata-token-ttl-seconds": fmt.Sprint(ImdsTokenDuration),
	}

	// 새로운 토큰 발급 요청
	err = PutHttpWithHeaderResponseLines(tokenUrl, tokenHeader, nil, func(respbytes []byte) {
		token = string(respbytes)
		imdsV2Supported = true // IMDSv2 지원 여부를 true로 설정
	})

	// 오류 발생 시 함수 종료
	if err != nil {
		return err, ""
	}

	// 새로운 토큰을 반환
	return nil, token
}
