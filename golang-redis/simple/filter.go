package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	// "time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/go-redis/redis/v8"
)

var UpdateUpstreamBody = "upstream response body updated by the simple plugin"

// The callbacks in the filter, like `DecodeHeaders`, can be implemented on demand.
// Because api.PassThroughStreamFilter provides a default implementation.
type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	path      string
	config    *config
}
var counter = 0

func init() {
	api.LogErrorf("TEst log")
	// print test log every second
	// go func() {
	// 	for {
	// 		api.LogInfof("test log")
	// 		time.Sleep(time.Second)
	// 	}
	// }()

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Respond to every request with "hello"
			fmt.Fprintln(w, "hello")
			api.LogInfof("handling server request")
			counter = counter+1
		})

		// Listen on port 8088 and log errors if any
		err := http.ListenAndServe(":8088", nil)
		if err != nil {
			api.LogErrorf("Error starting server: %v", err)
		}
	}()
}

func (f *filter) sendLocalReplyInternal() api.StatusType {
	body := fmt.Sprintf("%s, path: %s\r\n", f.config.echoBody, f.path)
	f.callbacks.DecoderFilterCallbacks().SendLocalReply(200, body, nil, 0, "")
	// Remember to return LocalReply when the request is replied locally
	return api.LocalReply
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	f.path, _ = header.Get(":path")
	api.LogInfof("get path %s", f.path)
	body, err := fetchAPIResponse()
	if err != nil {
		api.LogErrorf("Error occured while calling out: %+v", err)
	}
	api.LogInfof("got response %s", body)
	call_redis()

	if f.path == "/localreply_by_config" {
		return f.sendLocalReplyInternal()
	}
	return api.Continue
	/*
		// If the code is time-consuming, to avoid blocking the Envoy,
		// we need to run the code in a background goroutine
		// and suspend & resume the filter
		go func() {
			defer f.callbacks.DecoderFilterCallbacks().RecoverPanic()
			// do time-consuming jobs

			// resume the filter
			f.callbacks.DecoderFilterCallbacks().Continue(status)
		}()

		// suspend the filter
		return api.Running
	*/
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

// Callbacks which are called in response path
// The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if f.path == "/update_upstream_response" {
		header.Set("Content-Length", strconv.Itoa(len(UpdateUpstreamBody)))
	}
	header.Set("Rsp-Header-From-Go", fmt.Sprintf("bar-test-%d", counter))
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

// EncodeData might be called multiple times during handling the response body.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if f.path == "/update_upstream_response" {
		if endStream {
			buffer.SetString(UpdateUpstreamBody)
		} else {
			buffer.Reset()
		}
	}
	// support suspending & resuming the filter in a background goroutine
	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

// OnLog is called when the HTTP stream is ended on HTTP Connection Manager filter.
func (f *filter) OnLog(reqHeader api.RequestHeaderMap, reqTrailer api.RequestTrailerMap, respHeader api.ResponseHeaderMap, respTrailer api.ResponseTrailerMap) {
	code, _ := f.callbacks.StreamInfo().ResponseCode()
	respCode := strconv.Itoa(int(code))
	api.LogDebug(respCode)

	/*
		// It's possible to kick off a goroutine here.
		// But it's unsafe to access the f.callbacks because the FilterCallbackHandler
		// may be already released when the goroutine is scheduled.
		go func() {
			defer func() {
				if p := recover(); p != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					fmt.Printf("http: panic serving: %v\n%s", p, buf)
				}
			}()

			// do time-consuming jobs
		}()
	*/
}

// OnLogDownstreamStart is called when HTTP Connection Manager filter receives a new HTTP request
// (required the corresponding access log type is enabled)
func (f *filter) OnLogDownstreamStart(reqHeader api.RequestHeaderMap) {
	// also support kicking off a goroutine here, like OnLog.
}

// OnLogDownstreamPeriodic is called on any HTTP Connection Manager periodic log record
// (required the corresponding access log type is enabled)
func (f *filter) OnLogDownstreamPeriodic(reqHeader api.RequestHeaderMap, reqTrailer api.RequestTrailerMap, respHeader api.ResponseHeaderMap, respTrailer api.ResponseTrailerMap) {
	// also support kicking off a goroutine here, like OnLog.
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
	// One should not access f.callbacks here because the FilterCallbackHandler
	// is released. But we can still access other Go fields in the filter f.

	// goroutine can be used everywhere.
}

// Function to call the API and return the raw response as a string
func fetchAPIResponse() (string, error) {
	// Define the URL for the API endpoint
	url := "https://672c3c071600dda5a9f7a506.mockapi.io/test/users"
	url = "https://www.google.com"

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Error fetching data: %v", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error: Received non-OK HTTP status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %v", err)
	}

	// Return the response as a string
	return string(body), nil
}

func call_redis() {
	var ctx = context.Background()
	// Set up Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // Redis server address (hostname:port)
		Password: "",            // No password
		DB:       0,             // Default DB
	})

	// Ping Redis to check if connection is successful
	resp, err := rdb.Ping(ctx).Result()
	if err != nil {
		api.LogInfof("Could not connect to Redis: %v", err)
	}

	// Print response from Redis
	api.LogInfof("Response from Redis:", resp)
}