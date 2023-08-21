package emarsys_multiupdate

import (
	"bytes"
	"crypto/sha1"
	b64 "encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type E_MU interface {
	ObtainList(searchValue string) ([]int, error)
	UpdateAllWithValue(dups_list []int, target_key string /*emarsys_id*/, target_value string) error
}

type EdeData struct {
	Emarsys_auth   SuiteAPI
	SearchField    string
	SkipWhereEmpty string
}

type ReturnedDupsList struct {
	ReplyCode int    `json:"replyCode"`
	ReplyText string `json:"replyText"`
	Data      struct {
		Errors []interface{} `json:"errors"`
		Result []struct {
			Optin string `json:"31"`
			ID    string `json:"id"`
		} `json:"result"`
	} `json:"data"`
}

type ReturnedDupsListWithLimit struct {
	ReplyCode int    `json:"replyCode"`
	ReplyText string `json:"replyText"`
	Data      struct {
		Errors []interface{} `json:"errors"`
		Result []interface{} `json:"result"`
	} `json:"data"`
}

func (EData EdeData) ObtainList(searchValue string) ([]int, error) {

	var dups_slice []int

	switch EData.SkipWhereEmpty == "" {

	default:
		dups_json := ReturnedDupsList{}
		_, returnedDups := EData.Emarsys_auth.send("GET", "contact/query/?"+"return=31"+"&"+EData.SearchField+"="+searchValue, "")

		err := json.Unmarshal([]byte(returnedDups), &dups_json)

		if err != nil {
			fmt.Println("result: " + string(returnedDups))
			fmt.Println("url: " + "contact/query/?" + "return=31" + "&" + EData.SearchField + "=" + searchValue)
			return []int{}, err

		}

		var dups_slice []int

		for k := range dups_json.Data.Result {

			contact_id, err := strconv.Atoi(dups_json.Data.Result[k].ID)

			if err != nil {

				return []int{}, errors.New("non integer contact_id found \n" + "please report to TCS\n" + "contact_id: " + dups_json.Data.Result[k].ID)

			}

			//dups_slice += `"` + dups_json.Data.Result[k].ID + `"` + ","
			dups_slice = append(dups_slice, contact_id)

		}

	case false:
		dups_json := ReturnedDupsListWithLimit{}
		_, returnedDups := EData.Emarsys_auth.send("GET", "contact/query/?"+"return="+EData.SkipWhereEmpty+"&"+EData.SearchField+"="+searchValue, "")

		returnedDups = strings.ReplaceAll(returnedDups, "null", "\"\"")

		err := json.Unmarshal([]byte(returnedDups), &dups_json)

		if err != nil {
			fmt.Println("result: " + returnedDups)
			fmt.Println("url: " + "contact/query/?" + "return=31" + "&" + EData.SearchField + "=" + searchValue)
			return []int{}, err

		}

		for k := range dups_json.Data.Result {
			_, ok := dups_json.Data.Result[k].(map[string]string)
			if ok {

				if dups_json.Data.Result[k].(map[string]string)[EData.SkipWhereEmpty] == "" {

					continue
				}

				contact_id, err := strconv.Atoi(dups_json.Data.Result[k].(map[string]string)["id"])

				if err != nil {

					return []int{}, errors.New("non integer contact_id found \n" + "please report to TCS\n" + "contact_id: " + dups_json.Data.Result[k].(map[string]string)["id"])

				}

				//dups_slice += `"` + dups_json.Data.Result[k].ID + `"` + ","
				dups_slice = append(dups_slice, contact_id)

			}

		}
	}

	return dups_slice, nil

}

func (EData EdeData) UpdateAllWithValue(dups_list []int, target_key string /*emarsys_id*/, target_value string) error {

	var contacts_str string

	for i := range dups_list {

		contacts_str += `,{"id":"` + strconv.Itoa(dups_list[i]) + `",

"` + target_key + `": "` + target_value + `" 

}`

	}

	j_str := `{
	"key_id": "id",
		"contacts": [` +
		contacts_str[1:] +
		`]
}`

	status, resp := EData.Emarsys_auth.send("PUT", "contact", j_str)

	fmt.Println(resp)

	if status != "200" {

		return errors.New("API returned an error: " + resp)

	}
	return nil

}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func generateRandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

type SuiteAPI struct {
	user   string
	secret string
}

// Sends an HTTP request to the Emarsys API
func (config SuiteAPI) send(method string, path string, body string) (status string, respBody string) {
	url := "https://api.emarsys.net/api/v2/" + path
	var timestamp = time.Now().Format(time.RFC3339)
	nonce := generateRandString(36)
	text := (nonce + timestamp + config.secret)
	h := sha1.New()
	h.Write([]byte(text))
	sha1 := hex.EncodeToString(h.Sum(nil))
	passwordDigest := b64.StdEncoding.EncodeToString([]byte(sha1))

	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	header := string(" UsernameToken Username=\"" + config.user + "\",PasswordDigest=\"" + passwordDigest + "\",Nonce=\"" + nonce + "\",Created=\"" + timestamp + "\"")

	req.Header.Set("X-WSSE", header)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	status = resp.Status
	responseBody, _ := ioutil.ReadAll(resp.Body)

	respBody = string(responseBody)
	return
}
