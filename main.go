package main

import (
	"fmt"
	"github.com/alfian853/resilix-go/config"
	"github.com/alfian853/resilix-go/resilix"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

func initResilix() {
	cfg := config.NewConfiguration()
	cfg.ErrorThreshold = 0.2
	cfg.SlidingWindowStrategy = config.SwStrategy_CountBased
	cfg.RetryStrategy = config.RetryStrategy_Pessimistic
	cfg.WaitDurationInOpenState = 5000
	cfg.SlidingWindowMaxSize = 10
	cfg.NumberOfRetryInHalfOpenState = 5
	cfg.MinimumCallToEvaluate = 2

	resilix.Register("foo", cfg)
	resilix.Register("bar", cfg)
}

func CallThirdPartyApi(clientId string) (result string) {

	url := ""

	if clientId == "foo" {
		url = "http://localhost:3000/foo"
	} else {
		url = "http://localhost:5000/bar"
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print(err.Error())
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err.Error())
	}

	return fmt.Sprintf("%+v", string(bodyBytes))
}

func GetTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func main() {
	router := gin.Default()
	args := os.Args
	initResilix()

	options := []string{"foo", "bar"}

	router.GET("/resilix", func(c *gin.Context) {
		time := GetTimestamp()
		for _, option := range options {

			executed, result, err := resilix.Go(option).ExecuteSupplier(func() interface{} {
				return CallThirdPartyApi(option) // return response
			})

			if err != nil {
				// deliberately return error response
				c.String(http.StatusInternalServerError, "%d: %s is down!", time, option)
				return
			}
			if executed {
				c.String(http.StatusOK, "%d: %s", time, result.(string))
				return
			}
		}

		c.String(http.StatusInternalServerError, "%d: everyone is down!", time)
	})
	fooCount := new(int32)
	*fooCount = 0
	router.GET("/foo", func(c *gin.Context) {
		c.String(http.StatusOK, "foo-%d", atomic.AddInt32(fooCount, 1))
	})

	barCount := new(int32)
	*barCount = 0
	router.GET("/bar", func(c *gin.Context) {
		c.String(http.StatusOK, "bar-%d", atomic.AddInt32(barCount, 1))

	})

	router.Run("localhost:" + args[1])
}

