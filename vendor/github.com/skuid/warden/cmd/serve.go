package cmd

import (
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/skuid/picard"
	"github.com/skuid/spec/lifecycle"
	_ "github.com/skuid/spec/metrics" // spec metrics setup
	"github.com/skuid/spec/middlewares"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/api/v1"
	"github.com/skuid/warden/pkg/api/v2"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/version"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the server",
	Run:   serve,
}

func init() {
	zap.L().Info("Init serve")
	RootCmd.AddCommand(serveCmd)

	serveCmd.PersistentFlags().String("host", "0.0.0.0", "Warden will serve requests on this host")
	serveCmd.PersistentFlags().IntP("port", "p", 3000, "Warden will serve requests on this port")
	serveCmd.PersistentFlags().Duration("read_timeout", 5*time.Second, "Time limit for reading requests")
	serveCmd.PersistentFlags().Duration("write_timeout", 5*time.Second, "Time limit for writing responses")
	serveCmd.PersistentFlags().String("pliny_address", "", "Pliny Host")

	serveCmd.PersistentFlags().Bool("tls_enabled", false, "Enable serving over TLS")
	serveCmd.PersistentFlags().String("tls_cert_file", "", "TLS Cert File")
	serveCmd.PersistentFlags().String("tls_key_file", "", "TLS Key File")

	serveCmd.PersistentFlags().Duration("client_timeout", 20*time.Second, "Timeout on the client connection to the proxy/seaquill. Set to -1 to turn off.")
	serveCmd.PersistentFlags().Duration("authcache_ttl", 0, "TTL for a user in the auth cache. If the value is zero, no caching will be used")
	serveCmd.PersistentFlags().Bool("gzip_load_response", false, "Do you want the load route's response from warden to be gzipped?")
	serveCmd.PersistentFlags().Bool("gzip_load_proxy", false, "Do you want the load route's response from warden to be gzipped?")
	serveCmd.PersistentFlags().Bool("stream", false, "Whether or not to stream the load response")
	serveCmd.PersistentFlags().Bool("xtream", false, "If stream = true, and xtream = true, then you get an extreme stream")
	if err := viper.BindPFlags(serveCmd.PersistentFlags()); err != nil {
		zap.L().Error("encountered an error on viper flag binding", zap.Error(err))
		os.Exit(1)
	}
	viper.AutomaticEnv()
}

// This is necessary because of a bug in viper / cobra that considers all
// variables that can be set with flags to be marked as set
func isExistingConfig(configName string) bool {
	return viper.IsSet(configName) && viper.GetString(configName) != ""
}

func getCORSProtectedHandler(router http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE"}),
		handlers.AllowedHeaders([]string{
			// client sends the following headers with all requests
			"x-skuid-auth-url",
			"x-skuid-public-key-endpoint",
			"x-skuid-session-id",
			"x-skuid-data-source",
			"x-skuid-options-schemas",
			"authorization",
			"content-type",
			// nginx adds the following headers on proxy pass
			"x-forwarded-for",
			"x-frame-options",
			"strict-transport-security",
			"host",
			"x-real-ip",
		}),
	)(router)
}

func notFound(isAPI bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := errors.ErrNotFound
		if isAPI {
			err = errors.WrapError(err, "NotFound", nil, "")
		}
		api.RespondNotFound(w, err)
	}
}

func checkRequiredConfs(required []string) {
	sugar := zap.L().Sugar()
	defer sugar.Sync()
	for _, envVar := range required {
		if !isExistingConfig(envVar) {
			sugar.Fatalf("STARTUP ERROR: '%s' configuration must be provided", envVar)
		}
	}
}

func serve(cmd *cobra.Command, args []string) {

	checkRequiredConfs([]string{
		"quill_address",
	})

	if viper.GetBool("tls_enabled") {
		checkRequiredConfs([]string{
			"tls_cert_file",
			"tls_key_file",
		})
	}

	listenHostPort := viper.GetString("host") + ":" + viper.GetString("port")

	apiRouter := mux.NewRouter()

	middlewareAPIHandler := middlewares.Apply(
		apiRouter,
		middlewares.InstrumentRoute(),
		middlewares.Logging(),
		middlewares.AddHeaders(map[string]string{
			"X-Frame-Options": "DENY",
			"content-type":    "application/json",
		}),
	)

	apiRouter.NotFoundHandler = http.HandlerFunc(notFound(true))

	v1.AddRoutes(apiRouter)
	v2.AddRoutes(apiRouter)

	router := mux.NewRouter()
	router.Handle("/api/{_:.*}", middlewareAPIHandler)
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/live", lifecycle.LivenessHandler)
	router.HandleFunc("/ready", lifecycle.ReadinessHandler)
	router.NotFoundHandler = http.HandlerFunc(notFound(false))

	profileMode := viper.GetBool("pprof")

	if profileMode {
		router.HandleFunc("/debug/pprof/", pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	handler := getCORSProtectedHandler(router)

	server := &http.Server{
		Addr:         listenHostPort,
		Handler:      handler,
		ReadTimeout:  viper.GetDuration("read_timeout"),
		WriteTimeout: viper.GetDuration("write_timeout"),
	}
	server.RegisterOnShutdown(picard.CloseConnection)
	lifecycle.ShutdownOnTerm(server)

	if viper.GetBool("tls_enabled") {
		zap.L().Sugar().Infof("Warden version %s protecting data over SSL at %s", version.Name, listenHostPort)
		if err := server.ListenAndServeTLS(viper.GetString("tls_cert_file"), viper.GetString("tls_key_file")); err != http.ErrServerClosed {
			zap.L().Fatal("encountered an error while serving over ssl", zap.Error(err))
		}
		zap.L().Info("Server gracefully stopped by sigterm")
	} else {
		zap.L().Sugar().Infof("Warden version %s protecting data at %s", version.Name, listenHostPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			zap.L().Fatal("encountered an error while serving normal http", zap.Error(err))
		}
		zap.L().Info("Server gracefully stopped by sigterm")
	}

}
