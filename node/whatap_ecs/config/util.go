package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"log"
)

var (
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func GetHttpResponseLines(url string, respcallback func(string)) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf(resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	httplog(nil, nil, resp, content)

	if resp.StatusCode < 300 {
		respcallback(string(content))
	} else {
		return fmt.Errorf(string(content))
	}

	return nil
}

func GetHttpWithHeaderResponseLines(url string, headers map[string]string, respcallback func([]byte)) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	httplog(req, nil, resp, content)

	if resp.StatusCode < 300 {
		respcallback(content)
	} else {
		return fmt.Errorf(string(content))
	}

	return nil
}

func PutHttpWithHeaderResponseLines(url string, headers map[string]string, body []byte, respcallback func([]byte)) error {
	// PUT 요청을 생성하며, body 데이터를 포함시킵니다.
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	// 헤더를 설정합니다.
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 클라이언트 요청을 수행합니다.
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 응답의 내용을 읽습니다.
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	httplog(req, body, resp, content)

	// 성공적인 응답(상태 코드가 300 미만)일 경우 콜백 함수를 호출합니다.
	if resp.StatusCode < 300 {
		respcallback(content)
	} else {
		// 오류가 발생한 경우, 응답 내용을 포함하여 오류를 반환합니다.
		return fmt.Errorf(string(content))
	}

	return nil
}

var debug = false

func httplog(request *http.Request, requestBody []byte, response *http.Response, responseBody []byte) {
	if debug {
		// 요청을 로깅합니다.
		if request != nil {
			log.Println(fmt.Sprintf("HTTP Request: %s %s\n", request.Method, request.URL.String()))
			log.Println(fmt.Sprintf("Request Headers: %v\n", request.Header))

		}
		if len(requestBody) > 0 {
			log.Println(fmt.Sprintf("Request Body: %s\n", string(requestBody)))
		}

		// 응답을 로깅합니다.
		if response != nil {
			log.Println(fmt.Sprintf("HTTP Response Status: %d\n", response.StatusCode))
			log.Println(fmt.Sprintf("Response Headers: %v\n", response.Header))
			if len(responseBody) > 0 {
				log.Println(fmt.Sprintf("Response Body: %s\n", string(responseBody)))
			}
		}
	}

}
