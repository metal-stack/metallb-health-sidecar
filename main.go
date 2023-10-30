package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	metricsEndpoint    = "http://localhost:7472/metrics"
	configStaleMetric  = "metallb_k8s_client_config_stale_bool"
	configLoadedMetric = "metallb_k8s_client_config_loaded_bool"
)

type metrics struct {
	configStale  bool
	configLoaded bool
}

func main() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
	log := slog.New(jsonHandler)

	k8s, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(fmt.Errorf("unable to create kubernetes client: %w", err))
	}

	s := gocron.NewScheduler(time.UTC)

	_, err = s.Every(30).Seconds().Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := getMetrics(ctx)
		if err != nil {
			log.Error("unable to get metrics", "error", err)
			return
		}

		log.Info("retrieved metrics", "stale", res.configStale, "loaded", res.configLoaded)

		health := corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "health",
				Namespace: "metallb-system",
			},
		}
		_, err = controllerutil.CreateOrUpdate(ctx, k8s, &health, func() error {
			health.Data["configStale"] = strconv.FormatBool(res.configStale)
			health.Data["configLoaded"] = strconv.FormatBool(res.configLoaded)
			return nil
		})
		if err != nil {
			log.Error("unable to write to health config map", "error", err)
			return
		}

		log.Info("successfully wrote health to config map")
	})
	if err != nil {
		panic(err)
	}

	s.StartBlocking()
}

func getMetrics(ctx context.Context) (*metrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body of metrics endpoint: %w", err)
	}

	var (
		lines        = map[string]string{}
		configStale  bool
		configLoaded bool
	)

	for _, line := range strings.Split(string(raw), "\n") {
		line := strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}

		lines[key] = value
	}

	readBoolMetric := func(metric string) (bool, error) {
		val, ok := lines[metric]
		if !ok {
			return false, fmt.Errorf("metrics not found in response: %q", metric)
		}

		res, err := strconv.ParseBool(val)
		if err != nil {
			return false, fmt.Errorf("unable to parse bool: %w", err)
		}

		return res, nil
	}

	configLoaded, err = readBoolMetric(configLoadedMetric)
	if err != nil {
		return nil, err
	}

	configStale, err = readBoolMetric(configStaleMetric)
	if err != nil {
		return nil, err
	}

	return &metrics{
		configStale:  configStale,
		configLoaded: configLoaded,
	}, nil
}
