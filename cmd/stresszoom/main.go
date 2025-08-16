package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

func main() {
	var url string
	var rate, conns, bodySz, workers int
	var dur time.Duration
	var token string
	flag.StringVar(&url, "url", "http://127.0.0.1:8080/webhook/zoom", "target URL")
	flag.IntVar(&rate, "rate", 3000, "requests per second")
	flag.IntVar(&conns, "conns", 800, "max parallel connections")
	flag.IntVar(&bodySz, "body", 512, "payload bytes")
	flag.DurationVar(&dur, "duration", 60*time.Second, "duration")
	flag.IntVar(&workers, "workers", runtime.NumCPU()*2, "workers")
	flag.StringVar(&token, "token", os.Getenv("ZOOM_WEBHOOK_SECRET_TOKEN"), "zoom secret (env by default)")
	flag.Parse()
	if token == "" { log.Fatal("set ZOOM_WEBHOOK_SECRET_TOKEN or -token") }

	body := bytes.Repeat([]byte("a"), bodySz)
	tr := &http.Transport{
		MaxIdleConns: conns,
		MaxConnsPerHost: conns,
		MaxIdleConnsPerHost: conns,
		DisableCompression: true,
		IdleConnTimeout: 30*time.Second,
		DialContext: (&net.Dialer{Timeout: 5*time.Second, KeepAlive: 30*time.Second}).DialContext,
	}
	client := &http.Client{Transport: tr, Timeout: 3*time.Second}

	var sent, ok, bad, terr uint64
	tick := time.NewTicker(time.Millisecond); defer tick.Stop()
	end := time.Now().Add(dur)
	work := make(chan struct{}, workers*2)
	for i:=0;i<workers;i++ {
		go func(){
			for range work {
				ts := strconv.FormatInt(time.Now().Unix(), 10)
				msg := []byte("v0:"+ts+":"+string(body))
				h := hmac.New(sha256.New, []byte(token))
				h.Write(msg)
				sum := make([]byte, hex.EncodedLen(h.Size())); hex.Encode(sum, h.Sum(nil))
				sig := "v0="+string(sum)
				req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
				req.Header.Set("Content-Type","application/json")
				req.Header.Set("x-zm-request-timestamp", ts)
				req.Header.Set("x-zm-signature", sig)
				resp, err := client.Do(req)
				atomic.AddUint64(&sent,1)
				if err != nil { atomic.AddUint64(&terr,1); continue }
				_, _ = io.Copy(io.Discard, resp.Body); resp.Body.Close()
				if resp.StatusCode == 202 || resp.StatusCode == 200 || resp.StatusCode == 204 {
					atomic.AddUint64(&ok,1)
				} else {
					atomic.AddUint64(&bad,1)
				}
			}
		}()
	}
	fmt.Printf("attack %s for %s @ %d rps (conns=%d)\n", url, dur, rate, conns)
loop:
	for {
		select {
		case <-tick.C:
			if time.Now().After(end) { break loop }
			perMs := rate // simplistic; rate is per sec, but we send per ms bursts of 'rate/1000' below
			if perMs > 1000 { perMs = perMs // keep burst size sane for laptop
			}
			for i:=0;i<rate//1000;i++ { work <- struct{}{} }
		}
	}
	close(work)
	time.Sleep(2*time.Second)
	s := atomic.LoadUint64(&sent); o := atomic.LoadUint64(&ok); b := atomic.LoadUint64(&bad); e := atomic.LoadUint64(&terr)
	fmt.Printf("sent=%d ok=%d bad=%d transport_err=%d\n", s,o,b,e)
