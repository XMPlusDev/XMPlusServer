package api
import (
	"log"
	"sync/atomic"
	"time"
	"sync"
	"fmt"
	
	"github.com/go-resty/resty/v2"
	"github.com/bitly/go-simplejson"
)

type Client struct {
	client           *resty.Client
	APIHost          string
	NodeID           int
	Key              string
	resp             atomic.Value
	eTags            map[string]string
	LastReportOnline map[int]int
	access           sync.Mutex
}

type ClientInfo struct {
	APIHost string
	NodeID  int
	Key     string
}

func New(apiConfig *Config) *Client {
	client := resty.New()
	client.SetRetryCount(5)
	if apiConfig.Timeout > 0 {
		client.SetTimeout(time.Duration(apiConfig.Timeout) * time.Second)
	} else {
		client.SetTimeout(30 * time.Second)
	}
	
	//client.SetQueryParam("key", apiConfig.Key)
	
	client.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			// v.Response contains the last response from the server
			// v.Err contains the original error
			log.Print(v.Err)
		}
	})
	
	client.SetBaseURL(apiConfig.APIHost)
	
	apiClient := &Client{
		client:           client,
		NodeID:           apiConfig.NodeID,
		Key:              apiConfig.Key,
		APIHost:          apiConfig.APIHost,
		LastReportOnline: make(map[int]int),
		eTags:            make(map[string]string),
	}
	
	return apiClient
}

func (c *Client) Describe() ClientInfo {
	return ClientInfo{APIHost: c.APIHost, NodeID: c.NodeID, Key: c.Key}
}

func (c *Client) Debug() {
	c.client.SetDebug(true)
}

func (c *Client) checkResponse(res *resty.Response, err error) (*simplejson.Json, error) {
	if err != nil {
		// Get request URL from response
		var requestURL string
		if res != nil && res.Request != nil && res.Request.RawRequest != nil {
			requestURL = res.Request.RawRequest.URL.String()
		}
				
		return nil, fmt.Errorf("request error occurred for URL %s: %s", requestURL, err)
	}
	
	if res.StatusCode() >= 400 {
		requestURL := "unknown"
		if res.Request != nil && res.Request.RawRequest != nil {
			requestURL = res.Request.RawRequest.URL.String()
		}
		
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %v", requestURL, string(body), err)
	}

	
	result, err := simplejson.NewJson(res.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %s", res.String())
	}
	
	return result, nil
}