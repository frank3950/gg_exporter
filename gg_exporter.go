package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/frank3950/cthun"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type exporter struct {
	gg_lag_at_chkpt_seconds     *prometheus.Desc
	gg_time_since_chkpt_seconds *prometheus.Desc
	gg_dirdat_bytes             *prometheus.Desc
}

func new() *exporter {
	return &exporter{
		gg_lag_at_chkpt_seconds: prometheus.NewDesc(
			"gg_lag_at_chkpt_seconds",
			"Lag at Chkpt",
			[]string{"name"},
			nil,
		),
		gg_time_since_chkpt_seconds: prometheus.NewDesc(
			"gg_time_since_chkpt_seconds",
			"Time Since Chkpt",
			[]string{"name"},
			nil,
		),
		gg_dirdat_bytes: prometheus.NewDesc(
			"gg_dirdat_bytes",
			"Size of dirdat firectory",
			nil,
			nil,
		),
	}
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.gg_lag_at_chkpt_seconds
	ch <- e.gg_time_since_chkpt_seconds
	ch <- e.gg_dirdat_bytes
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	i := cthun.ClassicGG{Home: *home}
	cthun.SetupGG(&i)
	m1, m2 := cthun.GetGGLag(i)

	go func() {
		wg.Add(1)
		defer wg.Done()
		s, err := cthun.GetGGDatSize(i)
		if err != nil {
			fmt.Println(err)
		}
		ch <- prometheus.MustNewConstMetric(
			e.gg_dirdat_bytes,
			prometheus.GaugeValue,
			float64(s),
		)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		for k, v := range m1 {
			ch <- prometheus.MustNewConstMetric(
				e.gg_lag_at_chkpt_seconds,
				prometheus.GaugeValue,
				float64(v),
				k,
			)
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		for k, v := range m2 {
			ch <- prometheus.MustNewConstMetric(
				e.gg_time_since_chkpt_seconds,
				prometheus.GaugeValue,
				float64(v),
				k,
			)
		}
	}()

	wg.Wait()
}

var (
	//命令行参数
	help bool //是否打印帮助信息
	//定义 Exporter 的版本（Version）、监听地址（listenAddress）、采集 url（metricPath）以及⾸⻚（landingPage）
	Version = "1.0.0"
	home    = flag.String("home", os.Getenv("OGG_HOME"), "Specify the ggs home directory, default $OGG_HOME")
	//端口规范https://github.com/prometheus/prometheus/wiki/Default-port-allocations
	listenAddress = flag.String("web.listen-address", ":9811", "Address to listen on for web interface and telemetry.")
	landingPage   = []byte("<html><head><title>GG Exporter" + Version + "</title></head><body><h1>GG Exporter " + Version + "</h1><p><a href='/metrics'>Metrics</a></p></body></html>")
)

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
	if *home == "" {
		fmt.Println("ERROR: can not find gg home. use --home or set $OGG_HOME")
		os.Exit(0)
	}
	prometheus.MustRegister(new())
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage)
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
