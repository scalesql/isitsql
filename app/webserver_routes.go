package app

import (
	"log"
	"net/http"
	"time"

	/* #nosec G108 - is conditionally bound to localhost below */
	_ "net/http/pprof"
	"strconv"

	"github.com/arl/statsviz"
	"github.com/go-pkgz/routegroup"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/internal/settings"
	"github.com/scalesql/isitsql/static"
)

func launchWebServer() {
	defer failure.HandlePanic()

	WinLogln("Launching Web Server...")

	s, err := settings.ReadConfig()
	if err != nil {
		WinLogln("readconfig", err)
		log.Fatal(err)
	}

	globalConfig.RLock()
	useLocal := globalConfig.AppConfig.UseLocalStatic
	globalConfig.RUnlock()

	group := routegroup.New(http.NewServeMux())
	group.DisableNotFoundHandler()
	group.Use(panicHandler)
	group.Use(prometheusMiddleware)

	//router := httprouter.New()

	//router.ServeFiles("/static/*filepath", http.Dir(path.Join(wd, "static")))
	//router.ServeFiles("/static/", http.FileServer(FS(false)))
	// router.ServeFiles("/static/",http.FileServer(FS(false)) )
	//router.ServeFiles("/static/*filepath", FS(useLocal))

	// This one works on a pure embed.FS
	//group.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static.FS)))

	// https://www.brandur.org/fragments/go-embed
	group.Handle("/static/", http.StripPrefix("/static/", http.FileServer(static.HttpFS(useLocal))))

	group.HandleFunc("GET /{$}", Home)
	group.HandleFunc("GET /connections", ConnectionsPage)
	group.HandleFunc("GET /infographic", infoGraphicPage)
	group.HandleFunc("GET /tag/{tag}", tagPage)
	group.HandleFunc("GET /user-tags", userTagPage)
	group.HandleFunc("GET /auto-tags", systemTagPage)
	group.HandleFunc("GET /log", logPage)
	group.HandleFunc("GET /info", infoPage)
	group.HandleFunc("GET /about", aboutPage)
	group.HandleFunc("GET /ips", ipPage)
	group.HandleFunc("GET /versions", versionPage)
	group.HandleFunc("GET /versions/csv", versionPageCSV)
	group.HandleFunc("GET /memory", memoryPage)
	group.HandleFunc("GET /usage", usagePage)
	group.HandleFunc("GET /usage/csv", usagePageCSV)

	group.HandleFunc("GET /snapshots", snapshotList)
	group.HandleFunc("GET /snapshots/json", snapshotList)

	group.HandleFunc("GET /slugs", slugsPage)
	group.HandleFunc("GET /server/{server}", serverPage)
	group.HandleFunc("GET /newchart/{server}", newchartPage)
	group.HandleFunc("GET /server/{server}/w2", serverW2Page)
	group.HandleFunc("GET /server/{server}/w2/diag", dynamicWaitsDiagnostic)
	group.HandleFunc("GET /server/{server}/waits", ServerWaitPage)
	group.HandleFunc("GET /server/{server}/info", ServerInfoPage)
	group.HandleFunc("GET /server/{server}/raw", serverRawPage)
	group.HandleFunc("GET /server/{server}/json", serverJSONPage)
	group.HandleFunc("GET /server/{server}/databases", serverDatabasesPage)

	group.HandleFunc("GET /server/{server}/jobs/all", ServerJobsPage)
	group.HandleFunc("GET /server/{server}/jobs/active", ServerJobsActivePage)
	group.HandleFunc("GET /server/{server}/jobs/{jobid}/history", ServerJobHistoryPage)
	group.HandleFunc("GET /server/{server}/jobs/{jobid}/history/{instanceid}", ServerJobMessagesPage)
	group.HandleFunc("GET /server/{server}/jobs/{jobid}/steplog", ServerJobStepLogPage)
	group.HandleFunc("GET /jobs", AgentJobsPage)

	group.HandleFunc("GET /server/{server}/qs", serverQueryStats)
	group.HandleFunc("GET /server/{server}/xe", serverXEPage)
	group.HandleFunc("GET /server/{server}/conn", serverConnPage)

	//group.HandleFunc("GET /api/", ApiTest)
	group.HandleFunc("GET /api/cpu/{server}", ApiCpu)
	group.HandleFunc("GET /api/disk/{server}", ApiDisk)
	//group.HandleFunc("GET /api2/", ApiDates)
	//group.HandleFunc("GET /apiall/", ApiAll)
	group.HandleFunc("GET /api/waits/{server}", APIServerWaits)
	group.HandleFunc("GET /api/waits2/{server}", APIServerWaits2)

	//group.HandleFunc("GET /hello/{server}", ApiServerJson)
	group.HandleFunc("GET /dashboard/{servers...}", dashboardPage)
	group.HandleFunc("GET /mirroring", mirroredDatabasesPage)
	group.HandleFunc("GET /ag", agPage)
	group.HandleFunc("GET /ag/json", agPage)
	group.HandleFunc("GET /backups", allBackupsPage)
	group.HandleFunc("GET /backups/json", allBackupsPage)

	group.HandleFunc("GET /login", loginPage)
	group.HandleFunc("POST /login", loginPage)
	group.HandleFunc("GET /logout", logoutPage)

	group.HandleFunc("GET /polling", pollingPage)

	// Settings pages
	group.HandleFunc("GET /settings", settingsPage)
	group.HandleFunc("POST /settings", settingsPage)

	// group.HandleFunc("GET /settings/advanced", wrapHTTPErrorHandling(settingsAdvancedPage))
	// group.HandleFunc("POST /settings/advanced", wrapHTTPErrorHandling(settingsAdvancedPage))

	group.HandleFunc("GET /settings/credentials", credentialListPage)
	group.HandleFunc("GET /settings/credentials/add", credentialAddPage)
	group.HandleFunc("POST /settings/credentials/add", credentialAddPage)
	group.HandleFunc("GET /settings/credentials/edit/{credential}", credentialEditPage)
	group.HandleFunc("POST /settings/credentials/edit/{credential}", credentialEditPage)
	group.HandleFunc("GET /settings/credentials/delete/{credential}", credentialDeletePage)
	group.HandleFunc("POST /settings/credentials/delete/{credential}", credentialDeletePage)

	group.HandleFunc("GET /settings/servers", serverListPage)
	group.HandleFunc("GET /settings/servers/add", serverAddPage)
	group.HandleFunc("POST /settings/servers/add", serverAddPage)
	group.HandleFunc("GET /settings/servers/edit/{server}", serverEditPage)
	group.HandleFunc("POST /settings/servers/edit/{server}", serverEditPage)
	group.HandleFunc("GET /settings/servers/delete/{server}", serverDeletePage)
	group.HandleFunc("POST /settings/servers/delete/{server}", serverDeletePage)

	group.HandleFunc("GET /settings/conns", connListPage)
	group.Handle("GET /metrics/isitsql", promhttp.Handler())
	// /metrics/mssql

	//group.HandleFunc("GET /panic", wrapHTTPErrorHandling(panicApp))
	//group.HandleFunc("GET /race", wrapHTTPErrorHandling(racePage))

	if getGlobalConfig().EnableStatsviz {
		svmux := http.NewServeMux()

		// Register Statsviz server on the mux, serving the user interface from
		// /foo/bar instead of /debug/statsviz and send metrics every 250
		// milliseconds instead of the default of once per second.
		err = statsviz.Register(svmux,
			statsviz.Root("/debug/statsviz"),
			// statsviz.SendFrequency(10*time.Second),
		)
		if err != nil {
			WinLogln(errors.Wrap(err, "statsviz.register"))
		} else {
			go func() {
				WinLogln("runtime metrics at http://localhost:8092/debug/statsviz ")
				log.Fatal(http.ListenAndServe(":8092", svmux))
			}()
		}
	}

	port := strconv.Itoa(s.Port)
	hostName := ":" + port

	go func() {
		//time.Sleep(250 * time.Millisecond)
		if s.Port == 80 {
			WinLogln("View the site at http://localhost ")
		} else {
			WinLogln("View the site on http://localhost:" + port + " ")
		}
	}()

	// setup a server to we can set timeouts
	srv := &http.Server{
		Addr:              hostName,
		Handler:           group,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       300 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	//err = http.ListenAndServe(hostName, router)
	err = srv.ListenAndServe()
	if err != nil {
		errMsg := err.Error()
		WinLogln(errMsg)
		log.Fatal("LaunchWebServer => ", err)
	}
}
