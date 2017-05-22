package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/kpango/httgo"
)

func main() {
	body, errs := httgo.Get("https://hogehoge.xyz").Do().GetRawBody()

	if errs != nil {
		for _, err := range errs {
			log.Println(err)
		}
		return
	}

	bstr, err := ioutil.ReadAll(body)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bstr)

	ctx, cancel := context.WithCancel(context.Background())

	var jsonBody map[string]interface{}
	var xmlBody map[string]interface{}

	errs = httgo.New().SetBasicAuth("user", "passowrd").
		EnableCache().
		EnableRedirct().
		SetRedirectCount(10).
		Post("http://hogehoge/api/v1/foofoo").
		DoWithContext(ctx).
		JSON(&jsonBody).
		ResetClient().
		Get("http://foofoo/api/v1").
		Do().
		XML(&xmlBody).
		GetErrors()

	cancel()
}
