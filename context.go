package stats

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MediaMath/govent/graphite"
	"gopkg.in/alexcesaro/statsd.v2"

	"golang.org/x/net/context"
)

type key int

const (
	prefixKey key = iota
	statsdURLKey
	runtimeIntervalKey
	graphiteURLKey
	graphiteUserKey
	graphitePasswordKey
	graphiteVerboseKey
)

//SetPrefix sets the stats prefix
func SetPrefix(ctx context.Context, prefix string) context.Context {
	return context.WithValue(ctx, prefixKey, prefix)
}

//GetPrefix gets the prefix
func GetPrefix(ctx context.Context) string {
	return getString(ctx, prefixKey, "")
}

//SetStatsdURL sets the stats prefix
func SetStatsdURL(ctx context.Context, url string) context.Context {
	return context.WithValue(ctx, statsdURLKey, url)
}

//SetGraphite sets the graphite client
func SetGraphite(ctx context.Context, url, user, password string, verbose bool) context.Context {
	ctx = context.WithValue(ctx, graphiteURLKey, url)
	ctx = context.WithValue(ctx, graphiteUserKey, user)
	ctx = context.WithValue(ctx, graphitePasswordKey, password)
	ctx = context.WithValue(ctx, graphiteVerboseKey, verbose)

	return ctx
}

//SetRuntimeInterval sets the runtime stats collector interval
func SetRuntimeInterval(ctx context.Context, interval time.Duration) context.Context {
	return context.WithValue(ctx, runtimeIntervalKey, interval)
}

//HasStats checks if the statsd url and graphite url are set
func HasStats(ctx context.Context) (hasStatsdURL bool, hasGraphiteURL bool) {
	statsdURL := getString(ctx, statsdURLKey, "")
	graphiteURL := getString(ctx, graphiteURLKey, "")

	return statsdURL != "", graphiteURL != ""
}

//RegisterStatsContext starts statsd and graphite based on the context
func RegisterStatsContext(ctx context.Context) error {
	prefix := GetPrefix(ctx)
	if prefix == "" {
		return fmt.Errorf("No prefix not starting stats consumers")
	}

	statsdURL := getString(ctx, statsdURLKey, "")
	if statsdURL == "" {
		return fmt.Errorf("No statsd URL not starting stats consumers")
	}

	graphiteURL := getString(ctx, graphiteURLKey, "")
	if graphiteURL == "" {
		return fmt.Errorf("No graphite URL not starting stats consumers")
	}

	log.Printf("Register statsd: %v %v", statsdURL, prefix)
	s, err := statsd.New(statsd.Address(statsdURL), statsd.Prefix(prefix))
	if err != nil {
		return err
	}

	go StartStatsd(ctx, DefaultBroker, s)

	graphiteUser := getString(ctx, graphiteUserKey, "")
	graphitePassword := getString(ctx, graphitePasswordKey, "")
	graphiteVerbose, _ := ctx.Value(graphiteVerboseKey).(bool)

	govent := &graphite.Graphite{
		Username: graphiteUser,
		Password: graphitePassword,
		Addr:     graphiteURL,
		Client:   &http.Client{Timeout: time.Second * 10},
		Verbose:  graphiteVerbose,
		Prefix:   GetPrefix(ctx),
	}

	log.Printf("Starting graphite %v %v", govent.Username, govent.Addr)
	go StartGraphite(ctx, DefaultBroker, govent)

	return nil
}

//RegisterRuntimeStatsContext starts runtime stats reporting based on the context
func RegisterRuntimeStatsContext(ctx context.Context) error {
	interval, has := ctx.Value(runtimeIntervalKey).(time.Duration)
	if !has {
		return fmt.Errorf("No runtime interval not reporting runtime stats")
	}

	return ReportRuntimeStats(ctx, interval)
}

func getString(ctx context.Context, key key, def string) string {
	val, has := ctx.Value(key).(string)
	if !has {
		return def
	}

	return val
}
