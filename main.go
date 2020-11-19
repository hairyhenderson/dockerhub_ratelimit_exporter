package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hairyhenderson/dockerhub_ratelimit_exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

const prog = "dockerhub_ratelimit_exporter"

func main() {
	exitCode := 0

	defer func() { os.Exit(exitCode) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := &cobra.Command{
		Use:     prog,
		Short:   "A Prometheus-format exporter to report on DockerHub per-image rate limits",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddress, _ := cmd.Flags().GetString("web.listen-address")

			log.Infof("Starting %s %s", prog, cmd.Version)

			hc := &http.Client{}

			http.Handle("/metrics", promhttp.Handler())
			http.Handle("/limits", limitsHandler(hc))

			log.Infoln("Listening on", listenAddress)

			return http.ListenAndServe(listenAddress, nil)
		},
	}

	cmd.Flags().String("web.listen-address", ":9764", "Address at which to listen")

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)

		exitCode = -1
	}
}

func createGaugeVec(ns, name, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: ns,
		Name:      name,
		Help:      help,
	}, labels)
}

func limitsHandler(hc *http.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		images := []string{}
		for k, vs := range r.URL.Query() {
			if k == "image" {
				images = append(images, vs...)

				break
			}
		}

		if len(images) == 0 {
			http.Error(w, "image parameter is missing", http.StatusBadRequest)

			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)

		registry := prometheus.NewRegistry()
		ns := "dockerhub_ratelimits"
		limit := createGaugeVec(ns, "limit", "total number of pulls that can be performed during the window", []string{"image"})
		remaining := createGaugeVec(ns, "remaining", "number of pulls remaining for the window", []string{"image"})
		window := createGaugeVec(ns, "window_seconds", "the length of the time window", []string{"image"})

		registry.MustRegister(limit, remaining, window)

		for _, image := range images {
			li, err := check(ctx, hc, image)
			if err != nil {
				http.Error(w, "failed to check limits", http.StatusBadGateway)

				return
			}

			l := prometheus.Labels{"image": image}
			limit.With(l).Set(float64(li.limit))
			remaining.With(l).Set(float64(li.remaining))
			window.With(l).Set(li.window.Seconds())
		}

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
}

type limitInfo struct {
	limit     int
	remaining int
	window    time.Duration
}

func check(ctx context.Context, hc *http.Client, img string) (l limitInfo, err error) {
	parts := strings.SplitN(img, ":", 2)
	image, tag := parts[0], "latest"

	if len(parts) > 1 {
		tag = parts[1]
	}

	if !strings.Contains(image, "/") {
		image = fmt.Sprintf("library/%s", image)
	}

	imageURL := fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", image, tag)

	tok, err := authenticate(ctx, hc, image)
	if err != nil {
		return l, fmt.Errorf("failed to authenticate: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, imageURL, nil)
	if err != nil {
		return l, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok))

	res, err := hc.Do(req)
	if err != nil {
		return l, fmt.Errorf("failed to HEAD: %w", err)
	}

	defer res.Body.Close()
	b, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return l, fmt.Errorf("HTTP status %d: %q", res.StatusCode, string(b))
	}

	limit := res.Header.Get("Ratelimit-Limit")
	remaining := res.Header.Get("Ratelimit-Remaining")

	l.limit, _ = parseHeader(limit)
	l.remaining, l.window = parseHeader(remaining)

	return l, nil
}

func parseHeader(s string) (int, time.Duration) {
	parts := strings.SplitN(s, ";w=", 2)

	num, _ := strconv.Atoi(parts[0])
	dur := time.Duration(0)

	if len(parts) > 1 {
		i, _ := strconv.Atoi(parts[1])
		dur = time.Duration(i) * time.Second
	}

	return num, dur
}

func authenticate(ctx context.Context, hc *http.Client, image string) (tok string, err error) {
	tokenURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", image)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}

	res, err := hc.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	r := &response{}
	dec := json.NewDecoder(res.Body)

	err = dec.Decode(r)
	if err != nil {
		return "", err
	}

	return r.Token, nil
}

type response struct {
	Token       string    `json:"token"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	IssuedAt    time.Time `json:"issued_at"`
}
