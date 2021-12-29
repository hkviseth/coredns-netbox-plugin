// Copyright 2020 Oz Tiram <oz.tiram@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package netbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	logger "log"
	"net/http"
	"strings"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/imkira/go-ttlmap"
)

type Record struct {
	Family   Family `json:"family"`
	Address  string `json:"address"`
	HostName string `json:"dns_name,omitempty"`
}

type Family struct {
	Version int    `json:"value"`
	Label   string `json:"label"`
}

type RecordsList struct {
	Records []Record `json:"results"`
}

var localCache = ttlmap.New(nil)

func query(url, token, dns_name string, duration time.Duration, family int) string {
	item, err := localCache.Get(dns_name)
	if err == nil {
		clog.Debug(fmt.Sprintf("Found in local cache %s", dns_name))
		logger.Printf("Found in local cache %s", dns_name)
		return item.Value().(string)
	} else {
		records := RecordsList{}
		client := &http.Client{}
		var resp *http.Response
		clog.Debug("Querying ", fmt.Sprintf("%s/?dns_name=%s", url, dns_name))
		logger.Printf("Querying %s ", fmt.Sprintf("%s/?dns_name=%s", url, dns_name))
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/?dns_name=%s", url, dns_name), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))

		for i := 1; i <= 10; i++ {
			resp, err = client.Do(req)

			if err != nil {
				clog.Fatalf("HTTP Error %v", err)
			}

			if resp.StatusCode == http.StatusOK {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if resp.StatusCode != http.StatusOK {
			return ""
		}

		body, err := ioutil.ReadAll(resp.Body)
		logger.Printf("%s", body)
		if err != nil {
			clog.Fatalf("Error reading body %v", err)
		}

		jsonAns := string(body)
		err = json.Unmarshal([]byte(jsonAns), &records)
		if err != nil {
			clog.Fatalf("could not unmarshal response %v", err)
		}

		if len(records.Records) == 0 {
			clog.Info("Recored not found in", jsonAns)
			return ""
		}

		var ip_address string
		switch family {
		case 4:
			for _, r := range records.Records {
				if r.Family.Version == 4 {
					ip_address = strings.Split(r.Address, "/")[0]
					localCache.Set(dns_name, ttlmap.NewItem(ip_address, ttlmap.WithTTL(duration)), nil)
					break
				}
			}
		case 6:
			for _, r := range records.Records {
				if r.Family.Version == 6 {
					ip_address = strings.Split(r.Address, "/")[0]
					localCache.Set(dns_name, ttlmap.NewItem(ip_address, ttlmap.WithTTL(duration)), nil)
					break
				}
			}

		}
		return ip_address
	}
}
