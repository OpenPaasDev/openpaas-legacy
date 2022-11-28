package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type IP struct {
	Query string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] //nolint
	}
	return string(b)
}

func GetPublicIP(ctx context.Context) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		e := resp.Body.Close()
		fmt.Println(e)
	}()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}
	var ip IP
	err = json.Unmarshal(body, &ip)
	if err != nil {
		return "", err
	}
	// fmt.Print(ip.Query)
	return ip.Query, nil
}
