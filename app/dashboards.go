package app

// var dashboards struct {
// 	sync.RWMutex
// 	Dashboards []dashboard
// }

// type ServerList struct {
// 	sync.RWMutex
// 	Servers map[string]SqlServer
// }

// type dashboards struct {
//     sync.RWMutex
//     Dashboards []dashboard
// }

// type dashboard struct {
// 	Name    string
// 	Servers []string
// }

// func setDashboards() error {

// 	WinLogln("Setting Dashboards...")
// 	dashboards.Lock()
// 	defer dashboards.Unlock()

// 	var wd string
// 	var err error
// 	wd, err = osext.ExecutableFolder()
// 	if err != nil {
// 		log.Fatalln(err)
// 		return errors.Wrap(err, "executablefolder")
// 	}

// 	// Work on the config folder
// 	configdir := filepath.Join(wd, "config")
// 	if _, err = os.Stat(configdir); os.IsNotExist(err) {
// 		err = os.Mkdir(configdir, 0644)
// 	}
// 	if err != nil {
// 		return errors.Wrap(err, "configcreate")
// 	}

// 	// Work on the file
// 	configfile := filepath.Join(wd, "config", "dashboards.json")

// 	// Does the file exist
// 	if _, err := os.Stat(configfile); os.IsNotExist(err) {

// 		// does dashboards have any entries?  Add one if needed
// 		if len(dashboards.Dashboards) == 0 {
// 			a := []string{"S1", "S2", "S3"}
// 			d := dashboard{
// 				Name:    "First Three Servers",
// 				Servers: a,
// 			}
// 			dashboards.Dashboards = append(dashboards.Dashboards, d)
// 		}

// 		// Overwrite the file
// 		dashboardJSON, err := json.Marshal(dashboards.Dashboards)
// 		if err != nil {
// 			return errors.Wrap(err, "Error marshalling Json")
// 		}

// 		err = ioutil.WriteFile(configfile, dashboardJSON, 0644)
// 		if err != nil {
// 			return errors.Wrap(err, "Error writing dashboards.json")
// 		}
// 	}

// 	file, err := ioutil.ReadFile(configfile)
// 	if err != nil {
// 		return errors.Wrap(err, "readdashboardfile")
// 	}
// 	//fmt.Println("Reading: ", string(file))
// 	var dlist []dashboard
// 	json.Unmarshal(file, &dlist)

// 	// fmt.Println(dlist)

// 	dashboards.Dashboards = dlist
// 	WinLogln("Dashboards: ", len(dashboards.Dashboards))

// 	return nil

// 	// Clear out dashboards

// 	// Read the file

// 	// blow up if configuration is bad

// 	// _, err = os.Create(fullfile)
// 	// fmt.Println(err)
// 	// if err != nil {
// 	// 	log.Fatalln("Failed to write config file", err)
// 	// }

// 	//multi := io.MultiWriter(file, os.Stdout)

// 	//log.SetFlags(log.Ldate | log.Ltime)
// 	//log.SetOutput(multi)
// }
